package render

import (
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

// A non-nil pipeline draws the same geometry into the SSAO depth prepass.
func (r *gpuRenderer) drawWorldOpaque(pass *wgpu.RenderPassEncoder, meshes []worldMeshBatch, world worldFrame, vertices, indices *wgpu.Buffer, pipeline *wgpu.RenderPipeline) error {
	state := renderPassState{}
	for _, batch := range meshes {
		if err := r.drawWorldMeshBatch(pass, batch, &state, pipeline); err != nil {
			return err
		}
	}
	if vertices == nil || indices == nil {
		return nil
	}
	state.setVertexBuffer(pass, vertices)
	state.setIndexBuffer(pass, indices)
	for _, batch := range world.batches {
		if batch.indexCount == 0 || !batch.key.options.DepthWrite {
			continue
		}
		if err := r.drawWorldBatch(pass, batch, &state, pipeline); err != nil {
			return err
		}
	}
	return nil
}

func (r *gpuRenderer) drawWorldBatch(pass *wgpu.RenderPassEncoder, batch drawBatch, state *renderPassState, pipeline *wgpu.RenderPipeline) error {
	tex, err := r.ensureTexture(batch.key.texture)
	if err != nil {
		return err
	}
	sampler, err := r.sampler(batch.key.options, batch.key.texture)
	if err != nil {
		return err
	}
	lightTex, err := r.ensureBatchLightTexture(batch.key)
	if err != nil {
		return err
	}
	bg, err := r.bindWorldGroup(r.worldUniform, 96, tex, lightTex, sampler)
	if err != nil {
		return err
	}
	if pipeline == nil {
		pipeline = r.worldPipelineFor(batch.key.options)
	}
	state.setPipeline(pass, pipeline)
	state.setBindGroup(pass, bg)
	pass.DrawIndexed(gputypes.DrawIndexedArgs{IndexCount: batch.indexCount, InstanceCount: 1, FirstIndex: batch.firstIndex})
	return nil
}
