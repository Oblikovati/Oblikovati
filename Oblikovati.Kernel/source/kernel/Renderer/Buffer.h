#pragma once

#include <vulkan/vulkan.h>

namespace Oblikovati::Kernel::VulkanRenderer
{

	// Forward declaration
	struct Vertex;

	class Buffer
	{
	public:
		Buffer(
			VkDevice device,
			VkPhysicalDevice physicalDevice,
			VkDeviceSize size,
			VkBufferUsageFlags usage,
			VkMemoryPropertyFlags properties
		);
		~Buffer();

		// Prevent copying
		Buffer(const Buffer&) = delete;
		Buffer& operator=(const Buffer&) = delete;

		// Copy data to buffer
		void copyData(const void* data, VkDeviceSize size, VkDeviceSize offset = 0);

		// Copy buffer to buffer
		static void copyBuffer(
			VkDevice device,
			VkQueue transferQueue,
			VkCommandPool commandPool,
			VkBuffer srcBuffer,
			VkBuffer dstBuffer,
			VkDeviceSize size
		);

		// Staging buffer operations
		static std::unique_ptr<Buffer> createStagingBuffer(
			VkDevice device,
			VkPhysicalDevice physicalDevice,
			VkDeviceSize size,
			const void* data = nullptr
		);

		// Create vertex buffer using staging buffer for transfer
		static std::unique_ptr<Buffer> createVertexBuffer(
			VkDevice device,
			VkPhysicalDevice physicalDevice,
			VkQueue transferQueue,
			VkCommandPool commandPool,
			const std::vector<Vertex>& vertices
		);

		// Create index buffer using staging buffer for transfer
		static std::unique_ptr<Buffer> createIndexBuffer(
			VkDevice device,
			VkPhysicalDevice physicalDevice,
			VkQueue transferQueue,
			VkCommandPool commandPool,
			const std::vector<uint32_t>& indices
		);

		// Getters
		VkBuffer getBuffer() const { return m_buffer; }
		VkDeviceMemory getMemory() const { return m_memory; }
		VkDeviceSize getSize() const { return m_size; }

	private:
		// Find memory type for buffer allocation
		static uint32_t findMemoryType(
			VkPhysicalDevice physicalDevice,
			uint32_t typeFilter,
			VkMemoryPropertyFlags properties
		);

		// Create command buffer for transfer operations
		static VkCommandBuffer beginSingleTimeCommands(
			VkDevice device,
			VkCommandPool commandPool
		);

		// End and submit command buffer for transfer operations
		static void endSingleTimeCommands(
			VkDevice device,
			VkQueue queue,
			VkCommandPool commandPool,
			VkCommandBuffer commandBuffer
		);

		VkDevice m_device;
		VkBuffer m_buffer;
		VkDeviceMemory m_memory;
		VkDeviceSize m_size;

		void* m_mappedData = nullptr;
		bool m_persistentlyMapped = false;
	};
}
