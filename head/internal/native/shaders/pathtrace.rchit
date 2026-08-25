#version 460
// Closest-hit (M45-F04 PBI-345): single-bounce direct lighting from one point light,
// evaluated with the F03 GLSL BSDF library (openpbr/base_lobes.glsl) — base diffuse
// (EON single-scatter term) + specular dielectric (GGX). Position fetch supplies the
// hit triangle's exact vertices for the geometric normal (same technique as
// raytrace.comp's ray-query path, PBI-333). Per-material hit-group dispatch to other
// OpenPBR lobes (coat/fuzz/thin-film/subsurface/transmission, PBI-341..344) is a
// documented follow-up: this harness proves the base lobe pipeline end to end first.
#extension GL_EXT_ray_tracing : require
#extension GL_EXT_ray_tracing_position_fetch : require
#extension GL_GOOGLE_include_directive : require
#include "openpbr/base_lobes.glsl"

layout(location = 0) rayPayloadInEXT vec3 payload;
layout(location = 1) rayPayloadEXT bool shadowed;

layout(set = 0, binding = 0) uniform accelerationStructureEXT tlas;

layout(set = 0, binding = 3) uniform Params {
    vec3 lightPos;
    float lightIntensity;
    vec3 lightColor;
    float pad0;
    vec3 baseColor;
    float baseWeight;
    float specularRoughness;
    float specularIOR;
    float baseMetalness; // unused by this base-lobes-only harness; reserved for the
                         // conductor branch once per-material dispatch lands
    float pad1;
} params;

// buildBasis constructs an arbitrary orthonormal (tangent, bitangent) pair around n —
// exact orientation doesn't matter for an isotropic BSDF (every lobe this harness
// evaluates is isotropic), only that it's orthonormal.
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

    vec3 toLight = params.lightPos - hitPoint;
    float dist = length(toLight);
    vec3 wi = toLight / dist;
    vec3 wo = -gl_WorldRayDirectionEXT;

    shadowed = true;
    traceRayEXT(tlas,
               gl_RayFlagsOpaqueEXT | gl_RayFlagsTerminateOnFirstHitEXT | gl_RayFlagsSkipClosestHitShaderEXT,
               0xFF, 0, 0, 1, hitPoint + normal * 1e-3, 1e-3, wi, dist - 2e-3, 1);
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
    // base_diffuse_roughness is a distinct OpenPBR parameter from specular_roughness
    // (types.OpenPBRBase.DiffuseRoughness) that this minimal harness doesn't expose yet
    // (Params has no field for it) — fixed at 0 (Lambertian, the spec's own default)
    // rather than incorrectly reusing specularRoughness for an unrelated lobe.
    vec3 diffuse = openpbrDiffuseSingleScatter(params.baseColor * params.baseWeight, 0.0, wiLocal, woLocal);

    vec3 h = normalize(wiLocal + woLocal);
    float d = openpbrDistributionGGX(h, alpha);
    float g = openpbrSmithG2(wiLocal, woLocal, alpha);
    float fr = openpbrDielectricFresnel(params.specularIOR, max(dot(wiLocal, h), 0.0));
    float specular = fr * d * g / (4.0 * wiLocal.z * woLocal.z);

    vec3 brdf = diffuse + vec3(specular);
    float attenuation = params.lightIntensity / (dist * dist);
    float cosTheta = wiLocal.z; // wi already expressed in the local frame, so z IS cosθ
    payload = brdf * params.lightColor * attenuation * cosTheta;
}
