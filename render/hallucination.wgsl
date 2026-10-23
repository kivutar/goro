struct Ripple {
    phase: vec4<f32>,
}

@group(0) @binding(0) var<uniform> ripple: Ripple;
@group(0) @binding(1) var source: texture_2d<f32>;

@vertex
fn fullscreen(@builtin(vertex_index) index: u32) -> @builtin(position) vec4<f32> {
    let uv = vec2<f32>(f32((index << 1u) & 2u), f32(index & 2u));
    return vec4<f32>(uv.x * 2.0 - 1.0, 1.0 - uv.y * 2.0, 0.0, 1.0);
}

@fragment
fn distort(@builtin(position) position: vec4<f32>) -> @location(0) vec4<f32> {
    let size = vec2<i32>(textureDimensions(source));
    var pixel = vec2<i32>(position.xy);
    // The original shifts each scanline by up to 1% of the screen width,
    // advancing the sine by two degrees per row from bottom to top.
    let phase = ripple.phase.x + f32(size.y - 1 - pixel.y) * 0.034906585;
    let shift = i32(sin(phase) * f32(size.x) * 0.01);
    let x = pixel.x - shift;
    // Row copies leave the uncovered edge pixels unchanged.
    if (x >= 0 && x < size.x) {
        pixel.x = x;
    }
    return textureLoad(source, pixel, 0);
}
