// Software ray-tracing scene: BVH traversal via a plain compute shader (M45-F01
// PBI-334, ADR-0053). No acceleration-structure extensions, no buffer-device-address —
// every buffer here is a regular bound storage/uniform buffer, which is why this file is
// noticeably smaller than raytrace.cpp: the whole vkGetDeviceProcAddr function-loading
// dance PBI-333 needed is unnecessary when nothing is an extension-only entry point.
#include "swtrace.h"
#include <cstring>

namespace {

bool ok(VkResult r) { return r == VK_SUCCESS; }

// kRealisticParamsBytes mirrors raytrace.cpp's own constant of the same name — the live
// per-pixel Realistic-viewport pipeline's Params UBO size (#2148): 56 float32/224 bytes.
// Distinct from (and independent of) the single-ray test-harness ptParamsBuf, which
// stays fixed at 16 floats/64 bytes.
constexpr VkDeviceSize kRealisticParamsBytes = 56 * sizeof(float);

struct SWBuffer {
    VkBuffer buffer = VK_NULL_HANDLE;
    VkDeviceMemory memory = VK_NULL_HANDLE;
    void* mapped = nullptr;
};

bool sw_create_buffer(HeadContext* c, SWBuffer* b, VkBufferUsageFlags usage, VkDeviceSize bytes) {
    VkBufferCreateInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
    bi.size = bytes;
    bi.usage = usage;
    bi.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    if (!ok(vkCreateBuffer(c->device, &bi, c->allocator, &b->buffer))) return false;
    VkMemoryRequirements req{};
    vkGetBufferMemoryRequirements(c->device, b->buffer, &req);
    VkMemoryAllocateInfo ai{};
    ai.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    ai.allocationSize = req.size;
    ai.memoryTypeIndex = obk_find_memory_type(
        c->physical, req.memoryTypeBits, VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT);
    if (!ok(vkAllocateMemory(c->device, &ai, c->allocator, &b->memory))) return false;
    if (!ok(vkBindBufferMemory(c->device, b->buffer, b->memory, 0))) return false;
    return true;
}

bool sw_upload_buffer(HeadContext* c, SWBuffer* b, VkBufferUsageFlags usage, const void* data, VkDeviceSize bytes) {
    if (bytes == 0) return true; // an empty BVH/triangle list is legal (nothing to trace)
    if (!sw_create_buffer(c, b, usage, bytes)) return false;
    void* mapped = nullptr;
    vkMapMemory(c->device, b->memory, 0, bytes, 0, &mapped);
    std::memcpy(mapped, data, (size_t)bytes);
    vkUnmapMemory(c->device, b->memory);
    return true;
}

void sw_destroy_buffer(HeadContext* c, SWBuffer* b) {
    if (b->mapped) vkUnmapMemory(c->device, b->memory);
    if (b->buffer) vkDestroyBuffer(c->device, b->buffer, c->allocator);
    if (b->memory) vkFreeMemory(c->device, b->memory, c->allocator);
    *b = SWBuffer{};
}

// sw_build_compute_pipeline is shared by obk_sw_scene_build, obk_sw_scene_build_pathtrace_pipeline,
// and obk_sw_scene_build_realistic_pathtrace_pipeline: all three compile the same single-entry-point
// "main" compute shader and pipeline shape, differing only in the descriptor-set layout feeding it.
bool sw_build_compute_pipeline(HeadContext* c, VkPipelineLayout layout, const uint32_t* spv, int spvLen,
                               VkPipeline& outPipeline) {
    VkShaderModuleCreateInfo smInfo{};
    smInfo.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO;
    smInfo.codeSize = (size_t)spvLen;
    smInfo.pCode = spv;
    VkShaderModule shader = VK_NULL_HANDLE;
    if (!ok(vkCreateShaderModule(c->device, &smInfo, c->allocator, &shader))) return false;

    VkPipelineShaderStageCreateInfo stage{};
    stage.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stage.stage = VK_SHADER_STAGE_COMPUTE_BIT;
    stage.module = shader;
    stage.pName = "main";
    VkComputePipelineCreateInfo pInfo{};
    pInfo.sType = VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO;
    pInfo.stage = stage;
    pInfo.layout = layout;
    VkResult pipeResult = vkCreateComputePipelines(c->device, VK_NULL_HANDLE, 1, &pInfo, c->allocator, &outPipeline);
    vkDestroyShaderModule(c->device, shader, c->allocator);
    return ok(pipeResult);
}

} // namespace

struct SWScene {
    HeadContext* ctx = nullptr;
    VkCommandPool cmdPool = VK_NULL_HANDLE;
    VkFence fence = VK_NULL_HANDLE;
    SWBuffer nodesBuf, triOrderBuf, trianglesBuf, rayParamsBuf, resultBuf;
    VkDescriptorSetLayout dsLayout = VK_NULL_HANDLE;
    VkPipelineLayout pipeLayout = VK_NULL_HANDLE;
    VkPipeline pipeline = VK_NULL_HANDLE;
    VkDescriptorPool descPool = VK_NULL_HANDLE;
    VkDescriptorSet descSet = VK_NULL_HANDLE;
    bool built = false;

    // Single-bounce path-tracing harness (PBI-346) — a second compute pipeline sharing
    // this scene's already-uploaded nodesBuf/triOrderBuf/trianglesBuf above.
    VkDescriptorSetLayout ptDsLayout = VK_NULL_HANDLE;
    VkPipelineLayout ptPipeLayout = VK_NULL_HANDLE;
    VkPipeline ptPipeline = VK_NULL_HANDLE;
    VkDescriptorPool ptDescPool = VK_NULL_HANDLE;
    VkDescriptorSet ptDescSet = VK_NULL_HANDLE;
    SWBuffer ptCamBuf, ptParamsBuf, ptOutputBuf;
    bool pathtraceBuilt = false;

    // Live per-pixel Realistic-viewport pipeline (M45-F05 PBI-350) — a THIRD compute
    // pipeline, independent of the single-ray pt* harness above (own descriptor set), so
    // PBI-346's already-verified test is never at risk from a change here.
    VkDescriptorSetLayout imgDsLayout = VK_NULL_HANDLE;
    VkPipelineLayout imgPipeLayout = VK_NULL_HANDLE;
    VkPipeline imgPipeline = VK_NULL_HANDLE;
    VkDescriptorPool imgDescPool = VK_NULL_HANDLE;
    VkDescriptorSet imgDescSet = VK_NULL_HANDLE;
    SWBuffer imgCamBuf, imgParamsBuf, imgOutputBuf;
    int imgOutputWidth = 0, imgOutputHeight = 0;
    bool imgPipelineBuilt = false;
};

namespace {

// sw_dispatch_and_wait is shared by obk_sw_scene_trace, obk_sw_scene_trace_pathtrace, and
// obk_sw_scene_trace_realistic_pathtrace_image: all three record one compute dispatch into a
// fresh one-time command buffer, submit it, and block on the scene's shared fence for the
// (deliberately synchronous — this is a test/single-shot harness, not the live per-frame path)
// result to land in host-visible memory, differing only in which pipeline/descriptor set to
// bind and how many workgroups to dispatch.
void sw_dispatch_and_wait(HeadContext* c, SWScene* s, VkPipeline pipeline, VkPipelineLayout layout,
                          VkDescriptorSet descSet, uint32_t groupsX, uint32_t groupsY, uint32_t groupsZ) {
    VkCommandBufferAllocateInfo cba{};
    cba.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    cba.commandPool = s->cmdPool;
    cba.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    cba.commandBufferCount = 1;
    VkCommandBuffer cmd = VK_NULL_HANDLE;
    vkAllocateCommandBuffers(c->device, &cba, &cmd);
    VkCommandBufferBeginInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    bi.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(cmd, &bi);
    vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, pipeline);
    vkCmdBindDescriptorSets(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, layout, 0, 1, &descSet, 0, nullptr);
    vkCmdDispatch(cmd, groupsX, groupsY, groupsZ);
    VkMemoryBarrier barrier{};
    barrier.sType = VK_STRUCTURE_TYPE_MEMORY_BARRIER;
    barrier.srcAccessMask = VK_ACCESS_SHADER_WRITE_BIT;
    barrier.dstAccessMask = VK_ACCESS_HOST_READ_BIT;
    vkCmdPipelineBarrier(cmd, VK_PIPELINE_STAGE_COMPUTE_SHADER_BIT, VK_PIPELINE_STAGE_HOST_BIT, 0, 1, &barrier, 0,
                         nullptr, 0, nullptr);
    vkEndCommandBuffer(cmd);
    VkSubmitInfo submit{};
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &cmd;
    vkResetFences(c->device, 1, &s->fence);
    vkQueueSubmit(c->queue, 1, &submit, s->fence);
    vkWaitForFences(c->device, 1, &s->fence, VK_TRUE, UINT64_MAX);
    vkFreeCommandBuffers(c->device, s->cmdPool, 1, &cmd);
}

} // namespace

extern "C" {

void* obk_sw_scene_create(void* h) {
    HeadContext* c = (HeadContext*)h;
    if (!c || c->device == VK_NULL_HANDLE) return nullptr;
    SWScene* s = new SWScene();
    s->ctx = c;
    VkCommandPoolCreateInfo cpi{};
    cpi.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO;
    cpi.queueFamilyIndex = c->queueFamily;
    cpi.flags = VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT;
    if (!ok(vkCreateCommandPool(c->device, &cpi, c->allocator, &s->cmdPool))) {
        delete s;
        return nullptr;
    }
    VkFenceCreateInfo fi{};
    fi.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO;
    vkCreateFence(c->device, &fi, c->allocator, &s->fence);
    return s;
}

int obk_sw_scene_build(void* scene, const void* nodes, int nodeCount, const int32_t* triOrder,
                       int triOrderCount, const void* triangles, int triangleCount, const uint32_t* spv,
                       int spvLen) {
    SWScene* s = (SWScene*)scene;
    if (!s || s->built) return 1;
    HeadContext* c = s->ctx;

    const VkDeviceSize nodeBytes = (VkDeviceSize)nodeCount * 32;      // renderer.BVHNode: 6 floats + 2 int32
    const VkDeviceSize triOrderBytes = (VkDeviceSize)triOrderCount * 4;
    const VkDeviceSize triBytes = (VkDeviceSize)triangleCount * 44; // renderer.Triangle: 9 floats + 2 uint32

    if (!sw_upload_buffer(c, &s->nodesBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT, nodes, nodeBytes)) return 1;
    if (!sw_upload_buffer(c, &s->triOrderBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT, triOrder, triOrderBytes)) return 1;
    if (!sw_upload_buffer(c, &s->trianglesBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT, triangles, triBytes)) return 1;

    VkDescriptorSetLayoutBinding bindings[5]{};
    bindings[0] = {0, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[1] = {1, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[2] = {2, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[3] = {3, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[4] = {4, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    VkDescriptorSetLayoutCreateInfo dslInfo{};
    dslInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
    dslInfo.bindingCount = 5;
    dslInfo.pBindings = bindings;
    if (!ok(vkCreateDescriptorSetLayout(c->device, &dslInfo, c->allocator, &s->dsLayout))) return 1;

    VkPipelineLayoutCreateInfo plInfo{};
    plInfo.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    plInfo.setLayoutCount = 1;
    plInfo.pSetLayouts = &s->dsLayout;
    if (!ok(vkCreatePipelineLayout(c->device, &plInfo, c->allocator, &s->pipeLayout))) return 1;

    if (!sw_build_compute_pipeline(c, s->pipeLayout, spv, spvLen, s->pipeline)) return 1;

    VkDescriptorPoolSize poolSizes[2] = {
        {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1},
        {VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 4},
    };
    VkDescriptorPoolCreateInfo poolInfo{};
    poolInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    poolInfo.maxSets = 1;
    poolInfo.poolSizeCount = 2;
    poolInfo.pPoolSizes = poolSizes;
    if (!ok(vkCreateDescriptorPool(c->device, &poolInfo, c->allocator, &s->descPool))) return 1;

    VkDescriptorSetAllocateInfo dsAlloc{};
    dsAlloc.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    dsAlloc.descriptorPool = s->descPool;
    dsAlloc.descriptorSetCount = 1;
    dsAlloc.pSetLayouts = &s->dsLayout;
    if (!ok(vkAllocateDescriptorSets(c->device, &dsAlloc, &s->descSet))) return 1;

    // RayParams: vec3 origin, float tMin, vec3 direction, float tMax — 32 bytes (same
    // layout as raytrace.comp's).
    sw_create_buffer(c, &s->rayParamsBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT, 32);
    vkMapMemory(c->device, s->rayParamsBuf.memory, 0, 32, 0, &s->rayParamsBuf.mapped);

    // HitResult: same std430 layout (with vec3 padding) as raytrace.comp's — 64 bytes.
    sw_create_buffer(c, &s->resultBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT, 64);
    vkMapMemory(c->device, s->resultBuf.memory, 0, 64, 0, &s->resultBuf.mapped);

    VkDescriptorBufferInfo rayInfo{s->rayParamsBuf.buffer, 0, 32};
    VkDescriptorBufferInfo nodesInfo{s->nodesBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo triOrderInfo{s->triOrderBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo trianglesInfo{s->trianglesBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo resInfo{s->resultBuf.buffer, 0, 64};
    VkWriteDescriptorSet writes[5]{};
    VkDescriptorBufferInfo* infos[5] = {&rayInfo, &nodesInfo, &triOrderInfo, &trianglesInfo, &resInfo};
    VkDescriptorType types[5] = {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
                                 VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
                                 VK_DESCRIPTOR_TYPE_STORAGE_BUFFER};
    for (uint32_t i = 0; i < 5; i++) {
        writes[i] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->descSet, i, 0, 1,
                    types[i],           nullptr, infos[i],  nullptr};
    }
    vkUpdateDescriptorSets(c->device, 5, writes, 0, nullptr);

    s->built = true;
    return 0;
}

void obk_sw_scene_trace(void* scene, float ox, float oy, float oz, float dx, float dy, float dz, float tMin,
                        float tMax, int* hit, float* t, float* px, float* py, float* pz, float* nx, float* ny,
                        float* nz, uint32_t* instanceID, uint32_t* primitiveID) {
    if (hit) *hit = 0;
    SWScene* s = (SWScene*)scene;
    if (!s || !s->built) return;
    HeadContext* c = s->ctx;

    float rayParams[8] = {ox, oy, oz, tMin, dx, dy, dz, tMax};
    std::memcpy(s->rayParamsBuf.mapped, rayParams, sizeof(rayParams));

    sw_dispatch_and_wait(c, s, s->pipeline, s->pipeLayout, s->descSet, 1, 1, 1);

    // Mirrors raytrace.cpp's HitResultLayout exactly (same std430 shape and padding).
    struct HitResultLayout {
        uint32_t hit;
        float t;
        float _pad0[2];
        float px, py, pz;
        float _pad1;
        float nx, ny, nz;
        uint32_t instanceID, primitiveID;
    };
    auto* r = reinterpret_cast<HitResultLayout*>(s->resultBuf.mapped);
    if (hit) *hit = (int)r->hit;
    if (!r->hit) return;
    if (t) *t = r->t;
    if (px) *px = r->px;
    if (py) *py = r->py;
    if (pz) *pz = r->pz;
    if (nx) *nx = r->nx;
    if (ny) *ny = r->ny;
    if (nz) *nz = r->nz;
    if (instanceID) *instanceID = r->instanceID;
    if (primitiveID) *primitiveID = r->primitiveID;
}

int obk_sw_scene_build_pathtrace_pipeline(void* scene, const uint32_t* spv, int spvLen) {
    SWScene* s = (SWScene*)scene;
    if (!s || !s->built || s->pathtraceBuilt) return 1;
    HeadContext* c = s->ctx;

    VkDescriptorSetLayoutBinding bindings[6]{};
    bindings[0] = {0, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[1] = {1, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[2] = {2, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[3] = {3, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[4] = {4, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[5] = {5, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    VkDescriptorSetLayoutCreateInfo dslInfo{};
    dslInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
    dslInfo.bindingCount = 6;
    dslInfo.pBindings = bindings;
    if (!ok(vkCreateDescriptorSetLayout(c->device, &dslInfo, c->allocator, &s->ptDsLayout))) return 1;

    VkPipelineLayoutCreateInfo plInfo{};
    plInfo.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    plInfo.setLayoutCount = 1;
    plInfo.pSetLayouts = &s->ptDsLayout;
    if (!ok(vkCreatePipelineLayout(c->device, &plInfo, c->allocator, &s->ptPipeLayout))) return 1;

    if (!sw_build_compute_pipeline(c, s->ptPipeLayout, spv, spvLen, s->ptPipeline)) return 1;

    VkDescriptorPoolSize poolSizes[2] = {
        {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 2},
        {VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 4},
    };
    VkDescriptorPoolCreateInfo poolInfo{};
    poolInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    poolInfo.maxSets = 1;
    poolInfo.poolSizeCount = 2;
    poolInfo.pPoolSizes = poolSizes;
    if (!ok(vkCreateDescriptorPool(c->device, &poolInfo, c->allocator, &s->ptDescPool))) return 1;

    VkDescriptorSetAllocateInfo dsAlloc{};
    dsAlloc.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    dsAlloc.descriptorPool = s->ptDescPool;
    dsAlloc.descriptorSetCount = 1;
    dsAlloc.pSetLayouts = &s->ptDsLayout;
    if (!ok(vkAllocateDescriptorSets(c->device, &dsAlloc, &s->ptDescSet))) return 1;

    sw_create_buffer(c, &s->ptCamBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT, 32);
    vkMapMemory(c->device, s->ptCamBuf.memory, 0, 32, 0, &s->ptCamBuf.mapped);
    sw_create_buffer(c, &s->ptParamsBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT, 64);
    vkMapMemory(c->device, s->ptParamsBuf.memory, 0, 64, 0, &s->ptParamsBuf.mapped);
    sw_create_buffer(c, &s->ptOutputBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT, 16);
    vkMapMemory(c->device, s->ptOutputBuf.memory, 0, 16, 0, &s->ptOutputBuf.mapped);

    VkDescriptorBufferInfo camInfo{s->ptCamBuf.buffer, 0, 32};
    VkDescriptorBufferInfo nodesInfo{s->nodesBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo triOrderInfo{s->triOrderBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo trianglesInfo{s->trianglesBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo paramInfo{s->ptParamsBuf.buffer, 0, 64};
    VkDescriptorBufferInfo outInfo{s->ptOutputBuf.buffer, 0, 16};
    VkDescriptorBufferInfo* infos[6] = {&camInfo, &nodesInfo, &triOrderInfo, &trianglesInfo, &paramInfo, &outInfo};
    VkDescriptorType types[6] = {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
                                 VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
                                 VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER};
    VkWriteDescriptorSet writes[6]{};
    for (uint32_t i = 0; i < 6; i++) {
        writes[i] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->ptDescSet, i, 0, 1,
                    types[i],           nullptr, infos[i],  nullptr};
    }
    vkUpdateDescriptorSets(c->device, 6, writes, 0, nullptr);

    s->pathtraceBuilt = true;
    return 0;
}

void obk_sw_scene_trace_pathtrace(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                                  float tMin, float tMax, const float* params, float* outR, float* outG,
                                  float* outB) {
    SWScene* s = (SWScene*)scene;
    if (!s || !s->pathtraceBuilt) return;
    HeadContext* c = s->ctx;

    float camParams[8] = {ox, oy, oz, tMin, dx, dy, dz, tMax};
    std::memcpy(s->ptCamBuf.mapped, camParams, sizeof(camParams));
    std::memcpy(s->ptParamsBuf.mapped, params, 16 * sizeof(float));

    sw_dispatch_and_wait(c, s, s->ptPipeline, s->ptPipeLayout, s->ptDescSet, 1, 1, 1);

    auto* out = reinterpret_cast<float*>(s->ptOutputBuf.mapped);
    if (outR) *outR = out[0];
    if (outG) *outG = out[1];
    if (outB) *outB = out[2];
}

// obk_sw_scene_build_realistic_pathtrace_pipeline mirrors
// obk_sw_scene_build_pathtrace_pipeline exactly (same 6-binding descriptor set shape:
// camera UBO, nodes/triOrder/triangles SSBOs reusing the scene's already-uploaded
// geometry buffers, params UBO, output SSBO), except the camera UBO is 64 bytes (a
// pinhole eye/forward/right/up basis, swpathtrace_realistic.comp's CameraParams) and the
// output buffer starts at a 1-pixel placeholder, resized to width*height by the first
// obk_sw_scene_trace_realistic_pathtrace_image call.
int obk_sw_scene_build_realistic_pathtrace_pipeline(void* scene, const uint32_t* spv, int spvLen) {
    SWScene* s = (SWScene*)scene;
    if (!s || !s->built || s->imgPipelineBuilt) return 1;
    HeadContext* c = s->ctx;

    VkDescriptorSetLayoutBinding bindings[6]{};
    bindings[0] = {0, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[1] = {1, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[2] = {2, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[3] = {3, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[4] = {4, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[5] = {5, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    VkDescriptorSetLayoutCreateInfo dslInfo{};
    dslInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
    dslInfo.bindingCount = 6;
    dslInfo.pBindings = bindings;
    if (!ok(vkCreateDescriptorSetLayout(c->device, &dslInfo, c->allocator, &s->imgDsLayout))) return 1;

    VkPipelineLayoutCreateInfo plInfo{};
    plInfo.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    plInfo.setLayoutCount = 1;
    plInfo.pSetLayouts = &s->imgDsLayout;
    if (!ok(vkCreatePipelineLayout(c->device, &plInfo, c->allocator, &s->imgPipeLayout))) return 1;

    if (!sw_build_compute_pipeline(c, s->imgPipeLayout, spv, spvLen, s->imgPipeline)) return 1;

    VkDescriptorPoolSize poolSizes[2] = {
        {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 2},
        {VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 4},
    };
    VkDescriptorPoolCreateInfo poolInfo{};
    poolInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    poolInfo.maxSets = 1;
    poolInfo.poolSizeCount = 2;
    poolInfo.pPoolSizes = poolSizes;
    if (!ok(vkCreateDescriptorPool(c->device, &poolInfo, c->allocator, &s->imgDescPool))) return 1;

    VkDescriptorSetAllocateInfo dsAlloc{};
    dsAlloc.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    dsAlloc.descriptorPool = s->imgDescPool;
    dsAlloc.descriptorSetCount = 1;
    dsAlloc.pSetLayouts = &s->imgDsLayout;
    if (!ok(vkAllocateDescriptorSets(c->device, &dsAlloc, &s->imgDescSet))) return 1;

    sw_create_buffer(c, &s->imgCamBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT, 64);
    vkMapMemory(c->device, s->imgCamBuf.memory, 0, 64, 0, &s->imgCamBuf.mapped);
    sw_create_buffer(c, &s->imgParamsBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT, kRealisticParamsBytes);
    vkMapMemory(c->device, s->imgParamsBuf.memory, 0, kRealisticParamsBytes, 0, &s->imgParamsBuf.mapped);
    sw_create_buffer(c, &s->imgOutputBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT, 16);
    vkMapMemory(c->device, s->imgOutputBuf.memory, 0, 16, 0, &s->imgOutputBuf.mapped);
    s->imgOutputWidth = 1;
    s->imgOutputHeight = 1;

    VkDescriptorBufferInfo camInfo{s->imgCamBuf.buffer, 0, 64};
    VkDescriptorBufferInfo nodesInfo{s->nodesBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo triOrderInfo{s->triOrderBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo trianglesInfo{s->trianglesBuf.buffer, 0, VK_WHOLE_SIZE};
    VkDescriptorBufferInfo paramInfo{s->imgParamsBuf.buffer, 0, kRealisticParamsBytes};
    VkDescriptorBufferInfo outInfo{s->imgOutputBuf.buffer, 0, 16};
    VkDescriptorBufferInfo* infos[6] = {&camInfo, &nodesInfo, &triOrderInfo, &trianglesInfo, &paramInfo, &outInfo};
    VkDescriptorType types[6] = {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
                                 VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
                                 VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER};
    VkWriteDescriptorSet writes[6]{};
    for (uint32_t i = 0; i < 6; i++) {
        writes[i] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->imgDescSet, i, 0, 1,
                    types[i],           nullptr, infos[i],  nullptr};
    }
    vkUpdateDescriptorSets(c->device, 6, writes, 0, nullptr);

    s->imgPipelineBuilt = true;
    return 0;
}

// obk_sw_scene_trace_realistic_pathtrace_image dispatches ceil(width/8) x ceil(height/8)
// work groups (swpathtrace_realistic.comp's local_size 8x8) and reads back the
// resulting width*height RGB image. Resizes imgOutputBuf (and re-writes its descriptor)
// only when width*height actually changes from the previous call.
int obk_sw_scene_trace_realistic_pathtrace_image(void* scene, int width, int height, const float* camera,
                                                 const float* params, float* outPixels) {
    SWScene* s = (SWScene*)scene;
    if (!s || !s->imgPipelineBuilt || width <= 0 || height <= 0) return 1;
    HeadContext* c = s->ctx;

    if (width != s->imgOutputWidth || height != s->imgOutputHeight) {
        vkDeviceWaitIdle(c->device);
        sw_destroy_buffer(c, &s->imgOutputBuf);
        VkDeviceSize bytes = (VkDeviceSize)width * height * 16;
        if (!sw_create_buffer(c, &s->imgOutputBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT, bytes)) return 1;
        vkMapMemory(c->device, s->imgOutputBuf.memory, 0, bytes, 0, &s->imgOutputBuf.mapped);
        s->imgOutputWidth = width;
        s->imgOutputHeight = height;

        VkDescriptorBufferInfo outInfo{s->imgOutputBuf.buffer, 0, bytes};
        VkWriteDescriptorSet write{VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->imgDescSet, 5, 0, 1,
                                   VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, nullptr, &outInfo, nullptr};
        vkUpdateDescriptorSets(c->device, 1, &write, 0, nullptr);
    }

    std::memcpy(s->imgCamBuf.mapped, camera, 16 * sizeof(float));
    std::memcpy(s->imgParamsBuf.mapped, params, kRealisticParamsBytes);

    sw_dispatch_and_wait(c, s, s->imgPipeline, s->imgPipeLayout, s->imgDescSet, (uint32_t)((width + 7) / 8),
                        (uint32_t)((height + 7) / 8), 1);

    auto* out = reinterpret_cast<float*>(s->imgOutputBuf.mapped);
    for (int i = 0; i < width * height; i++) {
        outPixels[i * 3 + 0] = out[i * 4 + 0];
        outPixels[i * 3 + 1] = out[i * 4 + 1];
        outPixels[i * 3 + 2] = out[i * 4 + 2];
    }
    return 0;
}

void obk_sw_scene_destroy(void* scene) {
    SWScene* s = (SWScene*)scene;
    if (!s) return;
    HeadContext* c = s->ctx;
    vkDeviceWaitIdle(c->device);
    if (s->pipeline) vkDestroyPipeline(c->device, s->pipeline, c->allocator);
    if (s->pipeLayout) vkDestroyPipelineLayout(c->device, s->pipeLayout, c->allocator);
    if (s->dsLayout) vkDestroyDescriptorSetLayout(c->device, s->dsLayout, c->allocator);
    if (s->descPool) vkDestroyDescriptorPool(c->device, s->descPool, c->allocator);
    sw_destroy_buffer(c, &s->rayParamsBuf);
    sw_destroy_buffer(c, &s->resultBuf);
    if (s->ptPipeline) vkDestroyPipeline(c->device, s->ptPipeline, c->allocator);
    if (s->ptPipeLayout) vkDestroyPipelineLayout(c->device, s->ptPipeLayout, c->allocator);
    if (s->ptDsLayout) vkDestroyDescriptorSetLayout(c->device, s->ptDsLayout, c->allocator);
    if (s->ptDescPool) vkDestroyDescriptorPool(c->device, s->ptDescPool, c->allocator);
    sw_destroy_buffer(c, &s->ptCamBuf);
    sw_destroy_buffer(c, &s->ptParamsBuf);
    sw_destroy_buffer(c, &s->ptOutputBuf);
    if (s->imgPipeline) vkDestroyPipeline(c->device, s->imgPipeline, c->allocator);
    if (s->imgPipeLayout) vkDestroyPipelineLayout(c->device, s->imgPipeLayout, c->allocator);
    if (s->imgDsLayout) vkDestroyDescriptorSetLayout(c->device, s->imgDsLayout, c->allocator);
    if (s->imgDescPool) vkDestroyDescriptorPool(c->device, s->imgDescPool, c->allocator);
    sw_destroy_buffer(c, &s->imgCamBuf);
    sw_destroy_buffer(c, &s->imgParamsBuf);
    sw_destroy_buffer(c, &s->imgOutputBuf);
    sw_destroy_buffer(c, &s->nodesBuf);
    sw_destroy_buffer(c, &s->triOrderBuf);
    sw_destroy_buffer(c, &s->trianglesBuf);
    if (s->fence) vkDestroyFence(c->device, s->fence, c->allocator);
    if (s->cmdPool) vkDestroyCommandPool(c->device, s->cmdPool, c->allocator);
    delete s;
}

} // extern "C"
