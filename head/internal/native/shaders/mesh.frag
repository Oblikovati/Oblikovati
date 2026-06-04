#version 450
// Surface shading for the viewport. Lines (camPosLit.w == 0) draw flat. Surfaces pick a
// shader from the per-vertex mode (mirrors renderer.Shading): 1 flat Lambert, 2 GGX PBR
// (Realistic), 3 Monochrome, 4 Cel/Illustration, 5 Gooch/Technical, 6 Watercolor. Modes 0/1
// keep the original headlight Lambert so UI overlays are unchanged (ADR-0023 §2,§4).
layout(push_constant) uniform PushConstants {
    mat4 mvp;
    vec4 camPosLit;
} pc;
layout(location = 0) in vec3      vNormal;
layout(location = 1) in vec4      vColor;
layout(location = 2) in vec3      vWorldPos;
layout(location = 3) in float     vMetallic;
layout(location = 4) in float     vRoughness;
layout(location = 5) in vec3      vEmissive;
layout(location = 6) in flat int  vMode;
layout(location = 0) out vec4 outColor;

const float PI = 3.14159265359;
const vec3  LIGHT_DIR = vec3(0.4, 0.6, 0.8); // headlight, normalized below

vec3 toLinear(vec3 c) { return pow(clamp(c, 0.0, 1.0), vec3(2.2)); }
vec3 toSRGB(vec3 c)   { return pow(clamp(c, 0.0, 1.0), vec3(1.0 / 2.2)); }

// Narkowicz ACES filmic tone-map approximation.
vec3 aces(vec3 x) {
    const float a = 2.51, b = 0.03, c = 2.43, d = 0.59, e = 0.14;
    return clamp((x * (a * x + b)) / (x * (c * x + d) + e), 0.0, 1.0);
}

float distGGX(float NoH, float a) {
    float a2 = a * a;
    float d = NoH * NoH * (a2 - 1.0) + 1.0;
    return a2 / max(PI * d * d, 1e-7);
}
float geomSmith(float NoV, float NoL, float a) {
    float k = a * 0.5;
    float gv = NoV / (NoV * (1.0 - k) + k);
    float gl = NoL / (NoL * (1.0 - k) + k);
    return gv * gl;
}
vec3 fresnel(float VoH, vec3 f0) { return f0 + (1.0 - f0) * pow(1.0 - VoH, 5.0); }

// pbr is the GGX metallic-roughness BRDF with a crude analytic ambient + ACES tone map.
vec4 pbr(vec3 N, vec3 V, vec3 L, vec3 albedo, float metal, float rough, vec3 emissive, float alpha) {
    vec3  lin   = toLinear(albedo);
    rough       = clamp(rough, 0.05, 1.0);
    metal       = clamp(metal, 0.0, 1.0);
    float a     = rough * rough;
    vec3  f0    = mix(vec3(0.04), lin, metal);
    vec3  H     = normalize(L + V);
    float NoV   = max(dot(N, V), 1e-3);
    float NoL   = max(dot(N, L), 0.0);
    float NoH   = max(dot(N, H), 0.0);
    float VoH   = max(dot(V, H), 0.0);
    float D     = distGGX(NoH, a);
    float G     = geomSmith(NoV, NoL, a);
    vec3  F     = fresnel(VoH, f0);
    vec3  spec  = (D * G) * F / max(4.0 * NoV * NoL, 1e-3);
    vec3  kd    = (1.0 - F) * (1.0 - metal);
    vec3  diff  = kd * lin / PI;
    vec3  sun   = vec3(3.0);
    vec3  amb   = lin * 0.18 + f0 * 0.08; // image-based-lighting stand-in (PBI-304 will refine)
    vec3  color = (diff + spec) * sun * NoL + amb + toLinear(emissive);
    return vec4(toSRGB(aces(color)), alpha);
}

void main() {
    // Lines / flat overlays: emit the unlit color.
    if (pc.camPosLit.w < 0.5) { outColor = vColor; return; }

    vec3 N = normalize(vNormal);
    vec3 V = normalize(pc.camPosLit.xyz - vWorldPos);
    if (dot(N, V) < 0.0) N = -N; // two-sided shading (CAD faces can present either way)
    vec3 L = normalize(LIGHT_DIR);
    float NoL = max(dot(N, L), 0.0);
    vec3 albedo = vColor.rgb;
    float lambert = NoL * 0.8 + 0.2; // the original headlight term

    if (vMode == 2) { // Realistic — physically based
        outColor = pbr(N, V, L, albedo, vMetallic, vRoughness, vEmissive, vColor.a);
        return;
    }
    if (vMode == 3) { // Monochrome — desaturate + posterize, warm-paper tint
        float lum = dot(albedo, vec3(0.299, 0.587, 0.114)) * lambert;
        lum = floor(lum * 4.0 + 0.5) / 4.0; // 4 tone bands
        vec3 ink = vec3(0.10, 0.10, 0.12);
        vec3 paper = vec3(0.92, 0.91, 0.87);
        outColor = vec4(mix(ink, paper, lum), vColor.a);
        return;
    }
    if (vMode == 4) { // Illustration — cel / flat banded
        float band = NoL <= 0.25 ? 0.45 : (NoL <= 0.6 ? 0.72 : 1.0);
        outColor = vec4(albedo * band, vColor.a);
        return;
    }
    if (vMode == 5) { // Technical Illustration — Gooch cool-to-warm
        vec3 cool = vec3(0.0, 0.0, 0.40) + 0.25 * albedo;
        vec3 warm = vec3(0.40, 0.30, 0.0) + 0.50 * albedo;
        float t = (dot(N, L) + 1.0) * 0.5;
        outColor = vec4(clamp(mix(cool, warm, t), 0.0, 1.0), vColor.a);
        return;
    }
    if (vMode == 6) { // Watercolor — soft washes on paper
        float w = floor(smoothstep(0.0, 1.0, lambert) * 3.0 + 0.5) / 3.0;
        vec3 paper = vec3(0.96, 0.95, 0.92);
        vec3 wash = mix(paper, albedo, 0.55);
        outColor = vec4(wash * (0.72 + 0.28 * w), vColor.a * 0.95);
        return;
    }
    // Modes 0/1 — original headlight Lambert (Shaded / overlays).
    outColor = vec4(albedo * lambert, vColor.a);
}
