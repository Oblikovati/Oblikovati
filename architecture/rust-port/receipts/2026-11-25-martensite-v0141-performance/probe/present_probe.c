// SPDX-License-Identifier: GPL-2.0-only
#define _POSIX_C_SOURCE 200809L
#include <vulkan/vk_layer.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

static PFN_vkGetInstanceProcAddr next_instance_proc;
static PFN_vkGetDeviceProcAddr next_device_proc;
static PFN_GetPhysicalDeviceProcAddr next_physical_proc;
static PFN_vkQueuePresentKHR next_present;
static PFN_vkCreateSwapchainKHR next_swapchain;
static VkInstance instance;
static VkPhysicalDevice physical;
static FILE *trace;

static VkLayerInstanceCreateInfo *instance_link(const void *node) {
    for (const VkBaseInStructure *item = node; item; item = item->pNext) {
        if (item->sType == VK_STRUCTURE_TYPE_LOADER_INSTANCE_CREATE_INFO) {
            VkLayerInstanceCreateInfo *info = (VkLayerInstanceCreateInfo *)item;
            if (info->function == VK_LAYER_LINK_INFO) return info;
        }
    }
    return NULL;
}

static VkLayerDeviceCreateInfo *device_link(const void *node) {
    for (const VkBaseInStructure *item = node; item; item = item->pNext) {
        if (item->sType == VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO) {
            VkLayerDeviceCreateInfo *info = (VkLayerDeviceCreateInfo *)item;
            if (info->function == VK_LAYER_LINK_INFO) return info;
        }
    }
    return NULL;
}

static VKAPI_ATTR VkResult VKAPI_CALL create_instance(
    const VkInstanceCreateInfo *info, const VkAllocationCallbacks *allocator, VkInstance *out
) {
    VkLayerInstanceCreateInfo *link = instance_link(info->pNext);
    if (!link) return VK_ERROR_INITIALIZATION_FAILED;
    next_instance_proc = link->u.pLayerInfo->pfnNextGetInstanceProcAddr;
    next_physical_proc = link->u.pLayerInfo->pfnNextGetPhysicalDeviceProcAddr;
    link->u.pLayerInfo = link->u.pLayerInfo->pNext;
    PFN_vkCreateInstance create = (PFN_vkCreateInstance)next_instance_proc(NULL, "vkCreateInstance");
    VkResult result = create(info, allocator, out);
    if (result != VK_SUCCESS) return result;
    instance = *out;
    const char *trace_path = getenv("OBK_PRESENT_TRACE");
    if (trace_path) trace = fopen(trace_path, "w");
    if (trace) {
        setvbuf(trace, NULL, _IOLBF, 0);
        fputs("entry_ns,return_ns,result,swapchains\n", trace);
    }
    return result;
}

static VKAPI_ATTR VkResult VKAPI_CALL create_device(
    VkPhysicalDevice gpu, const VkDeviceCreateInfo *info,
    const VkAllocationCallbacks *allocator, VkDevice *out
) {
    VkLayerDeviceCreateInfo *link = device_link(info->pNext);
    if (!link) return VK_ERROR_INITIALIZATION_FAILED;
    PFN_vkGetInstanceProcAddr gipa = link->u.pLayerInfo->pfnNextGetInstanceProcAddr;
    next_device_proc = link->u.pLayerInfo->pfnNextGetDeviceProcAddr;
    link->u.pLayerInfo = link->u.pLayerInfo->pNext;
    PFN_vkCreateDevice create = (PFN_vkCreateDevice)gipa(instance, "vkCreateDevice");
    VkResult result = create(gpu, info, allocator, out);
    if (result == VK_SUCCESS) {
        physical = gpu;
        next_present = (PFN_vkQueuePresentKHR)next_device_proc(*out, "vkQueuePresentKHR");
        next_swapchain = (PFN_vkCreateSwapchainKHR)next_device_proc(*out, "vkCreateSwapchainKHR");
    }
    return result;
}

static int immediate_supported(VkSurfaceKHR surface) {
    PFN_vkGetPhysicalDeviceSurfacePresentModesKHR get_modes =
        (PFN_vkGetPhysicalDeviceSurfacePresentModesKHR)next_instance_proc(
            instance, "vkGetPhysicalDeviceSurfacePresentModesKHR");
    VkPresentModeKHR modes[32];
    uint32_t count = 32;
    VkResult result = get_modes(physical, surface, &count, modes);
    if (result != VK_SUCCESS && result != VK_INCOMPLETE) return 0;
    for (uint32_t i = 0; i < count; i++) {
        if (modes[i] == VK_PRESENT_MODE_IMMEDIATE_KHR) return 1;
    }
    return 0;
}

static VKAPI_ATTR VkResult VKAPI_CALL create_swapchain(
    VkDevice device, const VkSwapchainCreateInfoKHR *info,
    const VkAllocationCallbacks *allocator, VkSwapchainKHR *out
) {
    VkPresentModeKHR immediate = VK_PRESENT_MODE_IMMEDIATE_KHR;
    VkSwapchainCreateInfoKHR adjusted = *info;
    VkSwapchainPresentModesCreateInfoEXT declared;
    if (getenv("OBK_PRESENT_IMMEDIATE")) {
        if (!immediate_supported(info->surface)) return VK_ERROR_FEATURE_NOT_PRESENT;
        adjusted.presentMode = immediate;
        if (info->pNext) {
            const VkBaseInStructure *first = info->pNext;
            if (first->sType != VK_STRUCTURE_TYPE_SWAPCHAIN_PRESENT_MODES_CREATE_INFO_EXT) {
                return VK_ERROR_FEATURE_NOT_PRESENT;
            }
            declared = *(const VkSwapchainPresentModesCreateInfoEXT *)first;
            declared.presentModeCount = 1;
            declared.pPresentModes = &immediate;
            adjusted.pNext = &declared;
        }
    }
    VkResult result = next_swapchain(device, &adjusted, allocator, out);
    fprintf(stderr, "present probe: swapchain %ux%u requested=%d used=%d result=%d\n",
        info->imageExtent.width, info->imageExtent.height, info->presentMode,
        adjusted.presentMode, result);
    return result;
}

static long long nanoseconds(void) {
    struct timespec now;
    clock_gettime(CLOCK_MONOTONIC, &now);
    return (long long)now.tv_sec * 1000000000LL + now.tv_nsec;
}

static VKAPI_ATTR VkResult VKAPI_CALL present(VkQueue queue, const VkPresentInfoKHR *info) {
    long long entered = nanoseconds();
    VkResult result = next_present(queue, info);
    long long returned = nanoseconds();
    if (trace) fprintf(trace, "%lld,%lld,%d,%u\n",
        entered, returned, result, info->swapchainCount);
    return result;
}

static VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL get_device(VkDevice device, const char *name) {
    if (!strcmp(name, "vkGetDeviceProcAddr")) return (PFN_vkVoidFunction)get_device;
    if (!strcmp(name, "vkQueuePresentKHR")) return (PFN_vkVoidFunction)present;
    if (!strcmp(name, "vkCreateSwapchainKHR")) return (PFN_vkVoidFunction)create_swapchain;
    return next_device_proc ? next_device_proc(device, name) : NULL;
}

static VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL get_instance(VkInstance handle, const char *name) {
    if (!strcmp(name, "vkGetInstanceProcAddr")) return (PFN_vkVoidFunction)get_instance;
    if (!strcmp(name, "vkGetDeviceProcAddr")) return (PFN_vkVoidFunction)get_device;
    if (!strcmp(name, "vkCreateInstance")) return (PFN_vkVoidFunction)create_instance;
    if (!strcmp(name, "vkCreateDevice")) return (PFN_vkVoidFunction)create_device;
    if (!strcmp(name, "vkQueuePresentKHR")) return (PFN_vkVoidFunction)present;
    if (!strcmp(name, "vkCreateSwapchainKHR")) return (PFN_vkVoidFunction)create_swapchain;
    return next_instance_proc ? next_instance_proc(handle, name) : NULL;
}

static VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL get_physical(VkInstance handle, const char *name) {
    return next_physical_proc ? next_physical_proc(handle, name) : NULL;
}

VKAPI_ATTR VkResult VKAPI_CALL vkNegotiateLoaderLayerInterfaceVersion(VkNegotiateLayerInterface *iface) {
    if (iface->loaderLayerInterfaceVersion < 2) return VK_ERROR_INITIALIZATION_FAILED;
    iface->loaderLayerInterfaceVersion = 2;
    iface->pfnGetInstanceProcAddr = get_instance;
    iface->pfnGetDeviceProcAddr = get_device;
    iface->pfnGetPhysicalDeviceProcAddr = get_physical;
    return VK_SUCCESS;
}
