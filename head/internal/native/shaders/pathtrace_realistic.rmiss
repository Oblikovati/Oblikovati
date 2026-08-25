#version 460
// Primary-ray miss for the LIVE Realistic-mode viewport (M45-F05 PBI-350, #2155): background
// is black — this pipeline has no environment/IBL yet (F04-PBI348). A dedicated miss
// shader (rather than reusing the shared pathtrace.rmiss) because #2155 grew this
// pipeline's location-0 payload from a plain vec3 to RTPathPayload — pathtrace.rmiss stays
// on the plain-vec3 layout for its own (untouched) PBI-345 pipeline.
#extension GL_EXT_ray_tracing : require
#extension GL_GOOGLE_include_directive : require
#include "rt_payload.glsl"

layout(location = 0) rayPayloadInEXT RTPathPayload payload;

void main() {
    payload.radiance = vec3(0.0);
    payload.hitDistance = gl_RayTmaxEXT;
}
