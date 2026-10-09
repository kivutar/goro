@group(0) @binding(0) var linear_sampler: sampler;
@group(0) @binding(1) var source: texture_2d<f32>;
@group(0) @binding(2) var glow: texture_2d<f32>;

struct ScreenVertex {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
}

@vertex
fn fullscreen(@builtin(vertex_index) index: u32) -> ScreenVertex {
    let uv = vec2<f32>(f32((index << 1u) & 2u), f32(index & 2u));
    var out: ScreenVertex;
    out.position = vec4<f32>(uv.x * 2.0 - 1.0, 1.0 - uv.y * 2.0, 0.0, 1.0);
    out.uv = uv;
    return out;
}

fn bright(uv: vec2<f32>) -> vec3<f32> {
    let color = textureSample(source, linear_sampler, uv).rgb;
    let peak = max(max(color.r, color.g), color.b);
    return color * smoothstep(0.7, 1.0, peak);
}

@fragment
fn extract(input: ScreenVertex) -> @location(0) vec4<f32> {
    // Four bilinear taps cover a 4x4 block of the full-resolution scene.
    let step = 1.0 / vec2<f32>(textureDimensions(source));
    let color = bright(input.uv + step * vec2<f32>(-1.0, -1.0))
              + bright(input.uv + step * vec2<f32>( 1.0, -1.0))
              + bright(input.uv + step * vec2<f32>(-1.0,  1.0))
              + bright(input.uv + step * vec2<f32>( 1.0,  1.0));
    return vec4<f32>(color * 0.25, 1.0);
}

fn blur(uv: vec2<f32>, direction: vec2<f32>) -> vec4<f32> {
    // Symmetric nine-tap Gaussian, paired into five bilinear samples.
    let step = direction / vec2<f32>(textureDimensions(source));
    var color = textureSample(source, linear_sampler, uv).rgb * 0.2270270270;
    color += textureSample(source, linear_sampler, uv + step * 1.3846153846).rgb * 0.3162162162;
    color += textureSample(source, linear_sampler, uv - step * 1.3846153846).rgb * 0.3162162162;
    color += textureSample(source, linear_sampler, uv + step * 3.2307692308).rgb * 0.0702702703;
    color += textureSample(source, linear_sampler, uv - step * 3.2307692308).rgb * 0.0702702703;
    return vec4<f32>(color, 1.0);
}

@fragment
fn blur_horizontal(input: ScreenVertex) -> @location(0) vec4<f32> {
    return blur(input.uv, vec2<f32>(1.0, 0.0));
}

@fragment
fn blur_vertical(input: ScreenVertex) -> @location(0) vec4<f32> {
    return blur(input.uv, vec2<f32>(0.0, 1.0));
}

@fragment
fn composite(input: ScreenVertex) -> @location(0) vec4<f32> {
    let color = textureSample(source, linear_sampler, input.uv);
    let bloom = textureSample(glow, linear_sampler, input.uv).rgb;
    return vec4<f32>(color.rgb + bloom * 0.18, color.a);
}
