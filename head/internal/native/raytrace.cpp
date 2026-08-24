// Hardware ray-tracing scene: BLAS-per-mesh + one TLAS, and a single-ray compute-shader
// Intersector query via GL_EXT_ray_query (M45-F01 PBI-333, ADR-0053). Self-contained
// (own command pool/fence), independent of the live per-frame Viewport render loop —
// that integration (wiring rebuild/refit to the tessellation-dirty signal inside the
// instanced draw path) is a later PBI; this establishes the correct, hardware-verified
// acceleration-structure and ray-query mechanics renderer.Intersector needs.
#include "raytrace.h"
#include <cstring>
#include <vector>

namespace {

bool ok(VkResult r) { return r == VK_SUCCESS; }

// VK_KHR_acceleration_structure's functions are not part of the Linux Vulkan loader's
// direct-link trampoline set (unlike e.g. VK_KHR_swapchain's), so they must be resolved
// via vkGetDeviceProcAddr once the extension is enabled on a device — direct linking
// against them fails at link time with "undefined reference". Loaded once per process
// (lazily, on the first RTScene) since every Window shares one physical device class.
struct RTFunctions {
    PFN_vkGetAccelerationStructureBuildSizesKHR getBuildSizes = nullptr;
    PFN_vkCreateAccelerationStructureKHR create = nullptr;
    PFN_vkDestroyAccelerationStructureKHR destroy = nullptr;
    PFN_vkCmdBuildAccelerationStructuresKHR cmdBuild = nullptr;
    PFN_vkGetAccelerationStructureDeviceAddressKHR getDeviceAddress = nullptr;
};

RTFunctions g_rtFn;
bool g_rtFnLoaded = false;

void rt_load_functions(VkDevice device) {
    if (g_rtFnLoaded) return;
    g_rtFn.getBuildSizes = (PFN_vkGetAccelerationStructureBuildSizesKHR)vkGetDeviceProcAddr(
        device, "vkGetAccelerationStructureBuildSizesKHR");
    g_rtFn.create =
        (PFN_vkCreateAccelerationStructureKHR)vkGetDeviceProcAddr(device, "vkCreateAccelerationStructureKHR");
    g_rtFn.destroy =
        (PFN_vkDestroyAccelerationStructureKHR)vkGetDeviceProcAddr(device, "vkDestroyAccelerationStructureKHR");
    g_rtFn.cmdBuild = (PFN_vkCmdBuildAccelerationStructuresKHR)vkGetDeviceProcAddr(
        device, "vkCmdBuildAccelerationStructuresKHR");
    g_rtFn.getDeviceAddress = (PFN_vkGetAccelerationStructureDeviceAddressKHR)vkGetDeviceProcAddr(
        device, "vkGetAccelerationStructureDeviceAddressKHR");
    g_rtFnLoaded = true;
}

struct RTBuffer {
    VkBuffer buffer = VK_NULL_HANDLE;
    VkDeviceMemory memory = VK_NULL_HANDLE;
    VkDeviceSize size = 0;
    VkDeviceAddress address = 0;
    void* mapped = nullptr;
};

struct RTBlas {
    RTBuffer vertexBuf, indexBuf, asBuffer;
    VkAccelerationStructureKHR handle = VK_NULL_HANDLE;
    VkDeviceAddress deviceAddress = 0;
    uint32_t instanceCustomIndex = 0;
    uint32_t primitiveCount = 0;
};

} // namespace

struct RTScene {
    HeadContext* ctx = nullptr;
    VkCommandPool cmdPool = VK_NULL_HANDLE;
    VkFence fence = VK_NULL_HANDLE;
    std::vector<RTBlas> blases;
    RTBuffer instanceBuf;
    RTBuffer tlasBuffer;
    VkAccelerationStructureKHR tlas = VK_NULL_HANDLE;
    VkDescriptorSetLayout dsLayout = VK_NULL_HANDLE;
    VkPipelineLayout pipeLayout = VK_NULL_HANDLE;
    VkPipeline pipeline = VK_NULL_HANDLE;
    VkDescriptorPool descPool = VK_NULL_HANDLE;
    VkDescriptorSet descSet = VK_NULL_HANDLE;
    RTBuffer rayParamsBuf;
    RTBuffer resultBuf;
    bool built = false;
};

namespace {

// rt_create_buffer allocates and binds device memory for a buffer, mirroring viewport.cpp's
// ensure_buffer (anonymous-namespace, not linkable across this translation unit) — this
// file's own equivalent, extended with the SHADER_DEVICE_ADDRESS flag every RT buffer needs.
bool rt_create_buffer(HeadContext* c, RTBuffer* b, VkBufferUsageFlags usage,
                      VkMemoryPropertyFlags props, VkDeviceSize bytes) {
    VkBufferCreateInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
    bi.size = bytes;
    bi.usage = usage | VK_BUFFER_USAGE_SHADER_DEVICE_ADDRESS_BIT;
    bi.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    if (!ok(vkCreateBuffer(c->device, &bi, c->allocator, &b->buffer))) return false;
    VkMemoryRequirements req{};
    vkGetBufferMemoryRequirements(c->device, b->buffer, &req);
    VkMemoryAllocateFlagsInfo flagsInfo{};
    flagsInfo.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_FLAGS_INFO;
    flagsInfo.flags = VK_MEMORY_ALLOCATE_DEVICE_ADDRESS_BIT;
    VkMemoryAllocateInfo ai{};
    ai.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    ai.pNext = &flagsInfo;
    ai.allocationSize = req.size;
    ai.memoryTypeIndex = obk_find_memory_type(c->physical, req.memoryTypeBits, props);
    if (!ok(vkAllocateMemory(c->device, &ai, c->allocator, &b->memory))) return false;
    if (!ok(vkBindBufferMemory(c->device, b->buffer, b->memory, 0))) return false;
    b->size = bytes;
    VkBufferDeviceAddressInfo addrInfo{};
    addrInfo.sType = VK_STRUCTURE_TYPE_BUFFER_DEVICE_ADDRESS_INFO;
    addrInfo.buffer = b->buffer;
    b->address = vkGetBufferDeviceAddress(c->device, &addrInfo);
    return true;
}

// rt_upload_buffer creates a host-visible buffer and copies data into it — every RT input
// (vertices, indices, instances) is small enough (a CPU-reference test scene, not a live
// frame's whole geometry) that host-visible avoids a separate staging-buffer copy.
bool rt_upload_buffer(HeadContext* c, RTBuffer* b, VkBufferUsageFlags usage, const void* data,
                      VkDeviceSize bytes) {
    if (!rt_create_buffer(c, b, usage,
                          VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, bytes))
        return false;
    void* mapped = nullptr;
    vkMapMemory(c->device, b->memory, 0, bytes, 0, &mapped);
    std::memcpy(mapped, data, (size_t)bytes);
    vkUnmapMemory(c->device, b->memory);
    return true;
}

void rt_destroy_buffer(HeadContext* c, RTBuffer* b) {
    if (b->mapped) vkUnmapMemory(c->device, b->memory);
    if (b->buffer) vkDestroyBuffer(c->device, b->buffer, c->allocator);
    if (b->memory) vkFreeMemory(c->device, b->memory, c->allocator);
    *b = RTBuffer{};
}

// rt_run_commands records cmds into a fresh one-shot command buffer and submits+waits —
// the same idiom as viewport.cpp's readback path (VkCommandBufferAllocateInfo → begin →
// record → end → submit → wait fence → free), using this scene's own pool/fence since an
// RTScene is not tied to a live Viewport's frame resources.
template <typename Fn>
void rt_run_commands(RTScene* s, Fn&& cmds) {
    VkCommandBufferAllocateInfo cba{};
    cba.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    cba.commandPool = s->cmdPool;
    cba.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    cba.commandBufferCount = 1;
    VkCommandBuffer cmd = VK_NULL_HANDLE;
    vkAllocateCommandBuffers(s->ctx->device, &cba, &cmd);
    VkCommandBufferBeginInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    bi.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(cmd, &bi);
    cmds(cmd);
    vkEndCommandBuffer(cmd);
    VkSubmitInfo submit{};
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &cmd;
    vkResetFences(s->ctx->device, 1, &s->fence);
    vkQueueSubmit(s->ctx->queue, 1, &submit, s->fence);
    vkWaitForFences(s->ctx->device, 1, &s->fence, VK_TRUE, UINT64_MAX);
    vkFreeCommandBuffers(s->ctx->device, s->cmdPool, 1, &cmd);
}

// rt_build_acceleration_structure runs the shared query-size/create-buffer/create-AS/
// build sequence common to both a BLAS (geometry = triangles) and the TLAS (geometry =
// instances): every acceleration structure in this file follows this same five-step
// dance, only geometry/primitiveCount/type differ.
VkAccelerationStructureKHR rt_build_acceleration_structure(RTScene* s, VkAccelerationStructureTypeKHR type,
                                                            const VkAccelerationStructureGeometryKHR& geom,
                                                            uint32_t primitiveCount, RTBuffer* outAsBuffer) {
    HeadContext* c = s->ctx;
    VkAccelerationStructureBuildGeometryInfoKHR buildInfo{};
    buildInfo.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_BUILD_GEOMETRY_INFO_KHR;
    buildInfo.type = type;
    buildInfo.flags = VK_BUILD_ACCELERATION_STRUCTURE_PREFER_FAST_TRACE_BIT_KHR;
    // VK_KHR_ray_tracing_position_fetch requires an acceleration structure be built with
    // this flag before rayQueryGetIntersectionTriangleVertexPositionsEXT may read it
    // (raytrace.comp's normal computation) — guarded on the device feature since the bit
    // is only valid to set when the extension is enabled.
    if (c->hwRayTracingPositionFetch) buildInfo.flags |= VK_BUILD_ACCELERATION_STRUCTURE_ALLOW_DATA_ACCESS_KHR;
    buildInfo.mode = VK_BUILD_ACCELERATION_STRUCTURE_MODE_BUILD_KHR;
    buildInfo.geometryCount = 1;
    buildInfo.pGeometries = &geom;

    VkAccelerationStructureBuildSizesInfoKHR sizes{};
    sizes.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_BUILD_SIZES_INFO_KHR;
    g_rtFn.getBuildSizes(c->device, VK_ACCELERATION_STRUCTURE_BUILD_TYPE_DEVICE_KHR,
                                            &buildInfo, &primitiveCount, &sizes);

    rt_create_buffer(c, outAsBuffer,
                     VK_BUFFER_USAGE_ACCELERATION_STRUCTURE_STORAGE_BIT_KHR,
                     VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT, sizes.accelerationStructureSize);

    VkAccelerationStructureCreateInfoKHR asCreate{};
    asCreate.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_CREATE_INFO_KHR;
    asCreate.buffer = outAsBuffer->buffer;
    asCreate.size = sizes.accelerationStructureSize;
    asCreate.type = type;
    VkAccelerationStructureKHR as = VK_NULL_HANDLE;
    g_rtFn.create(c->device, &asCreate, c->allocator, &as);

    RTBuffer scratch;
    rt_create_buffer(c, &scratch, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT, sizes.buildScratchSize);

    buildInfo.dstAccelerationStructure = as;
    buildInfo.scratchData.deviceAddress = scratch.address;
    VkAccelerationStructureBuildRangeInfoKHR range{};
    range.primitiveCount = primitiveCount;
    const VkAccelerationStructureBuildRangeInfoKHR* pRange = &range;

    rt_run_commands(s, [&](VkCommandBuffer cmd) {
        g_rtFn.cmdBuild(cmd, 1, &buildInfo, &pRange);
    });

    rt_destroy_buffer(c, &scratch);
    return as;
}

VkDeviceAddress rt_as_device_address(HeadContext* c, VkAccelerationStructureKHR as) {
    VkAccelerationStructureDeviceAddressInfoKHR info{};
    info.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_DEVICE_ADDRESS_INFO_KHR;
    info.accelerationStructure = as;
    return g_rtFn.getDeviceAddress(c->device, &info);
}

// rt_create_compute_pipeline builds the raytrace.comp pipeline (descriptor layout:
// binding 0 = TLAS, binding 1 = ray-params UBO, binding 2 = hit-result SSBO) and its
// descriptor pool/set, plus the two small host-visible buffers the descriptor set binds.
bool rt_create_compute_pipeline(RTScene* s, const uint32_t* spv, int spvLen) {
    HeadContext* c = s->ctx;

    VkDescriptorSetLayoutBinding bindings[3]{};
    bindings[0] = {0, VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[1] = {1, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    bindings[2] = {2, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_COMPUTE_BIT, nullptr};
    VkDescriptorSetLayoutCreateInfo dslInfo{};
    dslInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
    dslInfo.bindingCount = 3;
    dslInfo.pBindings = bindings;
    if (!ok(vkCreateDescriptorSetLayout(c->device, &dslInfo, c->allocator, &s->dsLayout))) return false;

    VkPipelineLayoutCreateInfo plInfo{};
    plInfo.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    plInfo.setLayoutCount = 1;
    plInfo.pSetLayouts = &s->dsLayout;
    if (!ok(vkCreatePipelineLayout(c->device, &plInfo, c->allocator, &s->pipeLayout))) return false;

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
    pInfo.layout = s->pipeLayout;
    // Known false positive (verified against real RADV hardware, M45-F01 PBI-333): the
    // Khronos validation layer's SPIR-V capability table doesn't yet recognize
    // RayQueryPositionFetchKHR and reports "Unhandled OpCapability" here — this is a
    // validation-layer database gap, not a driver rejection. Pipeline creation still
    // returns VK_SUCCESS and the shader dispatches correctly (TestRTSceneMatchesCPUOracle
    // cross-checks hit results against renderer.FakeIntersector on live hardware).
    VkResult pipeResult = vkCreateComputePipelines(c->device, VK_NULL_HANDLE, 1, &pInfo, c->allocator, &s->pipeline);
    vkDestroyShaderModule(c->device, shader, c->allocator);
    if (!ok(pipeResult)) return false;

    VkDescriptorPoolSize poolSizes[3] = {
        {VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, 1},
        {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1},
        {VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1},
    };
    VkDescriptorPoolCreateInfo poolInfo{};
    poolInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    poolInfo.maxSets = 1;
    poolInfo.poolSizeCount = 3;
    poolInfo.pPoolSizes = poolSizes;
    if (!ok(vkCreateDescriptorPool(c->device, &poolInfo, c->allocator, &s->descPool))) return false;

    VkDescriptorSetAllocateInfo dsAlloc{};
    dsAlloc.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    dsAlloc.descriptorPool = s->descPool;
    dsAlloc.descriptorSetCount = 1;
    dsAlloc.pSetLayouts = &s->dsLayout;
    if (!ok(vkAllocateDescriptorSets(c->device, &dsAlloc, &s->descSet))) return false;

    // RayParams: vec3 origin, float tMin, vec3 direction, float tMax — std140, 32 bytes.
    rt_create_buffer(c, &s->rayParamsBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, 32);
    vkMapMemory(c->device, s->rayParamsBuf.memory, 0, 32, 0, &s->rayParamsBuf.mapped);

    // HitResult: uint hit, float t, vec3 point(+pad), vec3 normal(+pad), uint instanceID,
    // uint primitiveID — std430, 64 bytes (vec3 aligns to 16).
    rt_create_buffer(c, &s->resultBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, 64);
    vkMapMemory(c->device, s->resultBuf.memory, 0, 64, 0, &s->resultBuf.mapped);

    VkWriteDescriptorSetAccelerationStructureKHR asWrite{};
    asWrite.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET_ACCELERATION_STRUCTURE_KHR;
    asWrite.accelerationStructureCount = 1;
    asWrite.pAccelerationStructures = &s->tlas;
    VkDescriptorBufferInfo rayInfo{s->rayParamsBuf.buffer, 0, 32};
    VkDescriptorBufferInfo resInfo{s->resultBuf.buffer, 0, 64};
    VkWriteDescriptorSet writes[3]{};
    writes[0] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, &asWrite, s->descSet, 0, 0, 1,
                VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, nullptr, nullptr, nullptr};
    writes[1] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->descSet, 1, 0, 1,
                VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, nullptr, &rayInfo, nullptr};
    writes[2] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->descSet, 2, 0, 1,
                VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, nullptr, &resInfo, nullptr};
    vkUpdateDescriptorSets(c->device, 3, writes, 0, nullptr);
    return true;
}

} // namespace

extern "C" {

void* obk_rt_scene_create(void* h) {
    HeadContext* c = (HeadContext*)h;
    if (!c || !c->hwRayTracingAvailable) return nullptr;
    rt_load_functions(c->device);
    RTScene* s = new RTScene();
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

int obk_rt_scene_add_mesh(void* scene, const float* vertices, int vertexCount,
                          const uint32_t* indices, int indexCount, uint32_t instanceCustomIndex) {
    RTScene* s = (RTScene*)scene;
    if (!s || s->built || vertexCount <= 0 || indexCount <= 0) return 1;
    HeadContext* c = s->ctx;

    RTBlas b{};
    b.instanceCustomIndex = instanceCustomIndex;
    b.primitiveCount = (uint32_t)indexCount / 3;
    if (!rt_upload_buffer(c, &b.vertexBuf, VK_BUFFER_USAGE_ACCELERATION_STRUCTURE_BUILD_INPUT_READ_ONLY_BIT_KHR,
                          vertices, (VkDeviceSize)vertexCount * 3 * sizeof(float)))
        return 1;
    if (!rt_upload_buffer(c, &b.indexBuf, VK_BUFFER_USAGE_ACCELERATION_STRUCTURE_BUILD_INPUT_READ_ONLY_BIT_KHR,
                          indices, (VkDeviceSize)indexCount * sizeof(uint32_t)))
        return 1;

    VkAccelerationStructureGeometryTrianglesDataKHR tri{};
    tri.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_TRIANGLES_DATA_KHR;
    tri.vertexFormat = VK_FORMAT_R32G32B32_SFLOAT;
    tri.vertexData.deviceAddress = b.vertexBuf.address;
    tri.vertexStride = 3 * sizeof(float);
    tri.maxVertex = (uint32_t)vertexCount - 1;
    tri.indexType = VK_INDEX_TYPE_UINT32;
    tri.indexData.deviceAddress = b.indexBuf.address;
    // Position fetch (VK_KHR_ray_tracing_position_fetch) requires geometries built with
    // this flag to be readable by rayQueryGetIntersectionTriangleVertexPositionsEXT.
    if (c->hwRayTracingPositionFetch) tri.transformData.deviceAddress = 0;

    VkAccelerationStructureGeometryKHR geom{};
    geom.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_KHR;
    geom.geometryType = VK_GEOMETRY_TYPE_TRIANGLES_KHR;
    geom.geometry.triangles = tri;
    // Opaque: the shader always confirms every candidate hit (no any-hit evaluation) —
    // this package's Triangle carries no alpha/cutout data.
    geom.flags = VK_GEOMETRY_OPAQUE_BIT_KHR;

    b.handle = rt_build_acceleration_structure(s, VK_ACCELERATION_STRUCTURE_TYPE_BOTTOM_LEVEL_KHR,
                                               geom, b.primitiveCount, &b.asBuffer);
    if (b.handle == VK_NULL_HANDLE) return 1;
    b.deviceAddress = rt_as_device_address(c, b.handle);
    s->blases.push_back(b);
    return 0;
}

int obk_rt_scene_build(void* scene, const uint32_t* spv, int spvLen) {
    RTScene* s = (RTScene*)scene;
    if (!s || s->built || s->blases.empty()) return 1;
    HeadContext* c = s->ctx;

    std::vector<VkAccelerationStructureInstanceKHR> instances(s->blases.size());
    for (size_t i = 0; i < s->blases.size(); i++) {
        VkAccelerationStructureInstanceKHR& inst = instances[i];
        // Identity transform: PBI-333's scope builds one BLAS per body in world space
        // (no retessellation), with per-instance placement transforms wired into the
        // TLAS deferred to the live-viewport integration (a later PBI) — see raytrace.h.
        inst.transform.matrix[0][0] = inst.transform.matrix[1][1] = inst.transform.matrix[2][2] = 1.0f;
        inst.instanceCustomIndex = s->blases[i].instanceCustomIndex;
        inst.mask = 0xFF;
        inst.instanceShaderBindingTableRecordOffset = 0;
        inst.flags = VK_GEOMETRY_INSTANCE_TRIANGLE_FACING_CULL_DISABLE_BIT_KHR;
        inst.accelerationStructureReference = s->blases[i].deviceAddress;
    }
    if (!rt_upload_buffer(c, &s->instanceBuf, VK_BUFFER_USAGE_ACCELERATION_STRUCTURE_BUILD_INPUT_READ_ONLY_BIT_KHR,
                          instances.data(), instances.size() * sizeof(VkAccelerationStructureInstanceKHR)))
        return 1;

    VkAccelerationStructureGeometryInstancesDataKHR instData{};
    instData.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_INSTANCES_DATA_KHR;
    instData.data.deviceAddress = s->instanceBuf.address;
    VkAccelerationStructureGeometryKHR geom{};
    geom.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_KHR;
    geom.geometryType = VK_GEOMETRY_TYPE_INSTANCES_KHR;
    geom.geometry.instances = instData;

    s->tlas = rt_build_acceleration_structure(s, VK_ACCELERATION_STRUCTURE_TYPE_TOP_LEVEL_KHR,
                                              geom, (uint32_t)instances.size(), &s->tlasBuffer);
    if (s->tlas == VK_NULL_HANDLE) return 1;

    if (!rt_create_compute_pipeline(s, spv, spvLen)) return 1;
    s->built = true;
    return 0;
}

void obk_rt_scene_trace(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                        float tMin, float tMax, int* hit, float* t, float* px, float* py, float* pz,
                        float* nx, float* ny, float* nz, uint32_t* instanceID, uint32_t* primitiveID) {
    if (hit) *hit = 0;
    RTScene* s = (RTScene*)scene;
    if (!s || !s->built) return;

    float rayParams[8] = {ox, oy, oz, tMin, dx, dy, dz, tMax};
    std::memcpy(s->rayParamsBuf.mapped, rayParams, sizeof(rayParams));

    rt_run_commands(s, [&](VkCommandBuffer cmd) {
        vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, s->pipeline);
        vkCmdBindDescriptorSets(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, s->pipeLayout, 0, 1, &s->descSet, 0, nullptr);
        vkCmdDispatch(cmd, 1, 1, 1);
        VkMemoryBarrier barrier{};
        barrier.sType = VK_STRUCTURE_TYPE_MEMORY_BARRIER;
        barrier.srcAccessMask = VK_ACCESS_SHADER_WRITE_BIT;
        barrier.dstAccessMask = VK_ACCESS_HOST_READ_BIT;
        vkCmdPipelineBarrier(cmd, VK_PIPELINE_STAGE_COMPUTE_SHADER_BIT, VK_PIPELINE_STAGE_HOST_BIT, 0, 1,
                             &barrier, 0, nullptr, 0, nullptr);
    });

    // Mirrors the shader's std430 HitResult block EXACTLY, including its layout rules:
    // a vec3 member's base alignment is 16 bytes, so 8 bytes of padding separate `t`
    // from `point` (offset 8→16), and 4 more separate `point` from `normal` (offset
    // 28→32) — point/normal do NOT start immediately after the preceding scalar the
    // way a naive C++ struct would lay them out.
    struct HitResultLayout {
        uint32_t hit;   // offset 0
        float t;        // offset 4
        float _pad0[2]; // offset 8..15 (padding to vec3's 16-byte alignment)
        float px, py, pz; // offset 16..27
        float _pad1;      // offset 28..31 (padding to normal's 16-byte alignment)
        float nx, ny, nz; // offset 32..43
        uint32_t instanceID, primitiveID; // offset 44, 48
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

void obk_rt_scene_destroy(void* scene) {
    RTScene* s = (RTScene*)scene;
    if (!s) return;
    HeadContext* c = s->ctx;
    vkDeviceWaitIdle(c->device);
    if (s->pipeline) vkDestroyPipeline(c->device, s->pipeline, c->allocator);
    if (s->pipeLayout) vkDestroyPipelineLayout(c->device, s->pipeLayout, c->allocator);
    if (s->dsLayout) vkDestroyDescriptorSetLayout(c->device, s->dsLayout, c->allocator);
    if (s->descPool) vkDestroyDescriptorPool(c->device, s->descPool, c->allocator);
    rt_destroy_buffer(c, &s->rayParamsBuf);
    rt_destroy_buffer(c, &s->resultBuf);
    if (s->tlas) g_rtFn.destroy(c->device, s->tlas, c->allocator);
    rt_destroy_buffer(c, &s->tlasBuffer);
    rt_destroy_buffer(c, &s->instanceBuf);
    for (auto& b : s->blases) {
        if (b.handle) g_rtFn.destroy(c->device, b.handle, c->allocator);
        rt_destroy_buffer(c, &b.asBuffer);
        rt_destroy_buffer(c, &b.vertexBuf);
        rt_destroy_buffer(c, &b.indexBuf);
    }
    if (s->fence) vkDestroyFence(c->device, s->fence, c->allocator);
    if (s->cmdPool) vkDestroyCommandPool(c->device, s->cmdPool, c->allocator);
    delete s;
}

} // extern "C"
