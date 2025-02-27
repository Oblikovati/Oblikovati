#include "KernelPCH.h"

#include "Renderer.h"


#include <glm/glm.hpp>
#include <glm/gtc/matrix_transform.hpp>

// For framebuffer capture and comparison
#define STB_IMAGE_WRITE_IMPLEMENTATION
#include <stb_image_write.h>
#define STB_IMAGE_IMPLEMENTATION
#include <stb_image.h>

#include "descriptor.h"
#include "MousePicking.h"
#include "Pipeline.h"
#include "Swapchain.h"
#include "VulkanContext.h"

namespace Oblikovati::Kernel::VulkanRenderer
{

	Renderer::Renderer(Window* window, bool enableValidation)
		: m_window(window), m_enableValidation(enableValidation)
	{
		init();
	}

	Renderer::~Renderer()
	{
		cleanup();
	}

	void Renderer::init()
	{
		// Create Vulkan context
		m_context = std::make_unique<VulkanContext>(m_window, m_enableValidation);

		// Create swapchain
		m_swapchain = std::make_unique<Swapchain>(m_context.get(), m_window);

		// Create command pool
		createCommandPool();

		// Create camera
		m_camera = std::make_unique<Camera>();

		// Default camera position
		m_camera->setPosition(glm::vec3(0.0f, 0.0f, 5.0f));
		m_camera->setTarget(glm::vec3(0.0f, 0.0f, 0.0f));

		// Create pipeline
		m_pipeline = std::make_unique<Pipeline>(
			m_context.get(),
			m_swapchain->getExtent(),
			m_swapchain->getImageFormat(),
			m_swapchain->getDepthFormat()
		);

		// Create framebuffers
		m_swapchain->createFramebuffers(m_pipeline->getRenderPass());

		// Create sync objects
		createSyncObjects();

		// Create uniform buffers and descriptor sets
		createDescriptorPool();
		createUniformBuffers();
		createDescriptorSets();

		// Create command buffers
		createCommandBuffers();

		// Initialize mouse picking
		m_mousePicking = std::make_unique<MousePicking>(
			m_context->getDevice(),
			m_context->getPhysicalDevice(),
			m_window,
			m_camera.get()
		);

		m_mousePicking->init(this, m_swapchain->getExtent().width, m_swapchain->getExtent().height);

		//Logger::info("Renderer initialized");
	}

	void Renderer::createCommandPool()
	{
		VkCommandPoolCreateInfo poolInfo{};
		poolInfo.sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO;

		// Get queue family index
		VulkanContext::QueueFamilyIndices queueFamilyIndices = m_context->getQueueFamilyIndices();
		poolInfo.queueFamilyIndex = queueFamilyIndices.graphicsFamily.value();

		// Allow command buffers to be reset
		poolInfo.flags = VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT;

		if (vkCreateCommandPool(m_context->getDevice(), &poolInfo, nullptr, &m_commandPool) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create command pool!");
		}

		//Logger::debug("Command pool created");
	}

	void Renderer::createCommandBuffers()
	{
		// Allocate command buffers
		m_commandBuffers.resize(MAX_FRAMES_IN_FLIGHT);

		VkCommandBufferAllocateInfo allocInfo{};
		allocInfo.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
		allocInfo.commandPool = m_commandPool;
		allocInfo.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
		allocInfo.commandBufferCount = static_cast<uint32_t>(m_commandBuffers.size());

		if (vkAllocateCommandBuffers(m_context->getDevice(), &allocInfo, m_commandBuffers.data()) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to allocate command buffers!");
		}

		//Logger::debug("Command buffers created");
	}

	void Renderer::createSyncObjects()
	{
		// Resize sync objects
		m_imageAvailableSemaphores.resize(MAX_FRAMES_IN_FLIGHT);
		m_renderFinishedSemaphores.resize(MAX_FRAMES_IN_FLIGHT);
		m_inFlightFences.resize(MAX_FRAMES_IN_FLIGHT);

		// Create semaphores and fences
		VkSemaphoreCreateInfo semaphoreInfo{};
		semaphoreInfo.sType = VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO;

		VkFenceCreateInfo fenceInfo{};
		fenceInfo.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO;
		fenceInfo.flags = VK_FENCE_CREATE_SIGNALED_BIT; // Start signaled so we don't wait indefinitely on first frame

		for (size_t i = 0; i < MAX_FRAMES_IN_FLIGHT; i++)
		{
			if (vkCreateSemaphore(m_context->getDevice(), &semaphoreInfo, nullptr, &m_imageAvailableSemaphores[i]) != VK_SUCCESS ||
				vkCreateSemaphore(m_context->getDevice(), &semaphoreInfo, nullptr, &m_renderFinishedSemaphores[i]) != VK_SUCCESS ||
				vkCreateFence(m_context->getDevice(), &fenceInfo, nullptr, &m_inFlightFences[i]) != VK_SUCCESS)
			{
				throw std::runtime_error("Failed to create synchronization objects!");
			}
		}

		//Logger::debug("Sync objects created");
	}

	void Renderer::createDescriptorPool()
	{
		// Create descriptor system
		m_descriptor = std::make_unique<Descriptor>(m_context->getDevice());

		// Create descriptor pool
		m_descriptorPool = m_descriptor->createDescriptorPool(MAX_FRAMES_IN_FLIGHT * 10); // Allow 10 sets per frame

		//Logger::debug("Descriptor pool created");
	}

	void Renderer::createUniformBuffers()
	{
		VkDeviceSize bufferSize = sizeof(UniformBufferObject);

		m_uniformBuffers.resize(MAX_FRAMES_IN_FLIGHT);

		for (size_t i = 0; i < MAX_FRAMES_IN_FLIGHT; i++)
		{
			m_uniformBuffers[i] = std::make_unique<Buffer>(
				m_context->getDevice(),
				m_context->getPhysicalDevice(),
				bufferSize,
				VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
				VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT
			);
		}

		//Logger::debug("Uniform buffers created");
	}

	void Renderer::createDescriptorSets()
	{
		// Create descriptor set layout
		VkDescriptorSetLayout layout = m_descriptor->createDescriptorSetLayout();

		// Allocate descriptor sets
		m_descriptorSets = m_descriptor->allocateDescriptorSets(
			layout,
			m_descriptorPool,
			static_cast<uint32_t>(MAX_FRAMES_IN_FLIGHT)
		);

		// Update descriptor sets with uniform buffers
		m_descriptor->updateDescriptorSets(m_descriptorSets, m_uniformBuffers);

		//Logger::debug("Descriptor sets created");
	}

	void Renderer::updateUniformBuffer(uint32_t currentImage)
	{
		// Calculate delta time
		static auto startTime = std::chrono::high_resolution_clock::now();
		auto currentTime = std::chrono::high_resolution_clock::now();
		float deltaTime = std::chrono::duration<float, std::chrono::seconds::period>(currentTime - startTime).count();

		// Update uniform buffer with current camera matrices
		UniformBufferObject ubo{};
		ubo.model = glm::mat4(1.0f); // Identity matrix for model (will be overridden per model)
		ubo.view = m_camera->getViewMatrix();
		ubo.proj = m_camera->getProjectionMatrix();

		// Copy to uniform buffer
		m_uniformBuffers[currentImage]->copyData(&ubo, sizeof(ubo));
	}

	void Renderer::renderFrame(const std::vector<std::unique_ptr<Model>>& models)
	{
		// Wait for previous frame to finish
		vkWaitForFences(m_context->getDevice(), 1, &m_inFlightFences[m_currentFrame], VK_TRUE, UINT64_MAX);

		// Acquire next image from swapchain
		uint32_t imageIndex;
		VkResult result = m_swapchain->acquireNextImage(&imageIndex, m_imageAvailableSemaphores[m_currentFrame], VK_NULL_HANDLE);

		// Handle swapchain recreation
		if (result == VK_ERROR_OUT_OF_DATE_KHR || result == VK_SUBOPTIMAL_KHR || m_window->wasResized())
		{
			m_window->resetResizedFlag();
			recreateSwapchain();
			return;
		}
		else if (result != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to acquire swapchain image!");
		}

		// Reset fence only if we're submitting work
		vkResetFences(m_context->getDevice(), 1, &m_inFlightFences[m_currentFrame]);

		// Reset and record command buffer
		vkResetCommandBuffer(m_commandBuffers[m_currentFrame], 0);
		recordCommandBuffer(m_commandBuffers[m_currentFrame], imageIndex, models);

		// Update uniform buffer
		updateUniformBuffer(m_currentFrame);

		// Submit command buffer
		VkSubmitInfo submitInfo{};
		submitInfo.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;

		VkSemaphore waitSemaphores[] = { m_imageAvailableSemaphores[m_currentFrame] };
		VkPipelineStageFlags waitStages[] = { VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT };
		submitInfo.waitSemaphoreCount = 1;
		submitInfo.pWaitSemaphores = waitSemaphores;
		submitInfo.pWaitDstStageMask = waitStages;
		submitInfo.commandBufferCount = 1;
		submitInfo.pCommandBuffers = &m_commandBuffers[m_currentFrame];

		VkSemaphore signalSemaphores[] = { m_renderFinishedSemaphores[m_currentFrame] };
		submitInfo.signalSemaphoreCount = 1;
		submitInfo.pSignalSemaphores = signalSemaphores;

		if (vkQueueSubmit(m_context->getGraphicsQueue(), 1, &submitInfo, m_inFlightFences[m_currentFrame]) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to submit draw command buffer!");
		}

		// Present rendered image
		result = m_swapchain->presentImage(imageIndex, m_renderFinishedSemaphores[m_currentFrame]);

		// Handle swapchain recreation
		if (result == VK_ERROR_OUT_OF_DATE_KHR || result == VK_SUBOPTIMAL_KHR)
		{
			recreateSwapchain();
		}
		else if (result != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to present swapchain image!");
		}

		// Advance to next frame
		m_currentFrame = (m_currentFrame + 1) % MAX_FRAMES_IN_FLIGHT;
	}

	void Renderer::waitIdle()
	{
		vkDeviceWaitIdle(m_context->getDevice());
	}

	void Renderer::recreateSwapchain()
	{
		// Wait for device to be idle
		vkDeviceWaitIdle(m_context->getDevice());

		// Recreate swapchain
		m_swapchain->recreate();

		// Recreate pipeline with new swapchain extent
		m_pipeline = std::make_unique<Pipeline>(
			m_context.get(),
			m_swapchain->getExtent(),
			m_swapchain->getImageFormat(),
			m_swapchain->getDepthFormat()
		);

		// Recreate framebuffers
		m_swapchain->createFramebuffers(m_pipeline->getRenderPass());

		// Update mouse picking resources
		m_mousePicking->updateResources(m_swapchain->getExtent().width, m_swapchain->getExtent().height);

		//Logger::info("Swapchain recreated");
	}

	void Renderer::recordCommandBuffer(VkCommandBuffer commandBuffer, uint32_t imageIndex, const std::vector<std::unique_ptr<Model>>& models)
	{
		// Begin command buffer
		VkCommandBufferBeginInfo beginInfo{};
		beginInfo.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
		beginInfo.flags = 0;
		beginInfo.pInheritanceInfo = nullptr;

		if (vkBeginCommandBuffer(commandBuffer, &beginInfo) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to begin recording command buffer!");
		}

		// Begin render pass
		m_pipeline->beginRenderPass(commandBuffer, m_swapchain->getFramebuffers()[imageIndex], m_swapchain->getExtent());

		// Bind pipeline
		m_pipeline->bindPipeline(commandBuffer);

		// Bind descriptor sets
		vkCmdBindDescriptorSets(
			commandBuffer,
			VK_PIPELINE_BIND_POINT_GRAPHICS,
			m_pipeline->getPipelineLayout(),
			0,
			1,
			&m_descriptorSets[m_currentFrame],
			0,
			nullptr
		);

		// Draw models
		for (const auto& model : models)
		{
			// Set model matrix via push constant
			glm::mat4 modelMatrix = model->getTransform();

			vkCmdPushConstants(
				commandBuffer,
				m_pipeline->getPipelineLayout(),
				VK_SHADER_STAGE_VERTEX_BIT,
				0,
				sizeof(glm::mat4),
				&modelMatrix
			);

			// Draw meshes
			for (const auto& mesh : model->getMeshes())
			{
				mesh->bind(commandBuffer);
				mesh->draw(commandBuffer);
			}
		}

		// End render pass
		m_pipeline->endRenderPass(commandBuffer);

		// End command buffer
		if (vkEndCommandBuffer(commandBuffer) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to record command buffer!");
		}
	}

	bool Renderer::captureFramebuffer(const std::string& outputPath)
	{
		// Wait for rendering to complete
		vkDeviceWaitIdle(m_context->getDevice());

		// Get swapchain info
		VkExtent2D extent = m_swapchain->getExtent();
		VkFormat format = m_swapchain->getImageFormat();

		// Read framebuffer to memory
		std::vector<uint8_t> buffer;
		if (!readFramebufferToMemory(buffer, format))
		{
			return false;
		}

		// Save to disk
		return saveImageToDisk(buffer, extent.width, extent.height, outputPath);
	}

	bool Renderer::readFramebufferToMemory(std::vector<uint8_t>& buffer, VkFormat format)
	{
		// Get image size
		VkExtent2D extent = m_swapchain->getExtent();
		VkDeviceSize imageSize = extent.width * extent.height * 4; // Assuming 4 bytes per pixel (RGBA8)

		// Resize buffer
		buffer.resize(imageSize);

		// Create staging buffer
		auto stagingBuffer = std::make_unique<Buffer>(
			m_context->getDevice(),
			m_context->getPhysicalDevice(),
			imageSize,
			VK_BUFFER_USAGE_TRANSFER_DST_BIT,
			VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT
		);

		// Begin command buffer
		VkCommandBuffer commandBuffer;
		VkCommandBufferAllocateInfo allocInfo{};
		allocInfo.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO;
		allocInfo.level = VK_COMMAND_BUFFER_LEVEL_PRIMARY;
		allocInfo.commandPool = m_commandPool;
		allocInfo.commandBufferCount = 1;

		vkAllocateCommandBuffers(m_context->getDevice(), &allocInfo, &commandBuffer);

		VkCommandBufferBeginInfo beginInfo{};
		beginInfo.sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO;
		beginInfo.flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT;

		vkBeginCommandBuffer(commandBuffer, &beginInfo);

		// Transition swapchain image layout for transfer
		VkImageMemoryBarrier barrier{};
		barrier.sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER;
		barrier.oldLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR;
		barrier.newLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
		barrier.srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED;
		barrier.dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED;
		barrier.image = m_swapchain->getImages()[0]; // Use first swapchain image
		barrier.subresourceRange.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT;
		barrier.subresourceRange.baseMipLevel = 0;
		barrier.subresourceRange.levelCount = 1;
		barrier.subresourceRange.baseArrayLayer = 0;
		barrier.subresourceRange.layerCount = 1;
		barrier.srcAccessMask = 0;
		barrier.dstAccessMask = VK_ACCESS_TRANSFER_READ_BIT;

		vkCmdPipelineBarrier(
			commandBuffer,
			VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT,
			VK_PIPELINE_STAGE_TRANSFER_BIT,
			0,
			0, nullptr,
			0, nullptr,
			1, &barrier
		);

		// Copy image to buffer
		VkBufferImageCopy region{};
		region.bufferOffset = 0;
		region.bufferRowLength = 0;
		region.bufferImageHeight = 0;
		region.imageSubresource.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT;
		region.imageSubresource.mipLevel = 0;
		region.imageSubresource.baseArrayLayer = 0;
		region.imageSubresource.layerCount = 1;
		region.imageOffset = { 0, 0, 0 };
		region.imageExtent = { extent.width, extent.height, 1 };

		vkCmdCopyImageToBuffer(
			commandBuffer,
			m_swapchain->getImages()[0],
			VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
			stagingBuffer->getBuffer(),
			1,
			&region
		);

		// Transition back to original layout
		barrier.oldLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
		barrier.newLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR;
		barrier.srcAccessMask = VK_ACCESS_TRANSFER_READ_BIT;
		barrier.dstAccessMask = 0;

		vkCmdPipelineBarrier(
			commandBuffer,
			VK_PIPELINE_STAGE_TRANSFER_BIT,
			VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT,
			0,
			0, nullptr,
			0, nullptr,
			1, &barrier
		);

		// End and submit command buffer
		vkEndCommandBuffer(commandBuffer);

		VkSubmitInfo submitInfo{};
		submitInfo.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
		submitInfo.commandBufferCount = 1;
		submitInfo.pCommandBuffers = &commandBuffer;

		vkQueueSubmit(m_context->getGraphicsQueue(), 1, &submitInfo, VK_NULL_HANDLE);
		vkQueueWaitIdle(m_context->getGraphicsQueue());

		vkFreeCommandBuffers(m_context->getDevice(), m_commandPool, 1, &commandBuffer);

		// Copy from staging buffer to output buffer
		void* mappedMemory = nullptr;
		vkMapMemory(m_context->getDevice(), stagingBuffer->getMemory(), 0, imageSize, 0, &mappedMemory);

		memcpy(buffer.data(), mappedMemory, static_cast<size_t>(imageSize));

		vkUnmapMemory(m_context->getDevice(), stagingBuffer->getMemory());

		return true;
	}

	bool Renderer::saveImageToDisk(const std::vector<uint8_t>& buffer, uint32_t width, uint32_t height, const std::string& filename)
	{
		// Write to file using stb_image_write
		int result = stbi_write_png(filename.c_str(), width, height, 4, buffer.data(), width * 4);

		if (result == 0)
		{
			//Logger::error("Failed to save image to disk: " + filename);
			return false;
		}

		//Logger::info("Saved framebuffer to: " + filename);
		return true;
	}

	bool Renderer::compareImages(const std::string& imageA, const std::string& imageB, float threshold)
	{
		// Load images
		int widthA, heightA, channelsA;
		int widthB, heightB, channelsB;

		unsigned char* dataA = stbi_load(imageA.c_str(), &widthA, &heightA, &channelsA, 4);
		unsigned char* dataB = stbi_load(imageB.c_str(), &widthB, &heightB, &channelsB, 4);

		if (!dataA || !dataB)
		{
			if (dataA) stbi_image_free(dataA);
			if (dataB) stbi_image_free(dataB);

			//Logger::error("Failed to load images for comparison");
			return false;
		}

		// Check dimensions
		if (widthA != widthB || heightA != heightB)
		{
			stbi_image_free(dataA);
			stbi_image_free(dataB);

			//Logger::error("Image dimensions do not match");
			return false;
		}

		// Compare pixels
		size_t pixelCount = widthA * heightA;
		size_t diffCount = 0;

		for (size_t i = 0; i < pixelCount; i++)
		{
			// Compare RGBA values
			size_t index = i * 4;

			int diffR = abs(dataA[index + 0] - dataB[index + 0]);
			int diffG = abs(dataA[index + 1] - dataB[index + 1]);
			int diffB = abs(dataA[index + 2] - dataB[index + 2]);
			int diffA = abs(dataA[index + 3] - dataB[index + 3]);

			// If any channel differs more than threshold, count as different pixel
			if (diffR > 3 || diffG > 3 || diffB > 3 || diffA > 3)
			{
				diffCount++;
			}
		}

		// Free image data
		stbi_image_free(dataA);
		stbi_image_free(dataB);

		// Calculate difference percentage
		float diffPercentage = static_cast<float>(diffCount) / static_cast<float>(pixelCount);

		// Compare with threshold
		bool result = diffPercentage <= threshold;

		if (result)
		{
			//Logger::info("Images match (difference: " + std::to_string(diffPercentage * 100.0f) + "%)");
		}
		else
		{
			//Logger::error("Images do not match (difference: " + std::to_string(diffPercentage * 100.0f) +
						 "%, threshold: " + std::to_string(threshold * 100.0f) + "%)");
		}

		return result;
	}

	void Renderer::cleanup()
	{
		// Wait for device to be idle
		if (m_context)
		{
			vkDeviceWaitIdle(m_context->getDevice());
		}

		// Cleanup mouse picking
		m_mousePicking.reset();

		// Cleanup descriptor sets and buffers
		m_uniformBuffers.clear();

		// Cleanup synchronization objects
		for (size_t i = 0; i < MAX_FRAMES_IN_FLIGHT; i++)
		{
			if (m_context)
			{
				vkDestroySemaphore(m_context->getDevice(), m_renderFinishedSemaphores[i], nullptr);
				vkDestroySemaphore(m_context->getDevice(), m_imageAvailableSemaphores[i], nullptr);
				vkDestroyFence(m_context->getDevice(), m_inFlightFences[i], nullptr);
			}
		}

		// Cleanup descriptor pool
		if (m_context && m_descriptorPool != VK_NULL_HANDLE)
		{
			vkDestroyDescriptorPool(m_context->getDevice(), m_descriptorPool, nullptr);
			m_descriptorPool = VK_NULL_HANDLE;
		}

		// Cleanup pipeline and descriptor
		m_pipeline.reset();
		m_descriptor.reset();

		// Cleanup command pool
		if (m_context && m_commandPool != VK_NULL_HANDLE)
		{
			vkDestroyCommandPool(m_context->getDevice(), m_commandPool, nullptr);
			m_commandPool = VK_NULL_HANDLE;
		}

		// Cleanup swapchain (must be before context)
		m_swapchain.reset();

		// Cleanup context (must be last)
		m_context.reset();

		//Logger::info("Renderer resources cleaned up");
	}

	VkQueue Renderer::getGraphicsQueue() const
	{
		return m_context->getGraphicsQueue();
	}

	MousePicking* Renderer::getMousePicking() const
	{
		return m_mousePicking.get();
	}

} // namespace vk_app
