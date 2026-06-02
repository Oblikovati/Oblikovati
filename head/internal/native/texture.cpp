// Ribbon-icon textures: upload a CPU-side RGBA bitmap (rasterized from an SVG glyph by
// the pure-Go head/icon package) into a device-local Vulkan image and hand it to Dear
// ImGui as a sampled texture, the same way viewport.cpp exposes the 3D color image.
// Unlike the viewport (a render target), these are static uploads, so the path is a
// staging-buffer copy with layout transitions rather than a render pass.
//
// Textures are cached by the Go side (head/ui) and created once per (icon, size), so
// this code optimizes for clarity over batching. Each created texture's image/memory/
// view is tracked so it can be freed on teardown via the returned descriptor set.
#include "head.h"
#include <cstring>
#include <unordered_map>

namespace {

struct IconTexture {
    VkImage         image = VK_NULL_HANDLE;
    VkDeviceMemory  memory = VK_NULL_HANDLE;
    VkImageView     view = VK_NULL_HANDLE;
};

constexpr VkFormat kIconFormat = VK_FORMAT_R8G8B8A8_UNORM;

// transition records an image layout barrier covering the whole color image — enough
// for the two transitions a static upload needs (UNDEFINED→TRANSFER_DST→SHADER_READ).
void transition(VkCommandBuffer cmd, VkImage image, VkImageLayout from, VkImageLayout to,
                VkAccessFlags srcAccess, VkAccessFlags dstAccess,
                VkPipelineStageFlags srcStage, VkPipelineStageFlags dstStage) {
    VkImageMemoryBarrier b{};
    b.sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER;
    b.oldLayout = from;
    b.newLayout = to;
    b.srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED;
    b.dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED;
    b.image = image;
    b.subresourceRange = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 1, 0, 1};
    b.srcAccessMask = srcAccess;
    b.dstAccessMask = dstAccess;
    vkCmdPipelineBarrier(cmd, srcStage, dstStage, 0, 0, nullptr, 0, nullptr, 1, &b);
}

} // namespace

// IconTextures owns the resources shared by every icon texture (a sampler plus a
// transient command pool/fence for the upload submits) and the per-texture registry
// keyed by the ImGui descriptor set that identifies the texture to Go.
struct IconTextures {
    VkSampler       sampler = VK_NULL_HANDLE;
    VkCommandPool   cmdPool = VK_NULL_HANDLE;
    VkFence         fence = VK_NULL_HANDLE;
    std::unordered_map<VkDescriptorSet, IconTexture> registry;
};

namespace {

// ensure_icons lazily builds the shared sampler/command pool/fence on first upload.
IconTextures* ensure_icons(HeadContext* c) {
    if (c->icons) return c->icons;
    IconTextures* it = new IconTextures();

    VkSamplerCreateInfo si{};
    si.sType = VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO;
    si.magFilter = si.minFilter = VK_FILTER_LINEAR;
    si.addressModeU = si.addressModeV = si.addressModeW = VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE;
    vkCreateSampler(c->device, &si, nullptr, &it->sampler);

    VkCommandPoolCreateInfo cp{};
    cp.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO;
    cp.flags = VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT;
    cp.queueFamilyIndex = c->queueFamily;
    vkCreateCommandPool(c->device, &cp, nullptr, &it->cmdPool);

    VkFenceCreateInfo fi{};
    fi.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO;
    vkCreateFence(c->device, &fi, nullptr, &it->fence);

    c->icons = it;
    return it;
}

// make_staging creates a host-visible buffer holding the RGBA pixels to copy.
bool make_staging(HeadContext* c, VkDeviceSize bytes, const void* src, VkBuffer* buf,
                  VkDeviceMemory* mem) {
    VkBufferCreateInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
    bi.size = bytes;
    bi.usage = VK_BUFFER_USAGE_TRANSFER_SRC_BIT;
    bi.sharingMode = VK_SHARING_MODE_EXCLUSIVE;
    if (vkCreateBuffer(c->device, &bi, nullptr, buf) != VK_SUCCESS) return false;
    VkMemoryRequirements req;
    vkGetBufferMemoryRequirements(c->device, *buf, &req);
    VkMemoryAllocateInfo ai{};
    ai.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    ai.allocationSize = req.size;
    ai.memoryTypeIndex = obk_find_memory_type(c->physical, req.memoryTypeBits,
        VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT);
    if (vkAllocateMemory(c->device, &ai, nullptr, mem) != VK_SUCCESS) return false;
    vkBindBufferMemory(c->device, *buf, *mem, 0);
    void* mapped = nullptr;
    vkMapMemory(c->device, *mem, 0, bytes, 0, &mapped);
    std::memcpy(mapped, src, (size_t)bytes);
    vkUnmapMemory(c->device, *mem);
    return true;
}

// make_device_image creates the sampled device-local image the icon lives in.
bool make_device_image(HeadContext* c, int w, int h, IconTexture* tex) {
    VkImageCreateInfo ii{};
    ii.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
    ii.imageType = VK_IMAGE_TYPE_2D;
    ii.format = kIconFormat;
    ii.extent = {(uint32_t)w, (uint32_t)h, 1};
    ii.mipLevels = 1;
    ii.arrayLayers = 1;
    ii.samples = VK_SAMPLE_COUNT_1_BIT;
    ii.tiling = VK_IMAGE_TILING_OPTIMAL;
    ii.usage = VK_IMAGE_USAGE_TRANSFER_DST_BIT | VK_IMAGE_USAGE_SAMPLED_BIT;
    ii.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
    if (vkCreateImage(c->device, &ii, nullptr, &tex->image) != VK_SUCCESS) return false;
    VkMemoryRequirements req;
    vkGetImageMemoryRequirements(c->device, tex->image, &req);
    VkMemoryAllocateInfo ai{};
    ai.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
    ai.allocationSize = req.size;
    ai.memoryTypeIndex = obk_find_memory_type(c->physical, req.memoryTypeBits,
                                              VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    if (vkAllocateMemory(c->device, &ai, nullptr, &tex->memory) != VK_SUCCESS) return false;
    vkBindImageMemory(c->device, tex->image, tex->memory, 0);
    VkImageViewCreateInfo vi{};
    vi.sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO;
    vi.image = tex->image;
    vi.viewType = VK_IMAGE_VIEW_TYPE_2D;
    vi.format = kIconFormat;
    vi.subresourceRange = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 1, 0, 1};
    return vkCreateImageView(c->device, &vi, nullptr, &tex->view) == VK_SUCCESS;
}

} // namespace

extern "C" {

// obk_create_texture uploads w×h RGBA8 pixels and returns the ImGui texture handle (a
// VkDescriptorSet) for drawing with obk_ig_image / obk_ig_image_button, or 0 on failure.
uint64_t obk_create_texture(void* h, const unsigned char* rgba, int w, int hh) {
    HeadContext* c = (HeadContext*)h;
    if (!c || !rgba || w <= 0 || hh <= 0) return 0;
    IconTextures* it = ensure_icons(c);

    VkDeviceSize bytes = (VkDeviceSize)w * hh * 4;
    VkBuffer staging = VK_NULL_HANDLE;
    VkDeviceMemory stagingMem = VK_NULL_HANDLE;
    IconTexture tex{};
    bool okBuf = make_staging(c, bytes, rgba, &staging, &stagingMem);
    bool okImg = okBuf && make_device_image(c, w, hh, &tex);
    if (!okImg) {
        if (staging) vkDestroyBuffer(c->device, staging, nullptr);
        if (stagingMem) vkFreeMemory(c->device, stagingMem, nullptr);
        if (tex.view) vkDestroyImageView(c->device, tex.view, nullptr);
        if (tex.image) vkDestroyImage(c->device, tex.image, nullptr);
        if (tex.memory) vkFreeMemory(c->device, tex.memory, nullptr);
        return 0;
    }

    VkCommandBufferAllocateInfo ca{};
    ca.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
    ca.commandPool = it->cmdPool;
    ca.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
    ca.commandBufferCount = 1;
    VkCommandBuffer cmd = VK_NULL_HANDLE;
    vkAllocateCommandBuffers(c->device, &ca, &cmd);

    VkCommandBufferBeginInfo bi{};
    bi.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
    bi.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;
    vkBeginCommandBuffer(cmd, &bi);

    transition(cmd, tex.image, VK_IMAGE_LAYOUT_UNDEFINED, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
               0, VK_ACCESS_TRANSFER_WRITE_BIT,
               VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, VK_PIPELINE_STAGE_TRANSFER_BIT);

    VkBufferImageCopy region{};
    region.imageSubresource = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, 1};
    region.imageExtent = {(uint32_t)w, (uint32_t)hh, 1};
    vkCmdCopyBufferToImage(cmd, staging, tex.image,
                           VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1, &region);

    transition(cmd, tex.image, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
               VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
               VK_ACCESS_TRANSFER_WRITE_BIT, VK_ACCESS_SHADER_READ_BIT,
               VK_PIPELINE_STAGE_TRANSFER_BIT, VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT);

    vkEndCommandBuffer(cmd);

    VkSubmitInfo submit{};
    submit.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
    submit.commandBufferCount = 1;
    submit.pCommandBuffers = &cmd;
    vkResetFences(c->device, 1, &it->fence);
    vkQueueSubmit(c->queue, 1, &submit, it->fence);
    vkWaitForFences(c->device, 1, &it->fence, VK_TRUE, UINT64_MAX);
    vkFreeCommandBuffers(c->device, it->cmdPool, 1, &cmd);
    vkDestroyBuffer(c->device, staging, nullptr);
    vkFreeMemory(c->device, stagingMem, nullptr);

    VkDescriptorSet set = ImGui_ImplVulkan_AddTexture(it->sampler, tex.view,
                                                      VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL);
    if (set == VK_NULL_HANDLE) {
        vkDestroyImageView(c->device, tex.view, nullptr);
        vkDestroyImage(c->device, tex.image, nullptr);
        vkFreeMemory(c->device, tex.memory, nullptr);
        return 0;
    }
    it->registry[set] = tex;
    return (uint64_t)set;
}

// obk_destroy_texture frees one icon texture by its handle (idempotent on an unknown
// or zero handle), removing it from ImGui and releasing its image/memory/view.
void obk_destroy_texture(void* h, uint64_t handle) {
    HeadContext* c = (HeadContext*)h;
    if (!c || !c->icons || handle == 0) return;
    VkDescriptorSet set = (VkDescriptorSet)handle;
    auto found = c->icons->registry.find(set);
    if (found == c->icons->registry.end()) return;
    vkDeviceWaitIdle(c->device);
    ImGui_ImplVulkan_RemoveTexture(set);
    IconTexture& tex = found->second;
    vkDestroyImageView(c->device, tex.view, nullptr);
    vkDestroyImage(c->device, tex.image, nullptr);
    vkFreeMemory(c->device, tex.memory, nullptr);
    c->icons->registry.erase(found);
}

// obk_icons_destroy tears the whole cache down on shutdown (see head.h / app.cpp).
void obk_icons_destroy(HeadContext* c) {
    IconTextures* it = c->icons;
    if (!it) return;
    vkDeviceWaitIdle(c->device);
    for (auto& kv : it->registry) {
        ImGui_ImplVulkan_RemoveTexture(kv.first);
        vkDestroyImageView(c->device, kv.second.view, nullptr);
        vkDestroyImage(c->device, kv.second.image, nullptr);
        vkFreeMemory(c->device, kv.second.memory, nullptr);
    }
    vkDestroyFence(c->device, it->fence, nullptr);
    vkDestroyCommandPool(c->device, it->cmdPool, nullptr);
    vkDestroySampler(c->device, it->sampler, nullptr);
    delete it;
    c->icons = nullptr;
}

} // extern "C"
