// Software BVH traversal (M45-F01 PBI-334, extracted M45-F05 PBI-350): the median-split
// BVH stack traversal + Möller-Trumbore triangle test, shared by swpathtrace.comp (the
// single-ray CPU-oracle harness) and swpathtrace_realistic.comp (the live per-pixel
// viewport pass) — both call it twice per shaded point (primary + shadow query) with
// explicit origin/direction, so a traversal bug fix applies to both at once. swtrace.comp
// (PBI-334's raw single-ray Intersector) keeps its own compact single-call variant
// (reads a fixed `ray` uniform rather than taking parameters) deliberately un-refactored
// — it predates this include and has no second call site that would need the same code.
//
// Every includer must #include "sw_bvh_types.glsl" and declare a Nodes SSBO (BVHNode
// nodes[]), a TriOrder SSBO (int triOrder[]), and a Triangles SSBO (GpuTriangle tris[])
// — at whatever binding numbers that shader's descriptor set uses — BEFORE this
// #include: GLSL requires a type/global to be declared before use, and this file's
// traceBVH() references nodes/triOrder/tris by name (the preprocessor just pastes text).

// A traversal hit result — same fields as raytrace.comp's HitResult, minus the extra
// std430 vec3 padding this local (non-buffer) struct doesn't need.
struct TraceResult {
    bool hit;
    float t;
    vec3 normal;
    int triIdx;
};

bool rayAABBHit(vec3 origin, vec3 dir, vec3 mn, vec3 mx, float tMin, float currentBestT) {
    vec3 invD = 1.0 / dir;
    vec3 t0 = (mn - origin) * invD;
    vec3 t1 = (mx - origin) * invD;
    vec3 tSmall = min(t0, t1);
    vec3 tBig = max(t0, t1);
    float tEnter = max(max(tSmall.x, tSmall.y), max(tSmall.z, tMin));
    float tExit = min(min(tBig.x, tBig.y), min(tBig.z, currentBestT));
    return tEnter <= tExit;
}

bool rayTriangleHit(vec3 origin, vec3 dir, GpuTriangle tri, out float t, out vec3 normal) {
    vec3 v0 = vec3(tri.v0x, tri.v0y, tri.v0z);
    vec3 v1 = vec3(tri.v1x, tri.v1y, tri.v1z);
    vec3 v2 = vec3(tri.v2x, tri.v2y, tri.v2z);
    vec3 e1 = v1 - v0;
    vec3 e2 = v2 - v0;
    vec3 pvec = cross(dir, e2);
    float det = dot(e1, pvec);
    if (abs(det) < 1e-8) return false;
    float invDet = 1.0 / det;
    vec3 tvec = origin - v0;
    float u = dot(tvec, pvec) * invDet;
    if (u < 0.0 || u > 1.0) return false;
    vec3 qvec = cross(tvec, e1);
    float v = dot(dir, qvec) * invDet;
    if (v < 0.0 || u + v > 1.0) return false;
    t = dot(e2, qvec) * invDet;
    normal = normalize(cross(e1, e2));
    return true;
}

// traceBVH is swtrace.comp's traversal loop: a fixed-depth (32) explicit stack over the
// median-split BVH (renderer.BuildBVH), same code for both a primary and a shadow query.
TraceResult traceBVH(vec3 origin, vec3 dir, float tMin, float tMax) {
    TraceResult result;
    result.hit = false;
    float bestT = tMax;
    int bestTri = -1;
    vec3 bestNormal = vec3(0);

    int stack[32];
    int sp = 0;
    stack[sp++] = 0;
    while (sp > 0) {
        int nodeIdx = stack[--sp];
        BVHNode node = nodes[nodeIdx];
        vec3 mn = vec3(node.minX, node.minY, node.minZ);
        vec3 mx = vec3(node.maxX, node.maxY, node.maxZ);
        if (!rayAABBHit(origin, dir, mn, mx, tMin, bestT)) continue;

        if (node.triCount > 0) {
            for (int i = 0; i < node.triCount; i++) {
                int triIdx = triOrder[node.leftFirst + i];
                float t;
                vec3 n;
                if (rayTriangleHit(origin, dir, tris[triIdx], t, n) && t >= tMin && t < bestT) {
                    bestT = t;
                    bestTri = triIdx;
                    bestNormal = n;
                }
            }
        } else if (sp < 30) {
            stack[sp++] = node.leftFirst;
            stack[sp++] = node.leftFirst + 1;
        }
    }

    if (bestTri >= 0) {
        result.hit = true;
        result.t = bestT;
        result.normal = bestNormal;
        result.triIdx = bestTri;
    }
    return result;
}
