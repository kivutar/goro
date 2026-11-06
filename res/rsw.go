package res

import (
	"encoding/binary"
	"fmt"
	"image/color"
	"math"
)

type RSW struct {
	VersionMajor byte
	VersionMinor byte
	BuildNumber  int32
	Files        RSWFiles
	Water        RSWWater
	Light        RSWLight
	Ground       RSWGround
	Models       []RSWModel
	Lights       []RSWObjectLight
	Sounds       []RSWSound
	Effects      []RSWEffect
}

type RSWFiles struct {
	INI string
	GND string
	GAT string
	SRC string
}

type RSWWater struct {
	Level      float32
	Type       int32
	WaveHeight float32
	WaveSpeed  float32
	WavePitch  float32
	AnimSpeed  int32
}

type RSWLight struct {
	Longitude int32
	Latitude  int32
	Diffuse   [3]float32
	Ambient   [3]float32
	Opacity   float32
}

type RSWGround struct {
	Top    int32
	Bottom int32
	Left   int32
	Right  int32
}

type RSWVector3 struct {
	X float32
	Y float32
	Z float32
}

type RSWModel struct {
	Name        string
	AnimType    int32
	AnimSpeed   float32
	BlockType   int32
	UnknownByte byte
	UnknownInt  int32
	Filename    string
	NodeName    string
	Position    RSWVector3
	Rotation    RSWVector3
	Scale       RSWVector3
}

type RSWObjectLight struct {
	Name     string
	Position RSWVector3
	Color    color.RGBA
	Range    float32
}

type RSWSound struct {
	Name     string
	File     string
	Position RSWVector3
	Volume   float32
	Width    int32
	Height   int32
	Range    float32
	Cycle    float32
}

type RSWEffect struct {
	Name     string
	Position RSWVector3
	ID       int32
	Delay    float32 // Emission interval in classic 60 Hz effect ticks.
	Param    [4]float32
}

func ParseRSW(data []byte) (*RSW, error) {
	reader := rswReader{data: data}
	if string(reader.bytes(4)) != "GRSW" {
		return nil, fmt.Errorf("invalid rsw signature")
	}

	rsw := &RSW{
		VersionMajor: reader.u8(),
		VersionMinor: reader.u8(),
		Water: RSWWater{
			WaveHeight: 1,
			WaveSpeed:  2,
			WavePitch:  50,
			AnimSpeed:  3,
		},
		Light: RSWLight{
			Longitude: 45,
			Latitude:  45,
			Diffuse:   [3]float32{1, 1, 1},
			Opacity:   1,
		},
		Ground: RSWGround{
			Top:    -500,
			Bottom: 500,
			Left:   -500,
			Right:  500,
		},
	}
	if reader.err != nil {
		return nil, reader.err
	}

	if rsw.versionAtLeast(2, 5) {
		rsw.BuildNumber = reader.i32()
	}
	if rsw.versionAtLeast(2, 2) {
		_ = reader.u8()
	}

	rsw.Files.INI = fixedBinaryString(reader.bytes(40))
	rsw.Files.GND = fixedBinaryString(reader.bytes(40))
	rsw.Files.GAT = fixedBinaryString(reader.bytes(40))
	if rsw.versionAtLeast(1, 4) {
		rsw.Files.SRC = fixedBinaryString(reader.bytes(40))
	}

	if !rsw.versionAtLeast(2, 6) {
		if rsw.versionAtLeast(1, 3) {
			rsw.Water.Level = -reader.f32() / 5
		}
		if rsw.versionAtLeast(1, 8) {
			rsw.Water.Type = reader.i32()
			rsw.Water.WaveHeight = reader.f32() / 5
			rsw.Water.WaveSpeed = reader.f32()
			rsw.Water.WavePitch = reader.f32()
		}
		if rsw.versionAtLeast(1, 9) {
			rsw.Water.AnimSpeed = reader.i32()
		}
	}

	if rsw.versionAtLeast(1, 5) {
		rsw.Light.Longitude = reader.i32()
		rsw.Light.Latitude = reader.i32()
		for i := range rsw.Light.Diffuse {
			rsw.Light.Diffuse[i] = reader.f32()
		}
		for i := range rsw.Light.Ambient {
			rsw.Light.Ambient[i] = reader.f32()
		}
		if rsw.versionAtLeast(1, 7) {
			rsw.Light.Opacity = reader.f32()
		}
	}

	if rsw.versionAtLeast(1, 6) {
		rsw.Ground.Top = reader.i32()
		rsw.Ground.Bottom = reader.i32()
		rsw.Ground.Left = reader.i32()
		rsw.Ground.Right = reader.i32()
	}

	if rsw.versionAtLeast(2, 7) {
		count := reader.i32()
		if count < 0 {
			return nil, fmt.Errorf("invalid rsw auxiliary count=%d", count)
		}
		reader.skip(int(count) * 4)
	}

	objectCount := reader.i32()
	if objectCount < 0 {
		return nil, fmt.Errorf("invalid rsw object count=%d", objectCount)
	}
	for i := 0; i < int(objectCount); i++ {
		objectType := reader.i32()
		switch objectType {
		case 1:
			rsw.Models = append(rsw.Models, readRSWModel(&reader, rsw))
		case 2:
			rsw.Lights = append(rsw.Lights, readRSWObjectLight(&reader))
		case 3:
			rsw.Sounds = append(rsw.Sounds, readRSWSound(&reader, rsw))
		case 4:
			rsw.Effects = append(rsw.Effects, readRSWEffect(&reader))
		default:
			return nil, fmt.Errorf("unsupported rsw object type %d at index %d", objectType, i)
		}
	}
	if reader.err != nil {
		return nil, reader.err
	}
	return rsw, nil
}

func (r *RSW) versionAtLeast(major, minor byte) bool {
	return r.VersionMajor > major || r.VersionMajor == major && r.VersionMinor >= minor
}

func readRSWModel(reader *rswReader, rsw *RSW) RSWModel {
	var model RSWModel
	if rsw.versionAtLeast(1, 3) {
		model.Name = fixedBinaryString(reader.bytes(40))
		model.AnimType = reader.i32()
		model.AnimSpeed = reader.f32()
		model.BlockType = reader.i32()
	}
	if rsw.versionAtLeast(2, 6) && rsw.BuildNumber >= 186 {
		model.UnknownByte = reader.u8()
	}
	if rsw.versionAtLeast(2, 7) {
		model.UnknownInt = reader.i32()
	}
	model.Filename = fixedBinaryString(reader.bytes(80))
	model.NodeName = fixedBinaryString(reader.bytes(80))
	model.Position = reader.scaledPosition3(5)
	model.Rotation = reader.vector3()
	model.Scale = reader.scaledVector3(5)
	return model
}

func readRSWObjectLight(reader *rswReader) RSWObjectLight {
	light := RSWObjectLight{
		Name:     fixedBinaryString(reader.bytes(80)),
		Position: reader.scaledPosition3(5),
	}
	light.Color = color.RGBA{
		R: clampByte(reader.i32()),
		G: clampByte(reader.i32()),
		B: clampByte(reader.i32()),
		A: 255,
	}
	light.Range = reader.f32()
	return light
}

func readRSWSound(reader *rswReader, rsw *RSW) RSWSound {
	sound := RSWSound{
		Name:     fixedBinaryString(reader.bytes(80)),
		File:     fixedBinaryString(reader.bytes(80)),
		Position: reader.scaledPosition3(5),
		Volume:   reader.f32(),
		Width:    reader.i32(),
		Height:   reader.i32(),
		Range:    reader.f32(),
	}
	if rsw.versionAtLeast(2, 0) {
		sound.Cycle = reader.f32()
	}
	return sound
}

func readRSWEffect(reader *rswReader) RSWEffect {
	effect := RSWEffect{
		Name:     fixedBinaryString(reader.bytes(80)),
		Position: reader.scaledPosition3(5),
		ID:       reader.i32(),
		Delay:    reader.f32(),
	}
	for i := range effect.Param {
		effect.Param[i] = reader.f32()
	}
	return effect
}

func clampByte(value int32) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}

type rswReader struct {
	data   []byte
	offset int
	err    error
}

func (r *rswReader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.offset+n > len(r.data) {
		r.err = fmt.Errorf("rsw truncated at offset %d reading %d bytes", r.offset, n)
		return nil
	}
	out := r.data[r.offset : r.offset+n]
	r.offset += n
	return out
}

func (r *rswReader) skip(n int) {
	_ = r.bytes(n)
}

func (r *rswReader) u8() byte {
	data := r.bytes(1)
	if len(data) == 0 {
		return 0
	}
	return data[0]
}

func (r *rswReader) i32() int32 {
	data := r.bytes(4)
	if len(data) < 4 {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(data))
}

func (r *rswReader) f32() float32 {
	data := r.bytes(4)
	if len(data) < 4 {
		return 0
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(data))
}

func (r *rswReader) vector3() RSWVector3 {
	return RSWVector3{X: r.f32(), Y: r.f32(), Z: r.f32()}
}

func (r *rswReader) scaledVector3(scale float32) RSWVector3 {
	v := r.vector3()
	return RSWVector3{X: v.X / scale, Y: v.Y / scale, Z: v.Z / scale}
}

func (r *rswReader) scaledPosition3(scale float32) RSWVector3 {
	v := r.vector3()
	return RSWVector3{X: v.X / scale, Y: -v.Y / scale, Z: v.Z / scale}
}
