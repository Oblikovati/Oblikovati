#version 450
// Headlight Lambert shading for surfaces; flat color for wireframe (pc.lit == 0).
layout(push_constant) uniform PushConstants {
    mat4 mvp;
    int  lit;
} pc;
layout(location = 0) in vec3 vNormal;
layout(location = 1) in vec4 vColor;
layout(location = 0) out vec4 outColor;
void main() {
    if (pc.lit == 0) { outColor = vColor; return; }
    vec3 n = normalize(vNormal);
    vec3 l = normalize(vec3(0.4, 0.6, 0.8));
    float d = max(dot(n, l), 0.0) * 0.8 + 0.2;
    outColor = vec4(vColor.rgb * d, vColor.a);
}
