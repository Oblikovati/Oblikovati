#pragma once

#include "VulkanContext.h"
#include "Window.h"

#include <vulkan/vulkan.h>

namespace Oblikovati::Kernel::VulkanRenderer
{

	class Swapchain
	{
		public:
		Swapchain(VulkanContext* context, Window* window);
		~Swapchain();

		DISABLE_COPY_AND_MOVE(Swapchain);

		void recreate();

		VkSwapchainKHR getSwapchain() const { return m_swapchain; }
		const std::vector<VkImage>& getImages() const { return m_images; }
		const std::vector<VkImageView>& getImageViews() const { return m_imageViews; }
		VkFormat getImageFormat() const { return m_imageFormat; }
		VkExtent2D getExtent() const { return m_extent; }

		void createFramebuffers(VkRenderPass renderPass);
		void destroyFramebuffers();

		const std::vector<VkFramebuffer>& getFramebuffers() const { return m_framebuffers; }

		void createDepthResources();
		VkImageView getDepthImageView() const { return m_depthImageView; }
		VkFormat getDepthFormat() const { return m_depthFormat; }

		VkResult acquireNextImage(uint32_t* imageIndex, VkSemaphore semaphore, VkFence fence, uint64_t timeout = UINT64_MAX);

		VkResult presentImage(uint32_t imageIndex, VkSemaphore waitSemaphore);

		private:
		void init();
		void createSwapchain();
		void createImageViews();

		void cleanup();

		VkSurfaceFormatKHR chooseSwapSurfaceFormat(const std::vector<VkSurfaceFormatKHR>& availableFormats);
		VkPresentModeKHR chooseSwapPresentMode(const std::vector<VkPresentModeKHR>& availablePresentModes);
		VkExtent2D chooseSwapExtent(const VkSurfaceCapabilitiesKHR& capabilities);

		VkFormat findDepthFormat();
		VkFormat findSupportedFormat(
			const std::vector<VkFormat>& candidates,
			VkImageTiling tiling,
			VkFormatFeatureFlags features
		);

		void createImage(
			uint32_t width,
			uint32_t height,
			VkFormat format,
			VkImageTiling tiling,
			VkImageUsageFlags usage,
			VkMemoryPropertyFlags properties,
			VkImage& image,
			VkDeviceMemory& imageMemory
		);

		VkImageView createImageView(
			VkImage image,
			VkFormat format,
			VkImageAspectFlags aspectFlags
		);

		VulkanContext* m_context;
		Window* m_window;

		VkSwapchainKHR m_swapchain;
		std::vector<VkImage> m_images;
		std::vector<VkImageView> m_imageViews;
		std::vector<VkFramebuffer> m_framebuffers;

		VkImage m_depthImage;
		VkDeviceMemory m_depthImageMemory;
		VkImageView m_depthImageView;
		VkFormat m_depthFormat;

		VkFormat m_imageFormat;
		VkExtent2D m_extent;

		VkDevice m_device;
		VkPhysicalDevice m_physicalDevice;
		VkQueue m_presentQueue;
	};
}
