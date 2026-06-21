// Persistent native head context: one GLFW window + Vulkan device/swapchain + Dear
// ImGui, created once and driven frame-by-frame from Go. This replaces the one-shot
// smoke path — Go owns the frame loop and (via the thin ImGui wrappers at the bottom)
// the chrome layout, so UI state lives in the Go model and ImGui just renders it each
// frame (ADR-0004). All Vulkan/GLFW/ImGui state is hidden behind the opaque handle
// returned by obk_head_create.
#include "head.h"
#include "imgui_internal.h" // DockBuilder* for the default dock layout
#include "backends/imgui_impl_glfw.h"
#include <cstring>
#include <cstdio>
#include <vector>

// obk_glfw_error surfaces GLFW init/window failures (otherwise obk_head_create just
// returns null and the Go side reports a generic "step 1" with no cause — on macOS the
// usual cause is calling GLFW off the main thread, see runtime.LockOSThread in the
// head commands).
static void obk_glfw_error(int code, const char* desc) {
    fprintf(stderr, "[head] GLFW error %d: %s\n", code, desc ? desc : "(null)");
}

// obk_find_memory_type: shared with viewport.cpp (declared in head.h).
uint32_t obk_find_memory_type(VkPhysicalDevice phys, uint32_t type_bits,
                              VkMemoryPropertyFlags props) {
    VkPhysicalDeviceMemoryProperties mp;
    vkGetPhysicalDeviceMemoryProperties(phys, &mp);
    for (uint32_t i = 0; i < mp.memoryTypeCount; i++) {
        if ((type_bits & (1u << i)) &&
            (mp.memoryTypes[i].propertyFlags & props) == props) {
            return i;
        }
    }
    return UINT32_MAX;
}

namespace {

bool ok(VkResult r) { return r == VK_SUCCESS; }

// instance_has_ext reports whether the loader advertises an instance extension, so we
// only opt into portability on platforms (macOS/MoltenVK) that actually expose it.
bool instance_has_ext(const char* name) {
    uint32_t n = 0;
    vkEnumerateInstanceExtensionProperties(nullptr, &n, nullptr);
    std::vector<VkExtensionProperties> props(n);
    vkEnumerateInstanceExtensionProperties(nullptr, &n, props.data());
    for (const auto& p : props)
        if (strcmp(p.extensionName, name) == 0) return true;
    return false;
}

// device_has_ext reports whether a physical device advertises a device extension.
bool device_has_ext(VkPhysicalDevice dev, const char* name) {
    uint32_t n = 0;
    vkEnumerateDeviceExtensionProperties(dev, nullptr, &n, nullptr);
    std::vector<VkExtensionProperties> props(n);
    vkEnumerateDeviceExtensionProperties(dev, nullptr, &n, props.data());
    for (const auto& p : props)
        if (strcmp(p.extensionName, name) == 0) return true;
    return false;
}

bool create_instance(HeadContext* c, const char** exts, uint32_t n) {
    VkApplicationInfo app{};
    app.sType = VK_STRUCTURE_TYPE_APPLICATION_INFO;
    app.pApplicationName = "Oblikovati";
    app.apiVersion = VK_API_VERSION_1_3;
    std::vector<const char*> enabled(exts, exts + n);
    VkInstanceCreateInfo ci{};
    ci.sType = VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO;
    // Portability drivers (MoltenVK on macOS) are hidden from the loader unless the
    // instance opts in, otherwise vkCreateInstance fails with INCOMPATIBLE_DRIVER
    // ("Found no drivers!"). The guard keeps this a no-op on Linux/Windows, where the
    // extension is absent.
    if (instance_has_ext(VK_KHR_PORTABILITY_ENUMERATION_EXTENSION_NAME)) {
        enabled.push_back(VK_KHR_PORTABILITY_ENUMERATION_EXTENSION_NAME);
        ci.flags |= VK_INSTANCE_CREATE_ENUMERATE_PORTABILITY_BIT_KHR;
    }
    ci.pApplicationInfo = &app;
    ci.enabledExtensionCount = (uint32_t)enabled.size();
    ci.ppEnabledExtensionNames = enabled.data();
    return ok(vkCreateInstance(&ci, c->allocator, &c->instance));
}

bool create_device(HeadContext* c) {
    c->physical = ImGui_ImplVulkanH_SelectPhysicalDevice(c->instance);
    if (c->physical == VK_NULL_HANDLE) return false;
    c->queueFamily = ImGui_ImplVulkanH_SelectQueueFamilyIndex(c->physical);
    if (c->queueFamily == (uint32_t)-1) return false;
    const float prio = 1.0f;
    VkDeviceQueueCreateInfo q{};
    q.sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO;
    q.queueFamilyIndex = c->queueFamily;
    q.queueCount = 1;
    q.pQueuePriorities = &prio;
    std::vector<const char*> dev_ext = {"VK_KHR_swapchain"};
    // A portability-subset device (MoltenVK) MUST enable VK_KHR_portability_subset when
    // it advertises it, or device creation is a validation error
    // (VUID-VkDeviceCreateInfo-pProperties-04451). String literal, not the *_EXTENSION_NAME
    // macro, which is gated behind VK_ENABLE_BETA_EXTENSIONS.
    if (device_has_ext(c->physical, "VK_KHR_portability_subset"))
        dev_ext.push_back("VK_KHR_portability_subset");
    VkDeviceCreateInfo ci{};
    ci.sType = VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO;
    ci.queueCreateInfoCount = 1;
    ci.pQueueCreateInfos = &q;
    ci.enabledExtensionCount = (uint32_t)dev_ext.size();
    ci.ppEnabledExtensionNames = dev_ext.data();
    if (!ok(vkCreateDevice(c->physical, &ci, c->allocator, &c->device))) return false;
    vkGetDeviceQueue(c->device, c->queueFamily, 0, &c->queue);
    return true;
}

bool create_descriptor_pool(HeadContext* c) {
    // Combined-image-sampler sets are shared by the font atlas, the offscreen viewport
    // image, and one per cached ribbon icon (texture.cpp). The icon set alone is ~25
    // and grows with the command set, so size generously to avoid AddTexture failing.
    VkDescriptorPoolSize sizes[] = {{VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER, 256}};
    VkDescriptorPoolCreateInfo ci{};
    ci.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
    ci.flags = VK_DESCRIPTOR_POOL_CREATE_FREE_DESCRIPTOR_SET_BIT;
    ci.maxSets = 256;
    ci.poolSizeCount = 1;
    ci.pPoolSizes = sizes;
    return ok(vkCreateDescriptorPool(c->device, &ci, c->allocator, &c->descriptorPool));
}

void setup_window(HeadContext* c, VkSurfaceKHR surface, int w, int h) {
    ImGui_ImplVulkanH_Window* wd = &c->window_data;
    wd->Surface = surface;
    const VkFormat req_fmt[] = {VK_FORMAT_B8G8R8A8_UNORM, VK_FORMAT_R8G8B8A8_UNORM};
    wd->SurfaceFormat = ImGui_ImplVulkanH_SelectSurfaceFormat(
        c->physical, surface, req_fmt, (size_t)IM_ARRAYSIZE(req_fmt),
        VK_COLOR_SPACE_SRGB_NONLINEAR_KHR);
    VkPresentModeKHR req_present[] = {VK_PRESENT_MODE_FIFO_KHR};
    wd->PresentMode = ImGui_ImplVulkanH_SelectPresentMode(
        c->physical, surface, req_present, IM_ARRAYSIZE(req_present));
    ImGui_ImplVulkanH_CreateOrResizeWindow(c->instance, c->physical, c->device, wd,
        c->queueFamily, c->allocator, w, h, c->minImageCount,
        VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT);
}

void frame_render(HeadContext* c, ImDrawData* dd) {
    ImGui_ImplVulkanH_Window* wd = &c->window_data;
    VkSemaphore acq = wd->FrameSemaphores[wd->SemaphoreIndex].ImageAcquiredSemaphore;
    VkSemaphore done = wd->FrameSemaphores[wd->SemaphoreIndex].RenderCompleteSemaphore;
    VkResult err = vkAcquireNextImageKHR(c->device, wd->Swapchain, UINT64_MAX, acq,
                                         VK_NULL_HANDLE, &wd->FrameIndex);
    if (err == VK_ERROR_OUT_OF_DATE_KHR || err == VK_SUBOPTIMAL_KHR) {
        c->swapChainRebuild = true;
        if (err == VK_ERROR_OUT_OF_DATE_KHR) return;
    }
    ImGui_ImplVulkanH_Frame* fd = &wd->Frames[wd->FrameIndex];
    vkWaitForFences(c->device, 1, &fd->Fence, VK_TRUE, UINT64_MAX);
    vkResetFences(c->device, 1, &fd->Fence);
    vkResetCommandPool(c->device, fd->CommandPool, 0);
    VkCommandBufferBeginInfo begin{};
    begin.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    begin.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(fd->CommandBuffer, &begin);
    VkRenderPassBeginInfo rp{};
    rp.sType = VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO;
    rp.renderPass = wd->RenderPass;
    rp.framebuffer = fd->Framebuffer;
    rp.renderArea.extent.width = wd->Width;
    rp.renderArea.extent.height = wd->Height;
    rp.clearValueCount = 1;
    rp.pClearValues = &wd->ClearValue;
    vkCmdBeginRenderPass(fd->CommandBuffer, &rp, VK_SUBPASS_CONTENTS_INLINE);
    ImGui_ImplVulkan_RenderDrawData(dd, fd->CommandBuffer);
    vkCmdEndRenderPass(fd->CommandBuffer);
    vkEndCommandBuffer(fd->CommandBuffer);
    VkPipelineStageFlags stage = VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT;
    VkSubmitInfo submit{};
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.waitSemaphoreCount = 1;
    submit.pWaitSemaphores = &acq;
    submit.pWaitDstStageMask = &stage;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &fd->CommandBuffer;
    submit.signalSemaphoreCount = 1;
    submit.pSignalSemaphores = &done;
    vkQueueSubmit(c->queue, 1, &submit, fd->Fence);
}

void frame_present(HeadContext* c) {
    if (c->swapChainRebuild) return;
    ImGui_ImplVulkanH_Window* wd = &c->window_data;
    VkSemaphore done = wd->FrameSemaphores[wd->SemaphoreIndex].RenderCompleteSemaphore;
    VkPresentInfoKHR info{};
    info.sType = VK_STRUCTURE_TYPE_PRESENT_INFO_KHR;
    info.waitSemaphoreCount = 1;
    info.pWaitSemaphores = &done;
    info.swapchainCount = 1;
    info.pSwapchains = &wd->Swapchain;
    info.pImageIndices = &wd->FrameIndex;
    VkResult err = vkQueuePresentKHR(c->queue, &info);
    if (err == VK_ERROR_OUT_OF_DATE_KHR || err == VK_SUBOPTIMAL_KHR) c->swapChainRebuild = true;
    wd->SemaphoreIndex = (wd->SemaphoreIndex + 1) % wd->SemaphoreCount;
}

} // namespace

extern "C" {

void* obk_head_create(int width, int height, const char* title) {
    glfwSetErrorCallback(obk_glfw_error);
    if (!glfwInit()) { fprintf(stderr, "[head] glfwInit failed\n"); return nullptr; }
    if (!glfwVulkanSupported()) {
        fprintf(stderr, "[head] glfwVulkanSupported=false (GLFW cannot find the Vulkan loader)\n");
        glfwTerminate(); return nullptr;
    }
    HeadContext* c = new HeadContext();
    glfwWindowHint(GLFW_CLIENT_API, GLFW_NO_API);
    c->window = glfwCreateWindow(width, height, title, nullptr, nullptr);
    if (!c->window) { fprintf(stderr, "[head] glfwCreateWindow failed\n"); delete c; glfwTerminate(); return nullptr; }
    uint32_t n = 0;
    const char** exts = glfwGetRequiredInstanceExtensions(&n);
    if (!create_instance(c, exts, n)) { fprintf(stderr, "[head] vkCreateInstance failed\n"); delete c; return nullptr; }
    if (!create_device(c)) { fprintf(stderr, "[head] device selection/creation failed (no Vulkan physical device?)\n"); delete c; return nullptr; }
    if (!create_descriptor_pool(c)) { fprintf(stderr, "[head] descriptor pool creation failed\n"); delete c; return nullptr; }
    VkSurfaceKHR surface = VK_NULL_HANDLE;
    if (!ok(glfwCreateWindowSurface(c->instance, c->window, c->allocator, &surface))) {
        delete c; return nullptr;
    }
    int w = 0, h = 0;
    glfwGetFramebufferSize(c->window, &w, &h);
    setup_window(c, surface, w, h);

    IMGUI_CHECKVERSION();
    ImGui::CreateContext();
    ImGui::GetIO().IniFilename = nullptr;
    ImGui::GetIO().ConfigFlags |= ImGuiConfigFlags_DockingEnable; // dockable panels (viewport, browser, ribbon)
    ImGui::StyleColorsDark();
    ImGui_ImplGlfw_InitForVulkan(c->window, true);
    ImGui_ImplVulkan_InitInfo init{};
    init.ApiVersion = VK_API_VERSION_1_3;
    init.Instance = c->instance;
    init.PhysicalDevice = c->physical;
    init.Device = c->device;
    init.QueueFamily = c->queueFamily;
    init.Queue = c->queue;
    init.DescriptorPool = c->descriptorPool;
    init.MinImageCount = c->minImageCount;
    init.ImageCount = c->window_data.ImageCount;
    init.PipelineInfoMain.RenderPass = c->window_data.RenderPass;
    init.PipelineInfoMain.MSAASamples = VK_SAMPLE_COUNT_1_BIT;
    if (!ImGui_ImplVulkan_Init(&init)) { delete c; return nullptr; }
    return c;
}

// obk_head_set_ui_font replaces ImGui's built-in default with a UI font from in-memory
// TTF bytes (the Go-embedded Helvetica-metric face). The bytes are copied into
// ImGui-owned memory, so the caller's buffer need not outlive this call; the atlas frees
// the copy on shutdown. Call once after obk_head_create, before the first frame.
void obk_head_set_ui_font(const unsigned char* data, int len, float sizePx) {
    if (!data || len <= 0) return;
    ImGuiIO& io = ImGui::GetIO();
    io.Fonts->Clear();
    void* copy = IM_ALLOC((size_t)len);
    std::memcpy(copy, data, (size_t)len);
    ImFontConfig cfg;
    cfg.FontDataOwnedByAtlas = true; // ImGui owns + frees `copy` (matches IM_ALLOC)
    io.Fonts->AddFontFromMemoryTTF(copy, len, sizePx, &cfg);
}

// g_monoFont is the Script Console code editor's fixed-width face (Liberation Mono), added to
// the atlas alongside the UI font before the first frame so the editor's columns align. The
// pointer is retained for the draw-list AddText-with-font path and the glyph metrics the Go
// editor lays out against. NULL until obk_head_set_mono_font runs (the editor then falls back
// to the default font).
static ImFont* g_monoFont = nullptr;

// obk_head_set_mono_font adds a fixed-width face from in-memory TTF bytes as a SECOND atlas
// font (it must run after obk_head_set_ui_font, which clears the atlas, and before the first
// frame). The bytes are copied into ImGui-owned memory like the UI font.
void obk_head_set_mono_font(const unsigned char* data, int len, float sizePx) {
    if (!data || len <= 0) return;
    ImGuiIO& io = ImGui::GetIO();
    void* copy = IM_ALLOC((size_t)len);
    std::memcpy(copy, data, (size_t)len);
    ImFontConfig cfg;
    cfg.FontDataOwnedByAtlas = true;
    g_monoFont = io.Fonts->AddFontFromMemoryTTF(copy, len, sizePx, &cfg);
}

// obk_head_mono_font exposes the retained mono face to the draw helpers in imgui_wrap.cpp.
ImFont* obk_head_mono_font(void) { return g_monoFont; }

int obk_head_should_close(void* h) {
    HeadContext* c = (HeadContext*)h;
    return glfwWindowShouldClose(c->window) ? 1 : 0;
}

// obk_head_set_should_close overrides the window close flag — used to CANCEL a close
// (v=0) while a "save changes?" prompt is shown, or to confirm one (v=1).
void obk_head_set_should_close(void* h, int v) {
    HeadContext* c = (HeadContext*)h;
    glfwSetWindowShouldClose(c->window, v);
}

// obk_head_get_window_state reports the window's current position (virtual-screen
// coordinates, so the value encodes which monitor it's on), size, and maximized flag, for
// persisting the placement across sessions.
void obk_head_get_window_state(void* h, int* x, int* y, int* w, int* hh, int* maximized) {
    HeadContext* c = (HeadContext*)h;
    int px = 0, py = 0, pw = 0, ph = 0;
    glfwGetWindowPos(c->window, &px, &py);
    glfwGetWindowSize(c->window, &pw, &ph);
    if (x) *x = px;
    if (y) *y = py;
    if (w) *w = pw;
    if (hh) *hh = ph;
    if (maximized) *maximized = glfwGetWindowAttrib(c->window, GLFW_MAXIMIZED);
}

// obk_head_apply_window_state restores a saved placement: move the window to (x,y) — which
// puts it back on the same monitor — and maximize it if it was maximized. The size is set
// at creation (obk_head_create) from the saved width/height.
void obk_head_apply_window_state(void* h, int x, int y, int maximized) {
    HeadContext* c = (HeadContext*)h;
    glfwSetWindowPos(c->window, x, y);
    if (maximized) glfwMaximizeWindow(c->window);
}

// --- synthetic input injection (for in-window UI tests) ---
// Production never calls the obk_inject_* setters, so g_inject stays inactive and the
// real GLFW input flows untouched. When a test sets state, obk_apply_inject pushes it
// into ImGui's IO *after* the GLFW backend's NewFrame and *before* ImGui::NewFrame, so
// the injected pointer/buttons/wheel win over the real cursor for that frame.
struct ObkInject {
    bool active;
    bool posSet;
    float mx, my;
    bool down[5];
    float wheel;
    bool shift;
    int fkey; // a held hold-to-navigate function key (0 none, else 2/3/4) (#911)
};
static ObkInject g_inject = {};

extern "C" {
void obk_inject_mouse_pos(float x, float y)  { g_inject.active = true; g_inject.posSet = true; g_inject.mx = x; g_inject.my = y; }
void obk_inject_mouse_button(int b, int down) { if (b >= 0 && b < 5) { g_inject.active = true; g_inject.down[b] = down != 0; } }
void obk_inject_mouse_wheel(float w)         { g_inject.active = true; g_inject.wheel = w; }
void obk_inject_key_shift(int down)          { g_inject.active = true; g_inject.shift = down != 0; }
void obk_inject_fkey(int n, int down)        { g_inject.active = true; g_inject.fkey = down ? n : 0; }
}

static void obk_apply_inject() {
    if (!g_inject.active) return;
    ImGuiIO& io = ImGui::GetIO();
    // Apply every injected event in the same frame (no trickling), so a pos+button+wheel
    // burst all land together and the test sees them on the frame it set them.
    io.ConfigInputTrickleEventQueue = false;
    if (g_inject.posSet) io.AddMousePosEvent(g_inject.mx, g_inject.my);
    for (int b = 0; b < 5; b++) io.AddMouseButtonEvent(b, g_inject.down[b]);
    if (g_inject.wheel != 0.0f) { io.AddMouseWheelEvent(0.0f, g_inject.wheel); g_inject.wheel = 0.0f; }
    io.AddKeyEvent(ImGuiMod_Shift, g_inject.shift);
    io.AddKeyEvent(ImGuiKey_F2, g_inject.fkey == 2);
    io.AddKeyEvent(ImGuiKey_F3, g_inject.fkey == 3);
    io.AddKeyEvent(ImGuiKey_F4, g_inject.fkey == 4);
}

// obk_ig_dockspace_over_main hosts a full-window dockspace (called every frame, after
// NewFrame, before the panels). The central node is pass-through so the docked 3D
// viewport shows through. Returns the stable dockspace id for the one-time layout below.
extern "C" unsigned int obk_ig_dockspace_over_main(void) {
    return ImGui::DockSpaceOverViewport(0, NULL, ImGuiDockNodeFlags_PassthruCentralNode);
}

// obk_ig_dock_default_layout builds the initial panel arrangement once: the model
// browser left, the status bar bottom, and the 3D viewport filling the central node.
// The ribbon is no longer part of the dockspace — it is a fixed band pinned above it
// (obk_ig_begin_ribbon_band). The window names come from Go (it owns panel identity);
// the split structure is the default layout. Without this the panels float and
// auto-size to nothing, so the viewport collapses to a sliver.
extern "C" void obk_ig_dock_default_layout(unsigned int dockId, const char* model,
        const char* viewport, const char* status,
        unsigned int* outLeft, unsigned int* outBottom, unsigned int* outCenter) {
    ImGui::DockBuilderRemoveNode(dockId);
    ImGui::DockBuilderAddNode(dockId, ImGuiDockNodeFlags_DockSpace);
    ImGui::DockBuilderSetNodeSize(dockId, ImGui::GetMainViewport()->Size);
    ImGuiID center = dockId;
    ImGuiID left = ImGui::DockBuilderSplitNode(center, ImGuiDir_Left, 0.22f, NULL, &center);
    ImGuiID bottom = ImGui::DockBuilderSplitNode(center, ImGuiDir_Down, 0.07f, NULL, &center);
    ImGui::DockBuilderDockWindow(model, left);
    ImGui::DockBuilderDockWindow(status, bottom);
    ImGui::DockBuilderDockWindow(viewport, center);
    ImGui::DockBuilderFinish(dockId);
    // Report the side node ids so late-created windows (add-in dockable windows,
    // M05-F03) can be docked into the arrangement with SetNextWindowDockID.
    *outLeft = left; *outBottom = bottom; *outCenter = center;
}

// obk_ig_dock_split carves a new node off an existing one at runtime (e.g. a right
// band for an add-in window when none exists yet), returning the new node id and
// updating *nodeId to the remainder.
extern "C" unsigned int obk_ig_dock_split(unsigned int* nodeId, int dir, float ratio) {
    ImGuiID remain = *nodeId;
    ImGuiID fresh = ImGui::DockBuilderSplitNode(remain, (ImGuiDir)dir, ratio, NULL, &remain);
    ImGui::DockBuilderFinish(remain);
    *nodeId = remain;
    return fresh;
}

// obk_ig_set_next_window_dock docks the NEXT Begin'd window into the given node the
// first time it appears; afterwards the user's own re-docking wins (FirstUseEver).
extern "C" void obk_ig_set_next_window_dock(unsigned int nodeId) {
    ImGui::SetNextWindowDockID(nodeId, ImGuiCond_FirstUseEver);
}

void obk_head_begin_frame(void* h) {
    HeadContext* c = (HeadContext*)h;
    glfwPollEvents();
    if (c->swapChainRebuild) {
        int w = 0, hh = 0;
        glfwGetFramebufferSize(c->window, &w, &hh);
        if (w > 0 && hh > 0) {
            ImGui_ImplVulkan_SetMinImageCount(c->minImageCount);
            ImGui_ImplVulkanH_CreateOrResizeWindow(c->instance, c->physical, c->device,
                &c->window_data, c->queueFamily, c->allocator, w, hh, c->minImageCount,
                VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT);
            c->window_data.FrameIndex = 0;
            c->swapChainRebuild = false;
        }
    }
    ImGui_ImplVulkan_NewFrame();
    ImGui_ImplGlfw_NewFrame();
    obk_apply_inject(); // override the real cursor with injected input, if a test set it
    ImGui::NewFrame();
}

void obk_head_end_frame(void* h, float r, float g, float b) {
    HeadContext* c = (HeadContext*)h;
    ImGui::Render();
    ImDrawData* dd = ImGui::GetDrawData();
    const bool minimized = dd->DisplaySize.x <= 0.0f || dd->DisplaySize.y <= 0.0f;
    c->window_data.ClearValue.color.float32[0] = r;
    c->window_data.ClearValue.color.float32[1] = g;
    c->window_data.ClearValue.color.float32[2] = b;
    c->window_data.ClearValue.color.float32[3] = 1.0f;
    if (!minimized) {
        frame_render(c, dd);
        frame_present(c);
    }
}

// obk_head_gpu_info reports the selected physical device's name and Vulkan API version for
// anonymous installation telemetry (#1182). nameOut is filled NUL-terminated up to nameCap
// bytes; apiVersionOut receives the raw VkPhysicalDeviceProperties.apiVersion (the Go side
// formats major.minor.patch). On macOS the device runs over MoltenVK, so this reports the
// MoltenVK-backed Vulkan version. Safe any time after obk_head_create; yields ""/0 if no
// device was selected.
void obk_head_gpu_info(void* h, char* nameOut, int nameCap, unsigned int* apiVersionOut) {
    HeadContext* c = (HeadContext*)h;
    if (apiVersionOut) *apiVersionOut = 0;
    if (nameCap > 0) nameOut[0] = '\0';
    if (!c || c->physical == VK_NULL_HANDLE) return;
    VkPhysicalDeviceProperties props;
    vkGetPhysicalDeviceProperties(c->physical, &props);
    if (apiVersionOut) *apiVersionOut = props.apiVersion;
    int i = 0;
    for (; i + 1 < nameCap && props.deviceName[i] != '\0'; i++) nameOut[i] = props.deviceName[i];
    if (nameCap > 0) nameOut[i] = '\0';
}

void obk_head_destroy(void* h) {
    HeadContext* c = (HeadContext*)h;
    if (!c) return;
    vkDeviceWaitIdle(c->device);
    obk_viewport_destroy(c);
    obk_icons_destroy(c); // frees icon textures before the Vulkan backend shuts down
    ImGui_ImplVulkan_Shutdown();
    ImGui_ImplGlfw_Shutdown();
    ImGui::DestroyContext();
    // ImGui_ImplVulkanH_DestroyWindow tears down the swapchain/views but NOT the
    // surface (the app owns it), so destroy it ourselves before the instance.
    VkSurfaceKHR surface = c->window_data.Surface;
    ImGui_ImplVulkanH_DestroyWindow(c->instance, c->device, &c->window_data, c->allocator);
    vkDestroyDescriptorPool(c->device, c->descriptorPool, c->allocator);
    vkDestroyDevice(c->device, c->allocator);
    if (surface != VK_NULL_HANDLE) vkDestroySurfaceKHR(c->instance, surface, c->allocator);
    vkDestroyInstance(c->instance, c->allocator);
    glfwDestroyWindow(c->window);
    glfwTerminate();
    delete c;
}

} // extern "C"
