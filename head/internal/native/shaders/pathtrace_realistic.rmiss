#version 460
// Primary/continuation-ray miss for the LIVE Realistic-mode viewport (M45-F05 PBI-350,
// #2135/#2155): a ray that misses all geometry samples the bound equirect environment map
// (binding 4, the same image the raster skybox shows via obk_viewport_env_binding) instead
// of returning flat black — background visibility only, not full hemisphere IBL sampling
// (openpbr/env_sample.glsl's own doc comment). A dedicated miss shader (rather than reusing
// the shared pathtrace.rmiss) because #2155 grew this pipeline's location-0 payload from a
// plain vec3 to RTPathPayload — pathtrace.rmiss stays on the plain-vec3 layout for its own
// (untouched) PBI-345 pipeline.
#extension GL_EXT_ray_tracing : require
#extension GL_GOOGLE_include_directive : require
#include "rt_payload.glsl"
#include "openpbr/env_sample.glsl"

layout(location = 0) rayPayloadInEXT RTPathPayload payload;

// Field list shared with pathtrace_realistic.rchit/swpathtrace_realistic.comp — see
// extended_lobes.glsl's OPENPBR_REALISTIC_PARAMS_FIELDS doc comment. Read here only for
// envEnabled/envRotation/envIntensity.
layout(set = 0, binding = 3) uniform Params { OPENPBR_REALISTIC_PARAMS_FIELDS } params;
layout(set = 0, binding = 4) uniform sampler2D envMap;

void main() {
    payload.radiance = openpbrSampleEnvironment(envMap, gl_WorldRayDirectionEXT, params.envEnabled,
                                                params.envRotation, params.envIntensity);
    payload.hitDistance = gl_RayTmaxEXT;
}
