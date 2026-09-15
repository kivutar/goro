package res

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

// One action, animation, and layer, with every field supported by the version.
func actCountFixture(major, minor byte) ([]byte, map[string]int) {
	version := &ACT{VersionMajor: int(major), VersionMinor: int(minor)}
	var buf bytes.Buffer
	buf.WriteString("AC")
	buf.Write([]byte{minor, major})
	offsets := map[string]int{"actions": buf.Len()}
	writeTestU16(&buf, 1)
	buf.Write(make([]byte, 10))
	offsets["animations"] = buf.Len()
	writeTestU32(&buf, 1)
	buf.Write(make([]byte, 32))
	offsets["layers"] = buf.Len()
	writeTestU32(&buf, 1)
	for _, value := range []int32{12, -34, 3, 1} {
		writeTestI32(&buf, value)
	}
	if version.versionAtLeast(2, 0) {
		buf.Write([]byte{255, 128, 64, 32})
		writeTestF32(&buf, 1.5)
		if version.versionAtLeast(2, 4) {
			writeTestF32(&buf, 0.75)
		}
		writeTestI32(&buf, 45)
		writeTestI32(&buf, 1)
		if version.versionAtLeast(2, 5) {
			writeTestI32(&buf, 32)
			writeTestI32(&buf, 64)
		}
		writeTestI32(&buf, -1) // Sound index.
	}
	if version.versionAtLeast(2, 3) {
		offsets["positions"] = buf.Len()
		writeTestU32(&buf, 1)
		for _, value := range []int32{0, 5, 6, 7} {
			writeTestI32(&buf, value)
		}
	}
	if version.versionAtLeast(2, 1) {
		offsets["sounds"] = buf.Len()
		writeTestU32(&buf, 1)
		sound := make([]byte, 40)
		copy(sound, "test.wav")
		buf.Write(sound)
	}
	if version.versionAtLeast(2, 2) {
		writeTestF32(&buf, 6)
	}
	return buf.Bytes(), offsets
}

func TestParseACTCountValidationAcrossVersions(t *testing.T) {
	for _, version := range [][2]byte{{1, 0}, {1, 1}, {2, 0}, {2, 1}, {2, 2}, {2, 3}, {2, 4}, {2, 5}} {
		t.Run(fmt.Sprintf("%d.%d", version[0], version[1]), func(t *testing.T) {
			data, offsets := actCountFixture(version[0], version[1])
			act, err := ParseACT(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(act.Actions) != 1 || len(act.Actions[0].Animations) != 1 || len(act.Actions[0].Animations[0].Layers) != 1 {
				t.Fatal("valid records were lost")
			}
			layer := act.Actions[0].Animations[0].Layers[0]
			if layer.X != 12 || layer.Y != -34 || layer.Index != 3 || !layer.Mirror {
				t.Fatalf("incorrect layer: %+v", layer)
			}
			if act.versionAtLeast(2, 3) {
				positions := act.Actions[0].Animations[0].Pos
				if len(positions) != 1 || positions[0] != (ACTPosition{X: 5, Y: 6, Attr: 7}) {
					t.Fatalf("incorrect positions: %+v", positions)
				}
			}
			if act.versionAtLeast(2, 1) && (len(act.Sounds) != 1 || act.Sounds[0] != "test.wav") {
				t.Fatalf("incorrect sounds: %v", act.Sounds)
			}
			for end := range len(data) {
				if _, err := ParseACT(data[:end]); err == nil {
					t.Fatalf("accepted truncated file of %d/%d bytes", end, len(data))
				}
			}
			for name, offset := range offsets {
				t.Run(name, func(t *testing.T) {
					bad := bytes.Clone(data)
					if name == "actions" {
						binary.LittleEndian.PutUint16(bad[offset:], 0xFFFF)
					} else {
						binary.LittleEndian.PutUint32(bad[offset:], 100000)
					}
					if _, err := ParseACT(bad); err == nil {
						t.Fatal("accepted count exceeding available records")
					}
					if name == "actions" {
						return
					}
					binary.LittleEndian.PutUint32(bad[offset:], 0xFFFFFFFF)
					if _, err := ParseACT(bad); err == nil {
						t.Fatal("accepted maximum unsigned/negative signed count")
					}
				})
			}
		})
	}
}

func TestParseACTAllowsEmptyRecords(t *testing.T) {
	for _, version := range [][2]byte{{1, 1}, {2, 0}, {2, 3}, {2, 5}} {
		var buf bytes.Buffer
		buf.WriteString("AC")
		buf.Write([]byte{version[1], version[0]})
		writeTestU16(&buf, 1)
		buf.Write(make([]byte, 10))
		writeTestU32(&buf, 0) // No animations.
		if version[0] == 2 && version[1] >= 1 {
			writeTestU32(&buf, 0) // No sounds.
		}
		if version[0] == 2 && version[1] >= 2 {
			writeTestF32(&buf, 6)
		}
		act, err := ParseACT(buf.Bytes())
		if err != nil || len(act.Actions) != 1 || len(act.Actions[0].Animations) != 0 {
			t.Fatalf("version %v empty action: act=%+v error=%v", version, act, err)
		}
	}
}

func BenchmarkParseACTTruncatedAnimationCount(b *testing.B) {
	data := make([]byte, 20)
	copy(data, "AC")
	data[2], data[3] = 4, 2
	binary.LittleEndian.PutUint16(data[4:], 1)
	binary.LittleEndian.PutUint32(data[16:], 100000)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ParseACT(data); err == nil {
			b.Fatal("expected truncated file error")
		}
	}
}
