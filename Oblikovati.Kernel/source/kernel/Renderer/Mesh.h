#pragma once

#include "vertex.h"
#include <vulkan/vulkan.h>
#include <vector>
#include <memory>

namespace Oblikovati::Kernel::VulkanRenderer
{
	class Buffer;
	class Renderer;

	class Mesh
	{
		public:
		Mesh(
			const std::vector<Vertex>& vertices,
			const std::vector<uint32_t>& indices,
			const std::vector<glm::vec3>& normals
		);
		~Mesh();

		DISABLE_COPY_AND_MOVE(Mesh);

		void createBuffers(VkDevice device, VkPhysicalDevice physicalDevice, VkQueue transferQueue, VkCommandPool commandPool);

		void cleanup();

		void bind(VkCommandBuffer commandBuffer);

		void draw(VkCommandBuffer commandBuffer);

		const std::vector<Vertex>& getVertices() const { return m_vertices; }
		const std::vector<uint32_t>& getIndices() const { return m_indices; }
		const std::vector<glm::vec3>& getNormals() const { return m_normals; }

		uint32_t getVertexCount() const { return static_cast<uint32_t>(m_vertices.size()); }
		uint32_t getIndexCount() const { return static_cast<uint32_t>(m_indices.size()); }

		private:
		std::vector<Vertex> m_vertices;
		std::vector<uint32_t> m_indices;
		std::vector<glm::vec3> m_normals;

		std::unique_ptr<Buffer> m_vertexBuffer;
		std::unique_ptr<Buffer> m_indexBuffer;

		bool m_initialized = false;
		VkDevice m_device = VK_NULL_HANDLE;
	};
}
