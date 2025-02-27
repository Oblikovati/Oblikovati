#pragma once

#include "mesh.h"
#include <string>
#include <vector>
#include <memory>
#include <glm/vec3.hpp>
#include <glm/ext/matrix_float4x4.hpp>

namespace Oblikovati::Kernel::VulkanRenderer
{
	struct Vertex;

	class Model
	{
	public:
		Model() = default;
		~Model() = default;

		DISABLE_COPY_AND_MOVE(Model);

		// Load model from file
		bool loadFromFile(const std::string& path);

		// Load model from raw data
		bool loadFromData(
			const std::vector<Vertex>& vertices,
			const std::vector<uint32_t>& indices,
			const std::vector<glm::vec3>& normals
		);

		// Getters
		const std::vector<std::unique_ptr<Mesh>>& getMeshes() const { return m_meshes; }

		// Model transformation
		const glm::mat4& getTransform() const { return m_transform; }
		void setTransform(const glm::mat4& transform) { m_transform = transform; }

		void setPosition(const glm::vec3& position);
		void setRotation(const glm::vec3& rotation);
		void setScale(const glm::vec3& scale);

	private:
		std::vector<std::unique_ptr<Mesh>> m_meshes;
		glm::mat4 m_transform = glm::mat4(1.0f);

		glm::vec3 m_position = glm::vec3(0.0f);
		glm::vec3 m_rotation = glm::vec3(0.0f);
		glm::vec3 m_scale = glm::vec3(1.0f);

		void updateTransform();
	};
}
