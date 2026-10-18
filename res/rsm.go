package res

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
)

type RSM struct {
	Signature          string
	VersionMajor       byte
	VersionMinor       byte
	AnimLength         int32
	ShadeType          int32
	Alpha              float32
	FrameRatePerSecond float32
	Textures           []string
	MainNodeName       string
	Nodes              []RSMNode
	PositionKeyframes  []RSMPositionKeyframe
	VolumeBoxes        []RSMVolumeBox
}

type RSMNode struct {
	Name                  string
	ParentName            string
	TextureRefs           []RSMTextureRef
	Matrix                [9]float32
	Offset                RSMVector3
	Position              RSMVector3
	RotationAngle         float32
	RotationAxis          RSMVector3
	Scale                 RSMVector3
	Vertices              []RSMVector3
	TextureVertices       []RSMTextureVertex
	Faces                 []RSMFace
	ScaleKeyframes        []RSMScaleKeyframe
	RotationKeyframes     []RSMRotationKeyframe
	PositionKeyframes     []RSMPositionKeyframe
	TextureKeyframeGroups []RSMTextureKeyframeGroup
}

type RSMTextureRef struct {
	Index int32
	Name  string
}

type RSMVector3 struct {
	X float32
	Y float32
	Z float32
}

type RSMTextureVertex struct {
	Color [4]float32
	U     float32
	V     float32
}

type RSMFace struct {
	VertexIndices        [3]uint16
	TextureVertexIndices [3]uint16
	TextureID            uint16
	TwoSide              int32
	SmoothGroup          int32
	SmoothGroupExtra     []int32
}

type RSMScaleKeyframe struct {
	Frame int32
	Scale RSMVector3
	Data  float32
}

type RSMRotationKeyframe struct {
	Frame      int32
	Quaternion [4]float32
}

type RSMPositionKeyframe struct {
	Frame int32
	Pos   RSMVector3
	Data  float32
	Int   int32
}

type RSMTextureKeyframeGroup struct {
	TextureID int32
	Animation []RSMTextureAnimation
}

type RSMTextureAnimation struct {
	Type   int32
	Frames []RSMTextureKeyframe
}

type RSMTextureKeyframe struct {
	Frame  int32
	Offset float32
}

type RSMVolumeBox struct {
	Size     RSMVector3
	Position RSMVector3
	Rotation RSMVector3
	Flag     int32
}

func ParseRSM(data []byte) (*RSM, error) {
	reader := rsmReader{data: data}
	signature := string(reader.bytes(4))
	if signature != "GRSM" && signature != "GRSX" {
		return nil, fmt.Errorf("invalid rsm signature")
	}

	rsm := &RSM{
		Signature:          signature,
		VersionMajor:       reader.u8(),
		VersionMinor:       reader.u8(),
		FrameRatePerSecond: 60,
		Alpha:              1,
	}
	rsm.AnimLength = reader.i32()
	rsm.ShadeType = reader.i32()
	if rsm.versionAtLeast(1, 4) {
		rsm.Alpha = float32(reader.u8()) / 255
	}

	var versionTextures []string
	switch {
	case rsm.versionAtLeast(2, 3):
		rsm.FrameRatePerSecond = reader.f32()
		versionTextures = readRSMStringTable(&reader, true)
	case rsm.versionAtLeast(2, 2):
		rsm.FrameRatePerSecond = reader.f32()
		rsm.Textures = readRSMStringTable(&reader, true)
		versionTextures = readRSMStringTable(&reader, true)
	default:
		reader.skip(16)
		rsm.Textures = readRSMStringTable(&reader, false)
		rsm.MainNodeName = fixedBinaryString(reader.bytes(40))
		versionTextures = append(versionTextures, rsm.MainNodeName)
	}

	nodeCount := reader.i32()
	if nodeCount < 0 {
		return nil, fmt.Errorf("invalid rsm node count=%d", nodeCount)
	}
	rsm.Nodes = make([]RSMNode, int(nodeCount))
	for i := range rsm.Nodes {
		rsm.Nodes[i] = readRSMNode(&reader, rsm)
	}

	if rsm.MainNodeName == "" && len(rsm.Nodes) > 0 {
		rsm.MainNodeName = rsm.Nodes[0].Name
	}

	if !rsm.versionAtLeast(1, 6) {
		rsm.PositionKeyframes = readRSMLegacyPositionKeyframes(&reader)
	}

	if reader.remaining() >= 4 {
		volumeCount := reader.i32()
		if volumeCount < 0 {
			return nil, fmt.Errorf("invalid rsm volume count=%d", volumeCount)
		}
		rsm.VolumeBoxes = make([]RSMVolumeBox, int(volumeCount))
		for i := range rsm.VolumeBoxes {
			rsm.VolumeBoxes[i] = RSMVolumeBox{
				Size:     reader.vector3(),
				Position: reader.vector3(),
				Rotation: reader.vector3(),
			}
			if rsm.versionAtLeast(1, 3) {
				rsm.VolumeBoxes[i].Flag = reader.i32()
			}
		}
	}

	if rsm.versionAtLeast(2, 3) {
		rsm.Textures = appendUniqueStrings(rsm.Textures, versionTextures...)
		for nodeIndex := range rsm.Nodes {
			for refIndex := range rsm.Nodes[nodeIndex].TextureRefs {
				name := rsm.Nodes[nodeIndex].TextureRefs[refIndex].Name
				if name == "" {
					continue
				}
				rsm.Textures = appendUniqueStrings(rsm.Textures, name)
				rsm.Nodes[nodeIndex].TextureRefs[refIndex].Index = int32(indexString(rsm.Textures, name))
			}
		}
	}

	if reader.err != nil {
		return nil, reader.err
	}
	return rsm, nil
}

func RSMModelCandidates(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	candidates := []string{
		"data\\model\\" + name,
		"data/model/" + strings.ReplaceAll(name, "\\", "/"),
		"model\\" + name,
		"model/" + strings.ReplaceAll(name, "\\", "/"),
		name,
	}
	return candidates
}

func (r *RSM) versionAtLeast(major, minor byte) bool {
	return r.VersionMajor > major || r.VersionMajor == major && r.VersionMinor >= minor
}

func readRSMStringTable(reader *rsmReader, lengthPrefixed bool) []string {
	count := reader.i32()
	if count <= 0 {
		return nil
	}
	out := make([]string, int(count))
	for i := range out {
		if lengthPrefixed {
			length := reader.i32()
			out[i] = fixedBinaryString(reader.bytes(int(length)))
		} else {
			out[i] = fixedBinaryString(reader.bytes(40))
		}
	}
	return out
}

func readRSMNode(reader *rsmReader, rsm *RSM) RSMNode {
	node := RSMNode{
		Position:     RSMVector3{},
		RotationAxis: RSMVector3{},
		Scale:        RSMVector3{X: 1, Y: 1, Z: 1},
	}
	if rsm.versionAtLeast(2, 2) {
		node.Name = fixedBinaryString(reader.bytes(int(reader.i32())))
		node.ParentName = fixedBinaryString(reader.bytes(int(reader.i32())))
	} else {
		node.Name = fixedBinaryString(reader.bytes(40))
		node.ParentName = fixedBinaryString(reader.bytes(40))
	}

	textureCount := reader.i32()
	if textureCount > 0 {
		node.TextureRefs = make([]RSMTextureRef, int(textureCount))
		for i := range node.TextureRefs {
			if rsm.versionAtLeast(2, 3) {
				length := reader.i32()
				node.TextureRefs[i].Name = fixedBinaryString(reader.bytes(int(length)))
			} else {
				node.TextureRefs[i].Index = reader.i32()
			}
		}
	}

	for i := range node.Matrix {
		node.Matrix[i] = reader.f32()
	}
	node.Offset = reader.vector3()

	if !rsm.versionAtLeast(2, 2) {
		node.Position = reader.vector3()
		node.RotationAngle = reader.f32()
		node.RotationAxis = reader.vector3()
		node.Scale = reader.vector3()
	}

	vertexCount := reader.i32()
	if vertexCount > 0 {
		node.Vertices = make([]RSMVector3, int(vertexCount))
		for i := range node.Vertices {
			node.Vertices[i] = reader.vector3()
		}
	}

	textureVertexCount := reader.i32()
	if textureVertexCount > 0 {
		node.TextureVertices = make([]RSMTextureVertex, int(textureVertexCount))
		for i := range node.TextureVertices {
			if rsm.versionAtLeast(1, 2) {
				node.TextureVertices[i].Color = [4]float32{
					float32(reader.u8()) / 255,
					float32(reader.u8()) / 255,
					float32(reader.u8()) / 255,
					float32(reader.u8()) / 255,
				}
			} else {
				node.TextureVertices[i].Color = [4]float32{1, 1, 1, 1}
			}
			node.TextureVertices[i].U = reader.f32()*0.98 + 0.01
			node.TextureVertices[i].V = reader.f32()*0.98 + 0.01
		}
	}

	faceCount := reader.i32()
	if faceCount > 0 {
		node.Faces = make([]RSMFace, int(faceCount))
		for i := range node.Faces {
			faceLen := int32(-1)
			if rsm.versionAtLeast(2, 2) {
				faceLen = reader.i32()
			}
			for j := range node.Faces[i].VertexIndices {
				node.Faces[i].VertexIndices[j] = reader.u16()
			}
			for j := range node.Faces[i].TextureVertexIndices {
				node.Faces[i].TextureVertexIndices[j] = reader.u16()
			}
			node.Faces[i].TextureID = reader.u16()
			_ = reader.u16()
			node.Faces[i].TwoSide = reader.i32()
			if rsm.versionAtLeast(1, 2) {
				node.Faces[i].SmoothGroup = reader.i32()
				if faceLen > 24 {
					node.Faces[i].SmoothGroupExtra = append(node.Faces[i].SmoothGroupExtra, reader.i32())
				}
				if faceLen > 28 {
					node.Faces[i].SmoothGroupExtra = append(node.Faces[i].SmoothGroupExtra, reader.i32())
				}
				if faceLen > 32 {
					reader.skip(int(faceLen - 32))
				}
			}
		}
	}

	if rsm.versionAtLeast(1, 6) {
		node.ScaleKeyframes = readRSMScaleKeyframes(reader)
	}
	node.RotationKeyframes = readRSMRotationKeyframes(reader)
	if rsm.versionAtLeast(2, 2) {
		node.PositionKeyframes = readRSMPositionKeyframes(reader)
	}
	if rsm.versionAtLeast(2, 3) {
		node.TextureKeyframeGroups = readRSMTextureKeyframeGroups(reader)
	}
	return node
}

func readRSMScaleKeyframes(reader *rsmReader) []RSMScaleKeyframe {
	count := reader.i32()
	if count <= 0 {
		return nil
	}
	out := make([]RSMScaleKeyframe, int(count))
	for i := range out {
		out[i] = RSMScaleKeyframe{
			Frame: reader.i32(),
			Scale: reader.vector3(),
			Data:  reader.f32(),
		}
	}
	return out
}

func readRSMRotationKeyframes(reader *rsmReader) []RSMRotationKeyframe {
	count := reader.i32()
	if count <= 0 {
		return nil
	}
	out := make([]RSMRotationKeyframe, int(count))
	for i := range out {
		out[i].Frame = reader.i32()
		for j := range out[i].Quaternion {
			out[i].Quaternion[j] = reader.f32()
		}
	}
	return out
}

func readRSMPositionKeyframes(reader *rsmReader) []RSMPositionKeyframe {
	count := reader.i32()
	if count <= 0 {
		return nil
	}
	out := make([]RSMPositionKeyframe, int(count))
	for i := range out {
		out[i] = RSMPositionKeyframe{
			Frame: reader.i32(),
			Pos:   reader.vector3(),
			Int:   reader.i32(),
		}
	}
	return out
}

func readRSMLegacyPositionKeyframes(reader *rsmReader) []RSMPositionKeyframe {
	count := reader.i32()
	if count <= 0 {
		return nil
	}
	out := make([]RSMPositionKeyframe, int(count))
	for i := range out {
		out[i] = RSMPositionKeyframe{
			Frame: reader.i32(),
			Pos:   reader.vector3(),
			Data:  reader.f32(),
		}
	}
	return out
}

func readRSMTextureKeyframeGroups(reader *rsmReader) []RSMTextureKeyframeGroup {
	count := reader.i32()
	if count <= 0 {
		return nil
	}
	out := make([]RSMTextureKeyframeGroup, int(count))
	for i := range out {
		out[i].TextureID = reader.i32()
		animCount := reader.i32()
		if animCount <= 0 {
			continue
		}
		out[i].Animation = make([]RSMTextureAnimation, int(animCount))
		for j := range out[i].Animation {
			out[i].Animation[j].Type = reader.i32()
			frameCount := reader.i32()
			if frameCount <= 0 {
				continue
			}
			out[i].Animation[j].Frames = make([]RSMTextureKeyframe, int(frameCount))
			for k := range out[i].Animation[j].Frames {
				out[i].Animation[j].Frames[k] = RSMTextureKeyframe{
					Frame:  reader.i32(),
					Offset: reader.f32(),
				}
			}
		}
	}
	return out
}

func appendUniqueStrings(values []string, additions ...string) []string {
	for _, addition := range additions {
		if addition == "" || indexString(values, addition) >= 0 {
			continue
		}
		values = append(values, addition)
	}
	return values
}

func indexString(values []string, target string) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

type rsmReader struct {
	data   []byte
	offset int
	err    error
}

func (r *rsmReader) remaining() int {
	return len(r.data) - r.offset
}

func (r *rsmReader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || r.offset+n > len(r.data) {
		r.err = fmt.Errorf("rsm truncated at offset %d reading %d bytes", r.offset, n)
		return nil
	}
	out := r.data[r.offset : r.offset+n]
	r.offset += n
	return out
}

func (r *rsmReader) skip(n int) {
	_ = r.bytes(n)
}

func (r *rsmReader) u8() byte {
	data := r.bytes(1)
	if len(data) == 0 {
		return 0
	}
	return data[0]
}

func (r *rsmReader) u16() uint16 {
	data := r.bytes(2)
	if len(data) < 2 {
		return 0
	}
	return binary.LittleEndian.Uint16(data)
}

func (r *rsmReader) i32() int32 {
	data := r.bytes(4)
	if len(data) < 4 {
		return 0
	}
	return int32(binary.LittleEndian.Uint32(data))
}

func (r *rsmReader) f32() float32 {
	data := r.bytes(4)
	if len(data) < 4 {
		return 0
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(data))
}

func (r *rsmReader) vector3() RSMVector3 {
	return RSMVector3{X: r.f32(), Y: r.f32(), Z: r.f32()}
}
