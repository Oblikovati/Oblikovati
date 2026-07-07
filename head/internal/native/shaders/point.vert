#version 450
// Point-cloud vertex shader (#645 perf): one GPU vertex per scan point, drawn on the POINT_LIST
// pipeline — no CPU 3-axis-cross expansion (6 line verts/point) and no per-frame re-upload; the
// buffer is retained in VRAM (obk_viewport_upload_points) and only the MVP changes as you orbit.
// The stream is a compact interleave [pos.xyz, rgba] (7 floats), distinct from the 16-float mesh
// vertex. The on-screen point size is a fixed pixel count carried in the push-constant clip.x slot:
// the point pipeline does no section clipping, so that vec4 is free to repurpose here.
layout(push_constant) uniform PushConstants {
    mat4 mvp;
    vec4 camPosLit;
    vec4 clip; // clip.x = point size in pixels (repurposed for the point pipeline)
} pc;
layout(location = 0) in vec3 inPos;
layout(location = 1) in vec4 inColor;
layout(location = 0) out vec4 vColor;
void main() {
    gl_Position = pc.mvp * vec4(inPos, 1.0);
    gl_PointSize = pc.clip.x; // clamped to 1.0 on devices without the largePoints feature
    vColor = inColor;
}
