//go:build nofakecgo

package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"path/filepath"
	"strings"

	"github.com/ebitengine/oto/v3"
	mp3codec "github.com/godexture/codec-mp3"
	"github.com/godexture/core/domain/media"
	mp3format "github.com/godexture/format-mp3"
	"github.com/godexture/sdk/engine"
	"github.com/kivutar/goro/res"
)

const defaultSampleRate = 44100
const minPCM16 = -32768
const maxPCM16 = 32767

type BGM struct {
	resources  *res.Manager
	context    *oto.Context
	sampleRate int
	player     *oto.Player
	table      map[string]string
	current    string
	playerID   int
	enabled    bool
	bgmVolume  float64
	sfxVolume  float64
	sfxPlayers []*oto.Player
}

func NewBGM(resources *res.Manager, enabled bool, bgmVolume, sfxVolume float64) *BGM {
	return &BGM{
		resources: resources,
		enabled:   enabled,
		bgmVolume: clampVolume(bgmVolume),
		sfxVolume: clampVolume(sfxVolume),
	}
}

func (b *BGM) Enabled() bool {
	if b == nil {
		return false
	}
	return b.enabled
}

func (b *BGM) SetEnabled(enabled bool) {
	if b == nil || b.enabled == enabled {
		return
	}
	b.enabled = enabled
	if !enabled {
		b.Stop()
	}
}

func (b *BGM) Volume() float64 {
	return b.BGMVolume()
}

func (b *BGM) BGMVolume() float64 {
	if b == nil {
		return 0
	}
	return b.bgmVolume
}

func (b *BGM) SFXVolume() float64 {
	if b == nil {
		return 0
	}
	return b.sfxVolume
}

func (b *BGM) SetVolume(volume float64) {
	b.SetBGMVolume(volume)
	b.SetSFXVolume(volume)
}

func (b *BGM) SetBGMVolume(volume float64) {
	if b == nil {
		return
	}
	volume = clampVolume(volume)
	b.bgmVolume = volume
	if b.player != nil {
		b.player.SetVolume(volume)
	}
}

func (b *BGM) SetSFXVolume(volume float64) {
	if b == nil {
		return
	}
	volume = clampVolume(volume)
	b.sfxVolume = volume
	for _, player := range b.sfxPlayers {
		if player != nil {
			player.SetVolume(volume)
		}
	}
}

func (b *BGM) PlayMap(mapName string) (string, error) {
	if b == nil || !b.enabled {
		return "", nil
	}
	path := b.ResolveMapBGM(mapName)
	log.Printf("bgm map request map=%s resolved=%s current=%s player=%t", mapName, path, b.current, b.player != nil)
	if path == "" {
		return "", nil
	}
	if sameAudioPath(path, b.current) && b.player != nil {
		if !b.player.IsPlaying() {
			log.Printf("bgm resume existing id=%d path=%s", b.playerID, b.current)
			b.player.Play()
		} else {
			log.Printf("bgm keep existing id=%d path=%s", b.playerID, b.current)
		}
		return b.current, nil
	}
	if err := b.Play(path); err != nil {
		return path, err
	}
	return b.current, nil
}

func (b *BGM) Play(path string) error {
	if b == nil || !b.enabled {
		return nil
	}
	path = normalizeAudioPath(path)
	if path == "" {
		return nil
	}
	data, source, err := readAudioFile(b.resources, path)
	if err != nil {
		return err
	}

	pcm, sourceRate, decoder, err := decodeNativePCM(data)
	if err != nil {
		return fmt.Errorf("decode %s: %w", source, err)
	}
	context := b.ensureContext(outputSampleRateForSource(sourceRate))
	if context == nil {
		return fmt.Errorf("audio context unavailable")
	}
	if sourceRate != b.sampleRate {
		pcm, err = resamplePCM16Stereo(pcm, sourceRate, b.sampleRate)
		if err != nil {
			return fmt.Errorf("resample %s: %w", source, err)
		}
		decoder = fmt.Sprintf("%s+resample %d->%d", decoder, sourceRate, b.sampleRate)
	} else {
		decoder = fmt.Sprintf("%s native %d", decoder, sourceRate)
	}
	loop := newInfinitePCM(pcm)
	player := context.NewPlayer(loop)
	player.SetVolume(b.bgmVolume)

	b.stopCurrent()
	b.playerID++
	b.player = player
	b.current = path
	player.Play()
	log.Printf("bgm playing id=%d path=%s source=%s decoder=%s bytes=%d pcm_len=%d sample_rate=%d volume=%.2f", b.playerID, path, source, decoder, len(data), len(pcm), b.sampleRate, b.bgmVolume)
	return nil
}

func (b *BGM) PlaySFX(path string) (string, error) {
	return b.PlaySFXVolume(path, 1)
}

func (b *BGM) PlaySFXVolume(path string, volume float64) (string, error) {
	if b == nil {
		return "", nil
	}
	path = normalizeSFXPath(path)
	if path == "" {
		return "", nil
	}
	data, source, err := readSFXFile(b.resources, path)
	if err != nil {
		return "", err
	}
	context := b.ensureContext(defaultSampleRate)
	if context == nil {
		return source, fmt.Errorf("audio context unavailable")
	}
	pcm, sourceRate, err := decodeWAVPCM16Stereo(data)
	if err != nil {
		return source, fmt.Errorf("decode sfx %s: %w", source, err)
	}
	if sourceRate != b.sampleRate {
		pcm, err = resamplePCM16Stereo(pcm, sourceRate, b.sampleRate)
		if err != nil {
			return source, fmt.Errorf("resample sfx %s: %w", source, err)
		}
	}
	player := context.NewPlayer(bytes.NewReader(pcm))
	player.SetVolume(b.sfxVolume * clampVolume(volume))
	b.trimSFXPlayers()
	b.sfxPlayers = append(b.sfxPlayers, player)
	player.Play()
	return source, nil
}

func decodeNativePCM(data []byte) ([]byte, int, string, error) {
	demuxer, err := mp3format.NewDemuxerEngine(bytes.NewReader(data))
	if err != nil {
		return nil, 0, "", err
	}
	streams, _, err := demuxer.Analyze()
	if err != nil {
		return nil, 0, "", err
	}
	sampleRate := defaultSampleRate
	if len(streams) > 0 && streams[0].Audio.SampleRate > 0 {
		sampleRate = streams[0].Audio.SampleRate
	}

	decoder := mp3codec.NewDecoderEngine(mp3codec.DecoderConfig{})
	pcm, err := decodeMP3Packets(demuxer, decoder)
	if err != nil {
		return nil, 0, "", err
	}
	if len(pcm)%4 != 0 {
		return nil, 0, "", fmt.Errorf("invalid stereo pcm length %d", len(pcm))
	}
	return pcm, sampleRate, "godec/codec-mp3", nil
}

func decodeMP3Packets(demuxer engine.DemuxerEngine, decoder engine.DecoderEngine) ([]byte, error) {
	var pcm []byte
	for {
		packet, _, err := demuxer.ReadPacket()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if err := decoder.SendPacket(packet); err != nil && err != engine.ErrEAGAIN {
			return nil, err
		}
		decoded, err := receiveMP3Frames(decoder)
		if err != nil {
			return nil, err
		}
		pcm = append(pcm, decoded...)
	}
	if err := decoder.Flush(); err != nil {
		return nil, err
	}
	decoded, err := receiveMP3Frames(decoder)
	if err != nil {
		return nil, err
	}
	pcm = append(pcm, decoded...)
	if len(pcm) == 0 {
		return nil, fmt.Errorf("empty decoded MP3")
	}
	return pcm, nil
}

func receiveMP3Frames(decoder engine.DecoderEngine) ([]byte, error) {
	var pcm []byte
	for {
		frame, err := decoder.ReceiveFrame()
		if err == engine.ErrEAGAIN || err == engine.ErrEOF {
			return pcm, nil
		}
		if err != nil {
			return nil, err
		}
		audioFrame, ok := (*frame).(*media.AudioFrame)
		if !ok {
			return nil, fmt.Errorf("decoded MP3 frame is %T, want *media.AudioFrame", *frame)
		}
		pcm = appendAudioFramePCM16Stereo(pcm, audioFrame)
	}
}

func appendAudioFramePCM16Stereo(dst []byte, frame *media.AudioFrame) []byte {
	if frame == nil || len(frame.Planes()) == 0 {
		return dst
	}
	channels := frame.Layout.ChannelCount()
	if channels <= 0 {
		channels = 2
	}
	samples := frame.Samples
	plane := frame.Planes()[0]
	for sample := 0; sample < samples; sample++ {
		left := readFrameFloat32(plane, sample*channels)
		right := left
		if channels > 1 {
			right = readFrameFloat32(plane, sample*channels+1)
		}
		dst = binary.LittleEndian.AppendUint16(dst, uint16(float32ToPCM16(left)))
		dst = binary.LittleEndian.AppendUint16(dst, uint16(float32ToPCM16(right)))
	}
	return dst
}

func readFrameFloat32(plane []byte, sample int) float32 {
	offset := sample * 4
	if offset+4 > len(plane) {
		return 0
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(plane[offset:]))
}

func float32ToPCM16(value float32) int16 {
	sample := int(math.Round(float64(value * 32768)))
	if sample < minPCM16 {
		return minPCM16
	}
	if sample > maxPCM16 {
		return maxPCM16
	}
	return int16(sample)
}

func outputSampleRateForSource(_ int) int {
	return defaultSampleRate
}

type infinitePCM struct {
	data []byte
	pos  int
}

func newInfinitePCM(data []byte) *infinitePCM {
	return &infinitePCM{data: data}
}

func (r *infinitePCM) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	written := 0
	for written < len(p) {
		n := copy(p[written:], r.data[r.pos:])
		written += n
		r.pos += n
		if r.pos >= len(r.data) {
			r.pos = 0
		}
	}
	return written, nil
}

func decodeWAVPCM16Stereo(data []byte) ([]byte, int, error) {
	if len(data) < 44 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, 0, fmt.Errorf("not a RIFF/WAVE file")
	}
	pos := 12
	var channels, bitsPerSample uint16
	var sampleRate uint32
	var pcm []byte
	for pos+8 <= len(data) {
		id := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		pos += 8
		if size < 0 || pos+size > len(data) {
			return nil, 0, fmt.Errorf("invalid wav chunk %q size %d", id, size)
		}
		chunk := data[pos : pos+size]
		switch id {
		case "fmt ":
			if len(chunk) < 16 {
				return nil, 0, fmt.Errorf("short fmt chunk")
			}
			format := binary.LittleEndian.Uint16(chunk[0:2])
			channels = binary.LittleEndian.Uint16(chunk[2:4])
			sampleRate = binary.LittleEndian.Uint32(chunk[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(chunk[14:16])
			if format != 1 {
				return nil, 0, fmt.Errorf("unsupported wav format %d", format)
			}
		case "data":
			pcm = append([]byte(nil), chunk...)
		}
		pos += size
		if pos%2 != 0 {
			pos++
		}
	}
	if sampleRate == 0 || len(pcm) == 0 {
		return nil, 0, fmt.Errorf("missing wav fmt or data")
	}
	if bitsPerSample != 16 {
		return nil, 0, fmt.Errorf("unsupported wav bits per sample %d", bitsPerSample)
	}
	switch channels {
	case 1:
		stereo, err := monoPCM16ToStereo(pcm)
		if err != nil {
			return nil, 0, err
		}
		pcm = stereo
	case 2:
		if len(pcm)%4 != 0 {
			return nil, 0, fmt.Errorf("invalid stereo pcm length %d", len(pcm))
		}
	default:
		return nil, 0, fmt.Errorf("unsupported wav channels %d", channels)
	}
	return pcm, int(sampleRate), nil
}

func monoPCM16ToStereo(src []byte) ([]byte, error) {
	if len(src)%2 != 0 {
		return nil, fmt.Errorf("invalid mono pcm length %d", len(src))
	}
	dst := make([]byte, len(src)*2)
	for frame := 0; frame < len(src)/2; frame++ {
		sample := int16(uint16(src[frame*2]) | uint16(src[frame*2+1])<<8)
		writePCM16(dst, frame, 0, sample)
		writePCM16(dst, frame, 1, sample)
	}
	return dst, nil
}

func resamplePCM16Stereo(src []byte, srcRate, dstRate int) ([]byte, error) {
	if srcRate <= 0 || dstRate <= 0 {
		return nil, fmt.Errorf("invalid sample rate %d -> %d", srcRate, dstRate)
	}
	if len(src)%4 != 0 {
		return nil, fmt.Errorf("invalid stereo pcm length %d", len(src))
	}
	if srcRate == dstRate {
		return src, nil
	}
	srcFrames := len(src) / 4
	if srcFrames == 0 {
		return nil, fmt.Errorf("empty decoded PCM")
	}
	if dstRate > srcRate {
		return resamplePCM16StereoLinear(src, srcRate, dstRate)
	}
	dstFrames := int(float64(srcFrames) * float64(dstRate) / float64(srcRate))
	if dstFrames <= 0 {
		return nil, fmt.Errorf("empty resampled PCM")
	}
	dst := make([]byte, dstFrames*4)

	const radius = 8.0
	cutoff := 1.0
	if dstRate < srcRate {
		cutoff = float64(dstRate) / float64(srcRate)
	}
	divisor := gcd(srcRate, dstRate)
	phaseCount := dstRate / divisor
	taps := make([][]resampleTap, phaseCount)
	for phase := range taps {
		fraction := float64(phase*divisor) / float64(dstRate)
		taps[phase] = buildResampleTaps(fraction, radius, cutoff)
	}
	for frame := 0; frame < dstFrames; frame++ {
		srcNumerator := int64(frame) * int64(srcRate)
		base := int(srcNumerator / int64(dstRate))
		phase := int(srcNumerator%int64(dstRate)) / divisor
		left := 0.0
		right := 0.0
		weightSum := 0.0
		for _, tap := range taps[phase] {
			sample := base + tap.offset
			if sample < 0 || sample >= srcFrames {
				continue
			}
			weightSum += tap.weight
			left += pcm16ToFloat(readPCM16(src, sample, 0)) * tap.weight
			right += pcm16ToFloat(readPCM16(src, sample, 1)) * tap.weight
		}
		if weightSum != 0 {
			left /= weightSum
			right /= weightSum
		}
		writePCM16(dst, frame, 0, floatToPCM16(left))
		writePCM16(dst, frame, 1, floatToPCM16(right))
	}
	return dst, nil
}

func resamplePCM16StereoLinear(src []byte, srcRate, dstRate int) ([]byte, error) {
	if srcRate <= 0 || dstRate <= 0 {
		return nil, fmt.Errorf("invalid sample rate %d -> %d", srcRate, dstRate)
	}
	if len(src)%4 != 0 {
		return nil, fmt.Errorf("invalid stereo pcm length %d", len(src))
	}
	srcFrames := len(src) / 4
	if srcFrames == 0 {
		return nil, fmt.Errorf("empty decoded PCM")
	}
	dstFrames := int(float64(srcFrames) * float64(dstRate) / float64(srcRate))
	if dstFrames <= 0 {
		return nil, fmt.Errorf("empty resampled PCM")
	}
	dst := make([]byte, dstFrames*4)
	for frame := 0; frame < dstFrames; frame++ {
		srcPosition := float64(frame) * float64(srcRate) / float64(dstRate)
		base := int(srcPosition)
		fraction := srcPosition - float64(base)
		next := base + 1
		if next >= srcFrames {
			next = srcFrames - 1
		}
		for channel := 0; channel < 2; channel++ {
			a := float64(readPCM16(src, base, channel))
			b := float64(readPCM16(src, next, channel))
			writePCM16(dst, frame, channel, int16(math.Round(a+(b-a)*fraction)))
		}
	}
	return dst, nil
}

type resampleTap struct {
	offset int
	weight float64
}

func buildResampleTaps(fraction float64, radius, cutoff float64) []resampleTap {
	start := int(math.Floor(fraction - radius))
	end := int(math.Ceil(fraction + radius))
	taps := make([]resampleTap, 0, end-start+1)
	sum := 0.0
	for offset := start; offset <= end; offset++ {
		distance := fraction - float64(offset)
		if math.Abs(distance) > radius {
			continue
		}
		weight := cutoff * sinc(cutoff*distance) * hann(distance/radius)
		if weight == 0 {
			continue
		}
		taps = append(taps, resampleTap{offset: offset, weight: weight})
		sum += weight
	}
	if sum != 0 {
		for i := range taps {
			taps[i].weight /= sum
		}
	}
	return taps
}

func gcd(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

func readPCM16(pcm []byte, frame, channel int) int16 {
	offset := frame*4 + channel*2
	return int16(uint16(pcm[offset]) | uint16(pcm[offset+1])<<8)
}

func writePCM16(pcm []byte, frame, channel int, value int16) {
	offset := frame*4 + channel*2
	u := uint16(value)
	pcm[offset] = byte(u)
	pcm[offset+1] = byte(u >> 8)
}

func pcm16ToFloat(value int16) float64 {
	return float64(value) / 32768
}

func floatToPCM16(value float64) int16 {
	if value > 1 {
		value = 1
	}
	if value < -1 {
		value = -1
	}
	value *= 32767
	if value > 32767 {
		return 32767
	}
	if value < -32768 {
		return -32768
	}
	return int16(value)
}

func sinc(x float64) float64 {
	if math.Abs(x) < 1e-8 {
		return 1
	}
	x *= math.Pi
	return math.Sin(x) / x
}

func hann(x float64) float64 {
	if x < 0 {
		x = -x
	}
	if x >= 1 {
		return 0
	}
	return 0.5 + 0.5*math.Cos(math.Pi*x)
}

func (b *BGM) ensureContext(preferredSampleRate int) *oto.Context {
	if b.context != nil {
		return b.context
	}
	if preferredSampleRate <= 0 {
		preferredSampleRate = defaultSampleRate
	}
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   preferredSampleRate,
		ChannelCount: 2,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		log.Printf("bgm create audio context failed sample_rate=%d: %v", preferredSampleRate, err)
		return nil
	}
	<-ready
	b.context = context
	b.sampleRate = preferredSampleRate
	log.Printf("bgm created audio context sample_rate=%d", b.sampleRate)
	return b.context
}

func (b *BGM) Stop() {
	if b == nil {
		return
	}
	b.stopCurrent()
	b.stopSFX()
	b.current = ""
}

func (b *BGM) stopCurrent() {
	if b.player == nil {
		return
	}
	log.Printf("bgm stopping id=%d path=%s playing=%t", b.playerID, b.current, b.player.IsPlaying())
	b.player.Pause()
	b.player = nil
}

func (b *BGM) trimSFXPlayers() {
	if len(b.sfxPlayers) == 0 {
		return
	}
	active := b.sfxPlayers[:0]
	for _, player := range b.sfxPlayers {
		if player == nil {
			continue
		}
		if player.IsPlaying() {
			active = append(active, player)
			continue
		}
	}
	const maxSFXPlayers = 32
	for len(active) > maxSFXPlayers {
		player := active[0]
		active = active[1:]
		if player != nil {
			player.Pause()
		}
	}
	b.sfxPlayers = active
}

func (b *BGM) stopSFX() {
	for _, player := range b.sfxPlayers {
		if player == nil {
			continue
		}
		player.Pause()
	}
	b.sfxPlayers = nil
}

func (b *BGM) ResolveMapBGM(mapName string) string {
	if b == nil {
		return ""
	}
	b.ensureTable()
	key := canonicalMapName(mapName)
	if key == "" {
		return "bgm\\01.mp3"
	}
	if path := b.table[key]; path != "" {
		return path
	}
	return "bgm\\01.mp3"
}

func (b *BGM) ensureTable() {
	if b.table != nil {
		return
	}
	b.table = make(map[string]string)
	if b.resources == nil {
		return
	}
	_, data, ok := b.resources.ReadFirst([]string{
		"data\\mp3nametable.txt",
		"data/mp3nametable.txt",
		"mp3nametable.txt",
	})
	if !ok {
		log.Printf("bgm table not found, falling back to bgm\\01.mp3")
		return
	}
	for key, path := range ParseMP3NameTable(data) {
		b.table[key] = path
	}
	log.Printf("bgm table loaded entries=%d", len(b.table))
}

func ParseMP3NameTable(data []byte) map[string]string {
	out := make(map[string]string)
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(strings.TrimSuffix(rawLine, "\r"))
		if line == "" {
			continue
		}
		parts := strings.Split(line, "#")
		if len(parts) < 2 {
			continue
		}
		key := canonicalMapName(parts[0])
		path := normalizeAudioPath(parts[1])
		if key == "" || path == "" {
			continue
		}
		out[key] = path
	}
	return out
}

func canonicalMapName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "/", "\\")
	name = strings.ToLower(name)
	name = strings.TrimPrefix(name, "data\\")
	name = strings.TrimSuffix(name, ".gat")
	name = strings.TrimSuffix(name, ".rsw")
	if name == "" {
		return ""
	}
	return name + ".rsw"
}

func normalizeAudioPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"")
	path = strings.ReplaceAll(path, "/", "\\")
	path = strings.TrimPrefix(path, ".\\")
	path = strings.TrimPrefix(path, "data\\")
	if path == "" {
		return ""
	}
	if !strings.Contains(path, "\\") {
		path = "bgm\\" + path
	}
	return path
}

func readAudioFile(manager *res.Manager, path string) ([]byte, string, error) {
	if manager == nil {
		return nil, "", fmt.Errorf("resource manager is nil")
	}
	for _, candidate := range audioPathCandidates(path) {
		data, err := manager.ReadFile(candidate)
		if err == nil {
			return data, candidate, nil
		}
	}
	return nil, "", fmt.Errorf("bgm not found: %s", path)
}

func readSFXFile(manager *res.Manager, path string) ([]byte, string, error) {
	if manager == nil {
		return nil, "", fmt.Errorf("resource manager is nil")
	}
	for _, candidate := range sfxPathCandidates(path) {
		data, err := manager.ReadFileExact(candidate)
		if err == nil {
			return data, candidate, nil
		}
	}
	return nil, "", fmt.Errorf("sfx not found: %s", path)
}

func audioPathCandidates(path string) []string {
	normalized := normalizeAudioPath(path)
	if normalized == "" {
		return nil
	}
	slash := strings.ReplaceAll(normalized, "\\", "/")
	candidates := []string{normalized, slash}
	if strings.HasPrefix(strings.ToLower(normalized), "bgm\\") {
		rest := normalized[4:]
		candidates = append(candidates,
			"BGM\\"+rest,
			"BGM/"+strings.ReplaceAll(rest, "\\", "/"),
			"bgm\\"+rest,
			"bgm/"+strings.ReplaceAll(rest, "\\", "/"),
		)
	}
	if filepath.Ext(normalized) == "" {
		candidates = append(candidates, normalized+".mp3", slash+".mp3")
	}
	return uniqueStrings(candidates)
}

func normalizeSFXPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"")
	path = strings.ReplaceAll(path, "/", "\\")
	path = strings.TrimPrefix(path, ".\\")
	path = strings.TrimPrefix(path, "data\\")
	if path == "" {
		return ""
	}
	if filepath.Ext(path) == "" {
		path += ".wav"
	}
	return path
}

func sfxPathCandidates(path string) []string {
	normalized := normalizeSFXPath(path)
	if normalized == "" {
		return nil
	}
	slash := strings.ReplaceAll(normalized, "\\", "/")
	candidates := []string{normalized, slash}
	lower := strings.ToLower(normalized)
	if strings.HasPrefix(lower, "wav\\") {
		candidates = append(candidates,
			"data\\"+normalized,
			"data/"+slash,
		)
	} else {
		candidates = append(candidates,
			"wav\\"+normalized,
			"wav/"+slash,
			"data\\wav\\"+normalized,
			"data/wav/"+slash,
		)
	}
	return uniqueStrings(candidates)
}

func sameAudioPath(a, b string) bool {
	return strings.EqualFold(normalizeAudioPath(a), normalizeAudioPath(b))
}

func uniqueStrings(values []string) []string {
	out := values[:0]
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
