#version 450
// Point-cloud fragment shader (#645): emit the scan point's color flat — its RGB, intensity ramp,
// or the default neutral grey (renderer.PointCloudColor, chosen host-side per display mode). Points
// are not lit, so unlike mesh.frag there is no normal/PBR path.
layout(location = 0) in vec4 vColor;
layout(location = 0) out vec4 outColor;
void main() {
    outColor = vColor;
}
