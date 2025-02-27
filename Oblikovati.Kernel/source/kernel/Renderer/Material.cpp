#include "KernelPCH.h"

#include "Material.h"
#include "Buffer.h"
#include "Descriptor.h"
#include "Texture.h"

namespace Oblikovati::Kernel::VulkanRenderer
{

	Material::Material(VkDevice device, VkPhysicalDevice physicalDevice, VkCommandPool commandPool, VkQueue transferQueue)
		: m_device(device),
		m_physicalDevice(physicalDevice),
		m_commandPool(commandPool),
		m_transferQueue(transferQueue)
	{
		init();
	}

	Material::Material(
		VkDevice device,
		VkPhysicalDevice physicalDevice,
		VkCommandPool commandPool,
		VkQueue transferQueue,
		const MaterialProperties& properties,
		AlphaMode alphaMode
	) : m_device(device),
		m_physicalDevice(physicalDevice),
		m_commandPool(commandPool),
		m_transferQueue(transferQueue),
		m_properties(properties),
		m_alphaMode(alphaMode)
	{
		init();
	}

	Material::~Material()
	{
		// Uniform buffers and textures are automatically cleaned up by unique_ptr
	}

	void Material::init()
	{
		// Create descriptor system
		m_descriptor = std::make_unique<Descriptor>(m_device);

		// Create default uniform buffers
		createUniformBuffers(MAX_FRAMES_IN_FLIGHT);

		//Logger::debug("Material initialized");
	}

	void Material::createUniformBuffers(uint32_t count)
	{
		m_uniformBuffers.resize(count);

		VkDeviceSize bufferSize = sizeof(MaterialProperties);

		for (size_t i = 0; i < count; i++)
		{
			m_uniformBuffers[i] = std::make_unique<Buffer>(
				m_device,
				m_physicalDevice,
				bufferSize,
				VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
				VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT
			);

			// Initialize with current properties
			m_uniformBuffers[i]->copyData(&m_properties, bufferSize);
		}

		//Logger::debug("Created " + std::to_string(count) + " material uniform buffers");
	}

	void Material::createDescriptors(VkDescriptorSetLayout layout, VkDescriptorPool descriptorPool)
	{
		// Allocate descriptor sets
		m_descriptorSets = m_descriptor->allocateDescriptorSets(
			layout,
			descriptorPool,
			static_cast<uint32_t>(m_uniformBuffers.size())
		);

		// Update descriptor sets with current buffers and textures
		updateDescriptorSets();

		//Logger::debug("Material descriptors created");
	}

	void Material::updateDescriptorSets()
	{
		// Check if we have any textures
		std::vector<std::unique_ptr<Texture>> textures;

		if (m_baseColorTexture)
		{
			textures.push_back(std::move(m_baseColorTexture));
		}

		if (textures.empty())
		{
			// No textures, just update uniform buffers
			m_descriptor->updateDescriptorSets(m_descriptorSets, m_uniformBuffers);
		}
		else
		{
			// Update with textures
			m_descriptor->updateDescriptorSets(m_descriptorSets, m_uniformBuffers, textures);
		}

		//Logger::debug("Material descriptor sets updated");
	}

	void Material::updateUniformBuffer(uint32_t currentFrame)
	{
		if (currentFrame >= m_uniformBuffers.size())
		{
			return;
		}

		// Update uniform buffer with current properties
		m_uniformBuffers[currentFrame]->copyData(&m_properties, sizeof(MaterialProperties));
	}

	bool Material::loadBaseColorTexture(const std::string& path)
	{
		try
		{
			m_baseColorTexture = std::make_unique<Texture>(
				m_device,
				m_physicalDevice,
				m_commandPool,
				m_transferQueue,
				path,
				VK_FORMAT_R8G8B8A8_SRGB
			);

			// Update descriptors if already created
			if (!m_descriptorSets.empty())
			{
				updateDescriptorSets();
			}

			//Logger::info("Loaded base color texture: " + path);
			return true;
		}
		catch (const std::exception& e)
		{
			//Logger::error("Failed to load base color texture: " + path + " - " + e.what());
			return false;
		}
	}

	bool Material::loadMetallicRoughnessTexture(const std::string& path)
	{
		try
		{
			m_metallicRoughnessTexture = std::make_unique<Texture>(
				m_device,
				m_physicalDevice,
				m_commandPool,
				m_transferQueue,
				path,
				VK_FORMAT_R8G8B8A8_UNORM
			);

			// Update descriptors if already created
			if (!m_descriptorSets.empty())
			{
				updateDescriptorSets();
			}

			//Logger::info("Loaded metallic-roughness texture: " + path);
			return true;
		}
		catch (const std::exception& e)
		{
			//Logger::error("Failed to load metallic-roughness texture: " + path + " - " + e.what());
			return false;
		}
	}

	bool Material::loadNormalTexture(const std::string& path)
	{
		try
		{
			m_normalTexture = std::make_unique<Texture>(
				m_device,
				m_physicalDevice,
				m_commandPool,
				m_transferQueue,
				path,
				VK_FORMAT_R8G8B8A8_UNORM
			);

			// Update descriptors if already created
			if (!m_descriptorSets.empty())
			{
				updateDescriptorSets();
			}

			//Logger::info("Loaded normal texture: " + path);
			return true;
		}
		catch (const std::exception& e)
		{
			//Logger::error("Failed to load normal texture: " + path + " - " + e.what());
			return false;
		}
	}

	bool Material::loadOcclusionTexture(const std::string& path)
	{
		try
		{
			m_occlusionTexture = std::make_unique<Texture>(
				m_device,
				m_physicalDevice,
				m_commandPool,
				m_transferQueue,
				path,
				VK_FORMAT_R8G8B8A8_UNORM
			);

			// Update descriptors if already created
			if (!m_descriptorSets.empty())
			{
				updateDescriptorSets();
			}

			//Logger::info("Loaded occlusion texture: " + path);
			return true;
		}
		catch (const std::exception& e)
		{
			//Logger::error("Failed to load occlusion texture: " + path + " - " + e.what());
			return false;
		}
	}

	bool Material::loadEmissiveTexture(const std::string& path)
	{
		try
		{
			m_emissiveTexture = std::make_unique<Texture>(
				m_device,
				m_physicalDevice,
				m_commandPool,
				m_transferQueue,
				path,
				VK_FORMAT_R8G8B8A8_SRGB
			);

			// Update descriptors if already created
			if (!m_descriptorSets.empty())
			{
				updateDescriptorSets();
			}

			//Logger::info("Loaded emissive texture: " + path);
			return true;
		}
		catch (const std::exception& e)
		{
			//Logger::error("Failed to load emissive texture: " + path + " - " + e.what());
			return false;
		}
	}

	VkDescriptorSet Material::getDescriptorSet(uint32_t frameIndex) const
	{
		if (frameIndex < m_descriptorSets.size())
		{
			return m_descriptorSets[frameIndex];
		}
		return VK_NULL_HANDLE;
	}

	Texture* Material::getBaseColorTexture() const
	{
		return m_baseColorTexture.get();
	}

	Texture* Material::getMetallicRoughnessTexture() const
	{
		return m_metallicRoughnessTexture.get();
	}

	Texture* Material::getNormalTexture() const
	{
		return m_normalTexture.get();
	}

	Texture* Material::getOcclusionTexture() const
	{
		return m_occlusionTexture.get();
	}

	Texture* Material::getEmissiveTexture() const
	{
		return m_emissiveTexture.get();
	}

	bool Material::hasBaseColorTexture() const
	{
		return m_baseColorTexture != nullptr;
	}

	bool Material::hasMetallicRoughnessTexture() const
	{
		return m_metallicRoughnessTexture != nullptr;
	}

	bool Material::hasNormalTexture() const
	{
		return m_normalTexture != nullptr;
	}

	bool Material::hasOcclusionTexture() const
	{
		return m_occlusionTexture != nullptr;
	}

	bool Material::hasEmissiveTexture() const
	{
		return m_emissiveTexture != nullptr;
	}

	Material::PushConstantData Material::getPushConstantData() const
	{
		PushConstantData data{};

		// Base color with alpha
		data.baseColorFactor = glm::vec4(m_properties.baseColor, m_properties.opacity);

		data.metallicFactor = m_properties.metallic;
		data.roughnessFactor = m_properties.roughness;
		data.occlusionStrength = m_properties.ambientOcclusion;
		data.emissiveStrength = m_properties.emissiveStrength;

		// Emissive color with padding
		data.emissiveFactor = glm::vec4(m_properties.emissive, 0.0f);

		// Texture flags (0 = false, 1 = true)
		data.hasBaseColorTexture = hasBaseColorTexture() ? 1 : 0;
		data.hasMetallicRoughnessTexture = hasMetallicRoughnessTexture() ? 1 : 0;
		data.hasNormalTexture = hasNormalTexture() ? 1 : 0;
		data.hasOcclusionTexture = hasOcclusionTexture() ? 1 : 0;
		data.hasEmissiveTexture = hasEmissiveTexture() ? 1 : 0;

		data.alphaCutoff = m_properties.alphaCutoff;
		data.textureScale = m_properties.textureScale;
		data.textureOffset = m_properties.textureOffset;

		return data;
	}

}
