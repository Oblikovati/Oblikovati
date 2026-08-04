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
// vec3 pos + vec3 normal + vec4 color + metallic + roughness + vec3 emissive + mode.
constexpr uint32_t kVertexFloats = 16;
// Point-cloud vertex: vec3 pos + vec4 rgba (#645). Compact and separate from the 16-float mesh
// vertex — points carry no normal/PBR/instance data, so the retained buffer stays small.
constexpr uint32_t kPointFloats = 7;

// Instanced draw records (ADR-0038). Each record is kDrawRecInts int32s describing one
// (source-mesh stream × instance set) draw: {stream, firstIndex, indexCount, vertexOffset,
// firstInstance, instanceCount, biased}. stream selects the pipeline; biased marks a depth-pushed
// overlay fill. Records are ordered by stream so the pipeline binds change minimally.
constexpr int kDrawRecInts = 7;
constexpr int32_t kStreamOcc = 0, kStreamTri = 1, kStreamLine = 2, kStreamHid = 3,
                  kStreamTopTri = 4, kStreamTopLine = 5,
                  kStreamWideLine = 6, kStreamTopWideLine = 7; // stroked lines (#2015)
constexpr VkFormat kDepthFormat = VK_FORMAT_D32_SFLOAT;

// Scene UBO layout in floats (std140), matching the Scene block in mesh.frag:
//   [0..99]   header (vec4) + 8 lights (3 vec4 each) — written by viewport.PackLighting
//   [100..103] env vec4    (enabled, rotation, iblIntensity, maxLod) — set_environment
//   [104..119] lightVP mat4 (sun shadow light-space matrix)          — set_shadow
//   [120..123] shadow vec4  (mapRendered, density, softness, texel)  — set_shadow
//   [124..127] shadow2 vec4 (castOnDirect, occludeAmbient, _, _)     — set_shadow
constexpr uint32_t kEnvBlock = 4 + 8 * 12; // 100
constexpr uint32_t kShadowVP = kEnvBlock + 4; // 104
constexpr uint32_t kShadowParams = kShadowVP + 16; // 120
constexpr uint32_t kShadow2 = kShadowParams + 4; // 124
constexpr uint32_t kSceneFloats = kShadow2 + 4; // 128
constexpr VkFormat kEnvFormat = VK_FORMAT_R32G32B32A32_SFLOAT;
constexpr uint32_t kShadowDim = 2048; // sun shadow map resolution

// camPosLit packs the camera eye (xyz, world space, for the PBR view vector) and the lit flag
// (w: 0 = line/flat, 1 = surface) into one vec4 so the push block stays 16-byte aligned.
struct PushConstants {
    float mvp[16];
    float camPosLit[4];
    float clip[4]; // section plane (M12-F04): xyz = world normal (0 ⇒ none), w = offset d
    // viewport.xy = framebuffer size in pixels, which the wide-line shader needs to turn a stroke
    // width in pixels into a clip-space offset (#2015). Shaders that do not use it declare a
    // shorter block; a pipeline layout range may exceed what a shader reads.
    float viewport[4];
};

struct GpuBuffer {
    VkBuffer       buffer = VK_NULL_HANDLE;
    VkDeviceMemory memory = VK_NULL_HANDLE;
    VkDeviceSize   size = 0;
    void*          mapped = nullptr; // persistent mapping for HOST_VISIBLE geometry (#1422)
};
} // namespace

// kMaxTiles bounds how many views can render at once (the quad layout); each gets its own
// offscreen Target so simultaneous tiles show distinct cameras (ImGui draws are deferred,
// so they cannot share one image). See ADR — per-document multiple views.
static const int kMaxTiles = 4;

// kFramesInFlight rings the per-frame offscreen resources so the CPU never blocks on the
// just-submitted fence (#1421). 2 is enough to overlap one frame's GPU work with the next frame's
// CPU recording; it matches the app's triple-buffered swapchain throttle well enough. A static
// scene being orbited then pipelines instead of fully serialising CPU↔GPU every frame.
static const int kFramesInFlight = 2;

// Target is one size-dependent offscreen render target: a color+depth image, its
// framebuffer, and the ImGui sampled-image set that draws it into a panel. One per tile.
struct Target {
    int             width = 0, height = 0;
    VkImage         colorImage = VK_NULL_HANDLE;
    VkDeviceMemory  colorMem = VK_NULL_HANDLE;
    VkImageView     colorView = VK_NULL_HANDLE;
    VkImage         depthImage = VK_NULL_HANDLE;
    VkDeviceMemory  depthMem = VK_NULL_HANDLE;
    VkImageView     depthView = VK_NULL_HANDLE;
    VkFramebuffer   framebuffer = VK_NULL_HANDLE;
    VkDescriptorSet texture = VK_NULL_HANDLE; // ImGui sampled-image set
};

// FrameRes is everything one tile renders into, kept per (slot, frame-in-flight) so tiles are
// independent and a frame's GPU work can stay in flight while the next frame records (#1421). The
// CPU never blocks on the just-submitted fence; it instead waits on the N-frames-old frameFence
// before reusing a ring slot. The shadow stack is lazy — only built when a shadowed render needs it
// (a 1×1 default keeps descriptor binding 2 valid until then). Each tile's offscreen submit signals
// `sem`, which the ImGui swapchain pass waits on, so the model image is ready without a CPU stall.
struct FrameRes {
    Target          target;                        // offscreen color+depth + ImGui sampled set
    GpuBuffer       instbuf;                        // per-instance model matrices (binding 1, ADR-0038)
    VkBuffer        uboBuf = VK_NULL_HANDLE;        // scene-lighting UBO (binding 0)
    VkDeviceMemory  uboMem = VK_NULL_HANDLE;
    void*           uboMapped = nullptr;
    VkDescriptorSet sceneSet = VK_NULL_HANDLE;      // binds this frame's ubo + shared env + this ring's shadow
    VkCommandBuffer cmd = VK_NULL_HANDLE;
    VkSemaphore     sem = VK_NULL_HANDLE;           // offscreen-done → the ImGui pass waits on it (#1421)
    bool            pending = false;                // recorded this frame, awaiting the batched submit
};

struct Viewport {
    VkFormat        colorFormat = VK_FORMAT_UNDEFINED;
    VkRenderPass    renderPass = VK_NULL_HANDLE;
    VkPipelineLayout layout = VK_NULL_HANDLE;
    VkPipeline      triPipeline = VK_NULL_HANDLE;
    VkPipeline      linePipeline = VK_NULL_HANDLE;
    VkPipeline      occluderPipeline = VK_NULL_HANDLE; // depth-only faces (hidden-line modes)
    VkPipeline      hiddenPipeline = VK_NULL_HANDLE;   // occluded edges (reversed depth test)
    VkPipeline      topTriPipeline = VK_NULL_HANDLE;   // on-top faces (depth test disabled, PBI-067)
    VkPipeline      topLinePipeline = VK_NULL_HANDLE;  // on-top lines (depth test disabled, PBI-067)
    VkPipeline      skyboxPipeline = VK_NULL_HANDLE;   // HDR environment background (no depth)
    VkPipeline      pointPipeline = VK_NULL_HANDLE;    // point clouds: GL points, depth-tested (#645)
    VkShaderModule  vertModule = VK_NULL_HANDLE;
    VkShaderModule  fragModule = VK_NULL_HANDLE;
    VkPipeline      wideLinePipeline = VK_NULL_HANDLE;    // stroked lines, depth-tested (#2015)
    VkPipeline      topWideLinePipeline = VK_NULL_HANDLE; // stroked lines, on top (#2015)
    VkShaderModule  wideLineVertModule = VK_NULL_HANDLE;
    VkShaderModule  wideLineFragModule = VK_NULL_HANDLE;
    VkShaderModule  pointVertModule = VK_NULL_HANDLE;
    VkShaderModule  pointFragModule = VK_NULL_HANDLE;
    VkShaderModule  skyVertModule = VK_NULL_HANDLE;
    VkShaderModule  skyFragModule = VK_NULL_HANDLE;
    VkSampler       sampler = VK_NULL_HANDLE;

    // Scene-lighting descriptor layout + pool (the per-frame descriptor SET + its UBO live in
    // FrameRes so each in-flight frame has its own — #1421). sceneData is the CPU-side copy
    // obk_viewport_set_lighting writes; each render memcpy's it into the current frame's UBO.
    VkDescriptorSetLayout setLayout = VK_NULL_HANDLE;
    VkDescriptorPool      descPool = VK_NULL_HANDLE;
    float                 sceneData[kSceneFloats] = {0};

    // Equirectangular HDR environment for image-based lighting (descriptor binding 1). A 1×1
    // default is bound at init so the binding is always valid; obk_viewport_set_environment
    // replaces it with a mip-mapped image and flips the env-enabled flag (ADR-0026 §4).
    VkImage         envImage = VK_NULL_HANDLE;
    VkDeviceMemory  envMem = VK_NULL_HANDLE;
    VkImageView     envView = VK_NULL_HANDLE;
    VkSampler       envSampler = VK_NULL_HANDLE;

    // Skybox draw state: the inverse view-projection (column-major) and whether to draw the
    // environment background this frame, set per frame by obk_viewport_set_skybox.
    float           skyboxInvVP[16] = {0};
    bool            skyboxShow = false;

    // Sun shadow map (descriptor binding 2): rendered from the primary light's POV when shadows are
    // enabled, sampled with PCF. The pass/sampler/pipeline are shared; the depth image is rung per
    // frame-in-flight (frameShadow*[r]) so a frame's shadow isn't overwritten while the previous
    // frame still samples it (#1421). It is rendered ONCE per frame (frameShadowDone[r]) — the light
    // is scene-global, identical for every tile — and all tiles' scene sets sample that ring's map.
    VkRenderPass    shadowPass = VK_NULL_HANDLE;
    VkSampler       shadowSampler = VK_NULL_HANDLE;
    VkPipeline      shadowPipeline = VK_NULL_HANDLE;
    VkImage         frameShadowImage[kFramesInFlight] = {VK_NULL_HANDLE};
    VkDeviceMemory  frameShadowMem[kFramesInFlight] = {VK_NULL_HANDLE};
    VkImageView     frameShadowView[kFramesInFlight] = {VK_NULL_HANDLE};
    VkFramebuffer   frameShadowFB[kFramesInFlight] = {VK_NULL_HANDLE};
    bool            frameShadowDone[kFramesInFlight] = {false};

    VkCommandPool   cmdPool = VK_NULL_HANDLE;
    VkFence         fence = VK_NULL_HANDLE; // synchronous one-off transfers (readback, window capture)

    // Offscreen frames-in-flight ring (#1421). Each tile renders into slots[slot][frameIndex % N];
    // the CPU waits on the N-frames-old frameFence (already signalled, no stall) before reusing a ring
    // slot, and the per-tile semaphores hand the finished images to the ImGui swapchain pass. One
    // batched submit per frame signals frameFence[r]. pendingSems collects this frame's tile semaphores.
    FrameRes        slots[kMaxTiles][kFramesInFlight];
    VkFence         frameFence[kFramesInFlight] = {VK_NULL_HANDLE};
    uint64_t        frameIndex = 0;
    VkSemaphore     pendingSems[kMaxTiles] = {VK_NULL_HANDLE};
    int             pendingCount = 0;
    bool            frameSubmitted = false; // this frame's offscreen batch already went out (e.g. a mid-frame readback flushed it)

    GpuBuffer       vbuf, ibuf; // concatenated vertex + index geometry (HOST_VISIBLE, persistently mapped)

    // The merged-mesh identity (FNV-64 from the Go atlas key) currently resident in vbuf/ibuf. When
    // the next render carries the same key the concatenation + re-upload is skipped entirely, so a
    // static scene being orbited touches only the MVP push-constant (#1422). It tracks what is in the
    // SHARED buffer (last uploaded by ANY tile), which is what makes the skip correct across tiles —
    // a different tile's upload changes the key and forces this tile to re-upload. 0 = unknown (the
    // legacy flatten path, or a freshly recreated target), which always re-uploads (the #1218 guard).
    uint64_t        geomKey = 0;
    uint64_t        geomUploads = 0; // count of actual vbuf/ibuf re-uploads, for the #1422 test instrument

    // Retained point-cloud buffer (#645). Scan points upload ONCE (obk_viewport_upload_points) into
    // this persistent VRAM buffer and redraw every frame with the frame MVP — the CloudCompare-style
    // static-buffer path. pointKey is the camera-independent content identity resident in pointBuf;
    // when an upload carries the same key the transfer is skipped, so orbiting a loaded scan touches
    // no PCIe bandwidth (mirrors the geomKey skip, #1422). Points never ride the overlay atlas, so a
    // scan no longer forces the whole-model geometry re-upload the marker batch used to.
    GpuBuffer       pointBuf;        // interleaved [pos.xyz, rgba] (kPointFloats), HOST_VISIBLE|COHERENT
    uint64_t        pointKey = 0;    // content resident in pointBuf; 0 = none, forces the next upload
    int             pointCount = 0;  // points currently resident (0 ⇒ nothing drawn)
    float           pointSizePx = 3.0f; // on-screen point size in framebuffer pixels
    uint64_t        pointUploads = 0;   // actual point re-uploads, for the retained-buffer test instrument

    // Background the 3D pass clears to (themed; ADR-0021). Defaults reproduce the
    // pre-theming look so an un-themed build is unchanged.
    float           clearR = 0.10f, clearG = 0.11f, clearB = 0.13f;

    // Normal-debug: when on, shaded triangles render front-facing green / back-facing red
    // (gl_FrontFacing) instead of lit, to spot inverted-winding / flipped-normal triangles.
    bool            normalDebug = false;
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

    // Two external dependencies keep the offscreen target hazard-free across frames. The
    // color+depth images are reused every frame and ImGui samples the color image in its
    // main pass between our frames, so BOTH attachments' prior use must be ordered before
    // this pass's layout transition + load. The depth attachment in particular needs the
    // early/late fragment-test stages with DEPTH_STENCIL_ATTACHMENT_WRITE; without it the
    // sync validator flags SYNC-HAZARD-WRITE-AFTER-WRITE on the attachment-1 depth load
    // (loadOp CLEAR) vs. the layout transition.
    VkSubpassDependency deps[2]{};
    // [0] Before the pass: the previous frame's color sample (fragment-shader read) and
    // depth write must finish before we transition layouts and clear/write again.
    deps[0].srcSubpass = VK_SUBPASS_EXTERNAL;
    deps[0].dstSubpass = 0;
    deps[0].srcStageMask = VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT |
                           VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT |
                           VK_PIPELINE_STAGE_LATE_FRAGMENT_TESTS_BIT;
    deps[0].dstStageMask = VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT |
                           VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT |
                           VK_PIPELINE_STAGE_LATE_FRAGMENT_TESTS_BIT;
    deps[0].srcAccessMask = VK_ACCESS_SHADER_READ_BIT |
                            VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT;
    deps[0].dstAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT |
                            VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_READ_BIT |
                            VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT;
    // [1] After the pass: the color write must be visible to ImGui sampling the image as a
    // texture in the main swapchain pass.
    deps[1].srcSubpass = 0;
    deps[1].dstSubpass = VK_SUBPASS_EXTERNAL;
    deps[1].srcStageMask = VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT;
    deps[1].dstStageMask = VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT;
    deps[1].srcAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT;
    deps[1].dstAccessMask = VK_ACCESS_SHADER_READ_BIT;

    VkRenderPassCreateInfo rp{};
    rp.sType = VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO;
    rp.attachmentCount = 2;
    rp.pAttachments = atts;
    rp.subpassCount = 1;
    rp.pSubpasses = &sub;
    rp.dependencyCount = 2;
    rp.pDependencies = deps;
    vkCreateRenderPass(c->device, &rp, nullptr, &v->renderPass);
}

// fill_instanced_vertex_input describes the vertex input shared by every geometry + shadow pipeline:
// binding 0 is the per-vertex 16-float interleave (pos, normal, color, metallic, roughness, emissive,
// mode — matching mesh.vert and viewport.Flatten); binding 1 is the per-INSTANCE 4×4 model matrix
// (locations 7..10), so one local mesh draws at many transforms (ADR-0038). binds/attrs are
// caller-owned storage kept alive until pipeline creation.
void fill_instanced_vertex_input(VkVertexInputBindingDescription binds[2],
                                 VkVertexInputAttributeDescription attrs[11],
                                 VkPipelineVertexInputStateCreateInfo& vi) {
    binds[0] = {0, kVertexFloats * sizeof(float), VK_VERTEX_INPUT_RATE_VERTEX};
    binds[1] = {1, 16 * sizeof(float), VK_VERTEX_INPUT_RATE_INSTANCE};
    const VkVertexInputAttributeDescription base[7] = {
        {0, 0, VK_FORMAT_R32G32B32_SFLOAT, 0},
        {1, 0, VK_FORMAT_R32G32B32_SFLOAT, 3 * sizeof(float)},
        {2, 0, VK_FORMAT_R32G32B32A32_SFLOAT, 6 * sizeof(float)},
        {3, 0, VK_FORMAT_R32_SFLOAT, 10 * sizeof(float)},
        {4, 0, VK_FORMAT_R32_SFLOAT, 11 * sizeof(float)},
        {5, 0, VK_FORMAT_R32G32B32_SFLOAT, 12 * sizeof(float)},
        {6, 0, VK_FORMAT_R32_SFLOAT, 15 * sizeof(float)},
    };
    for (int i = 0; i < 7; i++) attrs[i] = base[i];
    for (uint32_t i = 0; i < 4; i++) { // mat4 model = 4 vec4 rows, binding 1
        attrs[7 + i] = {7 + i, 1, VK_FORMAT_R32G32B32A32_SFLOAT, (uint32_t)(i * 4 * sizeof(float))};
    }
    vi = {};
    vi.sType = VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO;
    vi.vertexBindingDescriptionCount = 2;
    vi.pVertexBindingDescriptions = binds;
    vi.vertexAttributeDescriptionCount = 11;
    vi.pVertexAttributeDescriptions = attrs;
}

// vs/fs override the default mesh shader pair — the wide-line pipelines swap in the expanding
// vertex shader while keeping this pipeline's vertex input, blending and depth setup (#2015).
VkPipeline create_pipeline(HeadContext* c, Viewport* v, VkPrimitiveTopology topo,
                           VkPolygonMode poly, VkBool32 colorWrite, VkCompareOp depthOp,
                           VkBool32 depthWrite,
                           VkShaderModule vs = VK_NULL_HANDLE, VkShaderModule fs = VK_NULL_HANDLE) {
    VkPipelineShaderStageCreateInfo stages[2]{};
    stages[0].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[0].stage = VK_SHADER_STAGE_VERTEX_BIT;
    stages[0].module = vs ? vs : v->vertModule;
    stages[0].pName = "main";
    stages[1].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[1].stage = VK_SHADER_STAGE_FRAGMENT_BIT;
    stages[1].module = fs ? fs : v->fragModule;
    stages[1].pName = "main";

    VkVertexInputBindingDescription binds[2];
    VkVertexInputAttributeDescription attrs[11];
    VkPipelineVertexInputStateCreateInfo vi{};
    fill_instanced_vertex_input(binds, attrs, vi);

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
    // Depth bias is dynamic (set per draw via vkCmdSetDepthBias): zero for solid geometry, a
    // small push-back for coplanar reference overlays (work-plane fills) so the solid wins the
    // depth test instead of z-fighting. Enabled here so the dynamic state takes effect.
    rs.depthBiasEnable = VK_TRUE;

    VkPipelineMultisampleStateCreateInfo ms{};
    ms.sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO;
    ms.rasterizationSamples = VK_SAMPLE_COUNT_1_BIT;

    VkPipelineDepthStencilStateCreateInfo ds{};
    ds.sType = VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO;
    ds.depthTestEnable = VK_TRUE;
    ds.depthWriteEnable = depthWrite;
    ds.depthCompareOp = depthOp;

    VkPipelineColorBlendAttachmentState cba{};
    cba.colorWriteMask = colorWrite ? (VK_COLOR_COMPONENT_R_BIT | VK_COLOR_COMPONENT_G_BIT |
                                       VK_COLOR_COMPONENT_B_BIT | VK_COLOR_COMPONENT_A_BIT)
                                    : 0;
    // Standard alpha blending so translucent overlays (work-plane fills) show through.
    // Opaque geometry has alpha 1, so src*1 + dst*0 = src — solids are unaffected.
    cba.blendEnable = VK_TRUE;
    cba.srcColorBlendFactor = VK_BLEND_FACTOR_SRC_ALPHA;
    cba.dstColorBlendFactor = VK_BLEND_FACTOR_ONE_MINUS_SRC_ALPHA;
    cba.colorBlendOp = VK_BLEND_OP_ADD;
    cba.srcAlphaBlendFactor = VK_BLEND_FACTOR_ONE;
    cba.dstAlphaBlendFactor = VK_BLEND_FACTOR_ZERO;
    cba.alphaBlendOp = VK_BLEND_OP_ADD;
    VkPipelineColorBlendStateCreateInfo cb{};
    cb.sType = VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO;
    cb.attachmentCount = 1;
    cb.pAttachments = &cba;

    VkDynamicState dyn[] = {VK_DYNAMIC_STATE_VIEWPORT, VK_DYNAMIC_STATE_SCISSOR,
                            VK_DYNAMIC_STATE_DEPTH_BIAS};
    VkPipelineDynamicStateCreateInfo dynState{};
    dynState.sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO;
    dynState.dynamicStateCount = 3;
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

// create_point_pipeline builds the point-cloud pipeline (#645): POINT_LIST topology with its own
// compact vertex input (binding 0 = kPointFloats interleave [pos.xyz, rgba]; no instance binding).
// It shares the mesh pipeline layout (same push range + descriptor set) but uses the point shaders,
// which set gl_PointSize from the push-constant. Depth test + write are on, so points integrate with
// the model (occlude and are occluded); no depth-bias dynamic state (points don't z-fight faces).
VkPipeline create_point_pipeline(HeadContext* c, Viewport* v) {
    VkPipelineShaderStageCreateInfo stages[2]{};
    stages[0].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[0].stage = VK_SHADER_STAGE_VERTEX_BIT;
    stages[0].module = v->pointVertModule;
    stages[0].pName = "main";
    stages[1].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[1].stage = VK_SHADER_STAGE_FRAGMENT_BIT;
    stages[1].module = v->pointFragModule;
    stages[1].pName = "main";

    VkVertexInputBindingDescription bind{0, kPointFloats * sizeof(float), VK_VERTEX_INPUT_RATE_VERTEX};
    VkVertexInputAttributeDescription attrs[2] = {
        {0, 0, VK_FORMAT_R32G32B32_SFLOAT, 0},                 // pos.xyz
        {1, 0, VK_FORMAT_R32G32B32A32_SFLOAT, 3 * sizeof(float)}, // rgba
    };
    VkPipelineVertexInputStateCreateInfo vi{};
    vi.sType = VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO;
    vi.vertexBindingDescriptionCount = 1;
    vi.pVertexBindingDescriptions = &bind;
    vi.vertexAttributeDescriptionCount = 2;
    vi.pVertexAttributeDescriptions = attrs;

    VkPipelineInputAssemblyStateCreateInfo ia{};
    ia.sType = VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO;
    ia.topology = VK_PRIMITIVE_TOPOLOGY_POINT_LIST;

    VkPipelineViewportStateCreateInfo vp{};
    vp.sType = VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO;
    vp.viewportCount = 1;
    vp.scissorCount = 1;

    VkPipelineRasterizationStateCreateInfo rs{};
    rs.sType = VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO;
    rs.polygonMode = VK_POLYGON_MODE_FILL;
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
    cba.blendEnable = VK_TRUE; // opaque points (alpha 1) pass through; leaves room for translucent coloring
    cba.srcColorBlendFactor = VK_BLEND_FACTOR_SRC_ALPHA;
    cba.dstColorBlendFactor = VK_BLEND_FACTOR_ONE_MINUS_SRC_ALPHA;
    cba.colorBlendOp = VK_BLEND_OP_ADD;
    cba.srcAlphaBlendFactor = VK_BLEND_FACTOR_ONE;
    cba.dstAlphaBlendFactor = VK_BLEND_FACTOR_ZERO;
    cba.alphaBlendOp = VK_BLEND_OP_ADD;
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

// create_skybox_pipeline builds the background pipeline: a vertex-buffer-less fullscreen
// triangle (positions from gl_VertexIndex), no depth test or write so it fills the cleared
// background, opaque. It shares the mesh pipeline layout (same push range + descriptor set).
VkPipeline create_skybox_pipeline(HeadContext* c, Viewport* v) {
    VkPipelineShaderStageCreateInfo stages[2]{};
    stages[0].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[0].stage = VK_SHADER_STAGE_VERTEX_BIT;
    stages[0].module = v->skyVertModule;
    stages[0].pName = "main";
    stages[1].sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stages[1].stage = VK_SHADER_STAGE_FRAGMENT_BIT;
    stages[1].module = v->skyFragModule;
    stages[1].pName = "main";

    VkPipelineVertexInputStateCreateInfo vi{}; // no inputs — gl_VertexIndex drives the triangle
    vi.sType = VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO;
    VkPipelineInputAssemblyStateCreateInfo ia{};
    ia.sType = VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO;
    ia.topology = VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST;

    VkPipelineViewportStateCreateInfo vp{};
    vp.sType = VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO;
    vp.viewportCount = vp.scissorCount = 1;
    VkPipelineRasterizationStateCreateInfo rs{};
    rs.sType = VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO;
    rs.polygonMode = VK_POLYGON_MODE_FILL;
    rs.cullMode = VK_CULL_MODE_NONE;
    rs.frontFace = VK_FRONT_FACE_COUNTER_CLOCKWISE;
    rs.lineWidth = 1.0f;
    VkPipelineMultisampleStateCreateInfo ms{};
    ms.sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO;
    ms.rasterizationSamples = VK_SAMPLE_COUNT_1_BIT;
    VkPipelineDepthStencilStateCreateInfo ds{};
    ds.sType = VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO;
    ds.depthTestEnable = VK_FALSE;
    ds.depthWriteEnable = VK_FALSE;
    ds.depthCompareOp = VK_COMPARE_OP_ALWAYS;
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

// ensure_buffer (re)creates b sized >= bytes with the given usage and memory properties, growing
// (never shrinking) so a steady scene stops reallocating. It returns whether it (re)allocated.
bool ensure_buffer(HeadContext* c, GpuBuffer* b, VkBufferUsageFlags usage,
                   VkMemoryPropertyFlags props, VkDeviceSize bytes) {
    if (b->buffer && b->size >= bytes) return false;
    // A grow frees the old allocation; unmap its persistent mapping first so the handle does not
    // dangle (the new, larger allocation is remapped lazily by upload_geom — #1422).
    if (b->mapped) { vkUnmapMemory(c->device, b->memory); b->mapped = nullptr; }
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
    ai.memoryTypeIndex = obk_find_memory_type(c->physical, req.memoryTypeBits, props);
    vkAllocateMemory(c->device, &ai, nullptr, &b->memory);
    vkBindBufferMemory(c->device, b->buffer, b->memory, 0);
    b->size = bytes;
    return true;
}

void upload(HeadContext* c, GpuBuffer* b, VkBufferUsageFlags usage, const void* data,
            VkDeviceSize bytes) {
    if (bytes == 0) return;
    ensure_buffer(c, b, usage,
                  VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, bytes);
    if (!data) return; // allocation only (e.g. a readback staging buffer filled by the GPU)
    void* mapped = nullptr;
    vkMapMemory(c->device, b->memory, 0, bytes, 0, &mapped);
    std::memcpy(mapped, data, (size_t)bytes);
    vkUnmapMemory(c->device, b->memory);
}

// upload_geom (re)sizes a HOST_VISIBLE|COHERENT geometry buffer and memcpys data into its
// PERSISTENT mapping — mapped once at (re)allocation, never unmapped per frame. Coherent memory
// makes the write visible to the GPU without an explicit flush, so the steady-state orbit path
// does zero map/unmap syscalls (#1422), unlike the generic per-call upload() above (which the
// rare readback-staging path still uses).
void upload_geom(HeadContext* c, GpuBuffer* b, VkBufferUsageFlags usage, const void* data,
                 VkDeviceSize bytes) {
    if (bytes == 0) return;
    ensure_buffer(c, b, usage,
                  VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT, bytes);
    if (!b->mapped) vkMapMemory(c->device, b->memory, 0, VK_WHOLE_SIZE, 0, &b->mapped);
    if (data) std::memcpy(b->mapped, data, (size_t)bytes);
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

void destroy_target(HeadContext* c, Target* t) {
    if (t->texture) { ImGui_ImplVulkan_RemoveTexture(t->texture); t->texture = VK_NULL_HANDLE; }
    if (t->framebuffer) vkDestroyFramebuffer(c->device, t->framebuffer, nullptr);
    if (t->colorView) vkDestroyImageView(c->device, t->colorView, nullptr);
    if (t->colorImage) vkDestroyImage(c->device, t->colorImage, nullptr);
    if (t->colorMem) vkFreeMemory(c->device, t->colorMem, nullptr);
    if (t->depthView) vkDestroyImageView(c->device, t->depthView, nullptr);
    if (t->depthImage) vkDestroyImage(c->device, t->depthImage, nullptr);
    if (t->depthMem) vkFreeMemory(c->device, t->depthMem, nullptr);
    t->framebuffer = VK_NULL_HANDLE;
    t->colorView = t->depthView = VK_NULL_HANDLE;
    t->colorImage = t->depthImage = VK_NULL_HANDLE;
    t->colorMem = t->depthMem = VK_NULL_HANDLE;
}

// ensure_target (re)creates target t at w×h, reusing the viewport's shared render pass and
// sampler. A no-op when t already matches the size.
void ensure_target(HeadContext* c, Viewport* v, Target* t, int w, int h) {
    if (w == t->width && h == t->height && t->framebuffer != VK_NULL_HANDLE) return;
    vkDeviceWaitIdle(c->device);
    // Recreating a target invalidates the geometry-resident assumption: M34-F4's geometry-skip
    // rendered blank across target-recreation / dock-layout transitions (#1218). Clearing the key
    // forces the next render to re-upload, so the recreated target is never sampled stale.
    v->geomKey = 0;
    destroy_target(c, t);
    t->width = w;
    t->height = h;
    t->colorView = make_image(c, v->colorFormat,
        VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT | VK_IMAGE_USAGE_SAMPLED_BIT |
            VK_IMAGE_USAGE_TRANSFER_SRC_BIT, // TRANSFER_SRC enables obk_viewport_readback
        VK_IMAGE_ASPECT_COLOR_BIT, w, h, &t->colorImage, &t->colorMem);
    t->depthView = make_image(c, kDepthFormat,
        VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT, VK_IMAGE_ASPECT_DEPTH_BIT, w, h,
        &t->depthImage, &t->depthMem);
    VkImageView atts[2] = {t->colorView, t->depthView};
    VkFramebufferCreateInfo fb{};
    fb.sType = VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO;
    fb.renderPass = v->renderPass;
    fb.attachmentCount = 2;
    fb.pAttachments = atts;
    fb.width = (uint32_t)w;
    fb.height = (uint32_t)h;
    fb.layers = 1;
    vkCreateFramebuffer(c->device, &fb, nullptr, &t->framebuffer);
    t->texture = ImGui_ImplVulkan_AddTexture(v->sampler, t->colorView,
                                             VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL);
}

// slotIndex clamps an external slot id into the valid tile range.
int slotIndex(int slot) {
    if (slot < 0) return 0;
    if (slot >= kMaxTiles) return kMaxTiles - 1;
    return slot;
}

// default_headlight fills the scene UBO with the LightingDefault rig (one directional
// headlight, ambience 0.18) so an un-configured viewport renders exactly as it did before
// lighting control existed (ADR-0026 §7). Layout matches viewport.PackLighting.
void default_headlight(float* d) {
    d[0] = 0.18f; d[1] = 1.0f; d[2] = 1.0f; d[3] = 1.0f; // ambience, brightness, exposure, count
    d[4] = 0.4f;  d[5] = 0.6f; d[6] = 0.8f; d[7] = 0.0f;  // dir.xyz + kind (directional)
    d[8] = 1.0f;  d[9] = 1.0f; d[10] = 1.0f; d[11] = 3.0f; // color.rgb + intensity
}

// create_scene_resources builds the lighting descriptor set: a fragment-stage UBO (binding 0)
// backed by a small host-visible, persistently mapped buffer that render refreshes each frame.
void create_scene_resources(HeadContext* c, Viewport* v) {
    // Binding 0: scene-lighting UBO. Binding 1: equirect environment (IBL). Binding 2: sun
    // shadow map.
    VkDescriptorSetLayoutBinding binds[3]{};
    binds[0].binding = 0;
    binds[0].descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
    binds[0].descriptorCount = 1;
    binds[0].stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT;
    binds[1].binding = 1;
    binds[1].descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
    binds[1].descriptorCount = 1;
    binds[1].stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT;
    binds[2].binding = 2;
    binds[2].descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
    binds[2].descriptorCount = 1;
    binds[2].stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT;
    VkDescriptorSetLayoutCreateInfo lci{};
    lci.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
    lci.bindingCount = 3;
    lci.pBindings = binds;
    vkCreateDescriptorSetLayout(c->device, &lci, nullptr, &v->setLayout);
    default_headlight(v->sceneData);

    // One descriptor set + UBO per (slot, frame-in-flight) so each in-flight frame binds its own
    // lighting/shadow without racing the previous frame still on the GPU (#1421).
    const uint32_t nSets = kMaxTiles * kFramesInFlight;
    VkDescriptorPoolSize ps[2] = {{VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, nSets},
                                  {VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER, 2 * nSets}};
    VkDescriptorPoolCreateInfo pci{};
    pci.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    pci.maxSets = nSets;
    pci.poolSizeCount = 2;
    pci.pPoolSizes = ps;
    vkCreateDescriptorPool(c->device, &pci, nullptr, &v->descPool);

    for (int s = 0; s < kMaxTiles; s++) {
        for (int f = 0; f < kFramesInFlight; f++) {
            FrameRes* fr = &v->slots[s][f];
            VkBufferCreateInfo bi{};
            bi.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
            bi.size = sizeof(v->sceneData);
            bi.usage = VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT;
            bi.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
            vkCreateBuffer(c->device, &bi, nullptr, &fr->uboBuf);
            VkMemoryRequirements req;
            vkGetBufferMemoryRequirements(c->device, fr->uboBuf, &req);
            VkMemoryAllocateInfo ai{};
            ai.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
            ai.allocationSize = req.size;
            ai.memoryTypeIndex = obk_find_memory_type(c->physical, req.memoryTypeBits,
                VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT);
            vkAllocateMemory(c->device, &ai, nullptr, &fr->uboMem);
            vkBindBufferMemory(c->device, fr->uboBuf, fr->uboMem, 0);
            vkMapMemory(c->device, fr->uboMem, 0, sizeof(v->sceneData), 0, &fr->uboMapped);
            std::memcpy(fr->uboMapped, v->sceneData, sizeof(v->sceneData));

            VkDescriptorSetAllocateInfo dai{};
            dai.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
            dai.descriptorPool = v->descPool;
            dai.descriptorSetCount = 1;
            dai.pSetLayouts = &v->setLayout;
            vkAllocateDescriptorSets(c->device, &dai, &fr->sceneSet);
            VkDescriptorBufferInfo dbi{fr->uboBuf, 0, sizeof(v->sceneData)};
            VkWriteDescriptorSet w{};
            w.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
            w.dstSet = fr->sceneSet;
            w.dstBinding = 0;
            w.descriptorCount = 1;
            w.descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
            w.pBufferInfo = &dbi;
            vkUpdateDescriptorSets(c->device, 1, &w, 0, nullptr);
        }
    }
}

// for_each_frame runs fn over every per-(slot,frame) FrameRes — used to fan descriptor updates
// (env image, shadow map) out to all the ringed scene descriptor sets.
template <typename F>
void for_each_frame(Viewport* v, F fn) {
    for (int s = 0; s < kMaxTiles; s++)
        for (int f = 0; f < kFramesInFlight; f++) fn(&v->slots[s][f]);
}

// img_barrier records a whole-image layout transition over all mip levels.
void img_barrier(VkCommandBuffer cmd, VkImage img, uint32_t levels, VkImageLayout oldL,
                 VkImageLayout newL, VkAccessFlags srcA, VkAccessFlags dstA,
                 VkPipelineStageFlags srcS, VkPipelineStageFlags dstS) {
    VkImageMemoryBarrier b{};
    b.sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER;
    b.oldLayout = oldL;
    b.newLayout = newL;
    b.srcQueueFamilyIndex = b.dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED;
    b.image = img;
    b.subresourceRange = {VK_IMAGE_ASPECT_COLOR_BIT, 0, levels, 0, 1};
    b.srcAccessMask = srcA;
    b.dstAccessMask = dstA;
    vkCmdPipelineBarrier(cmd, srcS, dstS, 0, 0, nullptr, 0, nullptr, 1, &b);
}

// write_env_descriptor points binding 1 (the environment image + sampler) of EVERY ringed scene set
// at the current env image — the env is shared across all in-flight frames (#1421).
void write_env_descriptor(HeadContext* c, Viewport* v) {
    VkDescriptorImageInfo ii{v->envSampler, v->envView, VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL};
    for_each_frame(v, [&](FrameRes* fr) {
        VkWriteDescriptorSet w{};
        w.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
        w.dstSet = fr->sceneSet;
        w.dstBinding = 1;
        w.descriptorCount = 1;
        w.descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
        w.pImageInfo = &ii;
        vkUpdateDescriptorSets(c->device, 1, &w, 0, nullptr);
    });
}

// destroy_env_image frees the current environment image/view/memory (the sampler persists).
void destroy_env_image(HeadContext* c, Viewport* v) {
    if (v->envView) vkDestroyImageView(c->device, v->envView, nullptr);
    if (v->envImage) vkDestroyImage(c->device, v->envImage, nullptr);
    if (v->envMem) vkFreeMemory(c->device, v->envMem, nullptr);
    v->envView = VK_NULL_HANDLE;
    v->envImage = VK_NULL_HANDLE;
    v->envMem = VK_NULL_HANDLE;
}

// make_env_image (re)creates the equirect environment image from a packed CPU mip chain (RGBA
// float32 levels concatenated; dims is w,h per level), uploads every level through a staging
// buffer, and points binding 1 at the result. Synchronous (waits on the fence) — it is called
// from the UI thread between frames, not in the render loop (ADR-0026 §4).
void make_env_image(HeadContext* c, Viewport* v, const float* data, const int* dims, int levels) {
    vkDeviceWaitIdle(c->device);
    destroy_env_image(c, v);

    VkImageCreateInfo ii{};
    ii.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
    ii.imageType = VK_IMAGE_TYPE_2D;
    ii.format = kEnvFormat;
    ii.extent = {(uint32_t)dims[0], (uint32_t)dims[1], 1};
    ii.mipLevels = (uint32_t)levels;
    ii.arrayLayers = 1;
    ii.samples = VK_SAMPLE_COUNT_1_BIT;
    ii.tiling = VK_IMAGE_TILING_OPTIMAL;
    ii.usage = VK_IMAGE_USAGE_TRANSFER_DST_BIT | VK_IMAGE_USAGE_SAMPLED_BIT;
    ii.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    vkCreateImage(c->device, &ii, nullptr, &v->envImage);
    VkMemoryRequirements req;
    vkGetImageMemoryRequirements(c->device, v->envImage, &req);
    VkMemoryAllocateInfo ai{};
    ai.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    ai.allocationSize = req.size;
    ai.memoryTypeIndex = obk_find_memory_type(c->physical, req.memoryTypeBits,
                                              VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    vkAllocateMemory(c->device, &ai, nullptr, &v->envMem);
    vkBindImageMemory(c->device, v->envImage, v->envMem, 0);

    uint64_t totalFloats = 0;
    for (int l = 0; l < levels; l++) totalFloats += (uint64_t)dims[l * 2] * dims[l * 2 + 1] * 4;
    GpuBuffer staging{};
    upload(c, &staging, VK_BUFFER_USAGE_TRANSFER_SRC_BIT, data, totalFloats * sizeof(float));

    VkCommandBufferAllocateInfo cba{};
    cba.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    cba.commandPool = v->cmdPool;
    cba.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    cba.commandBufferCount = 1;
    VkCommandBuffer cmd = VK_NULL_HANDLE;
    vkAllocateCommandBuffers(c->device, &cba, &cmd);
    VkCommandBufferBeginInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    bi.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(cmd, &bi);
    img_barrier(cmd, v->envImage, levels, VK_IMAGE_LAYOUT_UNDEFINED,
                VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 0, VK_ACCESS_TRANSFER_WRITE_BIT,
                VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, VK_PIPELINE_STAGE_TRANSFER_BIT);
    VkDeviceSize off = 0;
    for (int l = 0; l < levels; l++) {
        VkBufferImageCopy cp{};
        cp.bufferOffset = off;
        cp.imageSubresource = {VK_IMAGE_ASPECT_COLOR_BIT, (uint32_t)l, 0, 1};
        cp.imageExtent = {(uint32_t)dims[l * 2], (uint32_t)dims[l * 2 + 1], 1};
        vkCmdCopyBufferToImage(cmd, staging.buffer, v->envImage,
                               VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1, &cp);
        off += (VkDeviceSize)dims[l * 2] * dims[l * 2 + 1] * 4 * sizeof(float);
    }
    img_barrier(cmd, v->envImage, levels, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
                VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL, VK_ACCESS_TRANSFER_WRITE_BIT,
                VK_ACCESS_SHADER_READ_BIT, VK_PIPELINE_STAGE_TRANSFER_BIT,
                VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT);
    vkEndCommandBuffer(cmd);
    VkSubmitInfo submit{};
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &cmd;
    vkResetFences(c->device, 1, &v->fence);
    vkQueueSubmit(c->queue, 1, &submit, v->fence);
    vkWaitForFences(c->device, 1, &v->fence, VK_TRUE, UINT64_MAX);
    vkFreeCommandBuffers(c->device, v->cmdPool, 1, &cmd);
    vkDestroyBuffer(c->device, staging.buffer, nullptr);
    vkFreeMemory(c->device, staging.memory, nullptr);

    VkImageViewCreateInfo vi{};
    vi.sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO;
    vi.image = v->envImage;
    vi.viewType = VK_IMAGE_VIEW_TYPE_2D;
    vi.format = kEnvFormat;
    vi.subresourceRange = {VK_IMAGE_ASPECT_COLOR_BIT, 0, (uint32_t)levels, 0, 1};
    vkCreateImageView(c->device, &vi, nullptr, &v->envView);
    write_env_descriptor(c, v);
}

// create_env_sampler builds the IBL sampler: trilinear with azimuth wrap (U repeat) and a
// clamped pole (V), full mip range for roughness-driven LOD.
void create_env_sampler(HeadContext* c, Viewport* v) {
    VkSamplerCreateInfo si{};
    si.sType = VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO;
    si.magFilter = si.minFilter = VK_FILTER_LINEAR;
    si.mipmapMode = VK_SAMPLER_MIPMAP_MODE_LINEAR;
    si.addressModeU = VK_SAMPLER_ADDRESS_MODE_REPEAT;
    si.addressModeV = VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE;
    si.addressModeW = VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE;
    si.maxLod = VK_LOD_CLAMP_NONE;
    vkCreateSampler(c->device, &si, nullptr, &v->envSampler);
}

// init_default_env binds a 1×1 mid-grey image to binding 1 so the descriptor is valid before
// any environment is set; the env-enabled flag stays 0, so IBL is off until requested.
void init_default_env(HeadContext* c, Viewport* v) {
    const float grey[4] = {0.05f, 0.05f, 0.05f, 1.0f};
    const int dims[2] = {1, 1};
    make_env_image(c, v, grey, dims, 1);
}

// create_shadow_pass builds the depth-only render pass for the sun shadow map: one depth
// attachment cleared then stored and left readable by the surface shader.
void create_shadow_pass(HeadContext* c, Viewport* v) {
    VkAttachmentDescription att{};
    att.format = kDepthFormat;
    att.samples = VK_SAMPLE_COUNT_1_BIT;
    att.loadOp = VK_ATTACHMENT_LOAD_OP_CLEAR;
    att.storeOp = VK_ATTACHMENT_STORE_OP_STORE;
    att.stencilLoadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE;
    att.stencilStoreOp = VK_ATTACHMENT_STORE_OP_DONT_CARE;
    att.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    att.finalLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL;
    VkAttachmentReference depthRef{0, VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL};
    VkSubpassDescription sub{};
    sub.pipelineBindPoint = VK_PIPELINE_BIND_POINT_GRAPHICS;
    sub.pDepthStencilAttachment = &depthRef;
    // Order the depth write before the surface pass samples it, and after the previous frame's
    // sample finished.
    VkSubpassDependency deps[2]{};
    deps[0].srcSubpass = VK_SUBPASS_EXTERNAL;
    deps[0].dstSubpass = 0;
    deps[0].srcStageMask = VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT;
    deps[0].dstStageMask = VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT;
    deps[0].srcAccessMask = VK_ACCESS_SHADER_READ_BIT;
    deps[0].dstAccessMask = VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT;
    deps[1].srcSubpass = 0;
    deps[1].dstSubpass = VK_SUBPASS_EXTERNAL;
    deps[1].srcStageMask = VK_PIPELINE_STAGE_LATE_FRAGMENT_TESTS_BIT;
    deps[1].dstStageMask = VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT;
    deps[1].srcAccessMask = VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT;
    deps[1].dstAccessMask = VK_ACCESS_SHADER_READ_BIT;
    VkRenderPassCreateInfo rp{};
    rp.sType = VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO;
    rp.attachmentCount = 1;
    rp.pAttachments = &att;
    rp.subpassCount = 1;
    rp.pSubpasses = &sub;
    rp.dependencyCount = 2;
    rp.pDependencies = deps;
    vkCreateRenderPass(c->device, &rp, nullptr, &v->shadowPass);
}

// create_shadow_target allocates the per-frame-in-flight shadow depth images, their sampling views
// and framebuffers, plus the shared border-white sampler (so points outside the light frustum read
// as lit). One map per ring slot so a frame's shadow isn't overwritten while the previous frame
// still samples it (#1421).
void create_shadow_target(HeadContext* c, Viewport* v) {
    VkSamplerCreateInfo si{};
    si.sType = VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO;
    si.magFilter = si.minFilter = VK_FILTER_LINEAR;
    si.mipmapMode = VK_SAMPLER_MIPMAP_MODE_NEAREST;
    si.addressModeU = si.addressModeV = si.addressModeW = VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_BORDER;
    si.borderColor = VK_BORDER_COLOR_FLOAT_OPAQUE_WHITE;
    vkCreateSampler(c->device, &si, nullptr, &v->shadowSampler);
    for (int f = 0; f < kFramesInFlight; f++) {
        v->frameShadowView[f] = make_image(c, kDepthFormat,
            VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT | VK_IMAGE_USAGE_SAMPLED_BIT,
            VK_IMAGE_ASPECT_DEPTH_BIT, kShadowDim, kShadowDim,
            &v->frameShadowImage[f], &v->frameShadowMem[f]);
        VkFramebufferCreateInfo fb{};
        fb.sType = VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO;
        fb.renderPass = v->shadowPass;
        fb.attachmentCount = 1;
        fb.pAttachments = &v->frameShadowView[f];
        fb.width = fb.height = kShadowDim;
        fb.layers = 1;
        vkCreateFramebuffer(c->device, &fb, nullptr, &v->frameShadowFB[f]);
    }
}

// create_shadow_pipeline builds the depth-only caster pipeline: the mesh vertex shader (no
// fragment stage), depth write with a slope-scaled bias to suppress shadow acne, into the
// shadow pass. It reuses the mesh vertex layout + pipeline layout (the light matrix rides the
// push-constant mvp slot for the shadow draw).
void create_shadow_pipeline(HeadContext* c, Viewport* v) {
    VkPipelineShaderStageCreateInfo stage{};
    stage.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
    stage.stage = VK_SHADER_STAGE_VERTEX_BIT;
    stage.module = v->vertModule;
    stage.pName = "main";

    VkVertexInputBindingDescription binds[2];
    VkVertexInputAttributeDescription attrs[11];
    VkPipelineVertexInputStateCreateInfo vi{};
    fill_instanced_vertex_input(binds, attrs, vi);
    VkPipelineInputAssemblyStateCreateInfo ia{};
    ia.sType = VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO;
    ia.topology = VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST;
    VkPipelineViewportStateCreateInfo vp{};
    vp.sType = VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO;
    vp.viewportCount = vp.scissorCount = 1;
    VkPipelineRasterizationStateCreateInfo rs{};
    rs.sType = VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO;
    rs.polygonMode = VK_POLYGON_MODE_FILL;
    rs.cullMode = VK_CULL_MODE_NONE;
    rs.frontFace = VK_FRONT_FACE_COUNTER_CLOCKWISE;
    rs.lineWidth = 1.0f;
    rs.depthBiasEnable = VK_TRUE;
    rs.depthBiasConstantFactor = 1.5f;
    rs.depthBiasSlopeFactor = 2.5f;
    VkPipelineMultisampleStateCreateInfo ms{};
    ms.sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO;
    ms.rasterizationSamples = VK_SAMPLE_COUNT_1_BIT;
    VkPipelineDepthStencilStateCreateInfo ds{};
    ds.sType = VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO;
    ds.depthTestEnable = VK_TRUE;
    ds.depthWriteEnable = VK_TRUE;
    ds.depthCompareOp = VK_COMPARE_OP_LESS_OR_EQUAL;
    VkPipelineColorBlendStateCreateInfo cb{}; // no color attachments in the shadow pass
    cb.sType = VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO;
    VkDynamicState dyn[] = {VK_DYNAMIC_STATE_VIEWPORT, VK_DYNAMIC_STATE_SCISSOR};
    VkPipelineDynamicStateCreateInfo dynState{};
    dynState.sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO;
    dynState.dynamicStateCount = 2;
    dynState.pDynamicStates = dyn;
    VkGraphicsPipelineCreateInfo ci{};
    ci.sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO;
    ci.stageCount = 1;
    ci.pStages = &stage;
    ci.pVertexInputState = &vi;
    ci.pInputAssemblyState = &ia;
    ci.pViewportState = &vp;
    ci.pRasterizationState = &rs;
    ci.pMultisampleState = &ms;
    ci.pDepthStencilState = &ds;
    ci.pColorBlendState = &cb;
    ci.pDynamicState = &dynState;
    ci.layout = v->layout;
    ci.renderPass = v->shadowPass;
    vkCreateGraphicsPipelines(c->device, VK_NULL_HANDLE, 1, &ci, nullptr, &v->shadowPipeline);
}

// create_shadow_resources wires the whole shadow-map stack and points descriptor binding 2 at
// the shadow image. Shadows stay off (scene.shadow.x == 0) until obk_viewport_set_shadow.
void create_shadow_resources(HeadContext* c, Viewport* v) {
    create_shadow_pass(c, v);
    create_shadow_target(c, v);
    create_shadow_pipeline(c, v);
    // Each ring slot f's scene sets (all tiles) sample that ring's shadow map (#1421).
    for (int s = 0; s < kMaxTiles; s++) {
        for (int f = 0; f < kFramesInFlight; f++) {
            VkDescriptorImageInfo ii{v->shadowSampler, v->frameShadowView[f],
                                     VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL};
            VkWriteDescriptorSet w{};
            w.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
            w.dstSet = v->slots[s][f].sceneSet;
            w.dstBinding = 2;
            w.descriptorCount = 1;
            w.descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
            w.pImageInfo = &ii;
            vkUpdateDescriptorSets(c->device, 1, &w, 0, nullptr);
        }
    }
    v->sceneData[kShadowParams + 3] = 1.0f / float(kShadowDim); // texel size
}

} // namespace

extern "C" {

void obk_viewport_init(void* h, const uint32_t* vert, int vlen, const uint32_t* frag, int flen,
                       const uint32_t* pointVert, int pointVLen, const uint32_t* pointFrag, int pointFLen,
                       const uint32_t* wideVert, int wideVLen, const uint32_t* wideFrag, int wideFLen,
                       const uint32_t* skyVert, int skyVLen, const uint32_t* skyFrag,
                       int skyFLen) {
    HeadContext* c = (HeadContext*)h;
    Viewport* v = new Viewport();
    c->viewport = v;
    v->colorFormat = c->window_data.SurfaceFormat.format;
    v->vertModule = make_module(c->device, vert, vlen);
    v->fragModule = make_module(c->device, frag, flen);
    v->wideLineVertModule = make_module(c->device, wideVert, wideVLen);
    v->wideLineFragModule = make_module(c->device, wideFrag, wideFLen);
    v->pointVertModule = make_module(c->device, pointVert, pointVLen);
    v->pointFragModule = make_module(c->device, pointFrag, pointFLen);
    v->skyVertModule = make_module(c->device, skyVert, skyVLen);
    v->skyFragModule = make_module(c->device, skyFrag, skyFLen);

    // The scene-lighting descriptor set must exist before the pipeline layout references it.
    create_scene_resources(c, v);
    create_env_sampler(c, v);

    VkPushConstantRange pc{VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT, 0,
                           sizeof(PushConstants)};
    VkPipelineLayoutCreateInfo li{};
    li.sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO;
    li.setLayoutCount = 1;
    li.pSetLayouts = &v->setLayout;
    li.pushConstantRangeCount = 1;
    li.pPushConstantRanges = &pc;
    vkCreatePipelineLayout(c->device, &li, nullptr, &v->layout);

    create_render_pass(c, v);
    // Shaded triangles and solid lines: standard depth test, write depth. The occluder writes
    // depth only (no color) to hide edges behind unseen faces. The hidden-edge pipeline draws
    // only where it is behind geometry (GREATER) and does not write depth.
    v->triPipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
                                     VK_POLYGON_MODE_FILL, VK_TRUE, VK_COMPARE_OP_LESS_OR_EQUAL, VK_TRUE);
    v->linePipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_LINE_LIST,
                                      VK_POLYGON_MODE_FILL, VK_TRUE, VK_COMPARE_OP_LESS_OR_EQUAL, VK_TRUE);
    v->occluderPipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
                                          VK_POLYGON_MODE_FILL, VK_FALSE, VK_COMPARE_OP_LESS_OR_EQUAL, VK_TRUE);
    v->hiddenPipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_LINE_LIST,
                                        VK_POLYGON_MODE_FILL, VK_TRUE, VK_COMPARE_OP_GREATER, VK_FALSE);
    // On-top faces/lines: depth test always passes and depth is not written, so client-
    // graphics overlay/burn-through geometry draws over the model (PBI-067).
    v->topTriPipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
                                        VK_POLYGON_MODE_FILL, VK_TRUE, VK_COMPARE_OP_ALWAYS, VK_FALSE);
    v->topLinePipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_LINE_LIST,
                                         VK_POLYGON_MODE_FILL, VK_TRUE, VK_COMPARE_OP_ALWAYS, VK_FALSE);
    // Stroked lines (#2015): TRIANGLE topology, because each segment arrives as a quad the
    // wide-line vertex shader offsets in screen space. Depth setup mirrors the hairline lanes so a
    // stroked line occludes and is occluded exactly as its hairline equivalent would be.
    v->wideLinePipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
                                          VK_POLYGON_MODE_FILL, VK_TRUE, VK_COMPARE_OP_LESS_OR_EQUAL, VK_TRUE,
                                          v->wideLineVertModule, v->wideLineFragModule);
    v->topWideLinePipeline = create_pipeline(c, v, VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
                                             VK_POLYGON_MODE_FILL, VK_TRUE, VK_COMPARE_OP_ALWAYS, VK_FALSE,
                                             v->wideLineVertModule, v->wideLineFragModule);
    v->skyboxPipeline = create_skybox_pipeline(c, v);
    v->pointPipeline = create_point_pipeline(c, v); // point clouds (#645)
    create_shadow_resources(c, v);

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
    // One command buffer + offscreen-done semaphore per (slot, frame-in-flight); one fence per
    // frame-in-flight signalled by that frame's batched submit; plus a fence for synchronous
    // one-off transfers (readback / window capture). The frame fences start SIGNALLED so the first
    // kFramesInFlight frames don't block waiting on a never-submitted ring slot (#1421).
    VkCommandBufferAllocateInfo cb{};
    cb.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    cb.commandPool = v->cmdPool;
    cb.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    cb.commandBufferCount = 1;
    VkSemaphoreCreateInfo sci{};
    sci.sType = VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO;
    for (int s = 0; s < kMaxTiles; s++) {
        for (int f = 0; f < kFramesInFlight; f++) {
            vkAllocateCommandBuffers(c->device, &cb, &v->slots[s][f].cmd);
            vkCreateSemaphore(c->device, &sci, nullptr, &v->slots[s][f].sem);
        }
    }
    VkFenceCreateInfo fi{};
    fi.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO;
    fi.flags = VK_FENCE_CREATE_SIGNALED_BIT;
    for (int f = 0; f < kFramesInFlight; f++) vkCreateFence(c->device, &fi, nullptr, &v->frameFence[f]);
    fi.flags = 0;
    vkCreateFence(c->device, &fi, nullptr, &v->fence);

    // Bind a 1×1 default to the environment sampler now that the transfer fence/command pool
    // exist, so binding 1 is valid on the first frame (IBL stays off until an env is set).
    init_default_env(c, v);
}

// clear_viewport_depth resets the depth attachment to far (1.0) over the whole render area,
// mid-pass. It lets the on-top SOLID overlay (client-graphics glyphs, #1489) draw on top of the
// model yet still depth-test against ITSELF: after this clear, a depth-tested + depth-writing draw
// self-occludes (a cube reads as a cube) while no model fragment can hide it. Depth aspect only —
// the offscreen target is VK_FORMAT_D32_SFLOAT, no stencil.
static void clear_viewport_depth(VkCommandBuffer cmd, int w, int hh) {
    VkClearAttachment att{};
    att.aspectMask = VK_IMAGE_ASPECT_DEPTH_BIT;
    att.clearValue.depthStencil = {1.0f, 0};
    VkClearRect rect{};
    rect.rect.offset = {0, 0};
    rect.rect.extent = {(uint32_t)w, (uint32_t)hh};
    rect.baseArrayLayer = 0;
    rect.layerCount = 1;
    vkCmdClearAttachments(cmd, 1, &att, 1, &rect);
}

// obk_viewport_render uploads the geometry (only when geomKey marks it changed — #1422), records
// the offscreen pass, and submits it (waiting on a fence so the color image is ready to sample
// this frame). geomKey is the merged-mesh identity from the Go atlas cache; 0 means always upload.
void obk_viewport_render(void* h, int slot, int w, int hh, const float* mvp, const float* camPos,
                         const float* triV, int triVC, const uint32_t* triIdx, int triIC,
                         const float* occV, int occVC, const uint32_t* occIdx, int occIC,
                         const float* lineV, int lineVC, const uint32_t* lineIdx, int lineIC,
                         const float* hidV, int hidVC, const uint32_t* hidIdx, int hidIC,
                         const float* topTriV, int topTriVC, const uint32_t* topTriIdx, int topTriIC,
                         const float* topLineV, int topLineVC, const uint32_t* topLineIdx, int topLineIC,
                         const float* wideV, int wideVC, const uint32_t* wideIdx, int wideIC,
                         const float* topWideV, int topWideVC, const uint32_t* topWideIdx, int topWideIC,
                         int triBiasFirst, int topTriSolidFirst, const float* clip,
                         const float* mats, int matCount, const int32_t* recs, int recCount,
                         uint64_t geomKey) {
    HeadContext* c = (HeadContext*)h;
    Viewport* v = c->viewport;
    if (!v || w <= 0 || hh <= 0) return;
    // This tile renders into its own ring slot for the current frame-in-flight (#1421); the ring's
    // old fence was already waited in obk_viewport_frame_begin, so the resources here are free.
    const int r = (int)(v->frameIndex % kFramesInFlight);
    FrameRes* fr = &v->slots[slotIndex(slot)][r];
    Target* t = &fr->target;
    VkCommandBuffer cmd = fr->cmd;
    ensure_target(c, v, t, w, hh); // may clear v->geomKey (target recreated → re-upload, #1218)

    // One interleaved vertex buffer and one index buffer hold the streams back to back, in
    // draw order: occluder faces, shaded tris, solid lines, hidden lines, on-top tris, on-top
    // lines. Each stream's indices are 0-based within its own vertices, so each draw passes its
    // vertexOffset (the stream's vertex base) and firstIndex (its offset in the index buffer).
    const int occBase = 0, triBase = occVC, lineBase = occVC + triVC, hidBase = occVC + triVC + lineVC;
    const int topTriBase = hidBase + hidVC, topLineBase = topTriBase + topTriVC;
    // The stroked-line streams are concatenated last, so adding them left every base above unmoved.
    const int wideBase = topLineBase + topLineVC, topWideBase = wideBase + wideVC;
    const int occFirst = 0, triFirst = occIC, lineFirst = occIC + triIC, hidFirst = occIC + triIC + lineIC;
    const int topTriFirst = hidFirst + hidIC, topLineFirst = topTriFirst + topTriIC;
    const int wideFirst = topLineFirst + topLineIC, topWideFirst = wideFirst + wideIC;
    // Whether there is any geometry to draw this frame, independent of whether it was (re)uploaded —
    // a skipped upload (#1422) still draws the resident buffer, so the draw gate must use the counts.
    const bool haveGeometry = (occVC + triVC + lineVC + hidVC + topTriVC + topLineVC + wideVC + topWideVC) > 0;

    // Concatenate the six streams into one vertex + one index buffer and upload them — but ONLY when
    // the geometry actually changed. geomKey identifies the merged mesh (from the Go atlas cache); when
    // it matches what is already resident in vbuf/ibuf (last uploaded by any tile), the concatenation
    // and the whole-model PCIe transfer are skipped, so orbiting a static scene touches only the MVP
    // push-constant (#1422). geomKey == 0 (the legacy flatten path, or a freshly recreated target)
    // always re-uploads — that, plus ensure_target clearing the key, is the #1218 blank-after-recreation
    // guard. The draw offsets below are recomputed from the per-stream counts every frame regardless,
    // so a skip stays consistent with the resident buffer (identical geometry ⇒ identical counts).
    const bool geomResident = geomKey != 0 && geomKey == v->geomKey &&
                              v->vbuf.buffer != VK_NULL_HANDLE && v->ibuf.buffer != VK_NULL_HANDLE;
    if (!geomResident) {
        // vbuf/ibuf are SHARED across the in-flight ring; re-uploading (or growing) them while a
        // previous frame still reads them would corrupt that frame. Re-uploads happen only on a real
        // geometry change (rare — not the orbit path, #1422), so draining the ring here is cheap and
        // keeps the shared geometry safe under frames-in-flight (#1421).
        vkWaitForFences(c->device, kFramesInFlight, v->frameFence, VK_TRUE, UINT64_MAX);
        std::vector<float> verts;
        verts.reserve((size_t)(occVC + triVC + lineVC + hidVC + topTriVC + topLineVC + wideVC + topWideVC) * kVertexFloats);
        verts.insert(verts.end(), occV, occV + (size_t)occVC * kVertexFloats);
        verts.insert(verts.end(), triV, triV + (size_t)triVC * kVertexFloats);
        verts.insert(verts.end(), lineV, lineV + (size_t)lineVC * kVertexFloats);
        verts.insert(verts.end(), hidV, hidV + (size_t)hidVC * kVertexFloats);
        verts.insert(verts.end(), topTriV, topTriV + (size_t)topTriVC * kVertexFloats);
        verts.insert(verts.end(), topLineV, topLineV + (size_t)topLineVC * kVertexFloats);
        verts.insert(verts.end(), wideV, wideV + (size_t)wideVC * kVertexFloats);
        verts.insert(verts.end(), topWideV, topWideV + (size_t)topWideVC * kVertexFloats);
        std::vector<uint32_t> idx;
        idx.reserve((size_t)(occIC + triIC + lineIC + hidIC + topTriIC + topLineIC + wideIC + topWideIC));
        idx.insert(idx.end(), occIdx, occIdx + occIC);
        idx.insert(idx.end(), triIdx, triIdx + triIC);
        idx.insert(idx.end(), lineIdx, lineIdx + lineIC);
        idx.insert(idx.end(), hidIdx, hidIdx + hidIC);
        idx.insert(idx.end(), topTriIdx, topTriIdx + topTriIC);
        idx.insert(idx.end(), topLineIdx, topLineIdx + topLineIC);
        idx.insert(idx.end(), wideIdx, wideIdx + wideIC);
        idx.insert(idx.end(), topWideIdx, topWideIdx + topWideIC);
        upload_geom(c, &v->vbuf, VK_BUFFER_USAGE_VERTEX_BUFFER_BIT, verts.data(), verts.size() * sizeof(float));
        upload_geom(c, &v->ibuf, VK_BUFFER_USAGE_INDEX_BUFFER_BIT, idx.data(), idx.size() * sizeof(uint32_t));
        v->geomKey = geomKey; // what is now resident (0 stays 0 ⇒ legacy path never skips)
        v->geomUploads++;
    }

    // Per-instance model matrices (binding 1, ADR-0038) are small and change every frame (the
    // frustum-culled set), so they upload each frame — but into the same persistent mapping as the
    // geometry (upload_geom), so no per-frame map/unmap. With matrices supplied, recs drive instanced
    // per-(source,stream) draws; otherwise every stream draws as one identity instance (geometry in
    // its own world space — unchanged output).
    static const float kIdentity[16] = {1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1};
    if (matCount > 0 && mats) {
        upload_geom(c, &fr->instbuf, VK_BUFFER_USAGE_VERTEX_BUFFER_BIT, mats, (size_t)matCount * 16 * sizeof(float));
    } else {
        upload_geom(c, &fr->instbuf, VK_BUFFER_USAGE_VERTEX_BUFFER_BIT, kIdentity, sizeof(kIdentity));
    }

    // Refresh THIS frame's scene-lighting UBO from the CPU copy (coherent mapped memory, so the copy
    // is visible to the GPU without an explicit flush).
    if (fr->uboMapped) std::memcpy(fr->uboMapped, v->sceneData, sizeof(v->sceneData));

    vkResetCommandBuffer(cmd, 0);
    VkCommandBufferBeginInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    bi.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(cmd, &bi);

    // Shadow pass: render the casters' depth from the sun's POV into THIS ring's shadow map, when
    // shadows are enabled and there is geometry to cast. The light is scene-global, so the map is
    // rendered ONCE per frame (the first tile, frameShadowDone[r]) and every tile samples it (#1421).
    if (v->sceneData[kShadowParams] > 0.5f && (occIC > 0 || triIC > 0) && !v->frameShadowDone[r]) {
        v->frameShadowDone[r] = true;
        VkClearValue sclear;
        sclear.depthStencil = {1.0f, 0};
        VkRenderPassBeginInfo srp{};
        srp.sType = VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO;
        srp.renderPass = v->shadowPass;
        srp.framebuffer = v->frameShadowFB[r];
        srp.renderArea.extent = {kShadowDim, kShadowDim};
        srp.clearValueCount = 1;
        srp.pClearValues = &sclear;
        vkCmdBeginRenderPass(cmd, &srp, VK_SUBPASS_CONTENTS_INLINE);
        VkViewport sv{0, 0, (float)kShadowDim, (float)kShadowDim, 0.0f, 1.0f};
        VkRect2D ss{{0, 0}, {kShadowDim, kShadowDim}};
        vkCmdSetViewport(cmd, 0, 1, &sv);
        vkCmdSetScissor(cmd, 0, 1, &ss);
        VkDeviceSize zero = 0;
        VkBuffer vbufs[2] = {v->vbuf.buffer, fr->instbuf.buffer};
        VkDeviceSize voffs[2] = {0, 0};
        vkCmdBindVertexBuffers(cmd, 0, 2, vbufs, voffs); // binding 0 = verts, 1 = instance matrices
        vkCmdBindIndexBuffer(cmd, v->ibuf.buffer, 0, VK_INDEX_TYPE_UINT32);
        PushConstants sp{};
        std::memcpy(sp.mvp, &v->sceneData[kShadowVP], sizeof(sp.mvp));
        vkCmdPushConstants(cmd, v->layout,
            VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT, 0, sizeof(sp), &sp);
        vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->shadowPipeline);
        if (recCount > 0 && recs) { // instanced: each face-stream record casts at its instances
            for (int r = 0; r < recCount; r++) {
                const int32_t* rec = recs + (size_t)r * kDrawRecInts;
                if ((rec[0] == kStreamOcc || rec[0] == kStreamTri) && rec[2] > 0 && rec[5] > 0) {
                    vkCmdDrawIndexed(cmd, (uint32_t)rec[2], (uint32_t)rec[5], (uint32_t)rec[1], rec[3], (uint32_t)rec[4]);
                }
            }
        } else {
            if (occIC > 0) vkCmdDrawIndexed(cmd, (uint32_t)occIC, 1, (uint32_t)occFirst, occBase, 0);
            if (triIC > 0) vkCmdDrawIndexed(cmd, (uint32_t)triIC, 1, (uint32_t)triFirst, triBase, 0);
        }
        vkCmdEndRenderPass(cmd);
    }

    VkClearValue clears[2];
    clears[0].color = {{v->clearR, v->clearG, v->clearB, 1.0f}};
    clears[1].depthStencil = {1.0f, 0};
    VkRenderPassBeginInfo rp{};
    rp.sType = VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO;
    rp.renderPass = v->renderPass;
    rp.framebuffer = t->framebuffer;
    rp.renderArea.extent = {(uint32_t)w, (uint32_t)hh};
    rp.clearValueCount = 2;
    rp.pClearValues = clears;
    vkCmdBeginRenderPass(cmd, &rp, VK_SUBPASS_CONTENTS_INLINE);

    VkViewport vpRect{0, 0, (float)w, (float)hh, 0.0f, 1.0f};
    VkRect2D scissor{{0, 0}, {(uint32_t)w, (uint32_t)hh}};
    vkCmdSetViewport(cmd, 0, 1, &vpRect);
    vkCmdSetScissor(cmd, 0, 1, &scissor);
    vkCmdSetDepthBias(cmd, 0.0f, 0.0f, 0.0f); // default: solid geometry draws at zero bias

    // The scene-lighting + environment set (set 0) is shared by the skybox and every geometry
    // pipeline; bind it once, before any draw, so the background renders even with no geometry.
    vkCmdBindDescriptorSets(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->layout, 0, 1,
                            &fr->sceneSet, 0, nullptr);

    // Skybox background: drawn first (far plane, no depth write) so geometry overdraws it. The
    // push block carries the inverse view-projection in the mvp slot for ray reconstruction.
    if (v->skyboxShow) {
        PushConstants sky{};
        std::memcpy(sky.mvp, v->skyboxInvVP, sizeof(sky.mvp));
        sky.camPosLit[0] = camPos ? camPos[0] : 0.0f;
        sky.camPosLit[1] = camPos ? camPos[1] : 0.0f;
        sky.camPosLit[2] = camPos ? camPos[2] : 0.0f;
        sky.camPosLit[3] = 1.0f;
        vkCmdPushConstants(cmd, v->layout,
            VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT, 0, sizeof(sky), &sky);
        vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->skyboxPipeline);
        vkCmdDraw(cmd, 3, 1, 0, 0);
    }

    if (haveGeometry) {
        VkDeviceSize zero = 0;
        VkBuffer vbufs[2] = {v->vbuf.buffer, fr->instbuf.buffer};
        VkDeviceSize voffs[2] = {0, 0};
        vkCmdBindVertexBuffers(cmd, 0, 2, vbufs, voffs); // binding 0 = verts, 1 = instance matrices
        vkCmdBindIndexBuffer(cmd, v->ibuf.buffer, 0, VK_INDEX_TYPE_UINT32);
        PushConstants push{};
        std::memcpy(push.mvp, mvp, sizeof(push.mvp));
        push.camPosLit[0] = camPos ? camPos[0] : 0.0f;
        push.camPosLit[1] = camPos ? camPos[1] : 0.0f;
        push.camPosLit[2] = camPos ? camPos[2] : 0.0f;
        for (int i = 0; i < 4; i++) push.clip[i] = clip ? clip[i] : 0.0f; // section plane (M12-F04)
        // The stroked-line shader turns a pixel width into a clip-space offset, so it needs the
        // target size; every other pipeline ignores this slot (#2015).
        push.viewport[0] = (float)w;
        push.viewport[1] = (float)hh;
        push.viewport[2] = push.viewport[3] = 0.0f;
        auto pushLit = [&](float lit) {
            push.camPosLit[3] = lit;
            vkCmdPushConstants(cmd, v->layout,
                VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT, 0, sizeof(push), &push);
        };
        if (recCount > 0 && recs) {
            // Instanced (ADR-0038): one draw per (source stream × its instances), records ordered
            // by stream so the pipeline + lit flag change once per stream. Stream picks the same
            // pipeline/lit/bias the legacy per-stream path below uses.
            VkPipeline pipes[8] = {v->occluderPipeline, v->triPipeline, v->linePipeline,
                                   v->hiddenPipeline, v->topTriPipeline, v->topLinePipeline,
                                   v->wideLinePipeline, v->topWideLinePipeline};
            int curStream = -1;
            bool topSolidCleared = false;
            for (int r = 0; r < recCount; r++) {
                const int32_t* rec = recs + (size_t)r * kDrawRecInts;
                int stream = rec[0];
                if (stream < 0 || stream > 7 || rec[2] <= 0 || rec[5] <= 0) continue;
                // On-top SOLID tail (stream 4, flag rec[6]==1): the opaque client-graphics glyphs.
                // Clear depth once so they sit on top of the model, then draw them depth-tested +
                // lit through the shaded-tri pipeline so each self-occludes as a real solid (#1489).
                if (stream == kStreamTopTri && rec[6]) {
                    if (!topSolidCleared) { clear_viewport_depth(cmd, w, hh); topSolidCleared = true; }
                    vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->triPipeline);
                    pushLit(1.0f);
                    curStream = -1; // force a rebind for any following stream
                    vkCmdSetDepthBias(cmd, 0.0f, 0.0f, 0.0f);
                    vkCmdDrawIndexed(cmd, (uint32_t)rec[2], (uint32_t)rec[5],
                                     (uint32_t)rec[1], rec[3], (uint32_t)rec[4]);
                    continue;
                }
                if (stream != curStream) {
                    vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, pipes[stream]);
                    pushLit(stream == kStreamTri ? (v->normalDebug ? 2.0f : 1.0f)
                                                 : (stream == kStreamOcc ? 1.0f : 0.0f));
                    curStream = stream;
                }
                float bias = 0.0f;
                if (stream == kStreamLine || stream == kStreamWideLine) bias = -1.0f; // edges win the z-fight vs faces
                else if (stream == kStreamTri && rec[6]) bias = 2.0f; // overlay fill pushed back
                vkCmdSetDepthBias(cmd, bias, 0.0f, bias);
                vkCmdDrawIndexed(cmd, (uint32_t)rec[2], (uint32_t)rec[5],
                                 (uint32_t)rec[1], rec[3], (uint32_t)rec[4]);
            }
            vkCmdSetDepthBias(cmd, 0.0f, 0.0f, 0.0f);
        } else {
        // 1) occluder faces — depth only, hide edges behind unseen geometry.
        if (occIC > 0) {
            pushLit(1.0f);
            vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->occluderPipeline);
            vkCmdDrawIndexed(cmd, (uint32_t)occIC, 1, (uint32_t)occFirst, occBase, 0);
        }
        // 2) shaded triangles — color + depth, per-vertex shading mode. In normal-debug mode
        //    (lit flag 2.0) the shader colors them by raw facing (front green / back red).
        if (triIC > 0) {
            pushLit(v->normalDebug ? 2.0f : 1.0f);
            vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->triPipeline);
            int opaque = triBiasFirst;
            if (opaque < 0 || opaque > triIC) opaque = triIC; // guard against a bad split index
            if (opaque > 0) {
                vkCmdDrawIndexed(cmd, (uint32_t)opaque, 1, (uint32_t)triFirst, triBase, 0);
            }
            // Reference-overlay fills (work planes) draw last with a small depth push-back so a
            // coplanar solid face wins the depth test (no z-fighting); reset after.
            if (triIC - opaque > 0) {
                vkCmdSetDepthBias(cmd, 2.0f, 0.0f, 2.0f);
                vkCmdDrawIndexed(cmd, (uint32_t)(triIC - opaque), 1,
                                 (uint32_t)(triFirst + opaque), triBase, 0);
                vkCmdSetDepthBias(cmd, 0.0f, 0.0f, 0.0f);
            }
        }
        // 3) solid edges — depth-tested, so only the visible portions appear. A small negative
        //    depth bias pulls the edge fragments slightly toward the camera so they win the
        //    z-fight against the faces they lie on. Without it, an edge co-located with a face at
        //    the same depth flickers in and out of the depth test and renders broken — worst
        //    where two mated parts' coincident faces sit at the edge's exact depth. The bias is
        //    small enough that edges genuinely behind solid geometry still fail the test (the
        //    hidden-edge pass below handles those).
        if (lineIC > 0) {
            pushLit(0.0f);
            vkCmdSetDepthBias(cmd, -1.0f, 0.0f, -1.0f);
            vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->linePipeline);
            vkCmdDrawIndexed(cmd, (uint32_t)lineIC, 1, (uint32_t)lineFirst, lineBase, 0);
            vkCmdSetDepthBias(cmd, 0.0f, 0.0f, 0.0f);
        }
        // 4) hidden edges — reversed depth test, drawn only where occluded (dashed geometry).
        if (hidIC > 0) {
            pushLit(0.0f);
            vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->hiddenPipeline);
            vkCmdDrawIndexed(cmd, (uint32_t)hidIC, 1, (uint32_t)hidFirst, hidBase, 0);
        }
        // 5) on-top faces. The stream splits at topTriSolidFirst: the flat head (translucent
        //    ghosts/highlights, heatmap flood plots) draws with the depth test disabled so it
        //    burns over the model; the opaque-solid tail (client-graphics glyphs — support cubes,
        //    load arrows) draws after a depth clear, depth-tested + lit, so each glyph
        //    self-occludes as a real solid instead of a scatter of faces (#1489).
        if (topTriIC > 0) {
            int solidFirst = topTriSolidFirst;
            if (solidFirst < 0 || solidFirst > topTriIC) solidFirst = topTriIC;
            if (solidFirst > 0) {
                pushLit(0.0f);
                vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->topTriPipeline);
                vkCmdDrawIndexed(cmd, (uint32_t)solidFirst, 1, (uint32_t)topTriFirst, topTriBase, 0);
            }
            if (topTriIC - solidFirst > 0) {
                clear_viewport_depth(cmd, w, hh);
                pushLit(1.0f);
                vkCmdSetDepthBias(cmd, 0.0f, 0.0f, 0.0f);
                vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->triPipeline);
                vkCmdDrawIndexed(cmd, (uint32_t)(topTriIC - solidFirst), 1,
                                 (uint32_t)(topTriFirst + solidFirst), topTriBase, 0);
            }
        }
        // 6) on-top lines — depth test disabled (burn-through markers/edges).
        if (topLineIC > 0) {
            pushLit(0.0f);
            vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->topLinePipeline);
            vkCmdDrawIndexed(cmd, (uint32_t)topLineIC, 1, (uint32_t)topLineFirst, topLineBase, 0);
        }
        // 7) stroked lines (#2015) — quads the wide-line shader widens in screen space. Each draws
        //    after its hairline counterpart so a stroked overlay sits where a hairline one would.
        if (wideIC > 0) {
            pushLit(0.0f);
            vkCmdSetDepthBias(cmd, -1.0f, 0.0f, -1.0f); // same z-fight relief the solid edges take
            vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->wideLinePipeline);
            vkCmdDrawIndexed(cmd, (uint32_t)wideIC, 1, (uint32_t)wideFirst, wideBase, 0);
            vkCmdSetDepthBias(cmd, 0.0f, 0.0f, 0.0f);
        }
        if (topWideIC > 0) {
            pushLit(0.0f);
            vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->topWideLinePipeline);
            vkCmdDrawIndexed(cmd, (uint32_t)topWideIC, 1, (uint32_t)topWideFirst, topWideBase, 0);
        }
        } // end legacy (non-instanced) draws
    }

    // Point clouds (#645): the retained per-point buffer, drawn as native GL points with the frame
    // MVP — one vertex per scan point, depth-tested so they occlude and are occluded by the model.
    // Outside the haveGeometry gate so a scan-only scene still renders; the viewport/scissor and the
    // scene descriptor set were bound above. This is a pure VRAM redraw: no upload happens here (the
    // buffer was filled by obk_viewport_upload_points and is skipped on orbit).
    if (v->pointCount > 0 && v->pointBuf.buffer != VK_NULL_HANDLE) {
        PushConstants pp{};
        std::memcpy(pp.mvp, mvp, sizeof(pp.mvp));
        pp.clip[0] = v->pointSizePx; // point.vert reads gl_PointSize from clip.x
        vkCmdPushConstants(cmd, v->layout,
            VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT, 0, sizeof(pp), &pp);
        vkCmdBindPipeline(cmd, VK_PIPELINE_BIND_POINT_GRAPHICS, v->pointPipeline);
        VkDeviceSize off = 0;
        vkCmdBindVertexBuffers(cmd, 0, 1, &v->pointBuf.buffer, &off);
        vkCmdDraw(cmd, (uint32_t)v->pointCount, 1, 0, 0);
    }

    vkCmdEndRenderPass(cmd);
    vkEndCommandBuffer(cmd);

    // Don't submit per tile: mark this tile pending and queue its offscreen-done semaphore. The whole
    // frame's tiles go out in ONE batched submit at obk_viewport_frame_flush, and the ImGui swapchain
    // pass waits on these semaphores — so the CPU never blocks on the offscreen fence here (#1421).
    if (!fr->pending) {
        fr->pending = true;
        v->pendingSems[v->pendingCount++] = fr->sem;
    }
}

// obk_viewport_frame_begin opens an offscreen frame: it advances to the current ring slot and waits
// on that slot's fence (signalled kFramesInFlight frames ago, so already done — no stall), making the
// ring's command buffers / targets / instance+lighting buffers safe to overwrite. A no-op before the
// viewport exists. Called from obk_head_begin_frame.
void obk_viewport_frame_begin(HeadContext* c) {
    Viewport* v = c ? c->viewport : nullptr;
    if (!v) return;
    const int r = (int)(v->frameIndex % kFramesInFlight);
    vkWaitForFences(c->device, 1, &v->frameFence[r], VK_TRUE, UINT64_MAX);
    v->frameShadowDone[r] = false;
    v->frameSubmitted = false;
    v->pendingCount = 0;
    for (int s = 0; s < kMaxTiles; s++) v->slots[s][r].pending = false;
}

// submit_offscreen sends this frame's recorded tiles out in ONE vkQueueSubmit (fewer submits —
// #1421), each tile signalling its own semaphore (for the ImGui pass) and the batch signalling the
// ring fence (so the next reuse of this ring slot waits on it). Idempotent within a frame: a
// mid-frame readback flushes early, then obk_viewport_frame_flush skips the re-submit. Does NOT
// advance frameIndex (that happens once, at frame flush).
void submit_offscreen(HeadContext* c, Viewport* v, bool signalSems) {
    if (v->frameSubmitted) return;
    v->frameSubmitted = true;
    const int r = (int)(v->frameIndex % kFramesInFlight);
    if (v->pendingCount == 0) return;
    VkSubmitInfo subs[kMaxTiles]{};
    VkCommandBuffer cmds[kMaxTiles];
    int n = 0;
    for (int s = 0; s < kMaxTiles; s++) {
        FrameRes* fr = &v->slots[s][r];
        if (!fr->pending) continue;
        cmds[n] = fr->cmd;
        subs[n].sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
        subs[n].commandBufferCount = 1;
        subs[n].pCommandBuffers = &cmds[n];
        // Only signal the offscreen-done semaphore when the ImGui pass will actually wait on it this
        // frame. A minimized/skipped frame that signalled but never waited would leave the binary
        // semaphore signalled and trip a re-signal error next time the ring slot is reused (#1421).
        if (signalSems) {
            subs[n].signalSemaphoreCount = 1;
            subs[n].pSignalSemaphores = &fr->sem;
        }
        fr->pending = false;
        n++;
    }
    vkResetFences(c->device, 1, &v->frameFence[r]);
    vkQueueSubmit(c->queue, (uint32_t)n, subs, v->frameFence[r]);
}

// obk_viewport_frame_flush closes the offscreen frame: it submits the batch (if a readback didn't
// already), reports the tile semaphores via outSems/outCount so the caller's swapchain submit waits
// on them (no CPU stall — #1421), and advances the ring. present==0 (minimized frame, no swapchain
// submit to wait the semaphores) submits without signalling them and reports none. Called from
// obk_head_end_frame.
void obk_viewport_frame_flush(HeadContext* c, VkSemaphore* outSems, int* outCount, int present) {
    Viewport* v = c ? c->viewport : nullptr;
    if (!v) { if (outCount) *outCount = 0; return; }
    submit_offscreen(c, v, present != 0);
    int n = present ? v->pendingCount : 0;
    if (outSems) for (int i = 0; i < n; i++) outSems[i] = v->pendingSems[i];
    if (outCount) *outCount = n;
    v->pendingCount = 0;
    v->frameIndex++;
}

// obk_viewport_set_lighting copies the packed scene-lighting UBO (viewport.PackLighting's
// std140 float array) into the CPU-side copy; the next render uploads it. Extra floats beyond
// the UBO are ignored, and a short array leaves the remainder (a no-op before init). It is the
// single seam the app drives lights/ambient/exposure through (ADR-0026 §1,§3).
void obk_viewport_set_lighting(void* h, const float* data, int n) {
    HeadContext* c = (HeadContext*)h;
    if (!c->viewport || !data || n <= 0) return;
    uint32_t count = (uint32_t)n < kSceneFloats ? (uint32_t)n : kSceneFloats;
    std::memcpy(c->viewport->sceneData, data, count * sizeof(float));
}

// obk_viewport_set_environment replaces the IBL environment with a CPU mip chain (RGBA float32
// levels concatenated; dims = w,h per level) and enables image-based lighting with the given
// azimuth rotation (radians) and intensity. A non-positive level count (or null data) disables
// IBL, leaving the default image bound. A no-op before init. (ADR-0026 §4.)
void obk_viewport_set_environment(void* h, const float* data, const int* dims, int levels,
                                  float rotation, float intensity) {
    HeadContext* c = (HeadContext*)h;
    Viewport* v = c ? c->viewport : nullptr;
    if (!v) return;
    if (levels <= 0 || !data || !dims) {
        v->sceneData[kEnvBlock] = 0.0f; // disable IBL
        return;
    }
    make_env_image(c, v, data, dims, levels);
    v->sceneData[kEnvBlock + 0] = 1.0f;             // enabled
    v->sceneData[kEnvBlock + 1] = rotation;
    v->sceneData[kEnvBlock + 2] = intensity;
    v->sceneData[kEnvBlock + 3] = (float)(levels - 1); // max LOD for roughness sampling
}

// obk_viewport_set_skybox enables/disables drawing the environment as the background and stores
// the inverse view-projection (column-major, 16 floats) used to reconstruct view rays. A null
// matrix or show==0 disables it. A no-op before init. (ADR-0026 §5.)
void obk_viewport_set_skybox(void* h, const float* invVP, int show) {
    HeadContext* c = (HeadContext*)h;
    Viewport* v = c ? c->viewport : nullptr;
    if (!v) return;
    v->skyboxShow = (show != 0 && invVP != nullptr);
    if (v->skyboxShow) std::memcpy(v->skyboxInvVP, invVP, sizeof(v->skyboxInvVP));
}

// obk_viewport_set_shadow enables/disables the sun shadow map and stores the light-space matrix
// (column-major) plus density and softness ([0,1]). A null matrix or enabled==0 turns shadows
// off. A no-op before init. (ADR-0026 §6.)
void obk_viewport_set_shadow(void* h, const float* lightVP, int enabled, float density,
                             float softness, int castOnDirect, int occludeAmbient) {
    HeadContext* c = (HeadContext*)h;
    Viewport* v = c ? c->viewport : nullptr;
    if (!v) return;
    bool on = (enabled != 0 && lightVP != nullptr);
    v->sceneData[kShadowParams + 0] = on ? 1.0f : 0.0f; // map rendered this frame
    v->sceneData[kShadowParams + 1] = density;
    v->sceneData[kShadowParams + 2] = softness;
    v->sceneData[kShadow2 + 0] = (on && castOnDirect) ? 1.0f : 0.0f;   // darken direct light
    v->sceneData[kShadow2 + 1] = (on && occludeAmbient) ? 1.0f : 0.0f; // darken ambient
    if (on) std::memcpy(&v->sceneData[kShadowVP], lightVP, 16 * sizeof(float));
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

// obk_viewport_set_normal_debug toggles front-green/back-red facing visualization for shaded
// triangles (a tessellation debugging aid). Takes effect on the next render.
void obk_viewport_set_normal_debug(void* h, int on) {
    HeadContext* c = (HeadContext*)h;
    if (!c->viewport) return;
    c->viewport->normalDebug = (on != 0);
}

// obk_viewport_readback copies the offscreen color image into out (BGRA/RGBA8, row-major, top
// to bottom) for the current target size, returning the byte count written (0 if the target is
// not ready or out is too small) and reporting the dimensions. It is a synchronous transfer for
// headless verification/screenshots, not the per-frame path.
int obk_viewport_readback(void* h, int slot, unsigned char* out, int cap, int* w, int* hh) {
    HeadContext* c = (HeadContext*)h;
    Viewport* v = c ? c->viewport : nullptr;
    if (!v) return 0;
    // The offscreen pass is deferred (#1421): flush this frame's batch and wait for it so the color
    // image is actually rendered before we copy it (readback runs mid-frame, before the frame flush).
    const int r = (int)(v->frameIndex % kFramesInFlight);
    submit_offscreen(c, v, true);
    vkWaitForFences(c->device, 1, &v->frameFence[r], VK_TRUE, UINT64_MAX);
    Target* t = &v->slots[slotIndex(slot)][r].target;
    if (t->colorImage == VK_NULL_HANDLE || t->width <= 0 || t->height <= 0) return 0;
    int need = t->width * t->height * 4;
    if (w) *w = t->width;
    if (hh) *hh = t->height;
    if (!out || cap < need) return 0;

    GpuBuffer staging{};
    upload(c, &staging, VK_BUFFER_USAGE_TRANSFER_DST_BIT, nullptr, (VkDeviceSize)need);
    VkCommandBufferAllocateInfo cba{};
    cba.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    cba.commandPool = v->cmdPool;
    cba.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    cba.commandBufferCount = 1;
    VkCommandBuffer cmd = VK_NULL_HANDLE;
    vkAllocateCommandBuffers(c->device, &cba, &cmd);
    VkCommandBufferBeginInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    bi.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(cmd, &bi);
    img_barrier(cmd, t->colorImage, 1, VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
                VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, VK_ACCESS_SHADER_READ_BIT,
                VK_ACCESS_TRANSFER_READ_BIT, VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT,
                VK_PIPELINE_STAGE_TRANSFER_BIT);
    VkBufferImageCopy cp{};
    cp.imageSubresource = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, 1};
    cp.imageExtent = {(uint32_t)t->width, (uint32_t)t->height, 1};
    vkCmdCopyImageToBuffer(cmd, t->colorImage, VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
                           staging.buffer, 1, &cp);
    img_barrier(cmd, t->colorImage, 1, VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
                VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL, VK_ACCESS_TRANSFER_READ_BIT,
                VK_ACCESS_SHADER_READ_BIT, VK_PIPELINE_STAGE_TRANSFER_BIT,
                VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT);
    vkEndCommandBuffer(cmd);
    VkSubmitInfo submit{};
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &cmd;
    vkResetFences(c->device, 1, &v->fence);
    vkQueueSubmit(c->queue, 1, &submit, v->fence);
    vkWaitForFences(c->device, 1, &v->fence, VK_TRUE, UINT64_MAX);
    vkFreeCommandBuffers(c->device, v->cmdPool, 1, &cmd);

    void* mapped = nullptr;
    vkMapMemory(c->device, staging.memory, 0, need, 0, &mapped);
    std::memcpy(out, mapped, need);
    vkUnmapMemory(c->device, staging.memory);
    vkDestroyBuffer(c->device, staging.buffer, nullptr);
    vkFreeMemory(c->device, staging.memory, nullptr);
    return need;
}

// obk_window_capture copies the swapchain image for the current frame — the WHOLE window: the ImGui
// chrome, open dialogs, and the composited 3D viewport — into out as 8-bit BGRA (the surface order),
// row-major top to bottom. It returns the byte count written (0 if not ready or out is too small) and
// reports the window pixel size. A self-contained synchronous transfer like obk_viewport_readback, but
// it reads the backbuffer (the full composite) rather than the offscreen viewport target: it waits on
// the frame's render fence so the composite is complete, copies the backbuffer through a transient
// command pool (independent of the lazy viewport target), and restores the PRESENT_SRC layout so the
// presented image is untouched. Call after the frame has rendered (obk_head_end_frame).
int obk_window_capture(void* h, unsigned char* out, int cap, int* w, int* hh) {
    HeadContext* c = (HeadContext*)h;
    if (!c || c->device == VK_NULL_HANDLE) return 0;
    ImGui_ImplVulkanH_Window* wd = &c->window_data;
    if (wd->Width <= 0 || wd->Height <= 0 || (int)wd->FrameIndex >= wd->Frames.Size) return 0;
    ImGui_ImplVulkanH_Frame* fd = &wd->Frames[wd->FrameIndex];
    if (fd->Backbuffer == VK_NULL_HANDLE) return 0;
    int need = wd->Width * wd->Height * 4;
    if (w) *w = wd->Width;
    if (hh) *hh = wd->Height;
    if (!out || cap < need) return 0;

    // The ImGui render into this backbuffer must be complete before we read it.
    vkWaitForFences(c->device, 1, &fd->Fence, VK_TRUE, UINT64_MAX);

    GpuBuffer staging{};
    upload(c, &staging, VK_BUFFER_USAGE_TRANSFER_DST_BIT, nullptr, (VkDeviceSize)need);

    VkCommandPoolCreateInfo pci{};
    pci.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO;
    pci.flags = VK_COMMAND_POOL_CREATE_TRANSIENT_BIT;
    pci.queueFamilyIndex = c->queueFamily;
    VkCommandPool pool = VK_NULL_HANDLE;
    vkCreateCommandPool(c->device, &pci, nullptr, &pool);
    VkCommandBufferAllocateInfo cba{};
    cba.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    cba.commandPool = pool;
    cba.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    cba.commandBufferCount = 1;
    VkCommandBuffer cmd = VK_NULL_HANDLE;
    vkAllocateCommandBuffers(c->device, &cba, &cmd);

    VkCommandBufferBeginInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    bi.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(cmd, &bi);
    // The backbuffer is in PRESENT_SRC after the ImGui render pass; flip it to TRANSFER_SRC to copy,
    // then restore PRESENT_SRC.
    img_barrier(cmd, fd->Backbuffer, 1, VK_IMAGE_LAYOUT_PRESENT_SRC_KHR,
                VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, VK_ACCESS_MEMORY_READ_BIT,
                VK_ACCESS_TRANSFER_READ_BIT, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT,
                VK_PIPELINE_STAGE_TRANSFER_BIT);
    VkBufferImageCopy cp{};
    cp.imageSubresource = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, 1};
    cp.imageExtent = {(uint32_t)wd->Width, (uint32_t)wd->Height, 1};
    vkCmdCopyImageToBuffer(cmd, fd->Backbuffer, VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
                           staging.buffer, 1, &cp);
    img_barrier(cmd, fd->Backbuffer, 1, VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
                VK_IMAGE_LAYOUT_PRESENT_SRC_KHR, VK_ACCESS_TRANSFER_READ_BIT,
                VK_ACCESS_MEMORY_READ_BIT, VK_PIPELINE_STAGE_TRANSFER_BIT,
                VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT);
    vkEndCommandBuffer(cmd);

    VkFenceCreateInfo fci{};
    fci.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO;
    VkFence fence = VK_NULL_HANDLE;
    vkCreateFence(c->device, &fci, nullptr, &fence);
    VkSubmitInfo submit{};
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &cmd;
    vkQueueSubmit(c->queue, 1, &submit, fence);
    vkWaitForFences(c->device, 1, &fence, VK_TRUE, UINT64_MAX);

    void* mapped = nullptr;
    vkMapMemory(c->device, staging.memory, 0, need, 0, &mapped);
    std::memcpy(out, mapped, need);
    vkUnmapMemory(c->device, staging.memory);

    vkDestroyFence(c->device, fence, nullptr);
    vkDestroyCommandPool(c->device, pool, nullptr);
    vkDestroyBuffer(c->device, staging.buffer, nullptr);
    vkFreeMemory(c->device, staging.memory, nullptr);
    return need;
}

// obk_viewport_texture returns the ImGui texture handle for THIS frame's offscreen image (the
// current ring slot), to be drawn with ImGui::Image. The image is rendered by the batched offscreen
// submit at frame flush; the ImGui pass waits on its semaphore, so it is ready when sampled (#1421).
uint64_t obk_viewport_texture(void* h, int slot) {
    HeadContext* c = (HeadContext*)h;
    if (!c->viewport) return 0;
    const int r = (int)(c->viewport->frameIndex % kFramesInFlight);
    return (uint64_t)c->viewport->slots[slotIndex(slot)][r].target.texture;
}

// obk_viewport_geom_uploads returns the running count of actual geometry (vbuf/ibuf) re-uploads.
// A static scene being orbited holds this constant — the dirty-skip means only the MVP changes.
// Exposed for the #1422 regression test to assert "zero re-upload across frames"; 0 before init.
uint64_t obk_viewport_geom_uploads(void* h) {
    HeadContext* c = (HeadContext*)h;
    if (!c || !c->viewport) return 0;
    return c->viewport->geomUploads;
}

// obk_viewport_upload_points uploads the retained point-cloud stream (#645): count vertices of
// kPointFloats each ([pos.xyz, rgba], model space), and the on-screen point size in pixels. The
// transfer is SKIPPED when key matches what is already resident (key != 0), so the head can call
// this every frame while orbiting a loaded scan and pay nothing — the buffer redraws from VRAM in
// obk_viewport_render. key == 0 always uploads; count == 0 clears the cloud (nothing drawn). The
// head passes a camera-independent content key (placement/scale/budget/mode/color), so a re-upload
// happens only on a real change to the displayed set.
void obk_viewport_upload_points(void* h, const float* verts, int count, uint64_t key, float sizePx) {
    HeadContext* c = (HeadContext*)h;
    Viewport* v = c ? c->viewport : nullptr;
    if (!v) return;
    v->pointSizePx = sizePx; // cheap; kept current even when the buffer transfer is skipped
    const bool resident = key != 0 && key == v->pointKey && v->pointBuf.buffer != VK_NULL_HANDLE;
    if (resident) return;
    // pointBuf is shared across the in-flight ring; a (rare) re-upload drains the ring first so no
    // previous frame reads it mid-transfer (mirrors the vbuf re-upload guard, #1421). The fences are
    // signalled here — obk_viewport_frame_begin already waited this slot — so this does not stall.
    vkWaitForFences(c->device, kFramesInFlight, v->frameFence, VK_TRUE, UINT64_MAX);
    v->pointCount = count > 0 ? count : 0;
    if (v->pointCount > 0 && verts) {
        upload_geom(c, &v->pointBuf, VK_BUFFER_USAGE_VERTEX_BUFFER_BIT,
                    verts, (size_t)v->pointCount * kPointFloats * sizeof(float));
    }
    v->pointKey = key;
    v->pointUploads++;
}

// obk_viewport_point_uploads mirrors obk_viewport_geom_uploads for the point buffer: the count of
// actual re-uploads, so a test can assert a loaded scan re-uploads zero times across an orbit.
uint64_t obk_viewport_point_uploads(void* h) {
    HeadContext* c = (HeadContext*)h;
    if (!c || !c->viewport) return 0;
    return c->viewport->pointUploads;
}

void obk_viewport_destroy(HeadContext* c) {
    Viewport* v = c->viewport;
    if (!v) return;
    // Per-(slot, frame-in-flight) resources: offscreen target, instance buffer, lighting UBO and the
    // offscreen-done semaphore (the command buffers and descriptor sets are freed with their pools).
    for (int s = 0; s < kMaxTiles; s++) {
        for (int f = 0; f < kFramesInFlight; f++) {
            FrameRes* fr = &v->slots[s][f];
            destroy_target(c, &fr->target);
            if (fr->instbuf.mapped) vkUnmapMemory(c->device, fr->instbuf.memory);
            if (fr->instbuf.buffer) vkDestroyBuffer(c->device, fr->instbuf.buffer, nullptr);
            if (fr->instbuf.memory) vkFreeMemory(c->device, fr->instbuf.memory, nullptr);
            if (fr->uboMapped) vkUnmapMemory(c->device, fr->uboMem);
            if (fr->uboBuf) vkDestroyBuffer(c->device, fr->uboBuf, nullptr);
            if (fr->uboMem) vkFreeMemory(c->device, fr->uboMem, nullptr);
            if (fr->sem) vkDestroySemaphore(c->device, fr->sem, nullptr);
        }
    }
    // Free the geometry buffers before vkDestroyDevice so no VkDeviceMemory leaks past device
    // teardown (VUID-vkDestroyDevice-device-05137, surfaced by object-lifetime validation on a real
    // GPU — lavapipe did not flag it).
    GpuBuffer* geom[] = {&v->vbuf, &v->ibuf, &v->pointBuf};
    for (GpuBuffer* b : geom) {
        if (b->mapped) vkUnmapMemory(c->device, b->memory); // persistent map (#1422, #645)
        if (b->buffer) vkDestroyBuffer(c->device, b->buffer, nullptr);
        if (b->memory) vkFreeMemory(c->device, b->memory, nullptr);
    }
    destroy_env_image(c, v);
    if (v->envSampler) vkDestroySampler(c->device, v->envSampler, nullptr);
    for (int f = 0; f < kFramesInFlight; f++) {
        if (v->frameShadowFB[f]) vkDestroyFramebuffer(c->device, v->frameShadowFB[f], nullptr);
        if (v->frameShadowView[f]) vkDestroyImageView(c->device, v->frameShadowView[f], nullptr);
        if (v->frameShadowImage[f]) vkDestroyImage(c->device, v->frameShadowImage[f], nullptr);
        if (v->frameShadowMem[f]) vkFreeMemory(c->device, v->frameShadowMem[f], nullptr);
        if (v->frameFence[f]) vkDestroyFence(c->device, v->frameFence[f], nullptr);
    }
    if (v->shadowSampler) vkDestroySampler(c->device, v->shadowSampler, nullptr);
    if (v->shadowPipeline) vkDestroyPipeline(c->device, v->shadowPipeline, nullptr);
    if (v->shadowPass) vkDestroyRenderPass(c->device, v->shadowPass, nullptr);
    if (v->descPool) vkDestroyDescriptorPool(c->device, v->descPool, nullptr);
    if (v->setLayout) vkDestroyDescriptorSetLayout(c->device, v->setLayout, nullptr);
    vkDestroyFence(c->device, v->fence, nullptr);
    vkDestroyCommandPool(c->device, v->cmdPool, nullptr);
    vkDestroySampler(c->device, v->sampler, nullptr);
    vkDestroyPipeline(c->device, v->triPipeline, nullptr);
    vkDestroyPipeline(c->device, v->linePipeline, nullptr);
    vkDestroyPipeline(c->device, v->wideLinePipeline, nullptr);
    vkDestroyPipeline(c->device, v->topWideLinePipeline, nullptr);
    vkDestroyPipeline(c->device, v->occluderPipeline, nullptr);
    vkDestroyPipeline(c->device, v->hiddenPipeline, nullptr);
    vkDestroyPipeline(c->device, v->topTriPipeline, nullptr);
    vkDestroyPipeline(c->device, v->topLinePipeline, nullptr);
    vkDestroyPipeline(c->device, v->skyboxPipeline, nullptr);
    vkDestroyPipeline(c->device, v->pointPipeline, nullptr);
    vkDestroyPipelineLayout(c->device, v->layout, nullptr);
    vkDestroyRenderPass(c->device, v->renderPass, nullptr);
    vkDestroyShaderModule(c->device, v->vertModule, nullptr);
    vkDestroyShaderModule(c->device, v->fragModule, nullptr);
    vkDestroyShaderModule(c->device, v->wideLineVertModule, nullptr);
    vkDestroyShaderModule(c->device, v->wideLineFragModule, nullptr);
    vkDestroyShaderModule(c->device, v->pointVertModule, nullptr);
    vkDestroyShaderModule(c->device, v->pointFragModule, nullptr);
    vkDestroyShaderModule(c->device, v->skyVertModule, nullptr);
    vkDestroyShaderModule(c->device, v->skyFragModule, nullptr);
    delete v;
    c->viewport = nullptr;
}

} // extern "C"
