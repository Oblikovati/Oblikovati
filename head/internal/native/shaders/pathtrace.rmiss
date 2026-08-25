#version 460
// Primary-ray miss (M45-F04 PBI-345): background is black — this harness has no
// environment/IBL yet (F04-PBI348).
#extension GL_EXT_ray_tracing : require

layout(location = 0) rayPayloadInEXT vec3 payload;

void main() { payload = vec3(0.0); }
