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
//
// Transmission (#2155): when the material is transmissive, this shader fires ONE more
// continuation ray THROUGH the surface (real Snell refraction, or a straight-through ray
// for geometry_thin_walled) by invoking traceRayEXT on ITSELF (same hit group, same
// payload location 0) — the ray tracing pipeline re-invokes this exact shader for that
// child ray, which is how real recursion happens here (GLSL forbids recursive function
// calls; this is the hardware/driver-managed shader-re-invocation stack instead). The
// SAME location-0 payload variable is reused for the recursive call: by the time it's
// fired, everything this invocation still needs (depth/iorStack/channel selection) has
// already been read into local variables, so overwriting payload's fields for the child
// call is safe — see rt_payload.glsl's own doc comment. Each level fires its own shadow
// ray exactly like a primary hit would (directLightAt), so light reaching the viewer
// through several bounces of glass is still correctly shaded/occluded at every surface,
// not just the outermost one. raytrace.cpp requests enough maxPipelineRayRecursionDepth
// for OPENPBR_MAX_TRANSMISSION_BOUNCES levels of this plus one terminal shadow ray.
#extension GL_EXT_ray_tracing : require
#extension GL_EXT_ray_tracing_position_fetch : require
#extension GL_GOOGLE_include_directive : require
#include "rt_payload.glsl"
#include "openpbr/env_sample.glsl"

layout(location = 0) rayPayloadInEXT RTPathPayload payload;
layout(location = 1) rayPayloadEXT bool shadowed;

layout(set = 0, binding = 0) uniform accelerationStructureEXT tlas;

// Field list shared with swpathtrace_realistic.comp — see extended_lobes.glsl's
// OPENPBR_REALISTIC_PARAMS_FIELDS doc comment for the authoritative field order.
layout(set = 0, binding = 3) uniform Params { OPENPBR_REALISTIC_PARAMS_FIELDS } params;
// binding 4 (#2135/#2155's illumination-contribution follow-up): the same equirect
// environment map pathtrace_realistic.rmiss samples for background visibility —
// directLightAt below samples it again here at a DIFFERENT (light-importance-sampled)
// direction when params.lightIsEnvironment says this dispatch picked the environment
// over a discrete light.
layout(set = 0, binding = 4) uniform sampler2D envMap;

void buildBasis(vec3 n, out vec3 tangent, out vec3 bitangent) {
    vec3 up = abs(n.y) < 0.99 ? vec3(0, 1, 0) : vec3(1, 0, 0);
    tangent = normalize(cross(up, n));
    bitangent = cross(n, tangent);
}

OpenPBRRealisticMaterial realisticMaterialFromParams() {
    OpenPBRRealisticMaterial mat;
    mat.baseColor = params.baseColor; mat.baseWeight = params.baseWeight;
    mat.specularRoughness = params.specularRoughness; mat.specularIOR = params.specularIOR;
    mat.coatColor = params.coatColor; mat.coatWeight = params.coatWeight;
    mat.coatRoughness = params.coatRoughness; mat.coatIOR = params.coatIOR; mat.coatDarkening = params.coatDarkening;
    mat.fuzzColor = params.fuzzColor; mat.fuzzWeight = params.fuzzWeight; mat.fuzzRoughness = params.fuzzRoughness;
    mat.thinFilmWeight = params.thinFilmWeight; mat.thinFilmThicknessMicrons = params.thinFilmThicknessMicrons;
    mat.thinFilmIOR = params.thinFilmIOR;
    mat.subsurfaceColor = params.subsurfaceColor; mat.subsurfaceWeight = params.subsurfaceWeight;
    mat.subsurfaceAnisotropy = params.subsurfaceAnisotropy;
    mat.transmissionColor = params.transmissionColor; mat.transmissionWeight = params.transmissionWeight;
    mat.transmissionDepth = params.transmissionDepth; mat.dispersionScale = params.dispersionScale;
    mat.dispersionAbbeNumber = params.dispersionAbbeNumber;
    mat.thinWalled = params.thinWalled > 0.5;
    return mat;
}

// directLightAt mirrors swpathtrace_realistic.comp's function of the same name exactly:
// opaque base mixed against the translucent base's local (specular-only) term, coat/fuzz
// layered on top, shadow-tested. The traced transmitted contribution is added by main()
// separately, not here.
vec3 directLightAt(vec3 hitPoint, vec3 normal, vec3 wiLocal, vec3 woLocal, OpenPBRRealisticMaterial mat) {
    if (wiLocal.z <= 0.0 || woLocal.z <= 0.0) return vec3(0.0);
    vec3 wi = normalize(params.lightDirection);
    shadowed = true;
    traceRayEXT(tlas,
               gl_RayFlagsOpaqueEXT | gl_RayFlagsTerminateOnFirstHitEXT | gl_RayFlagsSkipClosestHitShaderEXT,
               0xFF, 0, 0, 1, hitPoint + normal * 1e-3, 1e-3, wi, 1e6, 1); // directional light: effectively at infinity
    if (shadowed) return vec3(0.0);

    vec3 opaqueBase = openpbrShadeBaseSubstrate(mat, wiLocal, woLocal);
    vec3 translucentLocal = openpbrBaseSpecular(mat, wiLocal, woLocal);
    vec3 base = openpbrMixTransmission(opaqueBase, translucentLocal, mat.transmissionWeight);
    vec3 shaded = openpbrLayerCoatFuzz(base, mat, wiLocal, woLocal);
    // #2135/#2155: this dispatch's single light pick is EITHER a discrete light
    // (lightColor holds its premultiplied color) OR the environment (lightIsEnvironment
    // != 0 — ui/realistic_render.go's pickLightParams picked an importance-sampled
    // direction from renderer.EnvironmentDistribution instead), in which case the color
    // comes from re-sampling envMap at wi (enabled=1.0 unconditionally: the CPU only
    // ever sets lightIsEnvironment when it already knows the environment is active,
    // independent of envEnabled's own background-visibility-only ShowImage gate).
    vec3 lightRadiance = params.lightIsEnvironment > 0.5
        ? openpbrSampleEnvironment(envMap, wi, 1.0, params.envRotation, params.envIntensity)
        : params.lightColor;
    return shaded * lightRadiance * params.lightIntensity * wiLocal.z;
}

void main() {
    // Capture everything this invocation needs from the incoming payload BEFORE it gets
    // overwritten (for a recursive continuation call, or for this invocation's own
    // result) — see this file's header comment on reusing location 0.
    int depth = payload.depth;
    int iorStackDepth = payload.iorStackDepth;
    float iorStack[OPENPBR_MAX_TRANSMISSION_BOUNCES + 2];
    for (int i = 0; i < iorStack.length(); i++) iorStack[i] = payload.iorStack[i];
    float channelIOR = payload.channelIOR;
    vec3 channelMask = payload.channelMask;

    vec3 hitPoint = gl_WorldRayOriginEXT + gl_WorldRayDirectionEXT * gl_HitTEXT;
    vec3 e1 = gl_HitTriangleVertexPositionsEXT[1] - gl_HitTriangleVertexPositionsEXT[0];
    vec3 e2 = gl_HitTriangleVertexPositionsEXT[2] - gl_HitTriangleVertexPositionsEXT[0];
    vec3 rawNormal = normalize(cross(e1, e2));
    vec3 normal = rawNormal;
    if (dot(normal, gl_WorldRayDirectionEXT) > 0.0) normal = -normal; // two-sided (CAD faces can present either way)

    vec3 wi = normalize(params.lightDirection);
    vec3 wo = -gl_WorldRayDirectionEXT;
    vec3 tangent, bitangent;
    buildBasis(normal, tangent, bitangent);
    vec3 wiLocal = vec3(dot(wi, tangent), dot(wi, bitangent), dot(wi, normal));
    vec3 woLocal = vec3(dot(wo, tangent), dot(wo, bitangent), dot(wo, normal));

    OpenPBRRealisticMaterial mat = realisticMaterialFromParams();

    vec3 radiance = channelMask * directLightAt(hitPoint, normal, wiLocal, woLocal, mat);

    if (mat.transmissionWeight > 0.0 && depth < OPENPBR_MAX_TRANSMISSION_BOUNCES && woLocal.z > 0.0) {
        // Entering/exiting is decided from the RAW (un-flipped) geometric normal — the
        // shading normal above always opposes the ray for two-sided convenience, which
        // would make every hit look like "entering" and break nested-medium tracking.
        bool entering = dot(rawNormal, gl_WorldRayDirectionEXT) < 0.0;
        float iorRatio;
        if (entering) {
            iorRatio = channelIOR / iorStack[iorStackDepth - 1];
            if (iorStackDepth < iorStack.length()) { iorStack[iorStackDepth] = channelIOR; iorStackDepth++; }
        } else {
            if (iorStackDepth > 1) iorStackDepth--;
            iorRatio = iorStack[iorStackDepth - 1] / channelIOR;
        }

        vec3 continueLocal;
        float transmittance;
        bool refracted = true;
        if (mat.thinWalled) {
            transmittance = openpbrThinWallTransmittance(channelIOR, woLocal.z);
            continueLocal = -woLocal; // straight through, undeviated (spec's thin-walled series)
        } else {
            refracted = openpbrRefract(woLocal, iorRatio, continueLocal);
            transmittance = 1.0 - openpbrDielectricFresnel(channelIOR, woLocal.z);
        }

        if (refracted) { // false = total internal reflection: no transmitted ray, chain ends here
            vec3 continueDirWorld = normalize(continueLocal.x * tangent + continueLocal.y * bitangent + continueLocal.z * normal);
            vec3 continueOrigin = hitPoint - normal * 1e-3; // offset to the transmitted side of the surface

            payload.depth = depth + 1;
            payload.iorStackDepth = iorStackDepth;
            for (int i = 0; i < iorStack.length(); i++) payload.iorStack[i] = iorStack[i];
            payload.channelIOR = channelIOR;
            payload.channelMask = channelMask;
            traceRayEXT(tlas, gl_RayFlagsOpaqueEXT, 0xFF, 0, 0, 0, continueOrigin, 1e-3, continueDirWorld, 1e6, 0);

            vec3 childRadiance = payload.radiance;
            float childDist = payload.hitDistance;
            vec3 extinction = (entering && !mat.thinWalled && mat.transmissionDepth > 0.0)
                ? openpbrTransmissionExtinction(mat.transmissionColor, mat.transmissionDepth) : vec3(0.0);
            radiance += mat.transmissionWeight * transmittance * (childRadiance * exp(-extinction * childDist));
        }
    }

    payload.radiance = radiance;
    payload.hitDistance = gl_HitTEXT;
}
