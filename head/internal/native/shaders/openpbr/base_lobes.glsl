// OpenPBR Surface base lobes (M45-F03 PBI-340, ADR-0053): GLSL translation of the Go CPU
// reference kernel/shading/openpbr/{diffuse,fresnel,ggx,emission}.go — see that package's
// doc comment for sourcing (OpenPBR spec index.html + Adobe's openpbr-bsdf impl/*.h) and
// the local-shading-space convention (Z = macrosurface normal, wi/wo point away from the
// surface).
//
// This is a math-only header meant to be #include'd by a future closest-hit/compute
// shader (F04, PBI-345/346) — no consuming shader stage exists yet, so unlike the raster
// .frag/.vert files alongside this directory it has no standalone .spv build. Every
// function here has a byte-identical Go counterpart with CPU-reference oracle tests
// (kernel/shading/openpbr/*_test.go); keep the two in lockstep by hand until a shared
// code-generation step exists.
//
// Kulla-Conty multi-scatter compensation (kernel/shading/openpbr/multiscatter.go) is
// intentionally NOT ported here yet: it is a numerically integrated CPU-test-only
// computation (too expensive to run per-shading-sample), and its GPU form needs a baked
// LUT texture this PBI does not build. Single-scatter-only lobes are physically valid,
// just slightly under-energized at high roughness — the same limitation the OpenPBR spec
// itself calls out (index.html line 425) for any renderer that skips this step.

#ifndef OPENPBR_BASE_LOBES_GLSL
#define OPENPBR_BASE_LOBES_GLSL

const float OPENPBR_PI = 3.14159265358979323846;

// --- GGX microfacet distribution + Smith masking-shadowing (ggx.go) ---

float openpbrAlphaFromRoughness(float roughness) { return roughness * roughness; }

float openpbrDistributionGGX(vec3 h, float alpha) {
    float denom = (h.x * h.x) / (alpha * alpha) + (h.y * h.y) / (alpha * alpha) + h.z * h.z;
    return 1.0 / (OPENPBR_PI * alpha * alpha * denom * denom);
}

float openpbrSmithG1(vec3 v, float alpha) {
    float vzSq = v.z * v.z;
    if (vzSq == 0.0) return 0.0;
    return 2.0 / (1.0 + sqrt(1.0 + (alpha * alpha * v.x * v.x + alpha * alpha * v.y * v.y) / vzSq));
}

float openpbrSmithG2(vec3 wi, vec3 wo, float alpha) {
    return openpbrSmithG1(wi, alpha) * openpbrSmithG1(wo, alpha);
}

// --- Fresnel (fresnel.go) ---

float openpbrF0FromIOR(float iorRatio) {
    float x = (iorRatio - 1.0) / (iorRatio + 1.0);
    return x * x;
}

// Exact unpolarized dielectric Fresnel reflectance; cosThetaI must be >= 0.
float openpbrDielectricFresnel(float iorRatio, float cosThetaI) {
    if (iorRatio == 1.0) return 0.0;
    float sinThetaISq = 1.0 - cosThetaI * cosThetaI;
    float sinThetaTSq = sinThetaISq / (iorRatio * iorRatio);
    if (sinThetaTSq >= 1.0) return 1.0;
    float cosThetaT = sqrt(1.0 - sinThetaTSq);
    float rs = (cosThetaI - iorRatio * cosThetaT) / (cosThetaI + iorRatio * cosThetaT);
    float rp = (cosThetaT - iorRatio * cosThetaI) / (cosThetaT + iorRatio * cosThetaI);
    return 0.5 * (rs * rs + rp * rp);
}

const float OPENPBR_F82_COS_THETA_MAX = 1.0 / 7.0;

vec3 openpbrF82SchlickBFactor(vec3 f0, vec3 tint) {
    float oneMinusMax = 1.0 - OPENPBR_F82_COS_THETA_MAX;
    float oneMinusMaxToFifth = pow(oneMinusMax, 5.0);
    float oneMinusMaxToSixth = oneMinusMaxToFifth * oneMinusMax;
    float denom = OPENPBR_F82_COS_THETA_MAX * oneMinusMaxToSixth;
    vec3 numer = (f0 + (vec3(1.0) - f0) * oneMinusMaxToFifth) * (vec3(1.0) - tint);
    return numer / denom;
}

// OpenPBR Metal section's F82-tint conductor Fresnel (spec eq. F_82). cosTheta must be >= 0.
vec3 openpbrF82TintFresnel(vec3 f0, vec3 tint, float cosTheta) {
    vec3 b = openpbrF82SchlickBFactor(f0, tint);
    float oneMinusCos = 1.0 - cosTheta;
    float oneMinusCosToFifth = pow(oneMinusCos, 5.0);
    vec3 v = f0 + ((vec3(1.0) - f0) - b * cosTheta * oneMinusCos) * oneMinusCosToFifth;
    return clamp(v, 0.0, 1.0);
}

// --- Energy-preserving Oren-Nayar diffuse (diffuse.go) ---

const float OPENPBR_FON_CONSTANT_A = 0.5 - 2.0 / (3.0 * OPENPBR_PI);
const float OPENPBR_FON_CONSTANT_B = 2.0 / 3.0 - 28.0 / (15.0 * OPENPBR_PI);

float openpbrDirectionalAlbedoFON(float mu, float roughness) {
    float af = 1.0 / (1.0 + OPENPBR_FON_CONSTANT_A * roughness);
    float bf = roughness * af;
    float clampedMu = clamp(mu, 0.0, 1.0);
    float si = sqrt(1.0 - clampedMu * clampedMu);
    float g = si * (acos(clampedMu) - si * clampedMu) +
              (2.0 / 3.0) * (si * clampedMu * (1.0 + si + si * si) / (1.0 + si) - si);
    return af + (bf / OPENPBR_PI) * g;
}

// Single-scatter term of the OpenPBR Base diffuse lobe (spec eq. FON_brdf). The
// multi-scatter compensation term (spec eq. EON_comp) is deferred per this file's header
// note — see kernel/shading/openpbr/diffuse.go's DiffuseEON for the full CPU reference.
vec3 openpbrDiffuseSingleScatter(vec3 rho, float roughness, vec3 wi, vec3 wo) {
    float muI = wi.z;
    float muO = wo.z;
    float s = dot(wi, wo) - muI * muO;
    float sOverT = s > 0.0 ? s / max(muI, muO) : s;
    float af = 1.0 / (1.0 + OPENPBR_FON_CONSTANT_A * roughness);
    return rho * (af * (1.0 + roughness * sOverT) / OPENPBR_PI);
}

// --- Emission (emission.go) ---

vec3 openpbrEmission(float luminance, vec3 color) { return color * luminance; }

#endif // OPENPBR_BASE_LOBES_GLSL
