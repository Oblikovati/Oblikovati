#version 460
// Closest-hit for the LIVE Realistic-mode viewport (M45-F05 PBI-350, ADR-0053): the
// same F03 base-lobes BSDF evaluation as pathtrace.rchit, but shading ONE directional
// light per dispatch (SceneLight.Direction is already a unit vector toward the light —
// renderer/lighting.go — so there is no inverse-square/toLight distance term, unlike
// pathtrace.rchit's point-light harness). The Go side picks which light via
// renderer.LightDistribution (PBI-348) before each dispatch and feeds only that one
// light's direction/premultiplied color here; accumulating many dispatches that each
// pick a possibly different light converges to the correct multi-light image (the same
// property PBI-348's CPU test already proved analytically).
//
// Material is a single scene-global value for now (not per-instance/per-body) — full
// per-body OpenPBR material binding is deferred to the appearance-editor/catalog PBIs
// (PBI-351/352), which is where materials become widely author-able; this PBI's job is
// getting Realistic mode's *rendering pipeline* correctly wired end to end.
#extension GL_EXT_ray_tracing : require
#extension GL_EXT_ray_tracing_position_fetch : require
#extension GL_GOOGLE_include_directive : require
#include "openpbr/base_lobes.glsl"

layout(location = 0) rayPayloadInEXT vec3 payload;
layout(location = 1) rayPayloadEXT bool shadowed;

layout(set = 0, binding = 0) uniform accelerationStructureEXT tlas;

layout(set = 0, binding = 3) uniform Params {
    vec3 lightDirection; // unit vector FROM the surface TOWARD the light
    float lightIntensity;
    vec3 lightColor;
    float pad0;
    vec3 baseColor;
    float baseWeight;
    float specularRoughness;
    float specularIOR;
    float baseMetalness; // unused by base_lobes.glsl's dielectric-only path (see pathtrace.rchit's note)
    float pad1;
} params;

void buildBasis(vec3 n, out vec3 tangent, out vec3 bitangent) {
    vec3 up = abs(n.y) < 0.99 ? vec3(0, 1, 0) : vec3(1, 0, 0);
    tangent = normalize(cross(up, n));
    bitangent = cross(n, tangent);
}

void main() {
    vec3 hitPoint = gl_WorldRayOriginEXT + gl_WorldRayDirectionEXT * gl_HitTEXT;
    vec3 e1 = gl_HitTriangleVertexPositionsEXT[1] - gl_HitTriangleVertexPositionsEXT[0];
    vec3 e2 = gl_HitTriangleVertexPositionsEXT[2] - gl_HitTriangleVertexPositionsEXT[0];
    vec3 normal = normalize(cross(e1, e2));
    if (dot(normal, gl_WorldRayDirectionEXT) > 0.0) normal = -normal; // two-sided (CAD faces can present either way)

    vec3 wi = normalize(params.lightDirection);
    vec3 wo = -gl_WorldRayDirectionEXT;

    shadowed = true;
    traceRayEXT(tlas,
               gl_RayFlagsOpaqueEXT | gl_RayFlagsTerminateOnFirstHitEXT | gl_RayFlagsSkipClosestHitShaderEXT,
               0xFF, 0, 0, 1, hitPoint + normal * 1e-3, 1e-3, wi, 1e6, 1); // directional light: effectively at infinity
    if (shadowed) {
        payload = vec3(0.0);
        return;
    }

    vec3 tangent, bitangent;
    buildBasis(normal, tangent, bitangent);
    vec3 wiLocal = vec3(dot(wi, tangent), dot(wi, bitangent), dot(wi, normal));
    vec3 woLocal = vec3(dot(wo, tangent), dot(wo, bitangent), dot(wo, normal));
    if (wiLocal.z <= 0.0 || woLocal.z <= 0.0) {
        payload = vec3(0.0);
        return;
    }

    float alpha = openpbrAlphaFromRoughness(params.specularRoughness);
    vec3 diffuse = openpbrDiffuseSingleScatter(params.baseColor * params.baseWeight, 0.0, wiLocal, woLocal);

    vec3 h = normalize(wiLocal + woLocal);
    float d = openpbrDistributionGGX(h, alpha);
    float g = openpbrSmithG2(wiLocal, woLocal, alpha);
    float fr = openpbrDielectricFresnel(params.specularIOR, max(dot(wiLocal, h), 0.0));
    float specular = fr * d * g / (4.0 * wiLocal.z * woLocal.z);

    vec3 brdf = diffuse + vec3(specular);
    float cosTheta = wiLocal.z;
    payload = brdf * params.lightColor * params.lightIntensity * cosTheta;
}
