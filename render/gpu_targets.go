package render

import (
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

// gpuRenderTarget owns an attachment whose format, sample count, and usage are
// fixed for the renderer's lifetime. Its storage follows the framebuffer size.
type gpuRenderTarget struct {
	texture *wgpu.Texture
	view    *wgpu.TextureView
	width   int
	height  int
}

func (t *gpuRenderTarget) ensure(dev *wgpu.Device, width, height int, format gputypes.TextureFormat, samples uint32, usage gputypes.TextureUsage) error {
	if t.view != nil && t.width == width && t.height == height {
		return nil
	}
	texture, err := dev.CreateTexture(&wgpu.TextureDescriptor{
		Label: "goro-world-attachment",
		Size: wgpu.Extent3D{
			Width: uint32(width), Height: uint32(height), DepthOrArrayLayers: 1,
		},
		MipLevelCount: 1,
		SampleCount:   samples,
		Dimension:     gputypes.TextureDimension2D,
		Format:        format,
		Usage:         usage,
	})
	if err != nil {
		return err
	}
	view, err := dev.CreateTextureView(texture, nil)
	if err != nil {
		texture.Release()
		return err
	}
	t.release()
	*t = gpuRenderTarget{texture: texture, view: view, width: width, height: height}
	return nil
}

func (t *gpuRenderTarget) release() {
	if t.view != nil {
		t.view.Release()
	}
	if t.texture != nil {
		t.texture.Release()
	}
	*t = gpuRenderTarget{}
}

func (r *gpuRenderer) worldPassDescriptor(surface *wgpu.TextureView, clear gputypes.Color) *wgpu.RenderPassDescriptor {
	color := wgpu.RenderPassColorAttachment{
		View: surface, LoadOp: gputypes.LoadOpClear, StoreOp: gputypes.StoreOpStore, ClearValue: clear,
	}
	if r.sampleCount > 1 {
		color.View = r.msaaTarget.view
		color.ResolveTarget = surface
		color.StoreOp = gputypes.StoreOpDiscard
	}
	return &wgpu.RenderPassDescriptor{
		ColorAttachments: []wgpu.RenderPassColorAttachment{color},
		DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
			View: r.depthTarget.view, DepthLoadOp: gputypes.LoadOpClear,
			DepthStoreOp: gputypes.StoreOpDiscard, DepthClearValue: 1,
		},
	}
}
