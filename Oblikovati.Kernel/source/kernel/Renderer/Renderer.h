#pragma once

#include <vulkan/vulkan.h>
#include <glm/matrix.hpp>

#include "buffer.h"
#include "MousePicking.h"
#include "Pipeline.h"
#include "Swapchain.h"
#include "VulkanContext.h"

namespace Oblikovati::Kernel::VulkanRenderer
{
	class Camera;
	class Model;
	class Window;

	class Renderer
	{
	public:
		Renderer(Window* window, bool enableValidation = true);
		~Renderer();

		// Prevent copying
		Renderer(const Renderer&) = delete;
		Renderer& operator=(const Renderer&) = delete;

		// Main rendering functions
		void renderFrame(const std::vector<std::unique_ptr<Model>>& models);
		void waitIdle();

		// Camera access
		Camera* getCamera() { return m_camera.get(); }

		// Framebuffer capture for testing
		bool captureFramebuffer(const std::string& outputPath);
		bool compareImages(const std::string& imageA, const std::string& imageB, float threshold);

	private:
		// Initialization
		void init();
		void createCommandPool();
		void createCommandBuffers();
		void createSyncObjects();
		void createDescriptorPool();
		void createUniformBuffers();
		void createDescriptorSets();
		void recreateSwapchain();
		// Cleanup
		void cleanup();

		// Update functions
		void updateUniformBuffer(uint32_t currentImage);

		// Record command buffer
		void recordCommandBuffer(VkCommandBuffer commandBuffer, uint32_t imageIndex, const std::vector<std::unique_ptr<Model>>& models);

		// Framebuffer operations for testing
		bool readFramebufferToMemory(std::vector<uint8_t>& buffer, VkFormat format);
		bool saveImageToDisk(const std::vector<uint8_t>& buffer, uint32_t width, uint32_t height, const std::string& filename);

		// Window reference
		Window* m_window;

		// Vulkan context
		std::unique_ptr<VulkanContext> m_context;
		std::unique_ptr<Swapchain> m_swapchain;
		std::unique_ptr<Pipeline> m_pipeline;
		std::unique_ptr<Camera> m_camera;

		// Command buffers
		VkCommandPool m_commandPool;
		std::vector<VkCommandBuffer> m_commandBuffers;

		// Synchronization
		std::vector<VkSemaphore> m_imageAvailableSemaphores;
		std::vector<VkSemaphore> m_renderFinishedSemaphores;
		std::vector<VkFence> m_inFlightFences;

		// Uniform buffers
		VkDescriptorPool m_descriptorPool;
		std::vector<std::unique_ptr<Buffer>> m_uniformBuffers;
		std::vector<VkDescriptorSet> m_descriptorSets;

		// Frame counter
		uint32_t m_currentFrame = 0;
		const int MAX_FRAMES_IN_FLIGHT = 2;

		// Validation
		bool m_enableValidation;
		std::unique_ptr<MousePicking> m_mousePicking;

		VkQueue getGraphicsQueue() const;
	};

	// Uniform buffer object structure
	struct UniformBufferObject
	{
		alignas(16) glm::mat4 model;
		alignas(16) glm::mat4 view;
		alignas(16) glm::mat4 proj;
	};

} // namespace vk_app
