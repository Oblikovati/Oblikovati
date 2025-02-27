#pragma once

#include <glm/glm.hpp>
#include <glm/gtc/matrix_transform.hpp>

namespace Oblikovati::Kernel::VulkanRenderer
{
	class Camera
	{
	public:
		enum class ProjectionType
		{
			Perspective,
			Orthographic
		};

		Camera(
			float fov = 45.0f,
			float aspectRatio = 16.0f / 9.0f,
			float nearPlane = 0.1f,
			float farPlane = 100.0f,
			ProjectionType projType = ProjectionType::Perspective
		);

		~Camera() = default;

		// Update camera matrices based on current parameters
		void updateMatrices();

		// Set camera position and orientation
		void setPosition(const glm::vec3& position);
		void setTarget(const glm::vec3& target);
		void setOrientation(float pitch, float yaw, float roll = 0.0f);

		// Movement functions
		void move(const glm::vec3& offset);
		void rotate(float deltaPitch, float deltaYaw, float deltaRoll = 0.0f);
		void zoom(float deltaFov);

		// Projection parameters
		void setProjectionType(ProjectionType type);
		void setPerspectiveProjection(float fov, float aspectRatio, float nearPlane, float farPlane);
		void setOrthographicProjection(float left, float right, float bottom, float top, float nearPlane, float farPlane);
		void setAspectRatio(float aspectRatio);

		// Getters
		const glm::mat4& getViewMatrix() const { return m_viewMatrix; }
		const glm::mat4& getProjectionMatrix() const { return m_projectionMatrix; }
		const glm::mat4& getViewProjectionMatrix() const { return m_viewProjectionMatrix; }

		const glm::vec3& getPosition() const { return m_position; }
		const glm::vec3& getTarget() const { return m_target; }
		const glm::vec3& getUp() const { return m_up; }
		const glm::vec3& getRight() const { return m_right; }
		const glm::vec3& getFront() const { return m_front; }

		float getPitch() const { return m_pitch; }
		float getYaw() const { return m_yaw; }
		float getRoll() const { return m_roll; }
		float getFov() const { return m_fov; }

	private:
		// Update view vectors based on orientation
		void updateVectors();

		// Camera position and orientation
		glm::vec3 m_position = glm::vec3(0.0f, 0.0f, 5.0f);
		glm::vec3 m_target = glm::vec3(0.0f, 0.0f, 0.0f);

		// Camera orientation (in degrees)
		float m_pitch = 0.0f;  // X-rotation (up/down)
		float m_yaw = -90.0f;  // Y-rotation (left/right)
		float m_roll = 0.0f;   // Z-rotation (roll)

		// Camera coordinate system
		glm::vec3 m_front = glm::vec3(0.0f, 0.0f, -1.0f);
		glm::vec3 m_up = glm::vec3(0.0f, 1.0f, 0.0f);
		glm::vec3 m_right = glm::vec3(1.0f, 0.0f, 0.0f);
		glm::vec3 m_worldUp = glm::vec3(0.0f, 1.0f, 0.0f);

		// Perspective parameters
		float m_fov;
		float m_aspectRatio;
		float m_nearPlane;
		float m_farPlane;

		// Orthographic parameters
		float m_left = -10.0f;
		float m_rightOrtho = 10.0f;
		float m_bottom = -10.0f;
		float m_top = 10.0f;

		// Projection type
		ProjectionType m_projectionType;

		// Matrices
		glm::mat4 m_viewMatrix = glm::mat4(1.0f);
		glm::mat4 m_projectionMatrix = glm::mat4(1.0f);
		glm::mat4 m_viewProjectionMatrix = glm::mat4(1.0f);

		// For smooth camera movement
		float m_movementSpeed = 2.5f;
		float m_mouseSensitivity = 0.1f;
		float m_zoomSensitivity = 1.0f;
	};

}
