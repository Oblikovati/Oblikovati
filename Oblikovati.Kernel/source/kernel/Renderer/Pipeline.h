#pragma once

#include <vulkan/vulkan.h>
#include <string>
#include <vector>
#include <memory>

namespace Oblikovati::Kernel::VulkanRenderer
{
	class ShaderModule;
	class Pipeline
	{
	public:
		Pipeline(VulkanContext* context, VkExtent2D swapchainExtent, VkFormat swapchainFormat, VkFormat depthFormat);
		~Pipeline();

		DISABLE_COPY_AND_MOVE(Pipeline);

		VkPipeline getPipeline() const { return m_graphicsPipeline; }
		VkRenderPass getRenderPass() const { return m_renderPass; }
		VkPipelineLayout getPipelineLayout() const { return m_pipelineLayout; }

		VkDescriptorSetLayout getDescriptorSetLayout() const { return m_descriptorSetLayout; }

		void beginRenderPass(VkCommandBuffer commandBuffer, VkFramebuffer framebuffer, VkExtent2D extent);
		void endRenderPass(VkCommandBuffer commandBuffer);
		void bindPipeline(VkCommandBuffer commandBuffer);

	private:
		void createRenderPass();
		void createDescriptorSetLayout();
		void createPipelineLayout();
		void createGraphicsPipeline();

		void cleanup();

		std::vector<char> readShaderFile(const std::string& filename);
		VkShaderModule createShaderModule(const std::vector<char>& code);

		VulkanContext* m_context;

		VkPipelineLayout m_pipelineLayout;
		VkRenderPass m_renderPass;
		VkPipeline m_graphicsPipeline;

		VkDescriptorSetLayout m_descriptorSetLayout;

		std::unique_ptr<ShaderModule> m_vertShaderModule;
		std::unique_ptr<ShaderModule> m_fragShaderModule;

		VkExtent2D m_swapchainExtent;
		VkFormat m_swapchainFormat;
		VkFormat m_depthFormat;

		VkDevice m_device;
	};
}
