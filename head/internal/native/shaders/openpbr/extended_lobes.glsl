// OpenPBR Surface extended lobes (#2148): GLSL translation of the Go CPU reference
// kernel/shading/openpbr/{coat,fuzz,thinfilm,transmission,subsurface}.go, ported into the
// live path tracer following base_lobes.glsl's own precedent (this #include's it, and
// reuses its GGX/Fresnel/diffuse primitives). Every function here has a byte-identical Go
// counterpart with CPU-reference oracle tests — keep the two in lockstep by hand.
//
// Two lobes are simplified relative to their CPU reference, each following an EXISTING
// precedent this file's sibling already set for the SAME class of problem (skipping an
// expensive/architecturally-unavailable term rather than inventing a new approximation):
//
//   - Fuzz's energy-compensation term (fuzzScalarAlbedo, a 24x24 hemisphere quadrature)
//     is skipped, exactly mirroring base_lobes.glsl's own documented skip of Kulla-Conty
//     multi-scatter compensation for the SAME reason ("too expensive to run per-shading-
//     sample", multiscatter.go's own doc comment). LayerFuzz here is a plain weighted mix,
//     not the exact energy-normalized blend.
//   - Subsurface is a LOCAL diffuse-albedo approximation (SubsurfaceSingleScatterAlbedo
//     fed through the same single-scatter diffuse shape base_lobes.glsl already has), not
//     a spatial (screen-space-blurred) diffusion profile — subsurface_radius has no
//     visible effect (no blur pass exists in this per-pixel-only architecture; a true
//     Christensen-Burley-style pass needs a G-buffer + separable blur post-process this
//     renderer does not build here). Still correctly colored/energy-preserving per the
//     van de Hulst inversion, just not spatially spread.
//
// Coat and Thin-film are full, unsimplified ports (both are purely local — no hemisphere
// integral, no extra ray).
//
// Transmission (#2155) IS wired in with a real continuation ray, traced by each backend's
// own main()/trace loop (openpbrShadeBaseSubstrate/openpbrBaseSpecular below expose the
// pieces a caller needs to splice a traced translucent contribution in before coat/fuzz
// layering — see transmission_lobe.glsl's header for why the actual traceRayEXT/BVH-walk
// control flow can't live in this shared, ray-tracing-agnostic file).

#ifndef OPENPBR_EXTENDED_LOBES_GLSL
#define OPENPBR_EXTENDED_LOBES_GLSL

#include "base_lobes.glsl"
#include "transmission_lobe.glsl"

// OPENPBR_MAX_TRANSMISSION_BOUNCES bounds how many continuation rays a single primary hit
// may chain through transmissive surfaces (entering + exiting a solid, or passing through
// a thin-walled sheet, each count as one bounce) — both backends' trace loops enforce
// this, and the HW backend's requested maxPipelineRayRecursionDepth (raytrace.cpp) is
// sized directly from it (1 primary + this many continuations + 1 terminal shadow ray).
const int OPENPBR_MAX_TRANSMISSION_BOUNCES = 4;

const float OPENPBR_MIN_DENOM = 1e-9;

// --- Complex arithmetic (vec2 = re,im) — GLSL has no native complex type. Only cxMul/
// cxDivSafe/cxSqrt need custom implementations; '+'/'-' and real*complex already work
// with GLSL's native component-wise vec2 operators. ---

vec2 openpbrCxMul(vec2 a, vec2 b) { return vec2(a.x * b.x - a.y * b.y, a.x * b.y + a.y * b.x); }

// openpbrCxDivSafe mirrors Adobe's/this repo's safeComplexDivide: returns 1+0i (not 0)
// when the denominator's magnitude is below OPENPBR_MIN_DENOM.
vec2 openpbrCxDivSafe(vec2 numer, vec2 denom) {
    float d2 = dot(denom, denom);
    if (sqrt(d2) < OPENPBR_MIN_DENOM) return vec2(1.0, 0.0);
    return vec2(numer.x * denom.x + numer.y * denom.y, numer.y * denom.x - numer.x * denom.y) / d2;
}

vec2 openpbrCxSqrt(vec2 z) {
    float r = length(z);
    float re = sqrt(max(0.0, (r + z.x) * 0.5));
    float im = sqrt(max(0.0, (r - z.x) * 0.5));
    return vec2(re, z.y < 0.0 ? -im : im);
}

// --- Coat (coat.go) ---

// openpbrDielectricAverageFresnel is the closed-form cosine-weighted hemispherical
// average of openpbrDielectricFresnel (fresnel.go's DielectricAverageFresnel).
float openpbrDielectricAverageFresnel(float iorRatio) {
    if (iorRatio > 1.0) return (iorRatio - 1.0) / (4.08567 + 1.00071 * iorRatio);
    return 0.997118 + 0.1014 * iorRatio - 0.965241 * iorRatio * iorRatio
         - 0.130607 * iorRatio * iorRatio * iorRatio;
}

// openpbrSpecularCoat is the coat's own single-scatter GGX+exact-Fresnel reflection
// (coat.go's SpecularCoat = SpecularDielectric at the coat's own roughness/ior) — the
// Kulla-Conty multi-scatter term SpecularDielectric also adds is skipped here, the same
// simplification base_lobes.glsl's openpbrDiffuseSingleScatter/specular already make.
float openpbrSpecularCoat(vec3 wi, vec3 wo, float roughness, float ior) {
    float cosI = wi.z, cosO = wo.z;
    if (cosI <= 0.0 || cosO <= 0.0) return 0.0;
    float alpha = openpbrAlphaFromRoughness(roughness);
    vec3 h = normalize(wi + wo);
    float d = openpbrDistributionGGX(h, alpha);
    float g = openpbrSmithG2(wi, wo, alpha);
    float fr = openpbrDielectricFresnel(ior, abs(dot(wi, h)));
    return fr * d * g / (4.0 * cosI * cosO);
}

// openpbrCoatDarkeningFactor ports coat.go's CoatDarkeningFactor exactly, using the
// Lambertian/rough-base K_r form (not the smooth/rough blend). baseAlbedoNormal is fixed
// at 1.0 by every caller here — see this file's callers' own note: it matches the base
// diffuse lobe's own roughness=0 EON normalization already in use, under which E_b=1.
float openpbrCoatDarkeningFactor(float baseAlbedoNormal, float weight, float coatDarkening, float ior) {
    if (weight <= 0.0 || coatDarkening <= 0.0) return 1.0;
    float fAvg = openpbrDielectricAverageFresnel(ior);
    float k = 1.0 - (1.0 - fAvg) / (ior * ior);
    float delta = (1.0 - k) / max(1e-4, 1.0 - baseAlbedoNormal * k);
    return mix(1.0, delta, clamp(weight * coatDarkening, 0.0, 1.0));
}

// openpbrLayerCoat ports coat.go's LayerCoat exactly: f_layer = f_coat + (1-E_coat(wo))
// * darkening * fSub * T_coat, blended in by weight.
vec3 openpbrLayerCoat(float fCoat, vec3 fSub, vec3 coatColor, float weight, float darkening,
                      float woZ, float ior) {
    if (weight <= 0.0) return fSub;
    float eCoat = openpbrDielectricFresnel(ior, max(woZ, 0.0));
    vec3 tCoat = sqrt(max(coatColor, vec3(0.0)));
    vec3 layered = fSub * tCoat * ((1.0 - eCoat) * darkening) + vec3(fCoat);
    return mix(fSub, layered, weight);
}

// --- Fuzz (fuzz.go) ---

float openpbrSheenDistributionCharlie(float sinThetaH, float alpha) {
    float a = max(alpha, 1e-3);
    float invAlpha = 1.0 / a;
    return (2.0 + invAlpha) * pow(sinThetaH, invAlpha) / (2.0 * OPENPBR_PI);
}

float openpbrVisibilityNeubelt(float cosI, float cosO) {
    return 1.0 / (4.0 * (cosI + cosO - cosI * cosO));
}

vec3 openpbrSpecularFuzz(vec3 wi, vec3 wo, float roughness, vec3 color) {
    float cosI = wi.z, cosO = wo.z;
    if (cosI <= 0.0 || cosO <= 0.0) return vec3(0.0);
    vec3 h = normalize(wi + wo);
    float sinThetaH = sqrt(max(0.0, 1.0 - h.z * h.z));
    float d = openpbrSheenDistributionCharlie(sinThetaH, roughness);
    float v = openpbrVisibilityNeubelt(cosI, cosO);
    return color * (d * v);
}

// openpbrLayerFuzz is a plain weighted mix, NOT fuzz.go's exact energy-normalized
// LayerFuzz (which needs fuzzScalarAlbedo's per-pixel hemisphere quadrature) — see this
// file's header note.
vec3 openpbrLayerFuzz(vec3 wi, vec3 wo, float roughness, vec3 color, float weight, vec3 coatedBase) {
    if (weight <= 0.0) return coatedBase;
    vec3 fFuzz = openpbrSpecularFuzz(wi, wo, roughness, color);
    return mix(coatedBase, fFuzz, weight);
}

// --- Thin-film (thinfilm.go) ---

// openpbrRgbWavelengthsNM mirrors rgbWavelengthsNM: the fixed R/G/B representative
// wavelengths (nm) the Airy summation evaluates at.
const vec3 OPENPBR_RGB_WAVELENGTHS_NM = vec3(611.0, 549.0, 466.0);

float openpbrSafeDenom(float d) {
    if (abs(d) < OPENPBR_MIN_DENOM) return d >= 0.0 ? OPENPBR_MIN_DENOM : -OPENPBR_MIN_DENOM;
    return d;
}

void openpbrFresnelAmplitudeDielectric(float cosThetaI, float etaI, float etaT,
        out float rs, out float rp, out float ts, out float tp, out float cosThetaT) {
    float sinThetaI = sqrt(clamp(1.0 - cosThetaI * cosThetaI, 0.0, 1.0));
    float sinThetaT = etaI / etaT * sinThetaI;
    if (sinThetaT >= 1.0) { rs = 1.0; rp = 1.0; ts = 0.0; tp = 0.0; cosThetaT = 0.0; return; }
    cosThetaT = sqrt(clamp(1.0 - sinThetaT * sinThetaT, 0.0, 1.0));
    float etaICosThetaI = etaI * cosThetaI;
    float etaTCosThetaT = etaT * cosThetaT;
    float etaTCosThetaI = etaT * cosThetaI;
    float etaICosThetaT = etaI * cosThetaT;
    float denomS = openpbrSafeDenom(etaICosThetaI + etaTCosThetaT);
    float denomP = openpbrSafeDenom(etaTCosThetaI + etaICosThetaT);
    rs = (etaICosThetaI - etaTCosThetaT) / denomS;
    rp = (etaTCosThetaI - etaICosThetaT) / denomP;
    ts = (2.0 * etaICosThetaI) / denomS;
    tp = (2.0 * etaICosThetaI) / denomP;
}

vec2 openpbrSnellCosComplex(float cosThetaI, float etaI, vec2 etaT) {
    float sinThetaI = sqrt(clamp(1.0 - cosThetaI * cosThetaI, 0.0, 1.0));
    vec2 etaIOverEtaT = openpbrCxDivSafe(vec2(etaI, 0.0), etaT);
    vec2 sinThetaT = openpbrCxMul(vec2(sinThetaI, 0.0), etaIOverEtaT);
    vec2 cosThetaTSq = vec2(1.0, 0.0) - openpbrCxMul(sinThetaT, sinThetaT);
    vec2 cosThetaT = openpbrCxSqrt(cosThetaTSq);
    if (openpbrCxMul(etaT, cosThetaT).y < 0.0) cosThetaT = -cosThetaT;
    return cosThetaT;
}

void openpbrFresnelAmplitudeComplex(float cosThetaI, float etaI, vec2 etaT, out vec2 rs, out vec2 rp) {
    vec2 cosThetaT = openpbrSnellCosComplex(cosThetaI, etaI, etaT);
    vec2 etaICosThetaI = vec2(etaI * cosThetaI, 0.0);
    vec2 etaTCosThetaT = openpbrCxMul(etaT, cosThetaT);
    vec2 etaTCosThetaI = etaT * cosThetaI;
    vec2 etaICosThetaT = etaI * cosThetaT;
    rs = openpbrCxDivSafe(etaICosThetaI - etaTCosThetaT, etaICosThetaI + etaTCosThetaT);
    rp = openpbrCxDivSafe(etaTCosThetaI - etaICosThetaT, etaTCosThetaI + etaICosThetaT);
}

float openpbrAiryReflectance(float r12, float t12, float r21, float t21, vec2 r23, vec2 expIDeltaPhi) {
    vec2 r23Exp = openpbrCxMul(r23, expIDeltaPhi);
    vec2 numerator = openpbrCxMul(vec2(t12 * t21, 0.0), r23Exp);
    vec2 denominator = vec2(1.0, 0.0) - openpbrCxMul(vec2(r21, 0.0), r23Exp);
    vec2 rTotal = vec2(r12, 0.0) + openpbrCxDivSafe(numerator, denominator);
    float m = length(rTotal);
    return m * m;
}

// openpbrThinFilmAiryAmplitudes packs thinFilmAmplitudes' fields into out params (GLSL
// has no multi-value struct return ergonomics as light as Go's); returns false on TIR
// at the outer (exterior->film) interface, matching thinFilmAiryAmplitudes' ok=false.
bool openpbrThinFilmAiryAmplitudes(float cosThetaI, float etaExterior, float etaFilm, float etaBase,
        float thicknessNM, out float r12s, out float r12p, out float t12s, out float t12p,
        out float r21s, out float r21p, out float t21s, out float t21p,
        out vec2 r23s, out vec2 r23p, out float opd) {
    float cosThetaTFilm;
    openpbrFresnelAmplitudeDielectric(cosThetaI, etaExterior, etaFilm, r12s, r12p, t12s, t12p, cosThetaTFilm);
    if (cosThetaTFilm <= 0.0) return false;
    r21s = -r12s; r21p = -r12p;
    float safeCosI = max(cosThetaI, OPENPBR_MIN_DENOM);
    float t21Scale = (etaFilm * cosThetaTFilm) / (etaExterior * safeCosI);
    t21s = t12s * t21Scale; t21p = t12p * t21Scale;
    openpbrFresnelAmplitudeComplex(cosThetaTFilm, etaFilm, vec2(etaBase, 0.0), r23s, r23p);
    opd = 2.0 * etaFilm * thicknessNM * cosThetaTFilm;
    return true;
}

// openpbrThinFilmReflectanceDielectric ports thinfilm.go's ThinFilmReflectanceDielectric
// exactly: Airy-summed interference reflectance at 3 representative wavelengths.
vec3 openpbrThinFilmReflectanceDielectric(float cosThetaI, float etaExterior, float etaFilm,
        float etaBase, float thicknessMicrons) {
    if (thicknessMicrons <= 0.0) return vec3(0.0);
    float thicknessNM = thicknessMicrons * 1000.0;
    float presence = smoothstep(OPENPBR_MIN_DENOM, 30.0, thicknessNM);

    float r12s, r12p, t12s, t12p, r21s, r21p, t21s, t21p, opd;
    vec2 r23s, r23p;
    bool ok = openpbrThinFilmAiryAmplitudes(cosThetaI, etaExterior, etaFilm, etaBase, thicknessNM,
        r12s, r12p, t12s, t12p, r21s, r21p, t21s, t21p, r23s, r23p, opd);
    if (!ok) return vec3(presence);

    vec3 result;
    for (int ch = 0; ch < 3; ch++) {
        float lambdaNM = OPENPBR_RGB_WAVELENGTHS_NM[ch];
        float phase = 2.0 * OPENPBR_PI * opd / lambdaNM;
        vec2 expI = vec2(cos(phase), sin(phase));
        float rs = openpbrAiryReflectance(r12s, t12s, r21s, t21s, r23s, expI);
        float rp = openpbrAiryReflectance(r12p, t12p, r21p, t21p, r23p, expI);
        result[ch] = presence * 0.5 * (rs + rp);
    }
    return result;
}

// openpbrFresnelWithThinFilm ports thinfilm.go's FresnelWithThinFilm exactly.
vec3 openpbrFresnelWithThinFilm(float cosThetaI, float ior, float filmIOR, float thicknessMicrons, float weight) {
    vec3 plain = vec3(openpbrDielectricFresnel(ior, cosThetaI));
    if (weight <= 0.0) return plain;
    vec3 film = openpbrThinFilmReflectanceDielectric(cosThetaI, 1.0, filmIOR, ior, thicknessMicrons);
    return mix(plain, film, weight);
}

// --- Subsurface (subsurface.go), local single-scatter-albedo approximation — see this
// file's header note on the missing spatial blur. ---

float openpbrSubsurfaceAlbedoFromColorChannel(float c, float g) {
    float s = 4.09712 + 4.20863 * c - sqrt(max(0.0, 9.59217 + 41.6808 * c + 17.7126 * c * c));
    float sSq = s * s;
    float denom = 1.0 - g * sSq;
    if (abs(denom) < 1e-9) return 0.0;
    return (1.0 - sSq) / denom;
}

// openpbrSubsurfaceSingleScatterAlbedo ports subsurface.go's SubsurfaceSingleScatterAlbedo
// (the van de Hulst inversion) exactly, per RGB channel.
vec3 openpbrSubsurfaceSingleScatterAlbedo(vec3 color, float anisotropy) {
    return vec3(
        openpbrSubsurfaceAlbedoFromColorChannel(color.r, anisotropy),
        openpbrSubsurfaceAlbedoFromColorChannel(color.g, anisotropy),
        openpbrSubsurfaceAlbedoFromColorChannel(color.b, anisotropy));
}

// openpbrSubsurfaceLocalApprox feeds the van de Hulst albedo through the SAME
// single-scatter diffuse shape base_lobes.glsl's openpbrDiffuseSingleScatter already
// uses (roughness=0, matching the base diffuse call's own convention) — a physically-
// motivated but spatially-local (zero mean-free-path) stand-in for the true volumetric
// random walk (see this file's header).
vec3 openpbrSubsurfaceLocalApprox(vec3 subsurfaceColor, float anisotropy, vec3 wiLocal, vec3 woLocal) {
    vec3 albedo = openpbrSubsurfaceSingleScatterAlbedo(subsurfaceColor, anisotropy);
    return openpbrDiffuseSingleScatter(albedo, 0.0, wiLocal, woLocal);
}

// openpbrMixSubsurface ports subsurface.go's MixSubsurface exactly.
vec3 openpbrMixSubsurface(vec3 diffuse, vec3 subsurface, float weight) {
    if (weight <= 0.0) return diffuse;
    return mix(diffuse, subsurface, weight);
}

// --- Shading combination + shared Params UBO field list ---
//
// OPENPBR_REALISTIC_PARAMS_FIELDS is the Params uniform block BODY shared by
// pathtrace_realistic.rchit and swpathtrace_realistic.comp — each file still declares
// its own `layout(set=..., binding=...) uniform Params { OPENPBR_REALISTIC_PARAMS_FIELDS
// } params;` (their binding indices differ), but the FIELD LIST itself lives in exactly
// one place, matching raytrace.go's RealisticLightParams.floats() field order
// byte-for-byte (60 floats/240 bytes total — see that struct's own doc comment).
#define OPENPBR_REALISTIC_PARAMS_FIELDS \
    vec3 lightDirection; \
    float lightIntensity; \
    vec3 lightColor; \
    float pad0; \
    vec3 baseColor; \
    float baseWeight; \
    float specularRoughness; \
    float specularIOR; \
    float baseMetalness; \
    float pad1; \
    vec3 coatColor; \
    float coatWeight; \
    float coatRoughness; \
    float coatIOR; \
    float coatDarkening; \
    float pad2; \
    vec3 fuzzColor; \
    float fuzzWeight; \
    float fuzzRoughness; \
    float pad3; \
    float pad4; \
    float pad5; \
    float thinFilmWeight; \
    float thinFilmThicknessMicrons; \
    float thinFilmIOR; \
    float pad6; \
    vec3 transmissionColor; \
    float transmissionWeight; \
    float transmissionDepth; \
    float dispersionScale; \
    float dispersionAbbeNumber; \
    float thinWalled; \
    vec3 subsurfaceColor; \
    float subsurfaceWeight; \
    vec3 subsurfaceRadiusScale; \
    float subsurfaceRadius; \
    float subsurfaceAnisotropy; \
    float envEnabled; \
    float envRotation; \
    float envIntensity; \
    float lightIsEnvironment; \
    float pad11; \
    float pad12; \
    float pad13;

// OpenPBRRealisticMaterial bundles the subset of a Params instance
// openpbrShadeSurface needs, so it can be a plain value parameter (a struct TYPE
// definition, unlike a global-variable reference, needs no `params` instance to already
// exist — safe to use before each shader's own Params block is declared).
struct OpenPBRRealisticMaterial {
    vec3 baseColor;
    float baseWeight;
    float specularRoughness, specularIOR;
    vec3 coatColor;
    float coatWeight, coatRoughness, coatIOR, coatDarkening;
    vec3 fuzzColor;
    float fuzzWeight, fuzzRoughness;
    float thinFilmWeight, thinFilmThicknessMicrons, thinFilmIOR;
    vec3 subsurfaceColor;
    float subsurfaceWeight, subsurfaceAnisotropy;
    vec3 transmissionColor;
    float transmissionWeight, transmissionDepth, dispersionScale, dispersionAbbeNumber;
    bool thinWalled;
};

// openpbrBaseSpecular is the base substrate's own dielectric/metal GGX+Fresnel reflection
// term (single-scatter, thin-film-modulated) — split out of openpbrShadeSurface (#2155)
// so a caller building a translucent base substrate (specular reflection PLUS a traced
// transmitted contribution, spec's S_translucent-base) can reuse it without recomputing
// the same GGX/Fresnel evaluation twice.
vec3 openpbrBaseSpecular(OpenPBRRealisticMaterial m, vec3 wiLocal, vec3 woLocal) {
    float alpha = openpbrAlphaFromRoughness(m.specularRoughness);
    vec3 h = normalize(wiLocal + woLocal);
    float d = openpbrDistributionGGX(h, alpha);
    float g = openpbrSmithG2(wiLocal, woLocal, alpha);
    float cosH = max(dot(wiLocal, h), 0.0);
    vec3 fr = openpbrFresnelWithThinFilm(cosH, m.specularIOR, m.thinFilmIOR,
                                         m.thinFilmThicknessMicrons, m.thinFilmWeight);
    return fr * (d * g / (4.0 * wiLocal.z * woLocal.z));
}

// openpbrShadeBaseSubstrate is the OPAQUE base substrate — diffuse(+subsurface)+specular,
// with NO transmission (spec's M_opaque-base) — split out of openpbrShadeSurface (#2155)
// so a caller with real ray-tracing capability can mix it against a translucent
// alternative (openpbrMixTransmission) before layering coat/fuzz. This file has no
// traceRay capability itself (also #include'd by the SW compute shader, which has none)
// — the actual continuation-ray trace that builds the translucent alternative lives in
// each backend's own main()/trace loop, not here.
vec3 openpbrShadeBaseSubstrate(OpenPBRRealisticMaterial m, vec3 wiLocal, vec3 woLocal) {
    vec3 diffuse = openpbrDiffuseSingleScatter(m.baseColor * m.baseWeight, 0.0, wiLocal, woLocal);
    vec3 specular = openpbrBaseSpecular(m, wiLocal, woLocal);

    vec3 diffuseSlab = diffuse;
    if (m.subsurfaceWeight > 0.0) {
        vec3 subsurface = openpbrSubsurfaceLocalApprox(m.subsurfaceColor, m.subsurfaceAnisotropy, wiLocal, woLocal);
        diffuseSlab = openpbrMixSubsurface(diffuse, subsurface, m.subsurfaceWeight);
    }
    return diffuseSlab + specular;
}

// openpbrLayerCoatFuzz layers coat then fuzz over an already-computed base substrate
// value (opaque, translucent, or a transmission_weight mix of the two) — split out of
// openpbrShadeSurface (#2155) so callers that build their own base (see
// openpbrShadeBaseSubstrate's doc comment) don't duplicate this layering.
vec3 openpbrLayerCoatFuzz(vec3 base, OpenPBRRealisticMaterial m, vec3 wiLocal, vec3 woLocal) {
    float fCoat = openpbrSpecularCoat(wiLocal, woLocal, m.coatRoughness, m.coatIOR);
    float darkening = openpbrCoatDarkeningFactor(1.0, m.coatWeight, m.coatDarkening, m.coatIOR);
    vec3 coated = openpbrLayerCoat(fCoat, base, m.coatColor, m.coatWeight, darkening, woLocal.z, m.coatIOR);
    return openpbrLayerFuzz(wiLocal, woLocal, m.fuzzRoughness, m.fuzzColor, m.fuzzWeight, coated);
}

// openpbrShadeSurface combines every lobe into one local reflected-radiance BRDF value
// (excluding light color/intensity/cosine — the caller applies those), in OpenPBR's own
// layering order: base (diffuse+specular, thin-film modulating the specular Fresnel) ->
// subsurface replaces diffuse only -> coat layers over the whole base -> fuzz layers over
// the coated result. Every extended lobe's weight=0 reproduces the PRIOR stage's output
// exactly (each Layer*/Mix* helper's own weight<=0 short-circuit), so a material with
// every extended field zeroed renders bit-identical to the pre-#2148 base-lobes-only
// shading — the same regression guard kernel/shading/openpbr's own CPU tests use. This is
// the NO-transmission composition (m.transmissionWeight ignored) — callers wiring in a
// traced translucent contribution use openpbrShadeBaseSubstrate/openpbrLayerCoatFuzz
// directly instead (pathtrace_realistic.rchit, swpathtrace_realistic.comp).
vec3 openpbrShadeSurface(OpenPBRRealisticMaterial m, vec3 wiLocal, vec3 woLocal) {
    return openpbrLayerCoatFuzz(openpbrShadeBaseSubstrate(m, wiLocal, woLocal), m, wiLocal, woLocal);
}

#endif // OPENPBR_EXTENDED_LOBES_GLSL
