// Software BVH buffer element types (M45-F01 PBI-334, extracted M45-F05 PBI-350),
// shared with sw_bvh_traverse.glsl. Split into its own file because GLSL requires a
// type to be declared before any buffer block that uses it, and every includer's own
// Nodes/TriOrder/Triangles SSBO declarations must come BEFORE sw_bvh_traverse.glsl's
// functions (which reference those SSBOs by name) — so this file is #include'd first,
// then the SSBO blocks, then sw_bvh_traverse.glsl.

struct BVHNode {
    float minX, minY, minZ;
    float maxX, maxY, maxZ;
    int leftFirst;
    int triCount;
};

struct GpuTriangle {
    float v0x, v0y, v0z;
    float v1x, v1y, v1z;
    float v2x, v2y, v2z;
    uint instanceID;
    uint primitiveID;
};
