#include "KernelPCH.h"

#include "camera.h"
#include <glm/gtc/matrix_transform.hpp>
#include <glm/gtx/quaternion.hpp>

namespace Oblikovati::Kernel::VulkanRenderer
{
	Camera::Camera(float fov, float aspectRatio, float nearPlane, float farPlane, ProjectionType projType)
		: m_fov(fov), m_aspectRatio(aspectRatio), m_nearPlane(nearPlane), m_farPlane(farPlane), m_projectionType(projType)
	{

		updateVectors();
		updateMatrices();
	}

	void Camera::updateMatrices()
	{
		// Update view matrix
		m_viewMatrix = glm::lookAt(m_position, m_position + m_front, m_up);

		// Update projection matrix based on projection type
		if (m_projectionType == ProjectionType::Perspective)
		{
			// Note: Vulkan uses different clip space than OpenGL, so we need to flip Y
			m_projectionMatrix = glm::perspective(glm::radians(m_fov), m_aspectRatio, m_nearPlane, m_farPlane);
			m_projectionMatrix[1][1] *= -1;  // Flip Y for Vulkan
		}
		else
		{
			// Orthographic projection
			m_projectionMatrix = glm::ortho(m_left, m_rightOrtho, m_bottom, m_top, m_nearPlane, m_farPlane);
			m_projectionMatrix[1][1] *= -1;  // Flip Y for Vulkan
		}

		// Precompute view-projection matrix
		m_viewProjectionMatrix = m_projectionMatrix * m_viewMatrix;
	}

	void Camera::updateVectors()
	{
		// Calculate front vector from Euler angles (pitch, yaw)
		glm::vec3 front;
		front.x = cos(glm::radians(m_yaw)) * cos(glm::radians(m_pitch));
		front.y = sin(glm::radians(m_pitch));
		front.z = sin(glm::radians(m_yaw)) * cos(glm::radians(m_pitch));
		m_front = glm::normalize(front);

		// Recalculate right and up vectors
		m_right = glm::normalize(glm::cross(m_front, m_worldUp));
		m_up = glm::normalize(glm::cross(m_right, m_front));

		// Also update target position based on front vector
		m_target = m_position + m_front;
	}

	void Camera::setPosition(const glm::vec3& position)
	{
		m_position = position;
		m_target = m_position + m_front;
		updateMatrices();
	}

	void Camera::setTarget(const glm::vec3& target)
	{
		m_target = target;
		m_front = glm::normalize(m_target - m_position);

		// Calculate pitch and yaw from front vector
		m_pitch = glm::degrees(asin(m_front.y));
		m_yaw = glm::degrees(atan2(m_front.z, m_front.x));

		updateVectors();
		updateMatrices();
	}

	void Camera::setOrientation(float pitch, float yaw, float roll)
	{
		m_pitch = std::clamp(pitch, -89.0f, 89.0f);  // Prevent gimbal lock
		m_yaw = yaw;
		m_roll = roll;

		updateVectors();
		updateMatrices();
	}

	void Camera::move(const glm::vec3& offset)
	{
		m_position += offset.x * m_right * m_movementSpeed;
		m_position += offset.y * m_up * m_movementSpeed;
		m_position += offset.z * m_front * m_movementSpeed;

		m_target = m_position + m_front;
		updateMatrices();
	}

	void Camera::rotate(float deltaPitch, float deltaYaw, float deltaRoll)
	{
		m_pitch += deltaPitch * m_mouseSensitivity;
		m_yaw += deltaYaw * m_mouseSensitivity;
		m_roll += deltaRoll * m_mouseSensitivity;

		// Constrain pitch to avoid gimbal lock
		m_pitch = std::clamp(m_pitch, -89.0f, 89.0f);

		// Keep yaw in range [0, 360]
		if (m_yaw > 360.0f) m_yaw -= 360.0f;
		else if (m_yaw < 0.0f) m_yaw += 360.0f;

		updateVectors();
		updateMatrices();
	}

	void Camera::zoom(float deltaFov)
	{
		if (m_projectionType == ProjectionType::Perspective)
		{
			m_fov -= deltaFov * m_zoomSensitivity;
			m_fov = std::clamp(m_fov, 1.0f, 120.0f);

			updateMatrices();
		}
	}

	void Camera::setProjectionType(ProjectionType type)
	{
		m_projectionType = type;
		updateMatrices();
	}

	void Camera::setPerspectiveProjection(float fov, float aspectRatio, float nearPlane, float farPlane)
	{
		m_projectionType = ProjectionType::Perspective;
		m_fov = fov;
		m_aspectRatio = aspectRatio;
		m_nearPlane = nearPlane;
		m_farPlane = farPlane;

		updateMatrices();
	}

	void Camera::setOrthographicProjection(float left, float right, float bottom, float top, float nearPlane, float farPlane)
	{
		m_projectionType = ProjectionType::Orthographic;
		m_left = left;
		m_rightOrtho = right;
		m_bottom = bottom;
		m_top = top;
		m_nearPlane = nearPlane;
		m_farPlane = farPlane;

		updateMatrices();
	}

	void Camera::setAspectRatio(float aspectRatio)
	{
		m_aspectRatio = aspectRatio;
		updateMatrices();
	}

}
