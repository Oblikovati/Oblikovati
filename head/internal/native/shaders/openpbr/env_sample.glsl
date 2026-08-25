// Equirectangular environment sampling shared by pathtrace_realistic.rmiss (hardware
// backend) and swpathtrace_realistic.comp (software backend) — #2135/#2155's IBL
// follow-up: a primary or transmission-continuation ray that misses all geometry samples
// the same HDR sky the raster skybox (skybox.frag) already shows, instead of returning
// flat black.

#ifndef OPENPBR_ENV_SAMPLE_GLSL
#define OPENPBR_ENV_SAMPLE_GLSL

// openpbrEnvDirUV mirrors skybox.frag's own dirUV exactly (equirectangular Z-up mapping),
// so the ray-traced background matches the rasterized skybox pixel-for-pixel. Callers
// must #include "openpbr/base_lobes.glsl" first for OPENPBR_PI — every consumer here
// already does, via extended_lobes.glsl.
vec2 openpbrEnvDirUV(vec3 d, float rot) {
    float u = atan(d.y, d.x) / (2.0 * OPENPBR_PI) + 0.5 + rot / (2.0 * OPENPBR_PI);
    float v = acos(clamp(d.z, -1.0, 1.0)) / OPENPBR_PI;
    return vec2(u, v);
}

// openpbrSampleEnvironment returns the LINEAR (ungraded) radiance a ray in direction dir
// sees when it misses all geometry: envMap sampled via openpbrEnvDirUV, scaled by
// intensity. No tone-map/gamma here — this codebase's Realistic-mode convention is that
// the path tracer accumulates linear radiance and defers tone-mapping to display time
// (head/ui/realistic_render.go's resolveDisplayRGBA), unlike skybox.frag which writes
// straight to a display-referred swapchain image and so tone-maps in-shader.
// enabled/rotation/intensity come from extended_lobes.glsl's
// OPENPBR_REALISTIC_PARAMS_FIELDS envEnabled/envRotation/envIntensity — enabled<=0.5
// returns black, matching skybox.frag's own scene.env.x gate.
vec3 openpbrSampleEnvironment(sampler2D envMap, vec3 dir, float enabled, float rotation, float intensity) {
    if (enabled <= 0.5) return vec3(0.0);
    return textureLod(envMap, openpbrEnvDirUV(dir, rotation), 0.0).rgb * intensity;
}

#endif
