#include "KernelPCH.h"

#include "Swapchain.h"

namespace Oblikovati::Kernel::VulkanRenderer
{

	Swapchain::Swapchain(VulkanContext* context, Window* window)
		: m_context(context),
		m_window(window),
		m_device(context->getDevice()),
		m_physicalDevice(context->getPhysicalDevice()),
		m_presentQueue(context->getPresentQueue())
	{
		init();
	}

	Swapchain::~Swapchain()
	{
		cleanup();
	}

	void Swapchain::init()
	{
		createSwapchain();
		createImageViews();
		m_depthFormat = findDepthFormat();
		createDepthResources();
	}

	void Swapchain::createSwapchain()
	{
		VulkanContext::SwapChainSupportDetails swapChainSupport = m_context->getSwapChainSupport();

		VkSurfaceFormatKHR surfaceFormat = chooseSwapSurfaceFormat(swapChainSupport.formats);
		VkPresentModeKHR presentMode = chooseSwapPresentMode(swapChainSupport.presentModes);
		VkExtent2D extent = chooseSwapExtent(swapChainSupport.capabilities);

		// Store format and extent for later use
		m_imageFormat = surfaceFormat.format;
		m_extent = extent;

		// Determine how many images to use in the swap chain
		uint32_t imageCount = swapChainSupport.capabilities.minImageCount + 1;

		// Make sure not to exceed maximum image count
		if (swapChainSupport.capabilities.maxImageCount > 0 && imageCount > swapChainSupport.capabilities.maxImageCount)
		{
			imageCount = swapChainSupport.capabilities.maxImageCount;
		}

		// Create swap chain
		VkSwapchainCreateInfoKHR createInfo{};
		createInfo.sType = VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR;
		createInfo.surface = m_context->getSurface();
		createInfo.minImageCount = imageCount;
		createInfo.imageFormat = surfaceFormat.format;
		createInfo.imageColorSpace = surfaceFormat.colorSpace;
		createInfo.imageExtent = extent;
		createInfo.imageArrayLayers = 1;
		createInfo.imageUsage = VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT;

		// Handle queue families
		VulkanContext::QueueFamilyIndices indices = m_context->getQueueFamilyIndices();
		uint32_t queueFamilyIndices[] = { indices.graphicsFamily.value(), indices.presentFamily.value() };

		if (indices.graphicsFamily != indices.presentFamily)
		{
			createInfo.imageSharingMode = VK_SHARING_MODE_CONCURRENT;
			createInfo.queueFamilyIndexCount = 2;
			createInfo.pQueueFamilyIndices = queueFamilyIndices;
		}
		else
		{
			createInfo.imageSharingMode = VK_SHARING_MODE_EXCLUSIVE;
			createInfo.queueFamilyIndexCount = 0;
			createInfo.pQueueFamilyIndices = nullptr;
		}

		createInfo.preTransform = swapChainSupport.capabilities.currentTransform;
		createInfo.compositeAlpha = VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR;
		createInfo.presentMode = presentMode;
		createInfo.clipped = VK_TRUE;
		createInfo.oldSwapchain = VK_NULL_HANDLE;

		if (vkCreateSwapchainKHR(m_device, &createInfo, nullptr, &m_swapchain) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create swap chain!");
		}

		// Get swap chain images
		vkGetSwapchainImagesKHR(m_device, m_swapchain, &imageCount, nullptr);
		m_images.resize(imageCount);
		vkGetSwapchainImagesKHR(m_device, m_swapchain, &imageCount, m_images.data());

		//Logger::info("Swapchain created with " + std::to_string(imageCount) + " images");
	}

	void Swapchain::createImageViews()
	{
		m_imageViews.resize(m_images.size());

		for (size_t i = 0; i < m_images.size(); i++)
		{
			VkImageViewCreateInfo createInfo{};
			createInfo.sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO;
			createInfo.image = m_images[i];
			createInfo.viewType = VK_IMAGE_VIEW_TYPE_2D;
			createInfo.format = m_imageFormat;
			createInfo.components.r = VK_COMPONENT_SWIZZLE_IDENTITY;
			createInfo.components.g = VK_COMPONENT_SWIZZLE_IDENTITY;
			createInfo.components.b = VK_COMPONENT_SWIZZLE_IDENTITY;
			createInfo.components.a = VK_COMPONENT_SWIZZLE_IDENTITY;
			createInfo.subresourceRange.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT;
			createInfo.subresourceRange.baseMipLevel = 0;
			createInfo.subresourceRange.levelCount = 1;
			createInfo.subresourceRange.baseArrayLayer = 0;
			createInfo.subresourceRange.layerCount = 1;

			if (vkCreateImageView(m_device, &createInfo, nullptr, &m_imageViews[i]) != VK_SUCCESS)
			{
				throw std::runtime_error("Failed to create image views!");
			}
		}

		//Logger::debug("Swapchain image views created");
	}

	void Swapchain::createFramebuffers(VkRenderPass renderPass)
	{
		m_framebuffers.resize(m_imageViews.size());

		for (size_t i = 0; i < m_imageViews.size(); i++)
		{
			std::array<VkImageView, 2> attachments = {
				m_imageViews[i],
				m_depthImageView
			};

			VkFramebufferCreateInfo framebufferInfo{};
			framebufferInfo.sType = VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO;
			framebufferInfo.renderPass = renderPass;
			framebufferInfo.attachmentCount = static_cast<uint32_t>(attachments.size());
			framebufferInfo.pAttachments = attachments.data();
			framebufferInfo.width = m_extent.width;
			framebufferInfo.height = m_extent.height;
			framebufferInfo.layers = 1;

			if (vkCreateFramebuffer(m_device, &framebufferInfo, nullptr, &m_framebuffers[i]) != VK_SUCCESS)
			{
				throw std::runtime_error("Failed to create framebuffer!");
			}
		}

		//Logger::debug("Swapchain framebuffers created");
	}

	void Swapchain::destroyFramebuffers()
	{
		for (auto framebuffer : m_framebuffers)
		{
			vkDestroyFramebuffer(m_device, framebuffer, nullptr);
		}
		m_framebuffers.clear();
	}

	void Swapchain::createDepthResources()
	{
		VkImageCreateInfo imageInfo{};
		imageInfo.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
		imageInfo.imageType = VK_IMAGE_TYPE_2D;
		imageInfo.extent.width = m_extent.width;
		imageInfo.extent.height = m_extent.height;
		imageInfo.extent.depth = 1;
		imageInfo.mipLevels = 1;
		imageInfo.arrayLayers = 1;
		imageInfo.format = m_depthFormat;
		imageInfo.tiling = VK_IMAGE_TILING_OPTIMAL;
		imageInfo.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
		imageInfo.usage = VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT;
		imageInfo.samples = VK_SAMPLE_COUNT_1_BIT;
		imageInfo.sharingMode = VK_SHARING_MODE_EXCLUSIVE;

		createImage(
			m_extent.width,
			m_extent.height,
			m_depthFormat,
			VK_IMAGE_TILING_OPTIMAL,
			VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT,
			VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
			m_depthImage,
			m_depthImageMemory
		);

		m_depthImageView = createImageView(m_depthImage, m_depthFormat, VK_IMAGE_ASPECT_DEPTH_BIT);

		//Logger::debug("Depth resources created");
	}

	void Swapchain::recreate()
	{
		// Wait until window is not minimized
		int width = 0, height = 0;
		while (width == 0 || height == 0)
		{
			width = m_window->getWidth();
			height = m_window->getHeight();
			glfwWaitEvents();
		}

		// Wait for the device to be idle before recreating resources
		vkDeviceWaitIdle(m_device);

		// Clean up old swapchain resources
		cleanup();

		// Create new swapchain resources
		init();

		//Logger::info("Swapchain recreated");
	}

	VkSurfaceFormatKHR Swapchain::chooseSwapSurfaceFormat(const std::vector<VkSurfaceFormatKHR>& availableFormats)
	{
		// Preferred format: SRGB with 8 bits per channel
		for (const auto& availableFormat : availableFormats)
		{
			if (availableFormat.format == VK_FORMAT_B8G8R8A8_SRGB &&
				availableFormat.colorSpace == VK_COLOR_SPACE_SRGB_NONLINEAR_KHR)
			{
				return availableFormat;
			}
		}

		// If preferred format is not available, just return the first format
		return availableFormats[0];
	}

	VkPresentModeKHR Swapchain::chooseSwapPresentMode(const std::vector<VkPresentModeKHR>& availablePresentModes)
	{
		// Preferred mode: Mailbox (triple buffering)
		for (const auto& availablePresentMode : availablePresentModes)
		{
			if (availablePresentMode == VK_PRESENT_MODE_MAILBOX_KHR)
			{
				return availablePresentMode;
			}
		}

		// Fallback to FIFO (V-Sync) which is guaranteed to be available
		return VK_PRESENT_MODE_FIFO_KHR;
	}

	VkExtent2D Swapchain::chooseSwapExtent(const VkSurfaceCapabilitiesKHR& capabilities)
	{
		if (capabilities.currentExtent.width != std::numeric_limits<uint32_t>::max())
		{
			return capabilities.currentExtent;
		}
		else
		{
			int width = m_window->getWidth();
			int height = m_window->getHeight();

			VkExtent2D actualExtent = {
				static_cast<uint32_t>(width),
				static_cast<uint32_t>(height)
			};

			actualExtent.width = std::clamp(actualExtent.width,
				capabilities.minImageExtent.width,
				capabilities.maxImageExtent.width);

			actualExtent.height = std::clamp(actualExtent.height,
				capabilities.minImageExtent.height,
				capabilities.maxImageExtent.height);

			return actualExtent;
		}
	}

	VkFormat Swapchain::findDepthFormat()
	{
		return findSupportedFormat(
			{ VK_FORMAT_D32_SFLOAT, VK_FORMAT_D32_SFLOAT_S8_UINT, VK_FORMAT_D24_UNORM_S8_UINT },
			VK_IMAGE_TILING_OPTIMAL,
			VK_FORMAT_FEATURE_DEPTH_STENCIL_ATTACHMENT_BIT
		);
	}

	VkFormat Swapchain::findSupportedFormat(const std::vector<VkFormat>& candidates, VkImageTiling tiling, VkFormatFeatureFlags features)
	{
		for (VkFormat format : candidates)
		{
			VkFormatProperties props;
			vkGetPhysicalDeviceFormatProperties(m_physicalDevice, format, &props);

			if (tiling == VK_IMAGE_TILING_LINEAR && (props.linearTilingFeatures & features) == features)
			{
				return format;
			}
			else if (tiling == VK_IMAGE_TILING_OPTIMAL && (props.optimalTilingFeatures & features) == features)
			{
				return format;
			}
		}

		throw std::runtime_error("Failed to find supported format!");
	}

	void Swapchain::createImage(uint32_t width, uint32_t height, VkFormat format, VkImageTiling tiling,
							   VkImageUsageFlags usage, VkMemoryPropertyFlags properties,
							   VkImage& image, VkDeviceMemory& imageMemory)
	{
		VkImageCreateInfo imageInfo{};
		imageInfo.sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
		imageInfo.imageType = VK_IMAGE_TYPE_2D;
		imageInfo.extent.width = width;
		imageInfo.extent.height = height;
		imageInfo.extent.depth = 1;
		imageInfo.mipLevels = 1;
		imageInfo.arrayLayers = 1;
		imageInfo.format = format;
		imageInfo.tiling = tiling;
		imageInfo.initialLayout = VK_IMAGE_LAYOUT_UNDEFINED;
		imageInfo.usage = usage;
		imageInfo.samples = VK_SAMPLE_COUNT_1_BIT;
		imageInfo.sharingMode = VK_SHARING_MODE_EXCLUSIVE;

		if (vkCreateImage(m_device, &imageInfo, nullptr, &image) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create image!");
		}

		VkMemoryRequirements memRequirements;
		vkGetImageMemoryRequirements(m_device, image, &memRequirements);

		VkMemoryAllocateInfo allocInfo{};
		allocInfo.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
		allocInfo.allocationSize = memRequirements.size;

		// Find memory type that satisfies requirements
		VkPhysicalDeviceMemoryProperties memProperties;
		vkGetPhysicalDeviceMemoryProperties(m_physicalDevice, &memProperties);

		uint32_t memoryTypeIndex = 0;
		for (uint32_t i = 0; i < memProperties.memoryTypeCount; i++)
		{
			if ((memRequirements.memoryTypeBits & (1 << i)) &&
				(memProperties.memoryTypes[i].propertyFlags & properties) == properties)
			{
				memoryTypeIndex = i;
				break;
			}
		}

		allocInfo.memoryTypeIndex = memoryTypeIndex;

		if (vkAllocateMemory(m_device, &allocInfo, nullptr, &imageMemory) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to allocate image memory!");
		}

		vkBindImageMemory(m_device, image, imageMemory, 0);
	}

	VkImageView Swapchain::createImageView(VkImage image, VkFormat format, VkImageAspectFlags aspectFlags)
	{
		VkImageViewCreateInfo viewInfo{};
		viewInfo.sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO;
		viewInfo.image = image;
		viewInfo.viewType = VK_IMAGE_VIEW_TYPE_2D;
		viewInfo.format = format;
		viewInfo.subresourceRange.aspectMask = aspectFlags;
		viewInfo.subresourceRange.baseMipLevel = 0;
		viewInfo.subresourceRange.levelCount = 1;
		viewInfo.subresourceRange.baseArrayLayer = 0;
		viewInfo.subresourceRange.layerCount = 1;

		VkImageView imageView;
		if (vkCreateImageView(m_device, &viewInfo, nullptr, &imageView) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create image view!");
		}

		return imageView;
	}

	VkResult Swapchain::acquireNextImage(uint32_t* imageIndex, VkSemaphore semaphore, VkFence fence, uint64_t timeout)
	{
		return vkAcquireNextImageKHR(m_device, m_swapchain, timeout, semaphore, fence, imageIndex);
	}

	VkResult Swapchain::presentImage(uint32_t imageIndex, VkSemaphore waitSemaphore)
	{
		VkPresentInfoKHR presentInfo{};
		presentInfo.sType = VK_STRUCTURE_TYPE_PRESENT_INFO_KHR;
		presentInfo.waitSemaphoreCount = 1;
		presentInfo.pWaitSemaphores = &waitSemaphore;

		VkSwapchainKHR swapChains[] = { m_swapchain };
		presentInfo.swapchainCount = 1;
		presentInfo.pSwapchains = swapChains;
		presentInfo.pImageIndices = &imageIndex;
		presentInfo.pResults = nullptr;

		return vkQueuePresentKHR(m_presentQueue, &presentInfo);
	}

	void Swapchain::cleanup()
	{
		// Destroy framebuffers
		destroyFramebuffers();

		// Destroy depth resources
		vkDestroyImageView(m_device, m_depthImageView, nullptr);
		vkDestroyImage(m_device, m_depthImage, nullptr);
		vkFreeMemory(m_device, m_depthImageMemory, nullptr);

		// Destroy image views
		for (auto imageView : m_imageViews)
		{
			vkDestroyImageView(m_device, imageView, nullptr);
		}

		// Destroy swapchain
		vkDestroySwapchainKHR(m_device, m_swapchain, nullptr);

		// Clear vectors
		m_images.clear();
		m_imageViews.clear();
	}
}
