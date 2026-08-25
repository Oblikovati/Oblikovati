// OpenPBR Transmission pure math (#2155): GLSL translation of the Go CPU reference
// kernel/shading/openpbr/transmission.go. Split out of extended_lobes.glsl (which
// #include's this) to keep that file under the project's per-file line budget — every
// function here has a byte-identical Go counterpart with CPU-reference oracle tests, kept
// in lockstep by hand like the rest of this shader library.
//
// Unlike coat/fuzz/thin-film (purely local reflection terms), transmission needs a real
// continuation ray traced through the surface — that ray-tracing control flow does NOT
// live here (this file is also #include'd by the SW compute backend, which has no
// traceRayEXT). Only the pure math each backend's own recursive/iterative trace loop
// calls — Fresnel/transmittance, Snell refraction, Beer's-law extinction, dispersion — is
// shared here.

#ifndef OPENPBR_TRANSMISSION_LOBE_GLSL
#define OPENPBR_TRANSMISSION_LOBE_GLSL

// --- Thin-walled evaluation mode (spec: a geometrical series over internal reflections
// in an infinitesimally thin dielectric sheet) ---

// openpbrThinWallFresnel ports transmission.go's ThinWallFresnel exactly: the closed-form
// sum of a thin sheet's internal-reflection series, 2R/(1+R).
float openpbrThinWallFresnel(float ior, float cosThetaI) {
    float r = openpbrDielectricFresnel(ior, cosThetaI);
    return (2.0 * r) / (1.0 + r);
}

// openpbrThinWallTransmittance ports transmission.go's ThinWallTransmittance exactly:
// (1-R)/(1+R), the fraction passing straight through undeflected.
float openpbrThinWallTransmittance(float ior, float cosThetaI) {
    float r = openpbrDielectricFresnel(ior, cosThetaI);
    return (1.0 - r) / (1.0 + r);
}

// --- Transmission color/depth -> volumetric extinction (Beer's law) ---

// openpbrTransmissionExtinction ports transmission.go's TransmissionExtinction (Beer's
// law inverted): the per-channel extinction coefficient such that white light becomes
// exactly transmissionColor after traveling transmissionDepth through the medium.
// depth<=0 means the interior medium is absent, so this returns zero extinction (the
// caller's transmissionColor is used as a plain multiplicative tint instead — see
// wherever this is applied over a REAL traced distance in each backend's trace loop).
vec3 openpbrTransmissionExtinction(vec3 color, float depth) {
    if (depth <= 0.0) return vec3(0.0);
    vec3 clamped = max(color, vec3(1e-6));
    return -log(clamped) / depth;
}

// --- Vector Snell refraction (standard form, e.g. Pharr/Jakob/Humphreys "Physically
// Based Rendering" — the OpenPBR spec's own #Pharr2023 citation) ---

// openpbrRefract ports transmission.go's Refract exactly: the refracted direction for wi
// (LOCAL shading space, z-up, pointing AWAY from the surface — this file's wi/wo
// convention throughout, matching base_lobes.glsl) crossing into a medium of relative IOR
// iorRatio = eta_t/eta_i. Returns false under total internal reflection (no refracted
// ray) — the caller falls back to reflection-only in that case.
bool openpbrRefract(vec3 wi, float iorRatio, out vec3 wt) {
    float cosThetaI = wi.z;
    float etaRel = 1.0 / iorRatio; // eta_i/eta_t, the ratio Snell's law scales the tangential component by
    float sinThetaISq = max(0.0, 1.0 - cosThetaI * cosThetaI);
    float sinThetaTSq = etaRel * etaRel * sinThetaISq;
    if (sinThetaTSq >= 1.0) return false;
    float cosThetaT = sqrt(1.0 - sinThetaTSq);
    wt = normalize(-etaRel * wi + vec3(0.0, 0.0, etaRel * cosThetaI - cosThetaT));
    return true;
}

// --- Abbe-number dispersion (spec: Cauchy empirical formula) ---

// Fraunhofer C/d/F spectral line wavelengths (nm) the spec's Abbe-number definition uses:
// C=656.3 (long/red), d=587.6 (medium/yellow, the reference wavelength specular_ior is
// defined at), F=486.1 (short/blue) — matches openpbrRgbWavelengthsNM's representative
// R/G/B samples used to evaluate this per-channel (extended_lobes.glsl).
const float OPENPBR_FRAUNHOFER_LAMBDA_C_NM = 656.3;
const float OPENPBR_FRAUNHOFER_LAMBDA_D_NM = 587.6;
const float OPENPBR_FRAUNHOFER_LAMBDA_F_NM = 486.1;

// openpbrCauchyCoefficients ports transmission.go's cauchyCoefficients exactly, for a
// dielectric whose IOR at the Fraunhofer d line is nd with Abbe number vd (already
// dispersion_scale-adjusted by the caller).
void openpbrCauchyCoefficients(float nd, float vd, out float a, out float b) {
    float invLambdaFSq = 1.0 / (OPENPBR_FRAUNHOFER_LAMBDA_F_NM * OPENPBR_FRAUNHOFER_LAMBDA_F_NM);
    float invLambdaCSq = 1.0 / (OPENPBR_FRAUNHOFER_LAMBDA_C_NM * OPENPBR_FRAUNHOFER_LAMBDA_C_NM);
    b = (nd - 1.0) / (vd * (invLambdaFSq - invLambdaCSq));
    a = nd - b / (OPENPBR_FRAUNHOFER_LAMBDA_D_NM * OPENPBR_FRAUNHOFER_LAMBDA_D_NM);
}

// openpbrDispersiveIOR ports transmission.go's DispersiveIOR exactly: the wavelength-
// dependent IOR n(lambda) of a dielectric whose reference IOR is nd (at the Fraunhofer d
// line — specular_ior) with the given transmission_dispersion_scale/abbe_number pair.
// dispersionScale<=0 returns nd unchanged at every wavelength (Go's vd=+Inf => B=0 path,
// short-circuited here since GLSL has no convenient infinity to carry through the Cauchy
// formula) — the caller only evaluates this per-channel when dispersionScale>0 anyway.
float openpbrDispersiveIOR(float nd, float abbeNumber, float dispersionScale, float wavelengthNM) {
    if (dispersionScale <= 0.0) return nd;
    float vd = abbeNumber / dispersionScale;
    float a, b;
    openpbrCauchyCoefficients(nd, vd, a, b);
    return a + b / (wavelengthNM * wavelengthNM);
}

// --- Base-substrate mixing (spec Base Substrate: M_dielectric-base = mix(M_opaque-base,
// S_translucent-base, transmission_weight)) ---

// openpbrMixTransmission ports transmission.go's MixTransmission exactly.
vec3 openpbrMixTransmission(vec3 opaque, vec3 transmission, float weight) {
    if (weight <= 0.0) return opaque;
    return mix(opaque, transmission, weight);
}

#endif // OPENPBR_TRANSMISSION_LOBE_GLSL
