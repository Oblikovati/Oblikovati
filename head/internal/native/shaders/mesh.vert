#version 450
// Mesh vertex shader: transform by the push-constant MVP and pass through the
// per-vertex normal and color. Shared by the triangle (lit) and line (unlit)
// pipelines; the `lit` flag selects shading in the fragment stage.
layout(push_constant) uniform PushConstants {
    mat4 mvp;
    int  lit;
} pc;
layout(location = 0) in vec3 inPos;
layout(location = 1) in vec3 inNormal;
layout(location = 2) in vec4 inColor;
layout(location = 0) out vec3 vNormal;
layout(location = 1) out vec4 vColor;
void main() {
    gl_Position = pc.mvp * vec4(inPos, 1.0);
    vNormal = inNormal;
    vColor = inColor;
}
