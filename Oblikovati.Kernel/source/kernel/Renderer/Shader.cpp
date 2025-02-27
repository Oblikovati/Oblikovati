#include "KernelPCH.h"

#include "Shader.h"


namespace Oblikovati::Kernel::VulkanRenderer
{

	ShaderModule::ShaderModule(VkDevice device, const std::string& filename, Type type)
		: m_device(device), m_entryPoint("main")
	{

		// Set shader stage based on type
		m_shaderStage = shaderTypeToVkShaderStage(type);

		// Read shader file
		std::vector<char> code = readFile(filename);

		// Create shader module
		createShaderModule(code);

		//Logger::debug("Shader module created: " + filename);
	}

	ShaderModule::~ShaderModule()
	{
		if (m_device != VK_NULL_HANDLE && m_shaderModule != VK_NULL_HANDLE)
		{
			vkDestroyShaderModule(m_device, m_shaderModule, nullptr);
			m_shaderModule = VK_NULL_HANDLE;
		}
	}

	std::vector<char> ShaderModule::readFile(const std::string& filename)
	{
		// Open file with cursor at the end
		std::ifstream file(filename, std::ios::ate | std::ios::binary);

		if (!file.is_open())
		{
			throw std::runtime_error("Failed to open shader file: " + filename);
		}

		// Get file size and allocate buffer
		size_t fileSize = (size_t)file.tellg();
		std::vector<char> buffer(fileSize);

		// Go back to beginning of file and read data
		file.seekg(0);
		file.read(buffer.data(), fileSize);

		file.close();

		//Logger::debug("Read shader file: " + filename + " (" + std::to_string(fileSize) + " bytes)");
		return buffer;
	}

	void ShaderModule::createShaderModule(const std::vector<char>& code)
	{
		VkShaderModuleCreateInfo createInfo{};
		createInfo.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO;
		createInfo.codeSize = code.size();
		createInfo.pCode = reinterpret_cast<const uint32_t*>(code.data());

		if (vkCreateShaderModule(m_device, &createInfo, nullptr, &m_shaderModule) != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create shader module");
		}
	}

	VkPipelineShaderStageCreateInfo ShaderModule::getPipelineShaderStageCreateInfo() const
	{
		VkPipelineShaderStageCreateInfo shaderStageInfo{};
		shaderStageInfo.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
		shaderStageInfo.stage = m_shaderStage;
		shaderStageInfo.module = m_shaderModule;
		shaderStageInfo.pName = m_entryPoint;

		return shaderStageInfo;
	}

	VkShaderStageFlagBits ShaderModule::shaderTypeToVkShaderStage(Type type)
	{
		switch (type)
		{
			case Type::Vertex:
				return VK_SHADER_STAGE_VERTEX_BIT;
			case Type::Fragment:
				return VK_SHADER_STAGE_FRAGMENT_BIT;
			case Type::Geometry:
				return VK_SHADER_STAGE_GEOMETRY_BIT;
			case Type::Compute:
				return VK_SHADER_STAGE_COMPUTE_BIT;
			case Type::TessControl:
				return VK_SHADER_STAGE_TESSELLATION_CONTROL_BIT;
			case Type::TessEvaluation:
				return VK_SHADER_STAGE_TESSELLATION_EVALUATION_BIT;
			default:
				throw std::runtime_error("Unknown shader type");
		}
	}

}
