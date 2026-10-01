package res

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const (
	grfHeaderSize = 46
	grfVersion200 = 0x200
	grfVersion300 = 0x300
)

var (
	ErrGRFUnsupportedEncryption = errors.New("grf entry uses unsupported encryption")
	ErrGRFUnsupportedVersion    = errors.New("unsupported grf version")
	ErrGRFNotFound              = errors.New("grf entry not found")
)

type GRF struct {
	path    string
	file    grfFile
	version uint32
	entries map[string]GRFEntry
}

type grfFile interface {
	io.Reader
	io.ReaderAt
	io.Closer
}

type GRFEntry struct {
	Name        string
	PackedSize  uint32
	AlignedSize uint32
	RealSize    uint32
	Type        byte
	Offset      uint64
}

func OpenGRF(path string) (*GRF, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return openGRFFile(path, file)
}

func openGRFFile(path string, file grfFile) (*GRF, error) {
	grf := &GRF{
		path:    path,
		file:    file,
		entries: make(map[string]GRFEntry),
	}
	if err := grf.load(); err != nil {
		_ = file.Close()
		return nil, err
	}
	return grf, nil
}

func (m *Manager) openArchive(path string) (*GRF, error) {
	if m.files == nil {
		return OpenGRF(path)
	}
	file, err := m.files.Open(filepath.ToSlash(path))
	if err != nil {
		return nil, err
	}
	reader, ok := file.(grfFile)
	if !ok {
		_ = file.Close()
		return nil, fmt.Errorf("archive %s does not support random access", path)
	}
	return openGRFFile(path, reader)
}

func (g *GRF) Close() error {
	if g.file == nil {
		return nil
	}
	return g.file.Close()
}

func (g *GRF) Path() string {
	return g.path
}

func (g *GRF) Count() int {
	return len(g.entries)
}

func (g *GRF) Has(name string) bool {
	_, ok := g.entries[normalizeGRFName(name)]
	return ok
}

func (g *GRF) Names() []string {
	names := make([]string, 0, len(g.entries))
	for _, entry := range g.entries {
		names = append(names, entry.Name)
	}
	sort.Strings(names)
	return names
}

func (g *GRF) NamesWithSuffix(suffix string) []string {
	suffix = normalizeGRFName(suffix)
	var names []string
	for key, entry := range g.entries {
		if grfPathSuffixMatch(key, suffix) {
			names = append(names, entry.Name)
		}
	}
	sort.Strings(names)
	return names
}

func grfPathSuffixMatch(name, suffix string) bool {
	if suffix == "" {
		return false
	}
	if name == suffix {
		return true
	}
	return strings.HasSuffix(name, "/"+suffix)
}

func (g *GRF) ReadFile(name string) ([]byte, error) {
	entry, ok := g.entries[normalizeGRFName(name)]
	if !ok {
		return nil, ErrGRFNotFound
	}
	raw := make([]byte, entry.AlignedSize)
	if _, err := g.file.ReadAt(raw, int64(entry.Offset+grfHeaderSize)); err != nil {
		return nil, err
	}
	if err := decryptGRFEntry(raw, entry); err != nil {
		return nil, err
	}

	if entry.Type&0x01 == 0 {
		if uint32(len(raw)) > entry.RealSize {
			raw = raw[:entry.RealSize]
		}
		return raw, nil
	}

	reader, err := zlib.NewReader(bytes.NewReader(raw[:entry.PackedSize]))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	out := bytes.NewBuffer(make([]byte, 0, entry.RealSize))
	if _, err := io.Copy(out, reader); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func decryptGRFEntry(raw []byte, entry GRFEntry) error {
	switch {
	case entry.Type&0x02 != 0:
		if shouldUseHeaderOnlyGRFDecrypt(entry.Name) {
			decryptGRFHeader(raw, entry.AlignedSize)
		} else {
			decryptGRFFull(raw, entry.AlignedSize, entry.PackedSize)
		}
	case entry.Type&0x04 != 0:
		decryptGRFHeader(raw, entry.AlignedSize)
	case entry.Type&^byte(0x07) != 0:
		return fmt.Errorf("%w: %s type=0x%02x", ErrGRFUnsupportedEncryption, entry.Name, entry.Type)
	}
	return nil
}

func shouldUseHeaderOnlyGRFDecrypt(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".gnd", ".gat", ".act", ".str":
		return true
	default:
		return false
	}
}

func (g *GRF) load() error {
	header := make([]byte, grfHeaderSize)
	if _, err := io.ReadFull(g.file, header); err != nil {
		return err
	}
	signature := strings.TrimRight(string(header[:15]), "\x00")
	if signature != "Master of Magic" && signature != "Event Horizon" {
		return fmt.Errorf("invalid grf signature %q", signature)
	}

	g.version = binary.LittleEndian.Uint32(header[42:46])
	switch g.version {
	case grfVersion200:
		tableOffset := uint64(binary.LittleEndian.Uint32(header[30:34])) + grfHeaderSize
		skip := binary.LittleEndian.Uint32(header[34:38])
		fileCountRaw := binary.LittleEndian.Uint32(header[38:42])
		if fileCountRaw < skip+7 {
			return fmt.Errorf("invalid grf file count")
		}
		return g.loadTable(tableOffset, fileCountRaw-skip-7, false)
	case grfVersion300:
		tableOffset := binary.LittleEndian.Uint64(header[30:38]) + grfHeaderSize + 4
		fileCount := binary.LittleEndian.Uint32(header[38:42])
		return g.loadTable(tableOffset, fileCount, true)
	default:
		return fmt.Errorf("%w: 0x%X", ErrGRFUnsupportedVersion, g.version)
	}
}

func (g *GRF) loadTable(offset uint64, count uint32, wideOffset bool) error {
	tableHeader := make([]byte, 8)
	if _, err := g.file.ReadAt(tableHeader, int64(offset)); err != nil {
		return err
	}
	packedSize := binary.LittleEndian.Uint32(tableHeader[:4])
	realSize := binary.LittleEndian.Uint32(tableHeader[4:8])

	packed := make([]byte, packedSize)
	if _, err := g.file.ReadAt(packed, int64(offset+8)); err != nil {
		return err
	}

	reader, err := zlib.NewReader(bytes.NewReader(packed))
	if err != nil {
		return err
	}
	defer reader.Close()

	table := bytes.NewBuffer(make([]byte, 0, realSize))
	if _, err := io.Copy(table, reader); err != nil {
		return err
	}
	return g.parseEntries(table.Bytes(), int(count), wideOffset)
}

func (g *GRF) parseEntries(table []byte, count int, wideOffset bool) error {
	pos := 0
	for i := 0; i < count; i++ {
		start := pos
		for pos < len(table) && table[pos] != 0 {
			pos++
		}
		if pos >= len(table) {
			return fmt.Errorf("unterminated grf filename at entry %d", i)
		}
		name := normalizeGRFName(decodeGRFName(table[start:pos]))
		pos++

		metaSize := 17
		if wideOffset {
			metaSize = 21
		}
		if len(table)-pos < metaSize {
			return fmt.Errorf("truncated grf metadata at entry %d", i)
		}

		entry := GRFEntry{
			Name:        name,
			PackedSize:  binary.LittleEndian.Uint32(table[pos : pos+4]),
			AlignedSize: binary.LittleEndian.Uint32(table[pos+4 : pos+8]),
			RealSize:    binary.LittleEndian.Uint32(table[pos+8 : pos+12]),
			Type:        table[pos+12],
		}
		if wideOffset {
			entry.Offset = binary.LittleEndian.Uint64(table[pos+13 : pos+21])
		} else {
			entry.Offset = uint64(binary.LittleEndian.Uint32(table[pos+13 : pos+17]))
		}
		pos += metaSize

		g.entries[name] = entry
	}
	return nil
}

func normalizeGRFName(name string) string {
	if !utf8.ValidString(name) {
		name = decodeGRFName([]byte(name))
	}
	name = strings.TrimLeft(name, `\/`)
	name = strings.ReplaceAll(name, "\\", "/")
	return strings.ToLower(name)
}

func decodeGRFName(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), data)
	if err != nil {
		return string(data)
	}
	return string(decoded)
}
