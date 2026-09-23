package render

import (
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/wgpu"
)

func TestReleasedImageGroupDropsOnlyItsResources(t *testing.T) {
	old, current := &ImageGroup{}, &ImageGroup{}
	oldImage, newImage := old.Add(NewImage(2, 2)), current.Add(NewImage(2, 2))
	neverUploaded := old.Add(NewImage(2, 2))
	oldTexture, newTexture := &gogpu.Texture{}, &gogpu.Texture{}
	oldMesh, newMesh := &WorldMesh{texture: oldImage}, &WorldMesh{texture: newImage}
	oldKey, newKey := bindGroupKey{texture: oldTexture}, bindGroupKey{texture: newTexture}
	lightKey := bindGroupKey{texture: newTexture, lightTexture: oldTexture}
	r := &gpuRenderer{
		imageGroups: map[*ImageGroup]struct{}{old: {}, current: {}},
		textures:    map[*Image]*gpuImageTexture{oldImage: {tex: oldTexture}, newImage: {tex: newTexture}},
		bindGroups:  map[bindGroupKey]*wgpu.BindGroup{oldKey: nil, newKey: nil, lightKey: nil},
		worldMeshes: map[*WorldMesh]*gpuWorldMesh{oldMesh: {}, newMesh: {}},
	}
	old.Release()
	old.Release()
	if oldImage.RGBA() != nil || neverUploaded.RGBA() != nil || newImage.RGBA() == nil {
		t.Fatal("release did not drop just the old map's CPU pixels")
	}
	r.releaseImageGroups()
	if len(r.imageGroups) != 1 || len(r.textures) != 1 || r.textures[newImage] == nil || len(r.worldMeshes) != 1 || r.worldMeshes[newMesh] == nil {
		t.Fatal("old GPU resources retained, or current resources removed")
	}
	if len(r.bindGroups) != 1 {
		t.Fatal("bind groups referencing released textures retained")
	}
	if _, ok := r.bindGroups[newKey]; !ok {
		t.Fatal("current bind group removed")
	}
	r.releaseImageGroups()
	if len(r.textures) != 1 {
		t.Fatal("second release removed an active resource")
	}
}

func TestPrepareImageDoesNotDrawAndClearsFrameReferences(t *testing.T) {
	f := NewFrame(100, 100)
	img := NewImage(2, 2)
	options := DrawTrianglesOptions{Filter: FilterLinear, Address: AddressRepeat}
	f.PrepareImage(img, &options)
	if len(f.imageUploads) != 1 || f.imageUploads[0].image != img || f.imageUploads[0].options != options || len(f.commands) != 0 || len(f.worldCommands) != 0 {
		t.Fatal("preparation drew the image or lost upload options")
	}
	f.BeginFrame()
	if len(f.imageUploads) != 0 || f.imageUploads[:cap(f.imageUploads)][0].image != nil {
		t.Fatal("old upload retained by the next frame")
	}
	g := &ImageGroup{}
	g.Add(img)
	g.Release()
	f.PrepareImage(img, nil)
	if len(f.imageUploads) != 0 {
		t.Fatal("released image queued for upload")
	}
}
