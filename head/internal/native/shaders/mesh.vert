#version 450
// Mesh vertex shader: transform by the push-constant MVP and pass through the per-vertex
// normal, albedo, world position, PBR material (metallic/roughness/emissive) and the shading
// mode. Shared by the triangle (shaded) and line (flat) pipelines; the fragment stage picks
// the shader from the mode (surfaces) or the camPosLit.w flag (lines are flat).
layout(push_constant) uniform PushConstants {
    mat4 mvp;
    vec4 camPosLit; // xyz = camera eye in world space, w = lit flag (0 = line/flat, 1 = surface)
    vec4 clip;      // section plane: xyz = world normal (0 ⇒ no section), w = plane offset d (M12-F04)
} pc;
layout(location = 0) in vec3  inPos;
layout(location = 1) in vec3  inNormal;
layout(location = 2) in vec4  inColor;
layout(location = 3) in float inMetallic;
layout(location = 4) in float inRoughness;
layout(location = 5) in vec3  inEmissive;
layout(location = 6) in float inMode;
layout(location = 0) out vec3      vNormal;
layout(location = 1) out vec4      vColor;
layout(location = 2) out vec3      vWorldPos;
layout(location = 3) out float     vMetallic;
layout(location = 4) out float     vRoughness;
layout(location = 5) out vec3      vEmissive;
layout(location = 6) out flat int  vMode;
void main() {
    gl_Position = pc.mvp * vec4(inPos, 1.0);
    vNormal = inNormal;
    vColor = inColor;
    vWorldPos = inPos;
    vMetallic = inMetallic;
    vRoughness = inRoughness;
    vEmissive = inEmissive;
    vMode = int(inMode + 0.5);
}
