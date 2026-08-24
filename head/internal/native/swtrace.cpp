// Software ray-tracing scene: BVH traversal via a plain compute shader (M45-F01
// PBI-334, ADR-0053). No acceleration-structure extensions, no buffer-device-address —
// every buffer here is a regular bound storage/uniform buffer, which is why this file is
// noticeably smaller than raytrace.cpp: the whole vkGetDeviceProcAddr function-loading
// dance PBI-333 needed is unnecessary when nothing is an extension-only entry point.
#include "swtrace.h"
#include <cstring>

namespace {

bool ok(VkResult r) { return r == VK_SUCCESS; }

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
};

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

    VkShaderModuleCreateInfo smInfo{};
    smInfo.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO;
    smInfo.codeSize = (size_t)spvLen;
    smInfo.pCode = spv;
    VkShaderModule shader = VK_NULL_HANDLE;
    if (!ok(vkCreateShaderModule(c->device, &smInfo, c->allocator, &shader))) return 1;

    VkPipelineShaderStageCreateInfo stage{};
    stage.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stage.stage = VK_SHADER_STAGE_COMPUTE_BIT;
    stage.module = shader;
    stage.pName = "main";
    VkComputePipelineCreateInfo pInfo{};
    pInfo.sType = VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO;
    pInfo.stage = stage;
    pInfo.layout = s->pipeLayout;
    VkResult pipeResult = vkCreateComputePipelines(c->device, VK_NULL_HANDLE, 1, &pInfo, c->allocator, &s->pipeline);
    vkDestroyShaderModule(c->device, shader, c->allocator);
    if (!ok(pipeResult)) return 1;

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
    vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, s->pipeline);
    vkCmdBindDescriptorSets(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, s->pipeLayout, 0, 1, &s->descSet, 0, nullptr);
    vkCmdDispatch(cmd, 1, 1, 1);
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

    VkShaderModuleCreateInfo smInfo{};
    smInfo.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO;
    smInfo.codeSize = (size_t)spvLen;
    smInfo.pCode = spv;
    VkShaderModule shader = VK_NULL_HANDLE;
    if (!ok(vkCreateShaderModule(c->device, &smInfo, c->allocator, &shader))) return 1;

    VkPipelineShaderStageCreateInfo stage{};
    stage.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stage.stage = VK_SHADER_STAGE_COMPUTE_BIT;
    stage.module = shader;
    stage.pName = "main";
    VkComputePipelineCreateInfo pInfo{};
    pInfo.sType = VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO;
    pInfo.stage = stage;
    pInfo.layout = s->ptPipeLayout;
    VkResult pipeResult = vkCreateComputePipelines(c->device, VK_NULL_HANDLE, 1, &pInfo, c->allocator, &s->ptPipeline);
    vkDestroyShaderModule(c->device, shader, c->allocator);
    if (!ok(pipeResult)) return 1;

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
    vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, s->ptPipeline);
    vkCmdBindDescriptorSets(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, s->ptPipeLayout, 0, 1, &s->ptDescSet, 0, nullptr);
    vkCmdDispatch(cmd, 1, 1, 1);
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

    auto* out = reinterpret_cast<float*>(s->ptOutputBuf.mapped);
    if (outR) *outR = out[0];
    if (outG) *outG = out[1];
    if (outB) *outB = out[2];
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
    sw_destroy_buffer(c, &s->nodesBuf);
    sw_destroy_buffer(c, &s->triOrderBuf);
    sw_destroy_buffer(c, &s->trianglesBuf);
    if (s->fence) vkDestroyFence(c->device, s->fence, c->allocator);
    if (s->cmdPool) vkDestroyCommandPool(c->device, s->cmdPool, c->allocator);
    delete s;
}

} // extern "C"
