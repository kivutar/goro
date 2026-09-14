package res

import (
	"encoding/binary"
	"fmt"
	"math"
)

type ACT struct {
	VersionMajor int
	VersionMinor int
	Actions      []ACTAction
	Sounds       []string
}

type ACTAction struct {
	Animations []ACTAnimation
	DelayMS    float32
}

type ACTAnimation struct {
	Layers []ACTLayer
	Sound  int
	Pos    []ACTPosition
}

type ACTLayer struct {
	X       int32
	Y       int32
	Index   int32
	Mirror  bool
	ScaleX  float32
	ScaleY  float32
	Color   [4]float32
	Angle   int32
	SPRType int32
	Width   int32
	Height  int32
}

type ACTPosition struct {
	X    int32
	Y    int32
	Attr int32
}

func ParseACT(data []byte) (*ACT, error) {
	reader := actReader{data: data}
	if string(reader.bytes(2)) != "AC" {
		return nil, fmt.Errorf("act invalid header")
	}
	act := &ACT{
		VersionMinor: int(reader.u8()),
		VersionMajor: int(reader.u8()),
	}
	actionCount := int(reader.u16())
	reader.skip(10)
	if err := reader.checkCount(uint32(actionCount), 4); err != nil {
		return nil, err
	}
	act.Actions = make([]ACTAction, actionCount)
	for i := range act.Actions {
		animations, err := reader.readAnimations(act)
		if err != nil {
			return nil, err
		}
		act.Actions[i] = ACTAction{Animations: animations, DelayMS: 150}
	}

	if act.versionAtLeast(2, 1) {
		soundCount := int(reader.i32())
		if soundCount < 0 {
			return nil, fmt.Errorf("act negative sound count")
		}
		if err := reader.checkCount(uint32(soundCount), 40); err != nil {
			return nil, err
		}
		act.Sounds = make([]string, soundCount)
		for i := range act.Sounds {
			act.Sounds[i] = fixedBinaryString(reader.bytes(40))
		}
		if act.versionAtLeast(2, 2) {
			for i := range act.Actions {
				act.Actions[i].DelayMS = reader.f32() * 25
			}
		}
	}
	if reader.err != nil {
		return nil, reader.err
	}
	return act, nil
}

func (a *ACT) versionAtLeast(major, minor int) bool {
	return a.VersionMajor > major || (a.VersionMajor == major && a.VersionMinor >= minor)
}

func (a *ACT) ActionFor(action, direction int) (ACTAction, bool) {
	if len(a.Actions) == 0 {
		return ACTAction{}, false
	}
	index := action*8 + ((direction%8)+8)%8
	index %= len(a.Actions)
	return a.Actions[index], true
}

func (r *actReader) readAnimations(act *ACT) ([]ACTAnimation, error) {
	count := r.u32()
	// Reserved bytes and layer count, followed by optional sound/anchor fields.
	minimumSize := 36
	if act.versionAtLeast(2, 0) {
		minimumSize += 4
	}
	if act.versionAtLeast(2, 3) {
		minimumSize += 4
	}
	if err := r.checkCount(count, minimumSize); err != nil {
		return nil, err
	}
	animations := make([]ACTAnimation, count)
	for i := range animations {
		r.skip(32)
		anim, err := r.readLayers(act)
		if err != nil {
			return nil, err
		}
		animations[i] = anim
	}
	return animations, nil
}

func (r *actReader) readLayers(act *ACT) (ACTAnimation, error) {
	count := r.u32()
	layerSize := 16
	if act.versionAtLeast(2, 0) {
		layerSize += 16
	}
	if act.versionAtLeast(2, 4) {
		layerSize += 4
	}
	if act.versionAtLeast(2, 5) {
		layerSize += 8
	}
	if err := r.checkCount(count, layerSize); err != nil {
		return ACTAnimation{}, err
	}
	anim := ACTAnimation{Layers: make([]ACTLayer, count), Sound: -1}
	for i := range anim.Layers {
		layer := ACTLayer{
			X:      r.i32(),
			Y:      r.i32(),
			Index:  r.i32(),
			Mirror: r.i32() != 0,
			ScaleX: 1,
			ScaleY: 1,
			Color:  [4]float32{1, 1, 1, 1},
		}
		if act.versionAtLeast(2, 0) {
			layer.Color[0] = float32(r.u8()) / 255
			layer.Color[1] = float32(r.u8()) / 255
			layer.Color[2] = float32(r.u8()) / 255
			layer.Color[3] = float32(r.u8()) / 255
			layer.ScaleX = r.f32()
			if act.versionAtLeast(2, 4) {
				layer.ScaleY = r.f32()
			} else {
				layer.ScaleY = layer.ScaleX
			}
			layer.Angle = r.i32()
			layer.SPRType = r.i32()
			if act.versionAtLeast(2, 5) {
				layer.Width = r.i32()
				layer.Height = r.i32()
			}
		}
		anim.Layers[i] = layer
	}
	if act.versionAtLeast(2, 0) {
		anim.Sound = int(r.i32())
	}
	if act.versionAtLeast(2, 3) {
		count := int(r.i32())
		if count < 0 {
			return ACTAnimation{}, fmt.Errorf("act negative position count")
		}
		if err := r.checkCount(uint32(count), 16); err != nil {
			return ACTAnimation{}, err
		}
		anim.Pos = make([]ACTPosition, count)
		for i := range anim.Pos {
			r.skip(4)
			anim.Pos[i] = ACTPosition{X: r.i32(), Y: r.i32(), Attr: r.i32()}
		}
	}
	if r.err != nil {
		return ACTAnimation{}, r.err
	}
	return anim, nil
}

type actReader struct {
	data   []byte
	offset int
	err    error
}

// Bound file-provided counts before allocation. Division avoids overflowing a
// count*recordSize multiplication, including on 32-bit targets.
func (r *actReader) checkCount(count uint32, recordSize int) error {
	if r.err != nil {
		return r.err
	}
	if uint64(count) > uint64((len(r.data)-r.offset)/recordSize) {
		return fmt.Errorf("act count %d at offset %d exceeds remaining data for %d-byte records", count, r.offset, recordSize)
	}
	return nil
}

func (r *actReader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > len(r.data)-r.offset {
		r.err = fmt.Errorf("act truncated at offset %d reading %d bytes", r.offset, n)
		return nil
	}
	out := r.data[r.offset : r.offset+n]
	r.offset += n
	return out
}

func (r *actReader) skip(n int) {
	_ = r.bytes(n)
}

func (r *actReader) u8() byte {
	data := r.bytes(1)
	if r.err != nil {
		return 0
	}
	return data[0]
}

func (r *actReader) u16() uint16 {
	data := r.bytes(2)
	if r.err != nil {
		return 0
	}
	return binary.LittleEndian.Uint16(data)
}

func (r *actReader) u32() uint32 {
	data := r.bytes(4)
	if r.err != nil {
		return 0
	}
	return binary.LittleEndian.Uint32(data)
}

func (r *actReader) i32() int32 {
	return int32(r.u32())
}

func (r *actReader) f32() float32 {
	data := r.bytes(4)
	if r.err != nil {
		return 0
	}
	value := math.Float32frombits(binary.LittleEndian.Uint32(data))
	if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
		return 0
	}
	return value
}
