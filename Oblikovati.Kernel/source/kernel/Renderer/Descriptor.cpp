#include "KernelPCH.h"

#include "descriptor.h"
#include "buffer.h"

#include <stdexcept>
#include <array>

namespace Oblikovati::Kernel::VulkanRenderer
{

	Descriptor::Descriptor(VkDevice device) : m_device(device)
	{

	}

	Descriptor::~Descriptor()
	{
		cleanup();
	}

	VkDescriptorSetLayout Descriptor::createDescriptorSetLayout()
	{
		// UBO binding
		VkDescriptorSetLayoutBinding uboLayoutBinding{};
		uboLayoutBinding.binding = 0;
		uboLayoutBinding.descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
		uboLayoutBinding.descriptorCount = 1;
		uboLayoutBinding.stageFlags = VK_SHADER_STAGE_VERTEX_BIT;
		uboLayoutBinding.pImmutableSamplers = nullptr;

		// Sampler binding
		VkDescriptorSetLayoutBinding samplerLayoutBinding{};
		samplerLayoutBinding.binding = 1;
		samplerLayoutBinding.descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
		samplerLayoutBinding.descriptorCount = 1;
		samplerLayoutBinding.stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT;
		samplerLayoutBinding.pImmutableSamplers = nullptr;

		std::array<VkDescriptorSetLayoutBinding, 2> bindings = { uboLayoutBinding, samplerLayoutBinding };

		VkDescriptorSetLayoutCreateInfo layoutInfo{};
		layoutInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO;
		layoutInfo.bindingCount = static_cast<uint32_t>(bindings.size());
		layoutInfo.pBindings = bindings.data();

		if (vkCreateDescriptorSetLayout(m_device, &layoutInfo, nullptr, &m_descriptorSetLayout) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create descriptor set layout!");
		}

		//Logger::debug("Descriptor set layout created");
		return m_descriptorSetLayout;
	}

	VkDescriptorPool Descriptor::createDescriptorPool(uint32_t maxSets)
	{
		// Descriptor pool sizes for different types
		std::array<VkDescriptorPoolSize, 2> poolSizes{};

		// Uniform buffers
		poolSizes[0].type = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
		poolSizes[0].descriptorCount = maxSets;

		// Combined image samplers
		poolSizes[1].type = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
		poolSizes[1].descriptorCount = maxSets;

		VkDescriptorPoolCreateInfo poolInfo{};
		poolInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO;
		poolInfo.poolSizeCount = static_cast<uint32_t>(poolSizes.size());
		poolInfo.pPoolSizes = poolSizes.data();
		poolInfo.maxSets = maxSets;

		if (vkCreateDescriptorPool(m_device, &poolInfo, nullptr, &m_descriptorPool) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create descriptor pool!");
		}

		//Logger::debug("Descriptor pool created with max sets: " + std::to_string(maxSets));
		return m_descriptorPool;
	}

	std::vector<VkDescriptorSet> Descriptor::allocateDescriptorSets(
		VkDescriptorSetLayout layout,
		VkDescriptorPool pool,
		uint32_t count
	)
	{
		std::vector<VkDescriptorSetLayout> layouts(count, layout);

		VkDescriptorSetAllocateInfo allocInfo{};
		allocInfo.sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
		allocInfo.descriptorPool = pool;
		allocInfo.descriptorSetCount = count;
		allocInfo.pSetLayouts = layouts.data();

		std::vector<VkDescriptorSet> descriptorSets(count);
		if (vkAllocateDescriptorSets(m_device, &allocInfo, descriptorSets.data()) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to allocate descriptor sets!");
		}

		//Logger::debug("Allocated " + std::to_string(count) + " descriptor sets");
		return descriptorSets;
	}

	void Descriptor::updateDescriptorSets(
		const std::vector<VkDescriptorSet>& descriptorSets,
		const std::vector<std::unique_ptr<Buffer>>& uniformBuffers
	)
	{
		for (size_t i = 0; i < descriptorSets.size(); i++)
		{
			VkDescriptorBufferInfo bufferInfo{};
			bufferInfo.buffer = uniformBuffers[i]->getBuffer();
			bufferInfo.offset = 0;
			bufferInfo.range = VK_WHOLE_SIZE;

			VkWriteDescriptorSet descriptorWrite{};
			descriptorWrite.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
			descriptorWrite.dstSet = descriptorSets[i];
			descriptorWrite.dstBinding = 0;
			descriptorWrite.dstArrayElement = 0;
			descriptorWrite.descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
			descriptorWrite.descriptorCount = 1;
			descriptorWrite.pBufferInfo = &bufferInfo;

			vkUpdateDescriptorSets(m_device, 1, &descriptorWrite, 0, nullptr);
		}

		//Logger::debug("Updated descriptor sets for uniform buffers");
	}

	void Descriptor::updateDescriptorSets(
		const std::vector<VkDescriptorSet>& descriptorSets,
		const std::vector<std::unique_ptr<Buffer>>& uniformBuffers,
		const std::vector<std::unique_ptr<Texture>>& textures
	)
	{
		// This implementation requires the Texture class
		// Since we haven't implemented that yet, we'll implement a basic version

		for (size_t i = 0; i < descriptorSets.size(); i++)
		{
			VkDescriptorBufferInfo bufferInfo{};
			bufferInfo.buffer = uniformBuffers[i]->getBuffer();
			bufferInfo.offset = 0;
			bufferInfo.range = VK_WHOLE_SIZE;

			// For each set, we'll have at most 2 writes (UBO and texture)
			std::array<VkWriteDescriptorSet, 2> descriptorWrites{};

			// UBO descriptor
			descriptorWrites[0].sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
			descriptorWrites[0].dstSet = descriptorSets[i];
			descriptorWrites[0].dstBinding = 0;
			descriptorWrites[0].dstArrayElement = 0;
			descriptorWrites[0].descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
			descriptorWrites[0].descriptorCount = 1;
			descriptorWrites[0].pBufferInfo = &bufferInfo;

			// Texture descriptor - would be filled if we had a Texture class
			// For now we'll just update the UBO

			vkUpdateDescriptorSets(m_device, 1, descriptorWrites.data(), 0, nullptr);
		}

		//Logger::debug("Updated descriptor sets for uniform buffers and textures");
	}

	void Descriptor::cleanup()
	{
		if (m_device == VK_NULL_HANDLE)
		{
			return;
		}

		if (m_descriptorPool != VK_NULL_HANDLE)
		{
			vkDestroyDescriptorPool(m_device, m_descriptorPool, nullptr);
			m_descriptorPool = VK_NULL_HANDLE;
		}

		if (m_descriptorSetLayout != VK_NULL_HANDLE)
		{
			vkDestroyDescriptorSetLayout(m_device, m_descriptorSetLayout, nullptr);
			m_descriptorSetLayout = VK_NULL_HANDLE;
		}

		//Logger::debug("Descriptor resources cleaned up");
	}

} // namespace vk_app
