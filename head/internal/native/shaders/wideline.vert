#version 450
// Wide-line vertex shader (#2015 line weight): expands a stroked line segment into a screen-space
// quad, so a CAD line weight renders as a constant PIXEL width at any zoom — which is what a line
// weight means; a world-space stroke would vanish when zoomed out and swell when zoomed in.
//
// The expansion has to happen here rather than on the CPU: the merged frame geometry is
// content-keyed and its upload is deliberately skipped while that key holds across an orbit
// (#1422), so camera-dependent vertices computed on the CPU would go stale on the GPU. It also
// cannot use vkCmdSetLineWidth: wideLines is an optional Vulkan feature and Metal-backed drivers
// report a [1,1] width range, so that path would silently draw hairlines on macOS.
//
// Vertices arrive in the standard 16-float mesh layout with three slots repurposed for this
// stream, where they are dead (a stroke is flat and unlit). head/viewport/flatten.go's
// appendWideLineItem is the only writer of this encoding:
//   inNormal    = the segment's OTHER endpoint (model space)
//   inMetallic  = which side of the stroke this corner sits on (+1 / -1)
//   inRoughness = the stroke width in pixels
layout(push_constant) uniform PushConstants {
    mat4 mvp;
    vec4 camPosLit;
    vec4 clip;
    vec4 viewport; // xy = framebuffer size in pixels
} pc;
layout(location = 0) in vec3  inPos;
layout(location = 1) in vec3  inNormal;    // the other endpoint
layout(location = 2) in vec4  inColor;
layout(location = 3) in float inMetallic;  // side, +1/-1
layout(location = 4) in float inRoughness; // width in pixels
layout(location = 5) in vec3  inEmissive;
layout(location = 6) in float inMode;
layout(location = 7) in mat4  inModel;
layout(location = 0) out vec4 vColor;
void main() {
    vec4 here  = pc.mvp * (inModel * vec4(inPos, 1.0));
    vec4 there = pc.mvp * (inModel * vec4(inNormal, 1.0));

    // Work in pixels so the offset is a true screen width. Both ends are divided by their own w:
    // using a single w would skew the stroke of a segment running steeply away from the camera.
    vec2 hereP  = (here.xy  / max(abs(here.w),  1e-6)) * pc.viewport.xy * 0.5;
    vec2 thereP = (there.xy / max(abs(there.w), 1e-6)) * pc.viewport.xy * 0.5;

    vec2 along = thereP - hereP;
    // A segment whose ends project to the same pixel has no direction to be perpendicular to;
    // pick a fixed axis so the quad collapses to a dot instead of producing NaNs.
    vec2 dir = length(along) > 1e-6 ? normalize(along) : vec2(1.0, 0.0);
    vec2 offsetPx = vec2(-dir.y, dir.x) * inMetallic * (max(inRoughness, 1.0) * 0.5);

    // Back to clip space: undo the halving above, then pre-multiply by w so the offset survives
    // the perspective divide at its intended pixel size.
    vec2 offsetNdc = offsetPx / (pc.viewport.xy * 0.5);
    gl_Position = vec4(here.xy + offsetNdc * here.w, here.z, here.w);
    vColor = inColor;
}
