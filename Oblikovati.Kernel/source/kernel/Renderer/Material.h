#pragma once

#include <vulkan/vulkan.h>
#include <glm/glm.hpp>

namespace Oblikovati::Kernel::VulkanRenderer
{
	class Texture;
	class Buffer;
	class Descriptor;

	/**
	 * @brief Material properties for PBR (Physically Based Rendering)
	 */
	struct MaterialProperties
	{
		glm::vec3 baseColor = glm::vec3(1.0f);     // Base color (albedo)
		float metallic = 0.0f;                     // Metallic factor (0 = dielectric, 1 = metal)
		float roughness = 0.5f;                    // Surface roughness
		float ambientOcclusion = 1.0f;             // Ambient occlusion factor
		glm::vec3 emissive = glm::vec3(0.0f);      // Emissive color
		float emissiveStrength = 0.0f;             // Emissive strength multiplier
		glm::vec2 textureScale = glm::vec2(1.0f);  // Texture scaling
		glm::vec2 textureOffset = glm::vec2(0.0f); // Texture offset
		float alphaCutoff = 0.5f;                  // Alpha cutoff threshold for alpha test
		float opacity = 1.0f;                      // Overall opacity
	};

	/**
	 * @brief Material class for 3D models
	 *
	 * Manages material properties, textures, and descriptors for 3D rendering.
	 */
	class Material
	{
	public:
		enum class AlphaMode
		{
			Opaque,
			Mask,
			Blend
		};

		// Creates a default material with basic properties
		Material(VkDevice device, VkPhysicalDevice physicalDevice, VkCommandPool commandPool, VkQueue transferQueue);

		// Creates a material with custom properties
		Material(
			VkDevice device,
			VkPhysicalDevice physicalDevice,
			VkCommandPool commandPool,
			VkQueue transferQueue,
			const MaterialProperties& properties,
			AlphaMode alphaMode = AlphaMode::Opaque
		);

		~Material();

		// Prevent copying
		Material(const Material&) = delete;
		Material& operator=(const Material&) = delete;

		// Create descriptors for this material
		void createDescriptors(VkDescriptorSetLayout layout, VkDescriptorPool descriptorPool);

		// Update uniform buffer with current properties
		void updateUniformBuffer(uint32_t currentFrame);

		// Load textures
		bool loadBaseColorTexture(const std::string& path);
		bool loadMetallicRoughnessTexture(const std::string& path);
		bool loadNormalTexture(const std::string& path);
		bool loadOcclusionTexture(const std::string& path);
		bool loadEmissiveTexture(const std::string& path);

		// Getters
		MaterialProperties& getProperties() { return m_properties; }
		const MaterialProperties& getProperties() const { return m_properties; }

		AlphaMode getAlphaMode() const { return m_alphaMode; }
		void setAlphaMode(AlphaMode mode) { m_alphaMode = mode; }

		VkDescriptorSet getDescriptorSet(uint32_t frameIndex) const;

		// Texture getters
		Texture* getBaseColorTexture() const;
		Texture* getMetallicRoughnessTexture() const;
		Texture* getNormalTexture() const;
		Texture* getOcclusionTexture() const;
		Texture* getEmissiveTexture() const;

		// Texture presence checks
		bool hasBaseColorTexture() const;
		bool hasMetallicRoughnessTexture() const;
		bool hasNormalTexture() const;
		bool hasOcclusionTexture() const;
		bool hasEmissiveTexture() const;

		// Material buffer for push constants
		struct PushConstantData
		{
			glm::vec4 baseColorFactor;      // Base color with alpha
			float metallicFactor;           // Metallic factor
			float roughnessFactor;          // Roughness factor
			float occlusionStrength;        // Occlusion strength
			float emissiveStrength;         // Emissive strength
			glm::vec4 emissiveFactor;       // Emissive color with padding
			int hasBaseColorTexture;        // Texture flags (0 = false, 1 = true)
			int hasMetallicRoughnessTexture;
			int hasNormalTexture;
			int hasOcclusionTexture;
			int hasEmissiveTexture;
			float alphaCutoff;              // Alpha cutoff for masked mode
			glm::vec2 textureScale;         // Texture scaling
			glm::vec2 textureOffset;        // Texture offset
		};

		// Get push constant data for rendering
		PushConstantData getPushConstantData() const;

	private:
		// Initialize material resources
		void init();

		// Create uniform buffer
		void createUniformBuffers(uint32_t count);

		// Update descriptor sets with current textures
		void updateDescriptorSets();

		VkDevice m_device;
		VkPhysicalDevice m_physicalDevice;
		VkCommandPool m_commandPool;
		VkQueue m_transferQueue;

		// Material properties
		MaterialProperties m_properties;
		AlphaMode m_alphaMode = AlphaMode::Opaque;

		// Textures
		std::unique_ptr<Texture> m_baseColorTexture;
		std::unique_ptr<Texture> m_metallicRoughnessTexture;
		std::unique_ptr<Texture> m_normalTexture;
		std::unique_ptr<Texture> m_occlusionTexture;
		std::unique_ptr<Texture> m_emissiveTexture;

		// Descriptor sets
		std::vector<VkDescriptorSet> m_descriptorSets;

		// Uniform buffers (one per frame in flight)
		std::vector<std::unique_ptr<Buffer>> m_uniformBuffers;

		// Descriptor system
		std::unique_ptr<Descriptor> m_descriptor;

		// Maximum number of frames in flight
		static constexpr int MAX_FRAMES_IN_FLIGHT = 2;
	};

}
