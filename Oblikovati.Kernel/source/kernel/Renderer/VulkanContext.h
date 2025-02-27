#pragma once

#include <vulkan/vulkan.h>

namespace Oblikovati::Kernel::VulkanRenderer
{

	class Window;

	class VulkanContext
	{
	public:
		struct QueueFamilyIndices
		{
			std::optional<uint32_t> graphicsFamily;
			std::optional<uint32_t> presentFamily;

			bool isComplete() const
			{
				return graphicsFamily.has_value() && presentFamily.has_value();
			}
		};

		struct SwapChainSupportDetails
		{
			VkSurfaceCapabilitiesKHR capabilities;
			std::vector<VkSurfaceFormatKHR> formats;
			std::vector<VkPresentModeKHR> presentModes;
		};

		VulkanContext(Window* window, bool enableValidation = true);
		~VulkanContext();

		DISABLE_COPY_AND_MOVE(VulkanContext);

		VkInstance getInstance() const { return m_instance; }
		VkDevice getDevice() const { return m_device; }
		VkPhysicalDevice getPhysicalDevice() const { return m_physicalDevice; }
		VkSurfaceKHR getSurface() const { return m_surface; }
		VkQueue getGraphicsQueue() const { return m_graphicsQueue; }
		VkQueue getPresentQueue() const { return m_presentQueue; }

		QueueFamilyIndices getQueueFamilyIndices() const { return m_queueFamilyIndices; }
		SwapChainSupportDetails getSwapChainSupport() const;

		VkPhysicalDeviceProperties getPhysicalDeviceProperties() const;
		VkPhysicalDeviceFeatures getPhysicalDeviceFeatures() const;
		VkPhysicalDeviceMemoryProperties getPhysicalDeviceMemoryProperties() const;

		bool isDeviceExtensionSupported(const char* extensionName) const;

		private:

		void createInstance();
		void setupDebugMessenger();
		void createSurface();
		void pickPhysicalDevice();
		void createLogicalDevice();

		bool checkValidationLayerSupport();
		std::vector<const char*> getRequiredExtensions();
		QueueFamilyIndices findQueueFamilies(VkPhysicalDevice device);
		bool isDeviceSuitable(VkPhysicalDevice device);
		bool checkDeviceExtensionSupport(VkPhysicalDevice device);
		SwapChainSupportDetails querySwapChainSupport(VkPhysicalDevice device);
		void populateDebugMessengerCreateInfo(VkDebugUtilsMessengerCreateInfoEXT& createInfo);
		int rateDeviceSuitability(VkPhysicalDevice device);
		SwapChainSupportDetails querySwapChainSupport(VkPhysicalDevice device) const;

		static VKAPI_ATTR VkBool32 VKAPI_CALL debugCallback(
			VkDebugUtilsMessageSeverityFlagBitsEXT messageSeverity,
			VkDebugUtilsMessageTypeFlagsEXT messageType,
			const VkDebugUtilsMessengerCallbackDataEXT* pCallbackData,
			void* pUserData
		);

		VkResult createDebugUtilsMessengerEXT(
			VkInstance instance,
			const VkDebugUtilsMessengerCreateInfoEXT* pCreateInfo,
			const VkAllocationCallbacks* pAllocator,
			VkDebugUtilsMessengerEXT* pDebugMessenger
		);

		void destroyDebugUtilsMessengerEXT(
			VkInstance instance,
			VkDebugUtilsMessengerEXT debugMessenger,
			const VkAllocationCallbacks* pAllocator
		);

		const std::vector<const char*> m_validationLayers = {
			"VK_LAYER_KHRONOS_validation"
		};

		const std::vector<const char*> m_deviceExtensions = {
			VK_KHR_SWAPCHAIN_EXTENSION_NAME
		};

		VkInstance m_instance;
		VkDebugUtilsMessengerEXT m_debugMessenger;
		VkSurfaceKHR m_surface;
		VkPhysicalDevice m_physicalDevice = VK_NULL_HANDLE;
		VkDevice m_device;

		VkQueue m_graphicsQueue;
		VkQueue m_presentQueue;

		QueueFamilyIndices m_queueFamilyIndices;

		Window* m_window;

		bool m_enableValidation;
	};

}
