#pragma once

#include <vulkan/vulkan.h>

namespace Oblikovati::Kernel::VulkanRenderer
{
	class Buffer;

	/**
	 * @brief Manages a texture resource in Vulkan
	 *
	 * Handles texture loading, creation of image resources, and sampling operations.
	 */
	class Texture
	{
	public:
		// Create texture from file
		Texture(
			VkDevice device,
			VkPhysicalDevice physicalDevice,
			VkCommandPool commandPool,
			VkQueue transferQueue,
			const std::string& filename,
			VkFormat format = VK_FORMAT_R8G8B8A8_SRGB
		);

		// Create texture from raw pixel data
		Texture(
			VkDevice device,
			VkPhysicalDevice physicalDevice,
			VkCommandPool commandPool,
			VkQueue transferQueue,
			const void* data,
			uint32_t width,
			uint32_t height,
			uint32_t channels,
			VkFormat format = VK_FORMAT_R8G8B8A8_SRGB
		);

		~Texture();

		// Prevent copying
		Texture(const Texture&) = delete;
		Texture& operator=(const Texture&) = delete;

		// Getters
		VkImageView getImageView() const { return m_imageView; }
		VkSampler getSampler() const { return m_sampler; }
		VkImage getImage() const { return m_image; }
		VkFormat getFormat() const { return m_format; }
		uint32_t getWidth() const { return m_width; }
		uint32_t getHeight() const { return m_height; }
		uint32_t getMipLevels() const { return m_mipLevels; }

		// Create descriptor info for this texture
		VkDescriptorImageInfo getDescriptorInfo() const;

		// Load common image formats (PNG, JPEG, etc.)
		static std::vector<uint8_t> loadImage(
			const std::string& filename,
			uint32_t& width,
			uint32_t& height,
			uint32_t& channels
		);

	private:
		// Initialize texture from raw pixel data
		void initialize(
			const void* data,
			uint32_t width,
			uint32_t height,
			uint32_t channels
		);

		// Create image resources
		void createImage(
			uint32_t width,
			uint32_t height,
			uint32_t mipLevels,
			VkSampleCountFlagBits samples,
			VkFormat format,
			VkImageTiling tiling,
			VkImageUsageFlags usage,
			VkMemoryPropertyFlags properties,
			VkImage& image,
			VkDeviceMemory& imageMemory
		);

		// Create image view
		VkImageView createImageView(
			VkImage image,
			VkFormat format,
			VkImageAspectFlags aspectFlags,
			uint32_t mipLevels
		);

		// Create texture sampler
		void createSampler();

		// Transition image layout
		void transitionImageLayout(
			VkImage image,
			VkFormat format,
			VkImageLayout oldLayout,
			VkImageLayout newLayout,
			uint32_t mipLevels
		);

		// Copy buffer to image
		void copyBufferToImage(
			VkBuffer buffer,
			VkImage image,
			uint32_t width,
			uint32_t height
		);

		// Generate mipmaps
		void generateMipmaps(
			VkImage image,
			VkFormat imageFormat,
			int32_t width,
			int32_t height,
			uint32_t mipLevels
		);

		// Check if format supports linear blitting (for mipmap generation)
		bool hasStencilComponent(VkFormat format);

		// Find supported format from candidates
		VkFormat findSupportedFormat(
			const std::vector<VkFormat>& candidates,
			VkImageTiling tiling,
			VkFormatFeatureFlags features
		);

		// Find depth format
		VkFormat findDepthFormat();

		// Begin single-time command buffer
		VkCommandBuffer beginSingleTimeCommands();

		// End and submit single-time command buffer
		void endSingleTimeCommands(VkCommandBuffer commandBuffer);

		// Device references
		VkDevice m_device = VK_NULL_HANDLE;
		VkPhysicalDevice m_physicalDevice = VK_NULL_HANDLE;
		VkCommandPool m_commandPool = VK_NULL_HANDLE;
		VkQueue m_transferQueue = VK_NULL_HANDLE;

		// Image resources
		VkImage m_image = VK_NULL_HANDLE;
		VkDeviceMemory m_imageMemory = VK_NULL_HANDLE;
		VkImageView m_imageView = VK_NULL_HANDLE;
		VkSampler m_sampler = VK_NULL_HANDLE;

		// Texture properties
		uint32_t m_width = 0;
		uint32_t m_height = 0;
		uint32_t m_mipLevels = 1;
		VkFormat m_format = VK_FORMAT_R8G8B8A8_SRGB;

		// Texture name for debugging
		std::string m_name;
	};
}
