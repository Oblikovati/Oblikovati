#pragma once

#include <vulkan/vulkan.h>

namespace Oblikovati::Kernel::VulkanRenderer
{
	class Buffer;
	class Texture;

	class Descriptor
	{
	public:
		Descriptor(VkDevice device);
		~Descriptor();

		// Prevent copying
		Descriptor(const Descriptor&) = delete;
		Descriptor& operator=(const Descriptor&) = delete;

		// Create descriptor set layout (call once during initialization)
		VkDescriptorSetLayout createDescriptorSetLayout();

		// Create descriptor pool (call once during initialization)
		VkDescriptorPool createDescriptorPool(uint32_t maxSets);

		// Allocate descriptor sets from the pool
		std::vector<VkDescriptorSet> allocateDescriptorSets(
			VkDescriptorSetLayout layout,
			VkDescriptorPool pool,
			uint32_t count
		);

		// Update descriptor sets for uniform buffers
		void updateDescriptorSets(
			const std::vector<VkDescriptorSet>& descriptorSets,
			const std::vector<std::unique_ptr<Buffer>>& uniformBuffers
		);

		// Update descriptor sets for uniform buffers and textures
		void updateDescriptorSets(
			const std::vector<VkDescriptorSet>& descriptorSets,
			const std::vector<std::unique_ptr<Buffer>>& uniformBuffers,
			const std::vector<std::unique_ptr<Texture>>& textures
		);

		// Cleanup resources
		void cleanup();

	private:
		VkDevice m_device;
		VkDescriptorSetLayout m_descriptorSetLayout = VK_NULL_HANDLE;
		VkDescriptorPool m_descriptorPool = VK_NULL_HANDLE;
	};

}
