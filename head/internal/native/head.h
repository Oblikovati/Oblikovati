// Shared native head state. HeadContext owns the GLFW window, the Vulkan device, and
// the Dear ImGui main-window swapchain (app.cpp), plus the offscreen 3D viewport
// target (viewport.cpp). It is defined here so both translation units can reach the
// device/queue/descriptor-pool without globals.
#pragma once
#include "imgui.h"
#include "backends/imgui_impl_vulkan.h"
#include <GLFW/glfw3.h>
#include <vulkan/vulkan.h>

struct Viewport;     // defined in viewport.cpp
struct IconTextures; // ribbon-icon texture cache, defined in texture.cpp

struct HeadContext {
    GLFWwindow*              window = nullptr;
    VkAllocationCallbacks*   allocator = nullptr;
    VkInstance               instance = VK_NULL_HANDLE;
    VkPhysicalDevice         physical = VK_NULL_HANDLE;
    VkDevice                 device = VK_NULL_HANDLE;
    uint32_t                 queueFamily = (uint32_t)-1;
    VkQueue                  queue = VK_NULL_HANDLE;
    VkDescriptorPool         descriptorPool = VK_NULL_HANDLE;
    ImGui_ImplVulkanH_Window window_data;
    // Triple buffering: with FIFO present, 2 images locks rendering to an integer
    // fraction of the vsync rate and stutters when a frame runs long; 3 lets a finished
    // frame queue while the next renders. (Vulkan best-practices
    // vkCreateSwapchainKHR-suboptimal-swapchain-image-count.) Clamped to surface caps by
    // ImGui_ImplVulkanH_CreateOrResizeWindow.
    uint32_t                 minImageCount = 3;
    bool                     swapChainRebuild = false;
    Viewport*                viewport = nullptr; // 3D scene render target (lazy)
    IconTextures*            icons = nullptr;    // ribbon-icon texture cache (lazy)
    // hwRayTracingAvailable is set once in create_device (app.cpp) after the real
    // VkPhysicalDeviceFeatures2 feature-bit query (M45-F01 PBI-333) — obk_head_ray_
    // tracing_support's presence-only probe (PBI-332) cannot tell whether the extension
    // is merely advertised or actually enabled on this device.
    bool                     hwRayTracingAvailable = false;
    // hwRayTracingPositionFetch additionally requires VK_KHR_ray_tracing_position_fetch
    // (only meaningful when hwRayTracingAvailable) — lets the ray-query compute shader
    // fetch a hit triangle's vertex positions directly for the geometric normal.
    bool                     hwRayTracingPositionFetch = false;
};

// Pick a memory type index satisfying type_bits and props, or UINT32_MAX. Shared by
// the buffer/image allocations in viewport.cpp.
uint32_t obk_find_memory_type(VkPhysicalDevice phys, uint32_t type_bits,
                              VkMemoryPropertyFlags props);

// obk_viewport_destroy frees the viewport target/pipelines; defined in viewport.cpp
// (C linkage) and called from app.cpp on teardown.
extern "C" void obk_viewport_destroy(HeadContext* c);

// Offscreen frames-in-flight ring hooks (#1421), defined in viewport.cpp and driven by the frame
// boundary in app.cpp. frame_begin waits on the ring slot that is about to be reused (no stall on
// the current frame). frame_flush submits this frame's offscreen tiles in one batch and returns the
// per-tile semaphores via outSems/outCount, which the swapchain submit waits on so the ImGui pass
// samples a finished image without a CPU stall. Both no-op before the viewport is initialized.
extern "C" void obk_viewport_frame_begin(HeadContext* c);
extern "C" void obk_viewport_frame_flush(HeadContext* c, VkSemaphore* outSems, int* outCount, int present);

// obk_icons_destroy frees the ribbon-icon texture cache (images + sampler); defined in
// texture.cpp (C linkage) and called from app.cpp on teardown.
extern "C" void obk_icons_destroy(HeadContext* c);
