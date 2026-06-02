// Shared native head state. HeadContext owns the GLFW window, the Vulkan device, and
// the Dear ImGui main-window swapchain (app.cpp), plus the offscreen 3D viewport
// target (viewport.cpp). It is defined here so both translation units can reach the
// device/queue/descriptor-pool without globals.
#pragma once
#include "imgui.h"
#include "backends/imgui_impl_vulkan.h"
#include <GLFW/glfw3.h>
#include <vulkan/vulkan.h>

struct Viewport; // defined in viewport.cpp

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
    uint32_t                 minImageCount = 2;
    bool                     swapChainRebuild = false;
    Viewport*                viewport = nullptr; // 3D scene render target (lazy)
};

// Pick a memory type index satisfying type_bits and props, or UINT32_MAX. Shared by
// the buffer/image allocations in viewport.cpp.
uint32_t obk_find_memory_type(VkPhysicalDevice phys, uint32_t type_bits,
                              VkMemoryPropertyFlags props);

// obk_viewport_destroy frees the viewport target/pipelines; defined in viewport.cpp
// (C linkage) and called from app.cpp on teardown.
extern "C" void obk_viewport_destroy(HeadContext* c);
