#version 450
// Skybox fragment shader: reconstructs the world-space view ray for each screen pixel from the
// inverse view-projection (carried in the push-constant mat4 slot for the skybox draw) and the
// camera eye, then samples the equirect environment. Drawn before geometry at the far plane, so
// it fills the background where no body covers it (ADR-0026 §5).
layout(push_constant) uniform PushConstants {
    mat4 invVP;     // inverse view-projection (the geometry pass puts the MVP here instead)
    vec4 camPosLit; // xyz = camera eye (world)
} pc;

struct GpuLight { vec4 dir; vec4 color; vec4 pos; };
layout(std140, set = 0, binding = 0) uniform Scene {
    vec4     header;
    GpuLight lights[8];
    vec4     env; // x=enabled y=rotation z=iblIntensity w=maxLod
} scene;
layout(set = 0, binding = 1) uniform sampler2D envMap;

layout(location = 0) in vec2 vNDC;
layout(location = 0) out vec4 outColor;

const float PI = 3.14159265359;

vec3 toSRGB(vec3 c) { return pow(clamp(c, 0.0, 1.0), vec3(1.0 / 2.2)); }
vec3 aces(vec3 x) {
    const float a = 2.51, b = 0.03, c = 2.43, d = 0.59, e = 0.14;
    return clamp((x * (a * x + b)) / (x * (c * x + d) + e), 0.0, 1.0);
}
vec2 dirUV(vec3 d, float rot) {
    float u = atan(d.y, d.x) / (2.0 * PI) + 0.5 + rot / (2.0 * PI);
    float v = acos(clamp(d.z, -1.0, 1.0)) / PI;
    return vec2(u, v);
}

void main() {
    vec4 world = pc.invVP * vec4(vNDC, 1.0, 1.0); // a point on the far plane, world space
    vec3 dir = normalize(world.xyz / world.w - pc.camPosLit.xyz);
    vec3 c = textureLod(envMap, dirUV(dir, scene.env.y), 0.0).rgb * scene.env.z;
    outColor = vec4(toSRGB(aces(c)), 1.0);
}
