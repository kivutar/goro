package res

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestManagerFSArchivePriorityLooseOverridesAndAliases(t *testing.T) {
	root := t.TempDir()
	for name, value := range map[string]string{"base.grf": "base", "patch.grf": "patch"} {
		if err := writeTestGRF(filepath.Join(root, name), `data\texture\target.bmp`, []byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "Data"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"DATA.INI":              "[Data]\n0=PATCH.GRF\n1=base.grf\n",
		"Data/ResNameTable.txt": "source.bmp#target.bmp#\n",
		"Data/clientinfo.xml":   "<clientinfo><servicetype>korea</servicetype><connection><display>Selected folder</display><address>192.0.2.10</address><port>6900</port></connection></clientinfo>",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Use a virtual root: successful reads cannot accidentally use native paths.
	manager, err := NewManagerFS(os.DirFS(root))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, archive := range manager.Archives {
			_ = archive.Close()
		}
	})
	if manager.ClientInfo.Connections[0].Address != "192.0.2.10" {
		t.Fatal("selected folder's clientinfo.xml was not loaded")
	}
	for _, name := range []string{`data\texture\target.bmp`, `data\texture\source.bmp`} {
		got, err := manager.ReadFileExact(name)
		if err != nil || string(got) != "patch" {
			t.Fatalf("read %q = %q, %v", name, got, err)
		}
	}
	if err := os.Mkdir(filepath.Join(root, "Data", "Texture"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Data", "Texture", "Target.bmp"), []byte("loose"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := manager.ReadFileExact(`data\texture\source.bmp`)
	if err != nil || string(got) != "loose" {
		t.Fatalf("loose alias override = %q, %v", got, err)
	}
}

func TestManagerFSRejectsArchiveOutsideGrantedFolder(t *testing.T) {
	for _, name := range []string{"../outside.grf", "/outside.grf"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "DATA.INI"), []byte("[Data]\n0="+name+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if manager, err := NewManagerFS(os.DirFS(root)); err == nil {
				for _, archive := range manager.Archives {
					_ = archive.Close()
				}
				t.Fatal("accepted an archive outside the granted root")
			}
		})
	}
}

type streamOnlyFS struct {
	fs.FS
	closed bool
}

type streamOnlyFile struct {
	fs.File // Deliberately does not expose io.ReaderAt.
	owner   *streamOnlyFS
}

func (s *streamOnlyFS) Open(name string) (fs.File, error) {
	file, err := s.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return &streamOnlyFile{File: file, owner: s}, nil
}

func (s *streamOnlyFile) Close() error {
	s.owner.closed = true
	return s.File.Close()
}

func TestManagerFSClosesUnseekableArchive(t *testing.T) {
	root := t.TempDir()
	if err := writeTestGRF(filepath.Join(root, "data.grf"), "test.txt", []byte("test")); err != nil {
		t.Fatal(err)
	}
	source := &streamOnlyFS{FS: os.DirFS(root)}
	manager := &Manager{Root: ".", files: source}
	if _, err := manager.openArchive("data.grf"); err == nil {
		t.Fatal("accepted an archive without random access")
	}
	if !source.closed {
		t.Fatal("failed archive leaked its descriptor")
	}
}
