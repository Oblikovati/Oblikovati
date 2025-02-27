#include "KernelPCH.h"

#include "buffer.h"
#include "Vertex.h"

#include <stdexcept>
#include <cstring>

namespace Oblikovati::Kernel::VulkanRenderer
{

	Buffer::Buffer(
		VkDevice device,
		VkPhysicalDevice physicalDevice,
		VkDeviceSize size,
		VkBufferUsageFlags usage,
		VkMemoryPropertyFlags properties
	) : m_device(device), m_size(size)
	{
		// Create buffer
		VkBufferCreateInfo bufferInfo{};
		bufferInfo.sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
		bufferInfo.size = size;
		bufferInfo.usage = usage;
		bufferInfo.sharingMode = VK_SHARING_MODE_EXCLUSIVE;

		if (vkCreateBuffer(device, &bufferInfo, nullptr, &m_buffer) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create buffer!");
		}

		// Get memory requirements
		VkMemoryRequirements memRequirements;
		vkGetBufferMemoryRequirements(device, m_buffer, &memRequirements);

		// Allocate memory
		VkMemoryAllocateInfo allocInfo{};
		allocInfo.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
		allocInfo.allocationSize = memRequirements.size;
		allocInfo.memoryTypeIndex = findMemoryType(
			physicalDevice,
			memRequirements.memoryTypeBits,
			properties
		);

		if (vkAllocateMemory(device, &allocInfo, nullptr, &m_memory) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to allocate buffer memory!");
		}

		// Bind memory to buffer
		vkBindBufferMemory(device, m_buffer, m_memory, 0);

		// If this is a host-visible buffer, map it persistently
		if (properties & VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT)
		{
			vkMapMemory(device, m_memory, 0, size, 0, &m_mappedData);
			m_persistentlyMapped = true;
		}
	}

	Buffer::~Buffer()
	{
		if (m_device != VK_NULL_HANDLE)
		{
			if (m_persistentlyMapped && m_mappedData)
			{
				vkUnmapMemory(m_device, m_memory);
				m_mappedData = nullptr;
			}

			if (m_buffer != VK_NULL_HANDLE)
			{
				vkDestroyBuffer(m_device, m_buffer, nullptr);
				m_buffer = VK_NULL_HANDLE;
			}

			if (m_memory != VK_NULL_HANDLE)
			{
				vkFreeMemory(m_device, m_memory, nullptr);
				m_memory = VK_NULL_HANDLE;
			}
		}
	}

	void Buffer::copyData(const void* data, VkDeviceSize size, VkDeviceSize offset)
	{
		if (offset + size > m_size)
		{
			throw std::runtime_error("Buffer copy out of bounds!");
		}

		if (m_persistentlyMapped)
		{
			// If buffer is already mapped, copy directly
			memcpy(static_cast<char*>(m_mappedData) + offset, data, static_cast<size_t>(size));
		}
		else
		{
			// Otherwise, map temporarily
			void* mappedData;
			vkMapMemory(m_device, m_memory, offset, size, 0, &mappedData);
			memcpy(mappedData, data, static_cast<size_t>(size));
			vkUnmapMemory(m_device, m_memory);
		}
	}

	void Buffer::copyBuffer(
		VkDevice device,
		VkQueue transferQueue,
		VkCommandPool commandPool,
		VkBuffer srcBuffer,
		VkBuffer dstBuffer,
		VkDeviceSize size
	)
	{
		// Create command buffer
		VkCommandBuffer commandBuffer = beginSingleTimeCommands(device, commandPool);

		// Record copy command
		VkBufferCopy copyRegion{};
		copyRegion.srcOffset = 0;
		copyRegion.dstOffset = 0;
		copyRegion.size = size;
		vkCmdCopyBuffer(commandBuffer, srcBuffer, dstBuffer, 1, &copyRegion);

		// Submit and cleanup
		endSingleTimeCommands(device, transferQueue, commandPool, commandBuffer);
	}

	std::unique_ptr<Buffer> Buffer::createStagingBuffer(
		VkDevice device,
		VkPhysicalDevice physicalDevice,
		VkDeviceSize size,
		const void* data
	)
	{
		// Create host-visible, host-coherent buffer for staging
		auto stagingBuffer = std::make_unique<Buffer>(
			device,
			physicalDevice,
			size,
			VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
			VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT
		);

		// Copy data to staging buffer
		if (data)
		{
			stagingBuffer->copyData(data, size);
		}

		return stagingBuffer;
	}

	std::unique_ptr<Buffer> Buffer::createVertexBuffer(VkDevice device,
		VkPhysicalDevice physicalDevice,
		VkQueue transferQueue,
		VkCommandPool commandPool,
		const std::vector<Vertex>& vertices
	)
	{
		VkDeviceSize bufferSize = sizeof(vertices[0]) * vertices.size();

		// Create staging buffer
		auto stagingBuffer = createStagingBuffer(
			device,
			physicalDevice,
			bufferSize,
			vertices.data()
		);

		// Create device-local buffer
		auto vertexBuffer = std::make_unique<Buffer>(
			device,
			physicalDevice,
			bufferSize,
			VK_BUFFER_USAGE_TRANSFER_DST_BIT | VK_BUFFER_USAGE_VERTEX_BUFFER_BIT,
			VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT
		);

		// Copy from staging to device-local
		copyBuffer(
			device,
			transferQueue,
			commandPool,
			stagingBuffer->getBuffer(),
			vertexBuffer->getBuffer(),
			bufferSize
		);

		return vertexBuffer;
	}

	std::unique_ptr<Buffer> Buffer::createIndexBuffer(VkDevice device, VkPhysicalDevice physicalDevice,	VkQueue transferQueue,
		VkCommandPool commandPool, const std::vector<uint32_t>& indices)
	{
		VkDeviceSize bufferSize = sizeof(indices[0]) * indices.size();

		// Create staging buffer
		auto stagingBuffer = createStagingBuffer(
			device,
			physicalDevice,
			bufferSize,
			indices.data()
		);

		// Create device-local buffer
		auto indexBuffer = std::make_unique<Buffer>(
			device,
			physicalDevice,
			bufferSize,
			VK_BUFFER_USAGE_TRANSFER_DST_BIT | VK_BUFFER_USAGE_INDEX_BUFFER_BIT,
			VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT
		);

		// Copy from staging to device-local
		copyBuffer(
			device,
			transferQueue,
			commandPool,
			stagingBuffer->getBuffer(),
			indexBuffer->getBuffer(),
			bufferSize
		);

		return indexBuffer;
	}

	uint32_t Buffer::findMemoryType(
		VkPhysicalDevice physicalDevice,
		uint32_t typeFilter,
		VkMemoryPropertyFlags properties
	)
	{
		VkPhysicalDeviceMemoryProperties memProperties;
		vkGetPhysicalDeviceMemoryProperties(physicalDevice, &memProperties);

		for (uint32_t i = 0; i < memProperties.memoryTypeCount; i++)
		{
			if ((typeFilter & (1 << i)) &&
				(memProperties.memoryTypes[i].propertyFlags & properties) == properties)
			{
				return i;
			}
		}

		throw std::runtime_error("Failed to find suitable memory type!");
	}

	VkCommandBuffer Buffer::beginSingleTimeCommands(VkDevice device, VkCommandPool commandPool)
	{
		VkCommandBufferAllocateInfo allocInfo{};
		allocInfo.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
		allocInfo.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
		allocInfo.commandPool = commandPool;
		allocInfo.commandBufferCount = 1;

		VkCommandBuffer commandBuffer;
		vkAllocateCommandBuffers(device, &allocInfo, &commandBuffer);

		VkCommandBufferBeginInfo beginInfo{};
		beginInfo.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
		beginInfo.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;

		vkBeginCommandBuffer(commandBuffer, &beginInfo);

		return commandBuffer;
	}

	void Buffer::endSingleTimeCommands(VkDevice device,	VkQueue queue, VkCommandPool commandPool, VkCommandBuffer commandBuffer)
	{
		vkEndCommandBuffer(commandBuffer);

		VkSubmitInfo submitInfo{};
		submitInfo.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
		submitInfo.commandBufferCount = 1;
		submitInfo.pCommandBuffers = &commandBuffer;

		vkQueueSubmit(queue, 1, &submitInfo, VK_NULL_HANDLE);
		vkQueueWaitIdle(queue);

		vkFreeCommandBuffers(device, commandPool, 1, &commandBuffer);
	}
}
