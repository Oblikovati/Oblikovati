// The 3D viewport: renders our geometry (renderer.DrawList, flattened by Go) into an
// OFFSCREEN color+depth target, then hands that color image to Dear ImGui as a sampled
// texture so it appears inside the dockable "Viewport" panel. Offscreen (rather than
// drawing into the swapchain pass) is what lets the 3D view live in a panel and gives
// us a depth buffer the ImGui main pass doesn't provide.
//
// Two pipelines share one vertex format (pos/normal/color, 10 floats): triangles are
// Lambert-lit, lines are flat (push-constant `lit` flag). Geometry is uploaded into
// host-visible buffers each render; fine at CAD-part vertex counts.
#include "head.h"
#include <cstring>
#include <vector>

namespace {
bool ok(VkResult r) { return r == VK_SUCCESS; }
constexpr uint32_t kVertexFloats = 10; // vec3 pos + vec3 normal + vec4 color
constexpr VkFormat kDepthFormat = VK_FORMAT_D32_SFLOAT;

struct PushConstants {
    float mvp[16];
    int32_t lit;
};

struct GpuBuffer {
    VkBuffer       buffer = VK_NULL_HANDLE;
    VkDeviceMemory memory = VK_NULL_HANDLE;
    VkDeviceSize   size = 0;
};
} // namespace

struct Viewport {
    VkFormat        colorFormat = VK_FORMAT_UNDEFINED;
    VkRenderPass    renderPass = VK_NULL_HANDLE;
    VkPipelineLayout layout = VK_NULL_HANDLE;
    VkPipeline      triPipeline = VK_NULL_HANDLE;
    VkPipeline      linePipeline = VK_NULL_HANDLE;
    VkShaderModule  vertModule = VK_NULL_HANDLE;
    VkShaderModule  fragModule = VK_NULL_HANDLE;
    VkSampler       sampler = VK_NULL_HANDLE;
    VkCommandPool   cmdPool = VK_NULL_HANDLE;
    VkCommandBuffer cmd = VK_NULL_HANDLE;
    VkFence         fence = VK_NULL_HANDLE;

    // Size-dependent target (recreated on resize).
    int             width = 0, height = 0;
    VkImage         colorImage = VK_NULL_HANDLE;
    VkDeviceMemory  colorMem = VK_NULL_HANDLE;
    VkImageView     colorView = VK_NULL_HANDLE;
    VkImage         depthImage = VK_NULL_HANDLE;
    VkDeviceMemory  depthMem = VK_NULL_HANDLE;
    VkImageView     depthView = VK_NULL_HANDLE;
    VkFramebuffer   framebuffer = VK_NULL_HANDLE;
    VkDescriptorSet texture = VK_NULL_HANDLE; // ImGui sampled-image set

    GpuBuffer       vbuf, ibuf;

    // Background the 3D pass clears to (themed; ADR-0021). Defaults reproduce the
    // pre-theming look so an un-themed build is unchanged.
    float           clearR = 0.10f, clearG = 0.11f, clearB = 0.13f;
};

namespace {

VkShaderModule make_module(VkDevice dev, const uint32_t* code, int len) {
    VkShaderModuleCreateInfo ci{};
    ci.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO;
    ci.codeSize = (size_t)len;
    ci.pCode = code;
    VkShaderModule m = VK_NULL_HANDLE;
    vkCreateShaderModule(dev, &ci, nullptr, &m);
    return m;
}

void create_render_pass(HeadContext* c, Viewport* v) {
    VkAttachmentDescription atts[2]{};
    atts[0].format = v->colorFormat;
    atts[0].samples = VK_SAMPLE_COUNT_1_BIT;
    atts[0].loadOp = VK_ATTACHMENT_LOAD_OP_CLEAR;
    atts[0].storeOp = VK_ATTACHMENT_STORE_OP_STORE;
    atts[0].stencilLoadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE;
    atts[0].stencilStoreOp = VK_ATTACHMENT_STORE_OP_DONT_CARE;
    atts[0].initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    atts[0].finalLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL;
    atts[1].format = kDepthFormat;
    atts[1].samples = VK_SAMPLE_COUNT_1_BIT;
    atts[1].loadOp = VK_ATTACHMENT_LOAD_OP_CLEAR;
    atts[1].storeOp = VK_ATTACHMENT_STORE_OP_DONT_CARE;
    atts[1].stencilLoadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE;
    atts[1].stencilStoreOp = VK_ATTACHMENT_STORE_OP_DONT_CARE;
    atts[1].initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    atts[1].finalLayout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL;

    VkAttachmentReference colorRef{0, VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL};
    VkAttachmentReference depthRef{1, VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL};
    VkSubpassDescription sub{};
    sub.pipelineBindPoint = VK_PIPELINE_BIND_POINT_GRAPHICS;
    sub.colorAttachmentCount = 1;
    sub.pColorAttachments = &colorRef;
    sub.pDepthStencilAttachment = &depthRef;

    VkSubpassDependency dep{};
    dep.srcSubpass = VK_SUBPASS_EXTERNAL;
    dep.dstSubpass = 0;
    dep.srcStageMask = VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT;
    dep.dstStageMask = VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT;
    dep.srcAccessMask = VK_ACCESS_SHADER_READ_BIT;
    dep.dstAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT;

    VkRenderPassCreateInfo rp{};
    rp.sType = VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO;
    rp.attachmentCount = 2;
    rp.pAttachments = atts;
    rp.subpassCount = 1;
    rp.pSubpasses = &sub;
    rp.dependencyCount = 1;
    rp.pDependencies = &dep;
    vkCreateRenderPass(c->device, &rp, nullptr, &v->renderPass);
}

VkPipeline create_pipeline(HeadContext* c, Viewport* v, VkPrimitiveTopology topo,
                           VkPolygonMode poly) {
    VkPipelineShaderStageCreateInfo stages[2]{};
    stages[0].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[0].stage = VK_SHADER_STAGE_VERTEX_BIT;
    stages[0].module = v->vertModule;
    stages[0].pName = "main";
    stages[1].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[1].stage = VK_SHADER_STAGE_FRAGMENT_BIT;
    stages[1].module = v->fragModule;
    stages[1].pName = "main";

    VkVertexInputBindingDescription bind{0, kVertexFloats * sizeof(float),
                                         VK_VERTEX_INPUT_RATE_VERTEX};
    VkVertexInputAttributeDescription attrs[3] = {
        {0, 0, VK_FORMAT_R32G32B32_SFLOAT, 0},
        {1, 0, VK_FORMAT_R32G32B32_SFLOAT, 3 * sizeof(float)},
        {2, 0, VK_FORMAT_R32G32B32A32_SFLOAT, 6 * sizeof(float)},
    };
    VkPipelineVertexInputStateCreateInfo vi{};
    vi.sType = VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO;
    vi.vertexBindingDescriptionCount = 1;
    vi.pVertexBindingDescriptions = &bind;
    vi.vertexAttributeDescriptionCount = 3;
    vi.pVertexAttributeDescriptions = attrs;

    VkPipelineInputAssemblyStateCreateInfo ia{};
    ia.sType = VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO;
    ia.topology = topo;

    VkPipelineViewportStateCreateInfo vp{};
    vp.sType = VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO;
    vp.viewportCount = 1;
    vp.scissorCount = 1;

    VkPipelineRasterizationStateCreateInfo rs{};
    rs.sType = VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO;
    rs.polygonMode = poly;
    rs.cullMode = VK_CULL_MODE_NONE;
    rs.frontFace = VK_FRONT_FACE_COUNTER_CLOCKWISE;
    rs.lineWidth = 1.0f;

    VkPipelineMultisampleStateCreateInfo ms{};
    ms.sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO;
    ms.rasterizationSamples = VK_SAMPLE_COUNT_1_BIT;

    VkPipelineDepthStencilStateCreateInfo ds{};
    ds.sType = VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO;
    ds.depthTestEnable = VK_TRUE;
    ds.depthWriteEnable = VK_TRUE;
    ds.depthCompareOp = VK_COMPARE_OP_LESS_OR_EQUAL;

    VkPipelineColorBlendAttachmentState cba{};
    cba.colorWriteMask = VK_COLOR_COMPONENT_R_BIT | VK_COLOR_COMPONENT_G_BIT |
                         VK_COLOR_COMPONENT_B_BIT | VK_COLOR_COMPONENT_A_BIT;
    VkPipelineColorBlendStateCreateInfo cb{};
    cb.sType = VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO;
    cb.attachmentCount = 1;
    cb.pAttachments = &cba;

    VkDynamicState dyn[] = {VK_DYNAMIC_STATE_VIEWPORT, VK_DYNAMIC_STATE_SCISSOR};
    VkPipelineDynamicStateCreateInfo dynState{};
    dynState.sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO;
    dynState.dynamicStateCount = 2;
    dynState.pDynamicStates = dyn;

    VkGraphicsPipelineCreateInfo ci{};
    ci.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO;
    ci.stageCount = 2;
    ci.pStages = stages;
    ci.pVertexInputState = &vi;
    ci.pInputAssemblyState = &ia;
    ci.pViewportState = &vp;
    ci.pRasterizationState = &rs;
    ci.pMultisampleState = &ms;
    ci.pDepthStencilState = &ds;
    ci.pColorBlendState = &cb;
    ci.pDynamicState = &dynState;
    ci.layout = v->layout;
    ci.renderPass = v->renderPass;
    VkPipeline p = VK_NULL_HANDLE;
    vkCreateGraphicsPipelines(c->device, VK_NULL_HANDLE, 1, &ci, nullptr, &p);
    return p;
}

void upload(HeadContext* c, GpuBuffer* b, VkBufferUsageFlags usage, const void* data,
            VkDeviceSize bytes) {
    if (bytes == 0) return;
    if (b->size < bytes) { // grow
        if (b->buffer) vkDestroyBuffer(c->device, b->buffer, nullptr);
        if (b->memory) vkFreeMemory(c->device, b->memory, nullptr);
        VkBufferCreateInfo bi{};
        bi.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
        bi.size = bytes;
        bi.usage = usage;
        bi.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
        vkCreateBuffer(c->device, &bi, nullptr, &b->buffer);
        VkMemoryRequirements req;
        vkGetBufferMemoryRequirements(c->device, b->buffer, &req);
        VkMemoryAllocateInfo ai{};
        ai.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
        ai.allocationSize = req.size;
        ai.memoryTypeIndex = obk_find_memory_type(c->physical, req.memoryTypeBits,
            VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT);
        vkAllocateMemory(c->device, &ai, nullptr, &b->memory);
        vkBindBufferMemory(c->device, b->buffer, b->memory, 0);
        b->size = bytes;
    }
    void* mapped = nullptr;
    vkMapMemory(c->device, b->memory, 0, bytes, 0, &mapped);
    std::memcpy(mapped, data, (size_t)bytes);
    vkUnmapMemory(c->device, b->memory);
}

VkImageView make_image(HeadContext* c, VkFormat fmt, VkImageUsageFlags usage,
                       VkImageAspectFlags aspect, int w, int h, VkImage* image,
                       VkDeviceMemory* mem) {
    VkImageCreateInfo ii{};
    ii.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
    ii.imageType = VK_IMAGE_TYPE_2D;
    ii.format = fmt;
    ii.extent = {(uint32_t)w, (uint32_t)h, 1};
    ii.mipLevels = 1;
    ii.arrayLayers = 1;
    ii.samples = VK_SAMPLE_COUNT_1_BIT;
    ii.tiling = VK_IMAGE_TILING_OPTIMAL;
    ii.usage = usage;
    ii.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    vkCreateImage(c->device, &ii, nullptr, image);
    VkMemoryRequirements req;
    vkGetImageMemoryRequirements(c->device, *image, &req);
    VkMemoryAllocateInfo ai{};
    ai.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    ai.allocationSize = req.size;
    ai.memoryTypeIndex = obk_find_memory_type(c->physical, req.memoryTypeBits,
                                              VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    vkAllocateMemory(c->device, &ai, nullptr, mem);
    vkBindImageMemory(c->device, *image, *mem, 0);
    VkImageViewCreateInfo vi{};
    vi.sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO;
    vi.image = *image;
    vi.viewType = VK_IMAGE_VIEW_TYPE_2D;
    vi.format = fmt;
    vi.subresourceRange = {aspect, 0, 1, 0, 1};
    VkImageView view = VK_NULL_HANDLE;
    vkCreateImageView(c->device, &vi, nullptr, &view);
    return view;
}

void destroy_target(HeadContext* c, Viewport* v) {
    if (v->texture) { ImGui_ImplVulkan_RemoveTexture(v->texture); v->texture = VK_NULL_HANDLE; }
    if (v->framebuffer) vkDestroyFramebuffer(c->device, v->framebuffer, nullptr);
    if (v->colorView) vkDestroyImageView(c->device, v->colorView, nullptr);
    if (v->colorImage) vkDestroyImage(c->device, v->colorImage, nullptr);
    if (v->colorMem) vkFreeMemory(c->device, v->colorMem, nullptr);
    if (v->depthView) vkDestroyImageView(c->device, v->depthView, nullptr);
    if (v->depthImage) vkDestroyImage(c->device, v->depthImage, nullptr);
    if (v->depthMem) vkFreeMemory(c->device, v->depthMem, nullptr);
    v->framebuffer = VK_NULL_HANDLE;
    v->colorView = v->depthView = VK_NULL_HANDLE;
    v->colorImage = v->depthImage = VK_NULL_HANDLE;
    v->colorMem = v->depthMem = VK_NULL_HANDLE;
}

void ensure_target(HeadContext* c, Viewport* v, int w, int h) {
    if (w == v->width && h == v->height && v->framebuffer != VK_NULL_HANDLE) return;
    vkDeviceWaitIdle(c->device);
    destroy_target(c, v);
    v->width = w;
    v->height = h;
    v->colorView = make_image(c, v->colorFormat,
        VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT | VK_IMAGE_USAGE_SAMPLED_BIT,
        VK_IMAGE_ASPECT_COLOR_BIT, w, h, &v->colorImage, &v->colorMem);
    v->depthView = make_image(c, kDepthFormat,
        VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT, VK_IMAGE_ASPECT_DEPTH_BIT, w, h,
        &v->depthImage, &v->depthMem);
    VkImageView atts[2] = {v->colorView, v->depthView};
    VkFramebufferCreateInfo fb{};
    fb.sType = VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO;
    fb.renderPass = v->renderPass;
    fb.attachmentCount = 2;
    fb.pAttachments = atts;
    fb.width = (uint32_t)w;
    fb.height = (uint32_t)h;
    fb.layers = 1;
    vkCreateFramebuffer(c->device, &fb, nullptr, &v->framebuffer);
    v->texture = ImGui_ImplVulkan_AddTexture(v->sampler, v->colorView,
                                             VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL);
}

} // namespace

extern "C" {

void obk_viewport_init(void* h, const uint32_t* vert, int vlen, const uint32_t* frag,
                       int flen) {
    HeadContext* c = (HeadContext*)h;
    Viewport* v = new Viewport();
    c->viewport = v;
    v->colorFormat = c->window_data.SurfaceFormat.format;
    v->vertModule = make_module(c->device, vert, vlen);
    v->fragModule = make_module(c->device, frag, flen);

    VkPushConstantRange pc{VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT, 0,
                           sizeof(PushConstants)};
    VkPipelineLayoutCreateInfo li{};
    li.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    li.pushConstantRangeCount = 1;
    li.pPushConstantRanges = &pc;
    vkCreatePipelineLayout(c->device, &li, nullptr, &v->layout);

    create_render_pass(c, v);
    v->triPipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
                                     VK_POLYGON_MODE_FILL);
    v->linePipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_LINE_LIST,
                                      VK_POLYGON_MODE_FILL);

    VkSamplerCreateInfo si{};
    si.sType = VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO;
    si.magFilter = si.minFilter = VK_FILTER_LINEAR;
    si.addressModeU = si.addressModeV = si.addressModeW = VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE;
    vkCreateSampler(c->device, &si, nullptr, &v->sampler);

    VkCommandPoolCreateInfo cp{};
    cp.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO;
    cp.flags = VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT;
    cp.queueFamilyIndex = c->queueFamily;
    vkCreateCommandPool(c->device, &cp, nullptr, &v->cmdPool);
    VkCommandBufferAllocateInfo cb{};
    cb.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    cb.commandPool = v->cmdPool;
    cb.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    cb.commandBufferCount = 1;
    vkAllocateCommandBuffers(c->device, &cb, &v->cmd);
    VkFenceCreateInfo fi{};
    fi.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO;
    vkCreateFence(c->device, &fi, nullptr, &v->fence);
}

// obk_viewport_render uploads the flattened geometry, records the offscreen pass, and
// submits it (waiting on a fence so the color image is ready to sample this frame).
void obk_viewport_render(void* h, int w, int hh, const float* mvp, int lit_unused,
                         const float* triV, int triVC, const uint32_t* triIdx, int triIC,
                         const float* lineV, int lineVC, const uint32_t* lineIdx, int lineIC) {
    (void)lit_unused;
    HeadContext* c = (HeadContext*)h;
    Viewport* v = c->viewport;
    if (!v || w <= 0 || hh <= 0) return;
    ensure_target(c, v, w, hh);

    // One interleaved vertex buffer (tris then lines), one index buffer (tri then line).
    std::vector<float> verts;
    verts.reserve((size_t)(triVC + lineVC) * kVertexFloats);
    verts.insert(verts.end(), triV, triV + (size_t)triVC * kVertexFloats);
    verts.insert(verts.end(), lineV, lineV + (size_t)lineVC * kVertexFloats);
    std::vector<uint32_t> idx;
    idx.reserve((size_t)(triIC + lineIC));
    idx.insert(idx.end(), triIdx, triIdx + triIC);
    idx.insert(idx.end(), lineIdx, lineIdx + lineIC);
    upload(c, &v->vbuf, VK_BUFFER_USAGE_VERTEX_BUFFER_BIT, verts.data(),
           verts.size() * sizeof(float));
    upload(c, &v->ibuf, VK_BUFFER_USAGE_INDEX_BUFFER_BIT, idx.data(),
           idx.size() * sizeof(uint32_t));

    vkResetCommandBuffer(v->cmd, 0);
    VkCommandBufferBeginInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    bi.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(v->cmd, &bi);

    VkClearValue clears[2];
    clears[0].color = {{v->clearR, v->clearG, v->clearB, 1.0f}};
    clears[1].depthStencil = {1.0f, 0};
    VkRenderPassBeginInfo rp{};
    rp.sType = VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO;
    rp.renderPass = v->renderPass;
    rp.framebuffer = v->framebuffer;
    rp.renderArea.extent = {(uint32_t)w, (uint32_t)hh};
    rp.clearValueCount = 2;
    rp.pClearValues = clears;
    vkCmdBeginRenderPass(v->cmd, &rp, VK_SUBPASS_CONTENTS_INLINE);

    VkViewport vpRect{0, 0, (float)w, (float)hh, 0.0f, 1.0f};
    VkRect2D scissor{{0, 0}, {(uint32_t)w, (uint32_t)hh}};
    vkCmdSetViewport(v->cmd, 0, 1, &vpRect);
    vkCmdSetScissor(v->cmd, 0, 1, &scissor);

    if (!verts.empty()) {
        VkDeviceSize zero = 0;
        vkCmdBindVertexBuffers(v->cmd, 0, 1, &v->vbuf.buffer, &zero);
        vkCmdBindIndexBuffer(v->cmd, v->ibuf.buffer, 0, VK_INDEX_TYPE_UINT32);
        PushConstants push{};
        std::memcpy(push.mvp, mvp, sizeof(push.mvp));
        if (triIC > 0) {
            push.lit = 1;
            vkCmdPushConstants(v->cmd, v->layout,
                VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT, 0,
                sizeof(push), &push);
            vkCmdBindPipeline(v->cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->triPipeline);
            vkCmdDrawIndexed(v->cmd, (uint32_t)triIC, 1, 0, 0, 0);
        }
        if (lineIC > 0) {
            push.lit = 0;
            vkCmdPushConstants(v->cmd, v->layout,
                VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT, 0,
                sizeof(push), &push);
            vkCmdBindPipeline(v->cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->linePipeline);
            vkCmdDrawIndexed(v->cmd, (uint32_t)lineIC, 1, (uint32_t)triIC, triVC, 0);
        }
    }

    vkCmdEndRenderPass(v->cmd);
    vkEndCommandBuffer(v->cmd);

    VkSubmitInfo submit{};
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &v->cmd;
    vkResetFences(c->device, 1, &v->fence);
    vkQueueSubmit(c->queue, 1, &submit, v->fence);
    vkWaitForFences(c->device, 1, &v->fence, VK_TRUE, UINT64_MAX);
}

// obk_viewport_set_clear sets the 3D pass background (themed). Takes effect on the next
// render; a no-op before the viewport is initialized.
void obk_viewport_set_clear(void* h, float r, float g, float b) {
    HeadContext* c = (HeadContext*)h;
    if (!c->viewport) return;
    c->viewport->clearR = r;
    c->viewport->clearG = g;
    c->viewport->clearB = b;
}

// obk_viewport_texture returns the ImGui texture handle for the rendered color image
// (0 before the first render), to be drawn with ImGui::Image.
uint64_t obk_viewport_texture(void* h) {
    HeadContext* c = (HeadContext*)h;
    if (!c->viewport) return 0;
    return (uint64_t)c->viewport->texture;
}

void obk_viewport_destroy(HeadContext* c) {
    Viewport* v = c->viewport;
    if (!v) return;
    destroy_target(c, v);
    if (v->vbuf.buffer) vkDestroyBuffer(c->device, v->vbuf.buffer, nullptr);
    if (v->vbuf.memory) vkFreeMemory(c->device, v->vbuf.memory, nullptr);
    if (v->ibuf.buffer) vkDestroyBuffer(c->device, v->ibuf.buffer, nullptr);
    if (v->ibuf.memory) vkFreeMemory(c->device, v->ibuf.memory, nullptr);
    vkDestroyFence(c->device, v->fence, nullptr);
    vkDestroyCommandPool(c->device, v->cmdPool, nullptr);
    vkDestroySampler(c->device, v->sampler, nullptr);
    vkDestroyPipeline(c->device, v->triPipeline, nullptr);
    vkDestroyPipeline(c->device, v->linePipeline, nullptr);
    vkDestroyPipelineLayout(c->device, v->layout, nullptr);
    vkDestroyRenderPass(c->device, v->renderPass, nullptr);
    vkDestroyShaderModule(c->device, v->vertModule, nullptr);
    vkDestroyShaderModule(c->device, v->fragModule, nullptr);
    delete v;
    c->viewport = nullptr;
}

} // extern "C"
