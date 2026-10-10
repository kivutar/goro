struct AOUniforms {
    inverse_view_projection: mat4x4<f32>,
    params: vec4<f32>, // world-space radius, strength
}

@group(0) @binding(0) var<uniform> ao: AOUniforms;
@group(0) @binding(1) var scene_depth: texture_2d<f32>;
@group(0) @binding(2) var occlusion: texture_2d<f32>;

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

fn depth_at(pixel: vec2<i32>) -> vec4<f32> {
    let size = vec2<i32>(textureDimensions(scene_depth));
    if (pixel.x < 0 || pixel.y < 0 || pixel.x >= size.x || pixel.y >= size.y) {
        return vec4<f32>(0.0);
    }
    return textureLoad(scene_depth, pixel, 0);
}

fn unpack_depth(value: vec4<f32>) -> f32 {
    return dot(value.rgb, vec3<f32>(65536.0, 256.0, 1.0)) * (255.0 / 16777215.0);
}

fn position_at(pixel: vec2<i32>, depth: f32) -> vec3<f32> {
    let uv = (vec2<f32>(pixel) + vec2<f32>(0.5)) / vec2<f32>(textureDimensions(scene_depth));
    // The game camera uses OpenGL clip depth; world shaders map it to [0,1].
    let p = ao.inverse_view_projection * vec4<f32>(uv.x * 2.0 - 1.0, 1.0 - uv.y * 2.0, depth * 2.0 - 1.0, 1.0);
    return p.xyz / p.w;
}

fn surface_step(pixel: vec2<i32>, offset: vec2<i32>, center: vec3<f32>, depth: f32) -> vec3<f32> {
    let a = depth_at(pixel + offset);
    let b = depth_at(pixel - offset);
    // Choose the neighbor on the same surface at silhouettes and creases.
    if (a.a > 0.0 && (b.a == 0.0 || abs(unpack_depth(a) - depth) <= abs(unpack_depth(b) - depth))) {
        return position_at(pixel + offset, unpack_depth(a)) - center;
    }
    if (b.a > 0.0) {
        return center - position_at(pixel - offset, unpack_depth(b));
    }
    return position_at(pixel + offset, depth) - center;
}

@fragment
fn estimate(input: ScreenVertex) -> @location(0) vec4<f32> {
    let pixel = vec2<i32>(input.uv * vec2<f32>(textureDimensions(scene_depth)));
    let encoded_depth = depth_at(pixel);
    if (encoded_depth.a == 0.0) {
        return vec4<f32>(1.0);
    }
    let depth = unpack_depth(encoded_depth);
    let center = position_at(pixel, depth);
    let dx = surface_step(pixel, vec2<i32>(1, 0), center, depth);
    let dy = surface_step(pixel, vec2<i32>(0, 1), center, depth);
    var normal = cross(dx, dy);
    normal = normal / max(length(normal), 0.000001);
    let toward_camera = position_at(pixel, 0.0) - center;
    if (dot(normal, toward_camera) < 0.0) {
        normal = -normal;
    }
    let radius = ao.params.x;
    let pixel_size = length(position_at(pixel + vec2<i32>(1, 0), depth) - center);
    let radius_pixels = clamp(radius / max(pixel_size, 0.000001), 1.0, 128.0);
    var shade = 0.0;
    // Eight fixed directions keep the result stable while the camera is still.
    for (var direction = 0; direction < 8; direction++) {
        let angle = f32(direction) * 0.7853981634;
        let ray = vec2<f32>(cos(angle), sin(angle));
        var horizon = 0.0;
        for (var step = 1; step <= 4; step++) {
            let offset = vec2<i32>(round(ray * radius_pixels * f32(step) * 0.25));
            let sample_pixel = pixel + offset;
            let sample_depth = depth_at(sample_pixel);
            if (sample_depth.a > 0.0) {
                let delta = position_at(sample_pixel, unpack_depth(sample_depth)) - center;
                let distance = length(delta);
                let elevation = max(0.0, dot(normal, delta) / max(distance, 0.000001) - 0.12);
                let falloff = max(0.0, 1.0 - distance * distance / (radius * radius));
                horizon = max(horizon, elevation * falloff);
            }
        }
        shade += horizon;
    }
    let visibility = 1.0 - ao.params.y * shade / 8.0;
    // Keep the sampled depth with AO so upsampling needs only one texture read.
    return vec4<f32>(encoded_depth.rgb, visibility);
}

@fragment
fn composite(input: ScreenVertex) -> @location(0) vec4<f32> {
    let size = vec2<i32>(textureDimensions(scene_depth));
    let pixel = vec2<i32>(input.uv * vec2<f32>(size));
    let encoded_depth = depth_at(pixel);
    if (encoded_depth.a == 0.0) {
        return vec4<f32>(1.0);
    }
    let center = position_at(pixel, unpack_depth(encoded_depth));
    let ao_size = vec2<i32>(textureDimensions(occlusion));
    let sample_position = input.uv * vec2<f32>(ao_size) - vec2<f32>(0.5);
    let base = vec2<i32>(floor(sample_position));
    let fraction = fract(sample_position);
    var total = 0.0;
    var weights = 0.0;
    // Four depth-weighted bilinear taps preserve silhouettes at full resolution.
    for (var y = 0; y < 2; y++) {
        for (var x = 0; x < 2; x++) {
            let q = clamp(base + vec2<i32>(x, y), vec2<i32>(0), ao_size - vec2<i32>(1));
            let ao_sample = textureLoad(occlusion, q, 0);
            let uv = (vec2<f32>(q) + vec2<f32>(0.5)) / vec2<f32>(ao_size);
            let sample_pixel = vec2<i32>(uv * vec2<f32>(size));
            let delta = position_at(sample_pixel, unpack_depth(ao_sample)) - center;
            let bilinear = mix(1.0 - fraction.x, fraction.x, f32(x)) * mix(1.0 - fraction.y, fraction.y, f32(y));
            let weight = bilinear * exp2(-dot(delta, delta) * 64.0 / (ao.params.x * ao.params.x));
            total += ao_sample.a * weight;
            weights += weight;
        }
    }
    var visibility = 1.0;
    if (weights > 0.0001) {
        visibility = total / weights;
    }
    visibility = mix(1.0, visibility, encoded_depth.a);
    return vec4<f32>(visibility, visibility, visibility, 1.0);
}
