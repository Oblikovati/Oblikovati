#pragma once

#include <vulkan/vulkan.h>
#include <string>
#include <vector>

namespace Oblikovati::Kernel::VulkanRenderer
{

	class ShaderModule
	{
	public:
		enum class Type
		{
			Vertex,
			Fragment,
			Geometry,
			Compute,
			TessControl,
			TessEvaluation
		};

		ShaderModule(VkDevice device, const std::string& filename, Type type);
		~ShaderModule();

		// Prevent copying
		ShaderModule(const ShaderModule&) = delete;
		ShaderModule& operator=(const ShaderModule&) = delete;

		// Getters
		VkShaderModule getShaderModule() const { return m_shaderModule; }
		VkShaderStageFlagBits getShaderStage() const { return m_shaderStage; }

		// Create pipeline shader stage info
		VkPipelineShaderStageCreateInfo getPipelineShaderStageCreateInfo() const;

	private:
		// Load shader from file
		std::vector<char> readFile(const std::string& filename);

		// Create shader module
		void createShaderModule(const std::vector<char>& code);

		// Convert shader type to VkShaderStageFlagBits
		static VkShaderStageFlagBits shaderTypeToVkShaderStage(Type type);

		// Device reference (not owned)
		VkDevice m_device;

		// Shader module
		VkShaderModule m_shaderModule;

		// Shader stage
		VkShaderStageFlagBits m_shaderStage;

		// Entry point
		const char* m_entryPoint = "main";
	};

}
