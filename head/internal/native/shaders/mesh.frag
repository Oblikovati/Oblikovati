#version 450
// Surface shading for the viewport. Lines (camPosLit.w == 0) draw flat. Surfaces pick a
// shader from the per-vertex mode (mirrors renderer.Shading): 1 flat Lambert, 2 GGX PBR
// (Realistic), 3 Monochrome, 4 Cel/Illustration, 5 Gooch/Technical, 6 Watercolor. Modes 0/1
// keep the original headlight Lambert so UI overlays are unchanged (ADR-0023 §2,§4).
//
// Lighting comes from the scene UBO (set 0, binding 0): a header (ambience/brightness/exposure/
// lightCount) plus an array of lights. The std140 layout here must match viewport.PackLighting
// (head/viewport/lighting_pack.go) exactly — see ADR-0026 §1,§3.
layout(push_constant) uniform PushConstants {
    mat4 mvp;
    vec4 camPosLit;
} pc;

// GpuLight: dir.xyz = unit vector toward the light, dir.w = kind (0 directional, 1 point,
// 2 spot); color.rgb = linear color, color.w = intensity; pos.xyz = world position (point/spot).
struct GpuLight {
    vec4 dir;
    vec4 color;
    vec4 pos;
};
layout(std140, set = 0, binding = 0) uniform Scene {
    vec4     header; // x=ambience y=brightness z=exposure w=lightCount
    GpuLight lights[8];
} scene;

layout(location = 0) in vec3      vNormal;
layout(location = 1) in vec4      vColor;
layout(location = 2) in vec3      vWorldPos;
layout(location = 3) in float     vMetallic;
layout(location = 4) in float     vRoughness;
layout(location = 5) in vec3      vEmissive;
layout(location = 6) in flat int  vMode;
layout(location = 0) out vec4 outColor;

const float PI = 3.14159265359;
const vec3  FALLBACK_DIR = vec3(0.4, 0.6, 0.8); // headlight when the scene has no lights

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

// lightDirAndRadiance resolves a GpuLight at the shaded point: the unit direction toward it and
// its incident radiance (color·intensity·brightness, with mild inverse-square falloff for
// point/spot lights so a positioned light dims with distance).
void lightDirAndRadiance(GpuLight lt, float brightness, out vec3 L, out vec3 radiance) {
    if (int(lt.dir.w) == 0) {            // directional
        L = normalize(lt.dir.xyz);
        radiance = lt.color.rgb * lt.color.w * brightness;
        return;
    }
    vec3 d = lt.pos.xyz - vWorldPos;     // point / spot
    float dist = length(d);
    L = d / max(dist, 1e-4);
    float atten = 1.0 / (1.0 + dist * dist * 1e-4);
    radiance = lt.color.rgb * lt.color.w * brightness * atten;
}

// pbr is the GGX metallic-roughness BRDF summed over the scene lights, with an analytic ambient
// (the image-based-lighting stand-in PBI-304 / Phase 4 will replace) scaled by ambience, then
// exposure + ACES tone map.
vec4 pbr(vec3 N, vec3 V, vec3 albedo, float metal, float rough, vec3 emissive, float alpha) {
    vec3  lin  = toLinear(albedo);
    rough      = clamp(rough, 0.05, 1.0);
    metal      = clamp(metal, 0.0, 1.0);
    float a    = rough * rough;
    vec3  f0   = mix(vec3(0.04), lin, metal);
    float NoV  = max(dot(N, V), 1e-3);

    int   count      = int(scene.header.w);
    float brightness = scene.header.y;
    vec3  color      = vec3(0.0);
    for (int i = 0; i < count && i < 8; i++) {
        vec3 L, radiance;
        lightDirAndRadiance(scene.lights[i], brightness, L, radiance);
        vec3  H   = normalize(L + V);
        float NoL = max(dot(N, L), 0.0);
        float NoH = max(dot(N, H), 0.0);
        float VoH = max(dot(V, H), 0.0);
        vec3  F   = fresnel(VoH, f0);
        vec3  spec = (distGGX(NoH, a) * geomSmith(NoV, NoL, a)) * F / max(4.0 * NoV * NoL, 1e-3);
        vec3  diff = (1.0 - F) * (1.0 - metal) * lin / PI;
        color += (diff + spec) * radiance * NoL;
    }
    vec3 amb = (lin + f0 * 0.44) * scene.header.x; // ambience-scaled analytic ambient
    color += amb + toLinear(emissive);
    return vec4(toSRGB(aces(color * scene.header.z)), alpha); // header.z = exposure
}

// headlightDir is the first scene light's direction (the key), or a constant fallback when the
// scene has no lights — used by the flat/NPR modes that want a single light vector.
vec3 headlightDir() {
    if (int(scene.header.w) > 0 && int(scene.lights[0].dir.w) == 0) {
        return normalize(scene.lights[0].dir.xyz);
    }
    return normalize(FALLBACK_DIR);
}

void main() {
    // Lines / flat overlays: emit the unlit color.
    if (pc.camPosLit.w < 0.5) { outColor = vColor; return; }

    vec3 N = normalize(vNormal);
    vec3 V = normalize(pc.camPosLit.xyz - vWorldPos);
    if (dot(N, V) < 0.0) N = -N; // two-sided shading (CAD faces can present either way)

    if (vMode == 2) { // Realistic — physically based, multi-light
        outColor = pbr(N, V, vColor.rgb, vMetallic, vRoughness, vEmissive, vColor.a);
        return;
    }

    vec3  L = headlightDir();
    float NoL = max(dot(N, L), 0.0);
    vec3  albedo = vColor.rgb;
    float lambert = NoL * 0.8 + 0.2; // the original headlight term

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
