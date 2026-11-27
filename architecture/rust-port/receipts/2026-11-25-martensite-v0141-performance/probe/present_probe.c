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
static FILE *phases;
static PFN_vkAcquireNextImageKHR next_acquire;
static PFN_vkWaitForFences next_wait_fences;
static PFN_vkQueueSubmit next_submit;
static PFN_vkQueueSubmit2 next_submit2;

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
    const char *phase_path = getenv("OBK_PHASE_TRACE");
    if (phase_path) phases = fopen(phase_path, "w");
    if (phases) {
        setvbuf(phases, NULL, _IOLBF, 0);
        fputs("function,entry_ns,return_ns,result\n", phases);
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
        next_acquire = (PFN_vkAcquireNextImageKHR)next_device_proc(*out, "vkAcquireNextImageKHR");
        next_wait_fences = (PFN_vkWaitForFences)next_device_proc(*out, "vkWaitForFences");
        next_submit = (PFN_vkQueueSubmit)next_device_proc(*out, "vkQueueSubmit");
        next_submit2 = (PFN_vkQueueSubmit2)next_device_proc(*out, "vkQueueSubmit2");
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

static VKAPI_ATTR VkResult VKAPI_CALL acquire(VkDevice device, VkSwapchainKHR swapchain, uint64_t timeout, VkSemaphore semaphore, VkFence fence, uint32_t *index) {
    long long entered = nanoseconds();
    VkResult result = next_acquire(device, swapchain, timeout, semaphore, fence, index);
    long long returned = nanoseconds();
    if (phases) fprintf(phases, "vkAcquireNextImageKHR,%lld,%lld,%d\n", entered, returned, result);
    return result;
}

static VKAPI_ATTR VkResult VKAPI_CALL wait_fences(VkDevice device, uint32_t count, const VkFence *fences, VkBool32 all, uint64_t timeout) {
    long long entered = nanoseconds();
    VkResult result = next_wait_fences(device, count, fences, all, timeout);
    long long returned = nanoseconds();
    if (phases) fprintf(phases, "vkWaitForFences,%lld,%lld,%d\n", entered, returned, result);
    return result;
}

static VKAPI_ATTR VkResult VKAPI_CALL submit(VkQueue queue, uint32_t count, const VkSubmitInfo *info, VkFence fence) {
    long long entered = nanoseconds();
    VkResult result = next_submit(queue, count, info, fence);
    long long returned = nanoseconds();
    if (phases) fprintf(phases, "vkQueueSubmit,%lld,%lld,%d\n", entered, returned, result);
    return result;
}

static VKAPI_ATTR VkResult VKAPI_CALL submit2(VkQueue queue, uint32_t count, const VkSubmitInfo2 *info, VkFence fence) {
    long long entered = nanoseconds();
    VkResult result = next_submit2(queue, count, info, fence);
    long long returned = nanoseconds();
    if (phases) fprintf(phases, "vkQueueSubmit2,%lld,%lld,%d\n", entered, returned, result);
    return result;
}

static VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL get_device(VkDevice device, const char *name) {
    if (!strcmp(name, "vkGetDeviceProcAddr")) return (PFN_vkVoidFunction)get_device;
    if (!strcmp(name, "vkQueuePresentKHR")) return (PFN_vkVoidFunction)present;
    if (phases && !strcmp(name, "vkAcquireNextImageKHR") && next_acquire) return (PFN_vkVoidFunction)acquire;
    if (phases && !strcmp(name, "vkWaitForFences") && next_wait_fences) return (PFN_vkVoidFunction)wait_fences;
    if (phases && !strcmp(name, "vkQueueSubmit") && next_submit) return (PFN_vkVoidFunction)submit;
    if (phases && !strcmp(name, "vkQueueSubmit2") && next_submit2) return (PFN_vkVoidFunction)submit2;
    if (!strcmp(name, "vkCreateSwapchainKHR")) return (PFN_vkVoidFunction)create_swapchain;
    return next_device_proc ? next_device_proc(device, name) : NULL;
}

static VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL get_instance(VkInstance handle, const char *name) {
    if (!strcmp(name, "vkGetInstanceProcAddr")) return (PFN_vkVoidFunction)get_instance;
    if (!strcmp(name, "vkGetDeviceProcAddr")) return (PFN_vkVoidFunction)get_device;
    if (!strcmp(name, "vkCreateInstance")) return (PFN_vkVoidFunction)create_instance;
    if (!strcmp(name, "vkCreateDevice")) return (PFN_vkVoidFunction)create_device;
    if (!strcmp(name, "vkQueuePresentKHR")) return (PFN_vkVoidFunction)present;
    if (phases && !strcmp(name, "vkAcquireNextImageKHR") && next_acquire) return (PFN_vkVoidFunction)acquire;
    if (phases && !strcmp(name, "vkWaitForFences") && next_wait_fences) return (PFN_vkVoidFunction)wait_fences;
    if (phases && !strcmp(name, "vkQueueSubmit") && next_submit) return (PFN_vkVoidFunction)submit;
    if (phases && !strcmp(name, "vkQueueSubmit2") && next_submit2) return (PFN_vkVoidFunction)submit2;
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
