#version 460
// Shadow-ray miss (M45-F04 PBI-345): reaching the light unoccluded — the shadow ray's
// only two outcomes are "hit something opaque" (skipped straight to this miss shader's
// sibling hit path via gl_RayFlagsSkipClosestHitShaderEXT — never runs a closest-hit) or
// this miss.
#extension GL_EXT_ray_tracing : require

layout(location = 1) rayPayloadInEXT bool shadowed;

void main() { shadowed = false; }
