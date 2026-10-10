// Reuse the world vertex shader and alpha test. Packing depth into a color
// target avoids depth resolve and depth sampling differences between backends.
@fragment
fn ssao_depth(input: VertexOutput) -> @location(0) vec4<f32> {
    let alpha = textureSample(tex, tex_sampler, input.uv).a * input.color.a;
    if (alpha < 0.01) {
        discard;
    }
    let depth = u32(clamp(input.clip.z, 0.0, 1.0) * 16777215.0);
    let encoded_depth = vec3<f32>(f32((depth >> 16u) & 255u), f32((depth >> 8u) & 255u), f32(depth & 255u)) / 255.0;
    let fog = clamp(smoothstep(uniforms.fog[0], uniforms.fog[1], input.fog_depth) * uniforms.fog[2] * uniforms.fog[3] * input.fog_enabled, 0.0, 1.0);
    return vec4<f32>(encoded_depth, 1.0 - fog);
}
