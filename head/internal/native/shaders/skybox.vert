#version 450
// Fullscreen-triangle skybox vertex shader: emits one oversized triangle covering the screen at
// the far plane (z=1), passing the clip-space XY so the fragment shader reconstructs a world ray.
// No vertex buffer — positions come from gl_VertexIndex (ADR-0026 §5).
layout(location = 0) out vec2 vNDC;
void main() {
    vec2 p = vec2(float((gl_VertexIndex << 1) & 2), float(gl_VertexIndex & 2)); // (0,0)(2,0)(0,2)
    vNDC = p * 2.0 - 1.0;
    gl_Position = vec4(vNDC, 1.0, 1.0); // far plane, so any geometry draws in front
}
