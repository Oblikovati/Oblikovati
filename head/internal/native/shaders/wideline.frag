#version 450
// Wide-line fragment shader (#2015 line weight): emit the stroke's colour flat. A stroke is a
// display annotation, not a surface — it carries no normal and takes no lighting, so unlike
// mesh.frag there is no PBR path here (mirroring point.frag).
layout(location = 0) in vec4 vColor;
layout(location = 0) out vec4 outColor;
void main() {
    outColor = vColor;
}
