// Hardware ray-tracing scene: BLAS-per-mesh + one TLAS, and a single-ray compute-shader
// Intersector query via GL_EXT_ray_query (M45-F01 PBI-333, ADR-0053). Self-contained
// (own command pool/fence), independent of the live per-frame Viewport render loop —
// that integration (wiring rebuild/refit to the tessellation-dirty signal inside the
// instanced draw path) is a later PBI; this establishes the correct, hardware-verified
// acceleration-structure and ray-query mechanics renderer.Intersector needs.
#include "raytrace.h"
#include <algorithm>
#include <cstring>
#include <vector>

namespace {

bool ok(VkResult r) { return r == VK_SUCCESS; }

// kRealisticParamsBytes is the live per-pixel Realistic-viewport pipeline's Params UBO
// size (#2148): 56 float32 / 224 bytes — the original 16-float base-lobes-only layout
// (RealisticLightParams' first 16 fields) plus 40 more for Coat/Fuzz/ThinFilm/
// Transmission+dispersion/Subsurface, laid out exactly as raytrace.go's
// RealisticLightParams.floats() and swpathtrace_realistic.comp/pathtrace_realistic.rchit's
// Params struct. Distinct from (and independent of) obk_rt_scene_build_pipeline's
// single-ray test-harness PipelineParams, which stays fixed at 16 floats/64 bytes.
constexpr VkDeviceSize kRealisticParamsBytes = 56 * sizeof(float);

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
    // Full RT pipeline (PBI-345) — also extension-only entry points.
    PFN_vkCreateRayTracingPipelinesKHR createRTPipeline = nullptr;
    PFN_vkGetRayTracingShaderGroupHandlesKHR getShaderGroupHandles = nullptr;
    PFN_vkCmdTraceRaysKHR cmdTraceRays = nullptr;
    // Pipeline stack sizing (#2155) — required once a pipeline actually recurses (a hit
    // shader tracing another ray from within itself, not just a single-level shadow
    // query): see rt_pipeline_stack_size's own doc comment for why.
    PFN_vkGetRayTracingShaderGroupStackSizeKHR getGroupStackSize = nullptr;
    PFN_vkCmdSetRayTracingPipelineStackSizeKHR cmdSetStackSize = nullptr;
};

// rt_load_functions resolves every extension-only entry point fresh, into fn, for
// device — always, never cached across calls. A device-level function pointer from
// vkGetDeviceProcAddr is valid ONLY for the device (and its child objects) it was
// retrieved from (Vulkan spec); a single process-global cache resolved once against
// the FIRST device ever seen (this file's original design) is undefined behavior for
// every RTScene created on a later, different device — e.g. a second test's window in
// the same process. Reproduced as an intermittent SIGSEGV inside obk_rt_scene_add_mesh
// when two windows/RTScenes were created in sequence within one test binary (found
// live while adding M45-F05 PBI-350's per-pixel image pipeline, but pre-existing since
// PBI-333 — any two RT-using tests in the same process could trigger it). Fixed by
// storing RTFunctions per-RTScene (RTScene::rtFn) instead of process-global; the
// re-resolution cost is a handful of vkGetDeviceProcAddr calls per scene, negligible.
void rt_load_functions(VkDevice device, RTFunctions& fn) {
    fn.getBuildSizes = (PFN_vkGetAccelerationStructureBuildSizesKHR)vkGetDeviceProcAddr(
        device, "vkGetAccelerationStructureBuildSizesKHR");
    fn.create =
        (PFN_vkCreateAccelerationStructureKHR)vkGetDeviceProcAddr(device, "vkCreateAccelerationStructureKHR");
    fn.destroy =
        (PFN_vkDestroyAccelerationStructureKHR)vkGetDeviceProcAddr(device, "vkDestroyAccelerationStructureKHR");
    fn.cmdBuild = (PFN_vkCmdBuildAccelerationStructuresKHR)vkGetDeviceProcAddr(
        device, "vkCmdBuildAccelerationStructuresKHR");
    fn.getDeviceAddress = (PFN_vkGetAccelerationStructureDeviceAddressKHR)vkGetDeviceProcAddr(
        device, "vkGetAccelerationStructureDeviceAddressKHR");
    fn.createRTPipeline =
        (PFN_vkCreateRayTracingPipelinesKHR)vkGetDeviceProcAddr(device, "vkCreateRayTracingPipelinesKHR");
    fn.getShaderGroupHandles = (PFN_vkGetRayTracingShaderGroupHandlesKHR)vkGetDeviceProcAddr(
        device, "vkGetRayTracingShaderGroupHandlesKHR");
    fn.cmdTraceRays = (PFN_vkCmdTraceRaysKHR)vkGetDeviceProcAddr(device, "vkCmdTraceRaysKHR");
    fn.getGroupStackSize = (PFN_vkGetRayTracingShaderGroupStackSizeKHR)vkGetDeviceProcAddr(
        device, "vkGetRayTracingShaderGroupStackSizeKHR");
    fn.cmdSetStackSize = (PFN_vkCmdSetRayTracingPipelineStackSizeKHR)vkGetDeviceProcAddr(
        device, "vkCmdSetRayTracingPipelineStackSizeKHR");
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
    RTFunctions rtFn; // resolved fresh for THIS scene's device in obk_rt_scene_create — see rt_load_functions
    VkCommandPool cmdPool = VK_NULL_HANDLE;
    VkFence fence = VK_NULL_HANDLE;
    std::vector<RTBlas> blases;
    RTBuffer instanceBuf;
    RTBuffer tlasBuffer;
    VkAccelerationStructureKHR tlas = VK_NULL_HANDLE;
    // Ray-query compute pipeline (PBI-333's single-TraceRay-call Intersector).
    VkDescriptorSetLayout rqDsLayout = VK_NULL_HANDLE;
    VkPipelineLayout rqPipeLayout = VK_NULL_HANDLE;
    VkPipeline rqPipeline = VK_NULL_HANDLE;
    VkDescriptorPool rqDescPool = VK_NULL_HANDLE;
    VkDescriptorSet rqDescSet = VK_NULL_HANDLE;
    RTBuffer rayParamsBuf;
    RTBuffer resultBuf;
    bool built = false;

    // Full RT pipeline (PBI-345's ray-gen/closest-hit/miss single-bounce harness) —
    // separate from the ray-query pipeline above; both can coexist on one scene.
    VkDescriptorSetLayout pipeDsLayout = VK_NULL_HANDLE;
    VkPipelineLayout pipePipeLayout = VK_NULL_HANDLE;
    VkPipeline pipePipeline = VK_NULL_HANDLE;
    VkDescriptorPool pipeDescPool = VK_NULL_HANDLE;
    VkDescriptorSet pipeDescSet = VK_NULL_HANDLE;
    RTBuffer sbtBuf;
    RTBuffer pipeCamBuf, pipeOutputBuf, pipeParamsBuf;
    uint32_t sbtHandleSize = 0, sbtHandleAlignment = 0, sbtBaseAlignment = 0;
    uint32_t pipeStackSize = 0; // vkCmdSetRayTracingPipelineStackSizeKHR — see rt_pipeline_stack_size
    bool pipelineBuilt = false;

    // Live per-pixel Realistic-viewport pipeline (M45-F05 PBI-350) — a THIRD, independent
    // pipeline/SBT/descriptor set alongside the ray-query (rq*) and single-ray test-harness
    // (pipe*) ones above, sharing only this scene's BLAS/TLAS. Kept independent rather than
    // reusing pipe*'s shader modules so PBI-345/346's already-verified single-ray harness is
    // never at risk of a regression from this pipeline's changes.
    VkDescriptorSetLayout imgDsLayout = VK_NULL_HANDLE;
    VkPipelineLayout imgPipeLayout = VK_NULL_HANDLE;
    VkPipeline imgPipeline = VK_NULL_HANDLE;
    VkDescriptorPool imgDescPool = VK_NULL_HANDLE;
    VkDescriptorSet imgDescSet = VK_NULL_HANDLE;
    RTBuffer imgSbtBuf, imgCamBuf, imgParamsBuf, imgOutputBuf;
    uint32_t imgSbtHandleSize = 0, imgSbtHandleAlignment = 0, imgSbtBaseAlignment = 0;
    uint32_t imgStackSize = 0; // vkCmdSetRayTracingPipelineStackSizeKHR — see rt_pipeline_stack_size
    int imgOutputWidth = 0, imgOutputHeight = 0; // 0 until the first trace_realistic_image call sizes imgOutputBuf
    bool imgPipelineBuilt = false;
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
    s->rtFn.getBuildSizes(c->device, VK_ACCELERATION_STRUCTURE_BUILD_TYPE_DEVICE_KHR,
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
    s->rtFn.create(c->device, &asCreate, c->allocator, &as);

    RTBuffer scratch;
    rt_create_buffer(c, &scratch, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT, sizes.buildScratchSize);

    buildInfo.dstAccelerationStructure = as;
    buildInfo.scratchData.deviceAddress = scratch.address;
    VkAccelerationStructureBuildRangeInfoKHR range{};
    range.primitiveCount = primitiveCount;
    const VkAccelerationStructureBuildRangeInfoKHR* pRange = &range;

    rt_run_commands(s, [&](VkCommandBuffer cmd) {
        s->rtFn.cmdBuild(cmd, 1, &buildInfo, &pRange);
    });

    rt_destroy_buffer(c, &scratch);
    return as;
}

VkDeviceAddress rt_as_device_address(RTScene* s, VkAccelerationStructureKHR as) {
    VkAccelerationStructureDeviceAddressInfoKHR info{};
    info.sType = VK_STRUCTURE_TYPE_ACCELERATION_STRUCTURE_DEVICE_ADDRESS_INFO_KHR;
    info.accelerationStructure = as;
    return s->rtFn.getDeviceAddress(s->ctx->device, &info);
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
    if (!ok(vkCreateDescriptorSetLayout(c->device, &dslInfo, c->allocator, &s->rqDsLayout))) return false;

    VkPipelineLayoutCreateInfo plInfo{};
    plInfo.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    plInfo.setLayoutCount = 1;
    plInfo.pSetLayouts = &s->rqDsLayout;
    if (!ok(vkCreatePipelineLayout(c->device, &plInfo, c->allocator, &s->rqPipeLayout))) return false;

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
    pInfo.layout = s->rqPipeLayout;
    // Known false positive (verified against real RADV hardware, M45-F01 PBI-333): the
    // Khronos validation layer's SPIR-V capability table doesn't yet recognize
    // RayQueryPositionFetchKHR and reports "Unhandled OpCapability" here — this is a
    // validation-layer database gap, not a driver rejection. Pipeline creation still
    // returns VK_SUCCESS and the shader dispatches correctly (TestRTSceneMatchesCPUOracle
    // cross-checks hit results against renderer.FakeIntersector on live hardware).
    VkResult pipeResult = vkCreateComputePipelines(c->device, VK_NULL_HANDLE, 1, &pInfo, c->allocator, &s->rqPipeline);
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
    if (!ok(vkCreateDescriptorPool(c->device, &poolInfo, c->allocator, &s->rqDescPool))) return false;

    VkDescriptorSetAllocateInfo dsAlloc{};
    dsAlloc.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    dsAlloc.descriptorPool = s->rqDescPool;
    dsAlloc.descriptorSetCount = 1;
    dsAlloc.pSetLayouts = &s->rqDsLayout;
    if (!ok(vkAllocateDescriptorSets(c->device, &dsAlloc, &s->rqDescSet))) return false;

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
    writes[0] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, &asWrite, s->rqDescSet, 0, 0, 1,
                VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, nullptr, nullptr, nullptr};
    writes[1] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->rqDescSet, 1, 0, 1,
                VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, nullptr, &rayInfo, nullptr};
    writes[2] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->rqDescSet, 2, 0, 1,
                VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, nullptr, &resInfo, nullptr};
    vkUpdateDescriptorSets(c->device, 3, writes, 0, nullptr);
    return true;
}

} // namespace

extern "C" {

void* obk_rt_scene_create(void* h) {
    HeadContext* c = (HeadContext*)h;
    if (!c || !c->hwRayTracingAvailable) return nullptr;
    RTScene* s = new RTScene();
    s->ctx = c;
    rt_load_functions(c->device, s->rtFn);
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
    b.deviceAddress = rt_as_device_address(s, b.handle);
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
        vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, s->rqPipeline);
        vkCmdBindDescriptorSets(cmd, VK_PIPELINE_BIND_POINT_COMPUTE, s->rqPipeLayout, 0, 1, &s->rqDescSet, 0, nullptr);
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

namespace {

VkDeviceSize align_up(VkDeviceSize v, VkDeviceSize a) { return (v + a - 1) / a * a; }

VkShaderModule rt_load_shader(HeadContext* c, const uint32_t* spv, int spvLen) {
    VkShaderModuleCreateInfo smInfo{};
    smInfo.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO;
    smInfo.codeSize = (size_t)spvLen;
    smInfo.pCode = spv;
    VkShaderModule shader = VK_NULL_HANDLE;
    vkCreateShaderModule(c->device, &smInfo, c->allocator, &shader);
    return shader;
}

// RTPipelineResources/rt_build_4stage_pipeline are shared by obk_rt_scene_build_pipeline and
// obk_rt_scene_build_realistic_pipeline: both build an identical 4-binding descriptor set
// (TLAS, output storage buffer, camera UBO, params UBO) and 4-stage (raygen/miss/shadow-miss/
// closest-hit) ray tracing pipeline shape, differing only in which RTScene fields the caller
// stores the result into (and the camera/params UBO sizes it allocates afterward).
struct RTPipelineResources {
    VkDescriptorSetLayout dsLayout = VK_NULL_HANDLE;
    VkPipelineLayout pipeLayout = VK_NULL_HANDLE;
    VkPipeline pipeline = VK_NULL_HANDLE;
    uint32_t handleSize = 0;
    uint32_t handleAlignment = 0;
    uint32_t baseAlignment = 0;
    uint32_t stackSize = 0;
};

// rt_pipeline_stack_size computes the pipeline's required call-stack size for
// vkCmdSetRayTracingPipelineStackSizeKHR (#2155). Without this call, the driver is free
// to use whatever default it likes for vkCmdTraceRaysKHR — RADV's default is sized for a
// pipeline that never recurses past a single shadow-ray query, which this pipeline no
// longer is (pathtrace_realistic.rchit now fires a bounded chain of continuation rays
// through transmissive surfaces, invoking itself again for each). An undersized stack
// overflows into unrelated GPU memory: observed live as an intermittent GPUVM fault/"CS
// cancelled" device-lost error that also corrupted LATER, unrelated dispatches on the
// same device — not a cosmetic bug. Formula per the VK_KHR_ray_tracing_pipeline spec's
// own "Ray Tracing Pipeline Stack" section (raygen's own frame, plus one frame of the
// worst-case hit/miss group per recursion level — this pipeline has no any-hit/
// intersection/callable shaders, so those terms are always zero here).
uint32_t rt_pipeline_stack_size(RTScene* s, VkPipeline pipeline, uint32_t recursionDepth) {
    auto groupStack = [&](uint32_t group, VkShaderGroupShaderKHR shader) -> uint32_t {
        return s->rtFn.getGroupStackSize(s->ctx->device, pipeline, group, shader);
    };
    uint32_t raygenStack = groupStack(0, VK_SHADER_GROUP_SHADER_GENERAL_KHR);
    uint32_t missStack = std::max(groupStack(1, VK_SHADER_GROUP_SHADER_GENERAL_KHR),
                                  groupStack(2, VK_SHADER_GROUP_SHADER_GENERAL_KHR));
    uint32_t hitStack = groupStack(3, VK_SHADER_GROUP_SHADER_CLOSEST_HIT_KHR);
    uint32_t perLevel = std::max(missStack, hitStack);
    return raygenStack + recursionDepth * perLevel;
}

bool rt_build_4stage_pipeline(HeadContext* c, RTScene* s, const uint32_t* rgenSpv, int rgenLen,
                              const uint32_t* missSpv, int missLen, const uint32_t* shadowMissSpv,
                              int shadowMissLen, const uint32_t* chitSpv, int chitLen,
                              RTPipelineResources& out) {
    VkDescriptorSetLayoutBinding bindings[4]{};
    bindings[0] = {0, VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, 1,
                  VK_SHADER_STAGE_RAYGEN_BIT_KHR | VK_SHADER_STAGE_CLOSEST_HIT_BIT_KHR, nullptr};
    bindings[1] = {1, VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1, VK_SHADER_STAGE_RAYGEN_BIT_KHR, nullptr};
    bindings[2] = {2, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1, VK_SHADER_STAGE_RAYGEN_BIT_KHR, nullptr};
    // #2155: RAYGEN_BIT added so pathtrace_realistic.rgen can read specularIOR/
    // dispersionScale/dispersionAbbeNumber to decide its dispersion loop — harmless for
    // the PBI-345 harness pipeline sharing this binding layout, whose rgen doesn't
    // declare (and so never reads) the Params UBO at all.
    bindings[3] = {3, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1,
                   VK_SHADER_STAGE_RAYGEN_BIT_KHR | VK_SHADER_STAGE_CLOSEST_HIT_BIT_KHR, nullptr};
    VkDescriptorSetLayoutCreateInfo dslInfo{};
    dslInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
    dslInfo.bindingCount = 4;
    dslInfo.pBindings = bindings;
    if (!ok(vkCreateDescriptorSetLayout(c->device, &dslInfo, c->allocator, &out.dsLayout))) return false;

    VkPipelineLayoutCreateInfo plInfo{};
    plInfo.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    plInfo.setLayoutCount = 1;
    plInfo.pSetLayouts = &out.dsLayout;
    if (!ok(vkCreatePipelineLayout(c->device, &plInfo, c->allocator, &out.pipeLayout))) return false;

    VkShaderModule rgen = rt_load_shader(c, rgenSpv, rgenLen);
    VkShaderModule miss = rt_load_shader(c, missSpv, missLen);
    VkShaderModule shadowMiss = rt_load_shader(c, shadowMissSpv, shadowMissLen);
    VkShaderModule chit = rt_load_shader(c, chitSpv, chitLen);
    if (!rgen || !miss || !shadowMiss || !chit) return false;

    VkPipelineShaderStageCreateInfo stages[4]{};
    stages[0] = {VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, nullptr, 0,
                VK_SHADER_STAGE_RAYGEN_BIT_KHR,      rgen,       "main", nullptr};
    stages[1] = {VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, nullptr, 0,
                VK_SHADER_STAGE_MISS_BIT_KHR,        miss,       "main", nullptr};
    stages[2] = {VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, nullptr, 0,
                VK_SHADER_STAGE_MISS_BIT_KHR,        shadowMiss, "main", nullptr};
    stages[3] = {VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, nullptr, 0,
                VK_SHADER_STAGE_CLOSEST_HIT_BIT_KHR, chit,       "main", nullptr};

    // Group indices mirror stages[] 1:1 for the two GENERAL (raygen/miss) groups; the
    // hit group is its own (4th) group referencing stage index 3 as its closest-hit.
    VkRayTracingShaderGroupCreateInfoKHR groups[4]{};
    for (int i = 0; i < 3; i++) {
        groups[i].sType = VK_STRUCTURE_TYPE_RAY_TRACING_SHADER_GROUP_CREATE_INFO_KHR;
        groups[i].type = VK_RAY_TRACING_SHADER_GROUP_TYPE_GENERAL_KHR;
        groups[i].generalShader = (uint32_t)i;
        groups[i].closestHitShader = VK_SHADER_UNUSED_KHR;
        groups[i].anyHitShader = VK_SHADER_UNUSED_KHR;
        groups[i].intersectionShader = VK_SHADER_UNUSED_KHR;
    }
    groups[3].sType = VK_STRUCTURE_TYPE_RAY_TRACING_SHADER_GROUP_CREATE_INFO_KHR;
    groups[3].type = VK_RAY_TRACING_SHADER_GROUP_TYPE_TRIANGLES_HIT_GROUP_KHR;
    groups[3].generalShader = VK_SHADER_UNUSED_KHR;
    groups[3].closestHitShader = 3;
    // Opaque geometry (VK_GEOMETRY_OPAQUE_BIT_KHR on every BLAS, obk_rt_scene_add_mesh)
    // never invokes any-hit, so this harness has none — the "any-hit" part of PBI-345's
    // scope is trivially satisfied by opaque-only geometry, same as raytrace.comp's
    // ray-query path; an alpha-cutout material would need a real any-hit shader here.
    groups[3].anyHitShader = VK_SHADER_UNUSED_KHR;
    groups[3].intersectionShader = VK_SHADER_UNUSED_KHR;

    VkRayTracingPipelineCreateInfoKHR pInfo{};
    pInfo.sType = VK_STRUCTURE_TYPE_RAY_TRACING_PIPELINE_CREATE_INFO_KHR;
    pInfo.stageCount = 4;
    pInfo.pStages = stages;
    pInfo.groupCount = 4;
    pInfo.pGroups = groups;
    // #2155: primary ray + up to OPENPBR_MAX_TRANSMISSION_BOUNCES (extended_lobes.glsl)
    // recursive continuation rays through transmissive surfaces + one terminal shadow ray
    // fired from whichever level is deepest. The PBI-345 test-harness pipeline (this
    // function's OTHER caller, obk_rt_scene_build_pipeline) never recurses past its own
    // primary+shadow pair, so the higher budget is unused there, not unsafe.
    pInfo.maxPipelineRayRecursionDepth = 6;
    pInfo.layout = out.pipeLayout;
    VkResult pipeResult =
        s->rtFn.createRTPipeline(c->device, VK_NULL_HANDLE, VK_NULL_HANDLE, 1, &pInfo, c->allocator, &out.pipeline);
    vkDestroyShaderModule(c->device, rgen, c->allocator);
    vkDestroyShaderModule(c->device, miss, c->allocator);
    vkDestroyShaderModule(c->device, shadowMiss, c->allocator);
    vkDestroyShaderModule(c->device, chit, c->allocator);
    if (!ok(pipeResult)) return false;

    VkPhysicalDeviceRayTracingPipelinePropertiesKHR rtProps{};
    rtProps.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_RAY_TRACING_PIPELINE_PROPERTIES_KHR;
    VkPhysicalDeviceProperties2 props2{};
    props2.sType = VK_STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2;
    props2.pNext = &rtProps;
    vkGetPhysicalDeviceProperties2(c->physical, &props2);
    out.handleSize = rtProps.shaderGroupHandleSize;
    out.handleAlignment = rtProps.shaderGroupHandleAlignment;
    out.baseAlignment = rtProps.shaderGroupBaseAlignment;
    out.stackSize = rt_pipeline_stack_size(s, out.pipeline, pInfo.maxPipelineRayRecursionDepth);
    return true;
}

} // namespace

int obk_rt_scene_build_pipeline(void* scene, const uint32_t* rgenSpv, int rgenLen, const uint32_t* missSpv,
                                int missLen, const uint32_t* shadowMissSpv, int shadowMissLen,
                                const uint32_t* chitSpv, int chitLen) {
    RTScene* s = (RTScene*)scene;
    if (!s || !s->built || s->pipelineBuilt || !s->ctx->hwRayTracingPipeline) return 1;
    HeadContext* c = s->ctx;

    RTPipelineResources res;
    if (!rt_build_4stage_pipeline(c, s, rgenSpv, rgenLen, missSpv, missLen, shadowMissSpv, shadowMissLen,
                                  chitSpv, chitLen, res))
        return 1;
    s->pipeDsLayout = res.dsLayout;
    s->pipePipeLayout = res.pipeLayout;
    s->pipePipeline = res.pipeline;
    s->sbtHandleSize = res.handleSize;
    s->sbtHandleAlignment = res.handleAlignment;
    s->sbtBaseAlignment = res.baseAlignment;
    s->pipeStackSize = res.stackSize;

    std::vector<uint8_t> handles(4 * (size_t)s->sbtHandleSize);
    if (!ok(s->rtFn.getShaderGroupHandles(c->device, s->pipePipeline, 0, 4, handles.size(), handles.data())))
        return 1;

    VkDeviceSize entryStride = align_up(s->sbtHandleSize, s->sbtHandleAlignment);
    VkDeviceSize raygenOffset = 0;
    VkDeviceSize missOffset = align_up(raygenOffset + entryStride, s->sbtBaseAlignment);
    VkDeviceSize hitOffset = align_up(missOffset + 2 * entryStride, s->sbtBaseAlignment);
    VkDeviceSize sbtSize = align_up(hitOffset + entryStride, s->sbtBaseAlignment);

    std::vector<uint8_t> sbtData(sbtSize, 0);
    std::memcpy(sbtData.data() + raygenOffset, handles.data() + 0 * s->sbtHandleSize, s->sbtHandleSize);
    std::memcpy(sbtData.data() + missOffset, handles.data() + 1 * s->sbtHandleSize, s->sbtHandleSize);
    std::memcpy(sbtData.data() + missOffset + entryStride, handles.data() + 2 * s->sbtHandleSize, s->sbtHandleSize);
    std::memcpy(sbtData.data() + hitOffset, handles.data() + 3 * s->sbtHandleSize, s->sbtHandleSize);

    if (!rt_upload_buffer(c, &s->sbtBuf,
                          VK_BUFFER_USAGE_SHADER_BINDING_TABLE_BIT_KHR | VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
                          sbtData.data(), sbtSize))
        return 1;

    rt_create_buffer(c, &s->pipeCamBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, 32);
    vkMapMemory(c->device, s->pipeCamBuf.memory, 0, 32, 0, &s->pipeCamBuf.mapped);
    rt_create_buffer(c, &s->pipeParamsBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, 64);
    vkMapMemory(c->device, s->pipeParamsBuf.memory, 0, 64, 0, &s->pipeParamsBuf.mapped);
    rt_create_buffer(c, &s->pipeOutputBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, 16);
    vkMapMemory(c->device, s->pipeOutputBuf.memory, 0, 16, 0, &s->pipeOutputBuf.mapped);

    VkDescriptorPoolSize poolSizes[3] = {
        {VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, 1},
        {VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1},
        {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 2},
    };
    VkDescriptorPoolCreateInfo poolInfo{};
    poolInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    poolInfo.maxSets = 1;
    poolInfo.poolSizeCount = 3;
    poolInfo.pPoolSizes = poolSizes;
    if (!ok(vkCreateDescriptorPool(c->device, &poolInfo, c->allocator, &s->pipeDescPool))) return 1;

    VkDescriptorSetAllocateInfo dsAlloc{};
    dsAlloc.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    dsAlloc.descriptorPool = s->pipeDescPool;
    dsAlloc.descriptorSetCount = 1;
    dsAlloc.pSetLayouts = &s->pipeDsLayout;
    if (!ok(vkAllocateDescriptorSets(c->device, &dsAlloc, &s->pipeDescSet))) return 1;

    VkWriteDescriptorSetAccelerationStructureKHR asWrite{};
    asWrite.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET_ACCELERATION_STRUCTURE_KHR;
    asWrite.accelerationStructureCount = 1;
    asWrite.pAccelerationStructures = &s->tlas;
    VkDescriptorBufferInfo outInfo{s->pipeOutputBuf.buffer, 0, 16};
    VkDescriptorBufferInfo camInfo{s->pipeCamBuf.buffer, 0, 32};
    VkDescriptorBufferInfo paramInfo{s->pipeParamsBuf.buffer, 0, 64};
    VkWriteDescriptorSet writes[4]{};
    writes[0] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, &asWrite, s->pipeDescSet, 0, 0, 1,
                VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, nullptr, nullptr, nullptr};
    writes[1] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->pipeDescSet, 1, 0, 1,
                VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, nullptr, &outInfo, nullptr};
    writes[2] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->pipeDescSet, 2, 0, 1,
                VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, nullptr, &camInfo, nullptr};
    writes[3] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->pipeDescSet, 3, 0, 1,
                VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, nullptr, &paramInfo, nullptr};
    vkUpdateDescriptorSets(c->device, 4, writes, 0, nullptr);

    s->pipelineBuilt = true;
    return 0;
}

void obk_rt_scene_trace_pipeline(void* scene, float ox, float oy, float oz, float dx, float dy, float dz,
                                 float tMin, float tMax, const float* params, float* outR, float* outG,
                                 float* outB) {
    RTScene* s = (RTScene*)scene;
    if (!s || !s->pipelineBuilt) return;
    HeadContext* c = s->ctx;

    float camParams[8] = {ox, oy, oz, tMin, dx, dy, dz, tMax};
    std::memcpy(s->pipeCamBuf.mapped, camParams, sizeof(camParams));
    std::memcpy(s->pipeParamsBuf.mapped, params, 16 * sizeof(float));

    VkDeviceAddress sbtBase = s->sbtBuf.address;
    VkDeviceSize entryStride = align_up(s->sbtHandleSize, s->sbtHandleAlignment);
    VkDeviceSize missOffset = align_up(entryStride, s->sbtBaseAlignment);
    VkDeviceSize hitOffset = align_up(missOffset + 2 * entryStride, s->sbtBaseAlignment);

    VkStridedDeviceAddressRegionKHR raygenRegion{sbtBase + 0, entryStride, entryStride};
    VkStridedDeviceAddressRegionKHR missRegion{sbtBase + missOffset, entryStride, 2 * entryStride};
    VkStridedDeviceAddressRegionKHR hitRegion{sbtBase + hitOffset, entryStride, entryStride};
    VkStridedDeviceAddressRegionKHR callableRegion{};

    rt_run_commands(s, [&](VkCommandBuffer cmd) {
        vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_RAY_TRACING_KHR, s->pipePipeline);
        s->rtFn.cmdSetStackSize(cmd, s->pipeStackSize);
        vkCmdBindDescriptorSets(cmd, VK_PIPELINE_BIND_POINT_RAY_TRACING_KHR, s->pipePipeLayout, 0, 1,
                                &s->pipeDescSet, 0, nullptr);
        s->rtFn.cmdTraceRays(cmd, &raygenRegion, &missRegion, &hitRegion, &callableRegion, 1, 1, 1);
        VkMemoryBarrier barrier{};
        barrier.sType = VK_STRUCTURE_TYPE_MEMORY_BARRIER;
        barrier.srcAccessMask = VK_ACCESS_SHADER_WRITE_BIT;
        barrier.dstAccessMask = VK_ACCESS_HOST_READ_BIT;
        vkCmdPipelineBarrier(cmd, VK_PIPELINE_STAGE_RAY_TRACING_SHADER_BIT_KHR, VK_PIPELINE_STAGE_HOST_BIT, 0, 1,
                             &barrier, 0, nullptr, 0, nullptr);
    });

    auto* out = reinterpret_cast<float*>(s->pipeOutputBuf.mapped);
    if (outR) *outR = out[0];
    if (outG) *outG = out[1];
    if (outB) *outB = out[2];
}

// obk_rt_scene_build_realistic_pipeline mirrors obk_rt_scene_build_pipeline exactly
// (same 4-binding descriptor set shape: TLAS, output storage buffer, camera UBO, params
// UBO), except the camera UBO is 64 bytes (an eye/forward/right/up pinhole basis, not a
// single fixed ray — pathtrace_realistic.rgen's CameraParams) and the output buffer
// starts at a 1-pixel placeholder size, resized to the real width*height by the first
// obk_rt_scene_trace_realistic_image call (image dimensions aren't known yet here).
int obk_rt_scene_build_realistic_pipeline(void* scene, const uint32_t* rgenSpv, int rgenLen,
                                          const uint32_t* missSpv, int missLen,
                                          const uint32_t* shadowMissSpv, int shadowMissLen,
                                          const uint32_t* chitSpv, int chitLen) {
    RTScene* s = (RTScene*)scene;
    if (!s || !s->built || s->imgPipelineBuilt || !s->ctx->hwRayTracingPipeline) return 1;
    HeadContext* c = s->ctx;

    RTPipelineResources res;
    if (!rt_build_4stage_pipeline(c, s, rgenSpv, rgenLen, missSpv, missLen, shadowMissSpv, shadowMissLen,
                                  chitSpv, chitLen, res))
        return 1;
    s->imgDsLayout = res.dsLayout;
    s->imgPipeLayout = res.pipeLayout;
    s->imgPipeline = res.pipeline;
    s->imgSbtHandleSize = res.handleSize;
    s->imgSbtHandleAlignment = res.handleAlignment;
    s->imgSbtBaseAlignment = res.baseAlignment;
    s->imgStackSize = res.stackSize;

    std::vector<uint8_t> handles(4 * (size_t)s->imgSbtHandleSize);
    if (!ok(s->rtFn.getShaderGroupHandles(c->device, s->imgPipeline, 0, 4, handles.size(), handles.data())))
        return 1;

    VkDeviceSize entryStride = align_up(s->imgSbtHandleSize, s->imgSbtHandleAlignment);
    VkDeviceSize raygenOffset = 0;
    VkDeviceSize missOffset = align_up(raygenOffset + entryStride, s->imgSbtBaseAlignment);
    VkDeviceSize hitOffset = align_up(missOffset + 2 * entryStride, s->imgSbtBaseAlignment);
    VkDeviceSize sbtSize = align_up(hitOffset + entryStride, s->imgSbtBaseAlignment);

    std::vector<uint8_t> sbtData(sbtSize, 0);
    std::memcpy(sbtData.data() + raygenOffset, handles.data() + 0 * s->imgSbtHandleSize, s->imgSbtHandleSize);
    std::memcpy(sbtData.data() + missOffset, handles.data() + 1 * s->imgSbtHandleSize, s->imgSbtHandleSize);
    std::memcpy(sbtData.data() + missOffset + entryStride, handles.data() + 2 * s->imgSbtHandleSize,
               s->imgSbtHandleSize);
    std::memcpy(sbtData.data() + hitOffset, handles.data() + 3 * s->imgSbtHandleSize, s->imgSbtHandleSize);

    if (!rt_upload_buffer(c, &s->imgSbtBuf,
                          VK_BUFFER_USAGE_SHADER_BINDING_TABLE_BIT_KHR | VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
                          sbtData.data(), sbtSize))
        return 1;

    rt_create_buffer(c, &s->imgCamBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, 64);
    vkMapMemory(c->device, s->imgCamBuf.memory, 0, 64, 0, &s->imgCamBuf.mapped);
    rt_create_buffer(c, &s->imgParamsBuf, VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, kRealisticParamsBytes);
    vkMapMemory(c->device, s->imgParamsBuf.memory, 0, kRealisticParamsBytes, 0, &s->imgParamsBuf.mapped);
    rt_create_buffer(c, &s->imgOutputBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT,
                     VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, 16);
    vkMapMemory(c->device, s->imgOutputBuf.memory, 0, 16, 0, &s->imgOutputBuf.mapped);
    s->imgOutputWidth = 1;
    s->imgOutputHeight = 1;

    VkDescriptorPoolSize poolSizes[3] = {
        {VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, 1},
        {VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, 1},
        {VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 2},
    };
    VkDescriptorPoolCreateInfo poolInfo{};
    poolInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    poolInfo.maxSets = 1;
    poolInfo.poolSizeCount = 3;
    poolInfo.pPoolSizes = poolSizes;
    if (!ok(vkCreateDescriptorPool(c->device, &poolInfo, c->allocator, &s->imgDescPool))) return 1;

    VkDescriptorSetAllocateInfo dsAlloc{};
    dsAlloc.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    dsAlloc.descriptorPool = s->imgDescPool;
    dsAlloc.descriptorSetCount = 1;
    dsAlloc.pSetLayouts = &s->imgDsLayout;
    if (!ok(vkAllocateDescriptorSets(c->device, &dsAlloc, &s->imgDescSet))) return 1;

    VkWriteDescriptorSetAccelerationStructureKHR asWrite{};
    asWrite.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET_ACCELERATION_STRUCTURE_KHR;
    asWrite.accelerationStructureCount = 1;
    asWrite.pAccelerationStructures = &s->tlas;
    VkDescriptorBufferInfo outInfo{s->imgOutputBuf.buffer, 0, 16};
    VkDescriptorBufferInfo camInfo{s->imgCamBuf.buffer, 0, 64};
    VkDescriptorBufferInfo paramInfo{s->imgParamsBuf.buffer, 0, kRealisticParamsBytes};
    VkWriteDescriptorSet writes[4]{};
    writes[0] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, &asWrite, s->imgDescSet, 0, 0, 1,
                VK_DESCRIPTOR_TYPE_ACCELERATION_STRUCTURE_KHR, nullptr, nullptr, nullptr};
    writes[1] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->imgDescSet, 1, 0, 1,
                VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, nullptr, &outInfo, nullptr};
    writes[2] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->imgDescSet, 2, 0, 1,
                VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, nullptr, &camInfo, nullptr};
    writes[3] = {VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->imgDescSet, 3, 0, 1,
                VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, nullptr, &paramInfo, nullptr};
    vkUpdateDescriptorSets(c->device, 4, writes, 0, nullptr);

    s->imgPipelineBuilt = true;
    return 0;
}

// obk_rt_scene_trace_realistic_image dispatches one vkCmdTraceRaysKHR(width,height,1)
// call — one primary+shadow ray pair per pixel — and reads back the resulting
// width*height RGB image. Resizes imgOutputBuf (and re-writes its descriptor) only when
// width*height actually changes from the previous call (e.g. a viewport panel resize),
// not on every call.
int obk_rt_scene_trace_realistic_image(void* scene, int width, int height, const float* camera,
                                       const float* params, float* outPixels) {
    RTScene* s = (RTScene*)scene;
    if (!s || !s->imgPipelineBuilt || width <= 0 || height <= 0) return 1;
    HeadContext* c = s->ctx;

    if (width != s->imgOutputWidth || height != s->imgOutputHeight) {
        vkDeviceWaitIdle(c->device); // the old buffer may still be read by a prior frame's GPU work
        rt_destroy_buffer(c, &s->imgOutputBuf);
        VkDeviceSize bytes = (VkDeviceSize)width * height * 16;
        if (!rt_create_buffer(c, &s->imgOutputBuf, VK_BUFFER_USAGE_STORAGE_BUFFER_BIT,
                              VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, bytes))
            return 1;
        vkMapMemory(c->device, s->imgOutputBuf.memory, 0, bytes, 0, &s->imgOutputBuf.mapped);
        s->imgOutputWidth = width;
        s->imgOutputHeight = height;

        VkDescriptorBufferInfo outInfo{s->imgOutputBuf.buffer, 0, bytes};
        VkWriteDescriptorSet write{VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr, s->imgDescSet, 1, 0, 1,
                                   VK_DESCRIPTOR_TYPE_STORAGE_BUFFER, nullptr, &outInfo, nullptr};
        vkUpdateDescriptorSets(c->device, 1, &write, 0, nullptr);
    }

    std::memcpy(s->imgCamBuf.mapped, camera, 16 * sizeof(float));
    std::memcpy(s->imgParamsBuf.mapped, params, kRealisticParamsBytes);

    VkDeviceAddress sbtBase = s->imgSbtBuf.address;
    VkDeviceSize entryStride = align_up(s->imgSbtHandleSize, s->imgSbtHandleAlignment);
    VkDeviceSize missOffset = align_up(entryStride, s->imgSbtBaseAlignment);
    VkDeviceSize hitOffset = align_up(missOffset + 2 * entryStride, s->imgSbtBaseAlignment);

    VkStridedDeviceAddressRegionKHR raygenRegion{sbtBase + 0, entryStride, entryStride};
    VkStridedDeviceAddressRegionKHR missRegion{sbtBase + missOffset, entryStride, 2 * entryStride};
    VkStridedDeviceAddressRegionKHR hitRegion{sbtBase + hitOffset, entryStride, entryStride};
    VkStridedDeviceAddressRegionKHR callableRegion{};

    rt_run_commands(s, [&](VkCommandBuffer cmd) {
        vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_RAY_TRACING_KHR, s->imgPipeline);
        s->rtFn.cmdSetStackSize(cmd, s->imgStackSize);
        vkCmdBindDescriptorSets(cmd, VK_PIPELINE_BIND_POINT_RAY_TRACING_KHR, s->imgPipeLayout, 0, 1,
                                &s->imgDescSet, 0, nullptr);
        s->rtFn.cmdTraceRays(cmd, &raygenRegion, &missRegion, &hitRegion, &callableRegion,
                            (uint32_t)width, (uint32_t)height, 1);
        VkMemoryBarrier barrier{};
        barrier.sType = VK_STRUCTURE_TYPE_MEMORY_BARRIER;
        barrier.srcAccessMask = VK_ACCESS_SHADER_WRITE_BIT;
        barrier.dstAccessMask = VK_ACCESS_HOST_READ_BIT;
        vkCmdPipelineBarrier(cmd, VK_PIPELINE_STAGE_RAY_TRACING_SHADER_BIT_KHR, VK_PIPELINE_STAGE_HOST_BIT, 0, 1,
                             &barrier, 0, nullptr, 0, nullptr);
    });

    auto* out = reinterpret_cast<float*>(s->imgOutputBuf.mapped);
    for (int i = 0; i < width * height; i++) {
        outPixels[i * 3 + 0] = out[i * 4 + 0];
        outPixels[i * 3 + 1] = out[i * 4 + 1];
        outPixels[i * 3 + 2] = out[i * 4 + 2];
    }
    return 0;
}

void obk_rt_scene_destroy(void* scene) {
    RTScene* s = (RTScene*)scene;
    if (!s) return;
    HeadContext* c = s->ctx;
    vkDeviceWaitIdle(c->device);
    if (s->rqPipeline) vkDestroyPipeline(c->device, s->rqPipeline, c->allocator);
    if (s->rqPipeLayout) vkDestroyPipelineLayout(c->device, s->rqPipeLayout, c->allocator);
    if (s->rqDsLayout) vkDestroyDescriptorSetLayout(c->device, s->rqDsLayout, c->allocator);
    if (s->rqDescPool) vkDestroyDescriptorPool(c->device, s->rqDescPool, c->allocator);
    rt_destroy_buffer(c, &s->rayParamsBuf);
    rt_destroy_buffer(c, &s->resultBuf);
    if (s->pipePipeline) vkDestroyPipeline(c->device, s->pipePipeline, c->allocator);
    if (s->pipePipeLayout) vkDestroyPipelineLayout(c->device, s->pipePipeLayout, c->allocator);
    if (s->pipeDsLayout) vkDestroyDescriptorSetLayout(c->device, s->pipeDsLayout, c->allocator);
    if (s->pipeDescPool) vkDestroyDescriptorPool(c->device, s->pipeDescPool, c->allocator);
    rt_destroy_buffer(c, &s->sbtBuf);
    rt_destroy_buffer(c, &s->pipeCamBuf);
    rt_destroy_buffer(c, &s->pipeOutputBuf);
    rt_destroy_buffer(c, &s->pipeParamsBuf);
    if (s->imgPipeline) vkDestroyPipeline(c->device, s->imgPipeline, c->allocator);
    if (s->imgPipeLayout) vkDestroyPipelineLayout(c->device, s->imgPipeLayout, c->allocator);
    if (s->imgDsLayout) vkDestroyDescriptorSetLayout(c->device, s->imgDsLayout, c->allocator);
    if (s->imgDescPool) vkDestroyDescriptorPool(c->device, s->imgDescPool, c->allocator);
    rt_destroy_buffer(c, &s->imgSbtBuf);
    rt_destroy_buffer(c, &s->imgCamBuf);
    rt_destroy_buffer(c, &s->imgOutputBuf);
    rt_destroy_buffer(c, &s->imgParamsBuf);
    if (s->tlas) s->rtFn.destroy(c->device, s->tlas, c->allocator);
    rt_destroy_buffer(c, &s->tlasBuffer);
    rt_destroy_buffer(c, &s->instanceBuf);
    for (auto& b : s->blases) {
        if (b.handle) s->rtFn.destroy(c->device, b.handle, c->allocator);
        rt_destroy_buffer(c, &b.asBuffer);
        rt_destroy_buffer(c, &b.vertexBuf);
        rt_destroy_buffer(c, &b.indexBuf);
    }
    if (s->fence) vkDestroyFence(c->device, s->fence, c->allocator);
    if (s->cmdPool) vkDestroyCommandPool(c->device, s->cmdPool, c->allocator);
    delete s;
}

} // extern "C"
