package render

import (
	"testing"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

func TestWorldPassResolvesOnlyMultisampledColor(t *testing.T) {
	surface, color, depth := &wgpu.TextureView{}, &wgpu.TextureView{}, &wgpu.TextureView{}
	for _, samples := range []uint32{1, 4} {
		r := &gpuRenderer{
			sampleCount: samples,
			msaaTarget:  gpuRenderTarget{view: color},
			depthTarget: gpuRenderTarget{view: depth},
		}
		pass := r.worldPassDescriptor(surface, gputypes.Color{A: 1})
		attachment := pass.ColorAttachments[0]
		if samples == 1 {
			if attachment.View != surface || attachment.ResolveTarget != nil || attachment.StoreOp != gputypes.StoreOpStore {
				t.Fatalf("single-sample world must preserve its output for the UI: %+v", attachment)
			}
		} else if attachment.View != color || attachment.ResolveTarget != surface {
			t.Fatalf("multisampled world must resolve into the UI target: %+v", attachment)
		}
		if attachment.LoadOp != gputypes.LoadOpClear || pass.DepthStencilAttachment.View != depth || pass.DepthStencilAttachment.DepthClearValue != 1 {
			t.Fatal("world color and depth must start fresh each frame")
		}
	}
}
