#include "KernelPCH.h"

#include "Window.h"
//#include "../utils/logger.h"

namespace Oblikovati::Kernel::VulkanRenderer
{

	Window::Window(int width, int height, const std::string& title)
		: m_width(width), m_height(height), m_title(title)
	{
		if (!glfwInit())
		{
			throw std::runtime_error("Failed to initialize GLFW");
		}

		glfwWindowHint(GLFW_CLIENT_API, GLFW_NO_API);
		glfwWindowHint(GLFW_RESIZABLE, GLFW_TRUE);

		m_window = glfwCreateWindow(width, height, title.c_str(), nullptr, nullptr);
		if (!m_window)
		{
			glfwTerminate();
			throw std::runtime_error("Failed to create GLFW window");
		}

		glfwSetWindowUserPointer(m_window, this);

		glfwSetFramebufferSizeCallback(m_window, framebufferResizeCallback);
		glfwSetKeyCallback(m_window, keyCallback);
		glfwSetMouseButtonCallback(m_window, mouseButtonCallback);
		glfwSetCursorPosCallback(m_window, cursorPosCallback);

		//Logger::info("Window created: " + title + " (" + std::to_string(width) + "x" + std::to_string(height) + ")");
	}

	Window::~Window()
	{
		if (m_window)
		{
			glfwDestroyWindow(m_window);
			m_window = nullptr;
		}

		glfwTerminate();
		//Logger::info("Window destroyed");
	}

	bool Window::shouldClose() const
	{
		return glfwWindowShouldClose(m_window);
	}

	void Window::pollEvents()
	{
		glfwPollEvents();
	}

	VkSurfaceKHR Window::createSurface(VkInstance instance)
	{
		VkSurfaceKHR surface;
		VkResult result = glfwCreateWindowSurface(instance, m_window, nullptr, &surface);

		if (result != VK_SUCCESS)
		{
			throw std::runtime_error("Failed to create window surface. Error code: " + std::to_string(result));
		}

		return surface;
	}

	void Window::framebufferResizeCallback(GLFWwindow* window, int width, int height)
	{
		auto* windowInstance = static_cast<Window*>(glfwGetWindowUserPointer(window));
		windowInstance->m_width = width;
		windowInstance->m_height = height;
		windowInstance->m_resized = true;

		//Logger::debug("Window resized: " + std::to_string(width) + "x" + std::to_string(height));
	}

	void Window::keyCallback(GLFWwindow* window, int key, int scancode, int action, int mods)
	{
		auto* windowInstance = static_cast<Window*>(glfwGetWindowUserPointer(window));

		if (key == GLFW_KEY_ESCAPE && action == GLFW_PRESS)
		{
			glfwSetWindowShouldClose(window, GLFW_TRUE);
		}

		if (windowInstance->m_keyCallback)
		{
			windowInstance->m_keyCallback(key, scancode, action, mods);
		}
	}

	void Window::mouseButtonCallback(GLFWwindow* window, int button, int action, int mods)
	{
		auto* windowInstance = static_cast<Window*>(glfwGetWindowUserPointer(window));

		if (windowInstance->m_mouseButtonCallback)
		{
			windowInstance->m_mouseButtonCallback(button, action, mods);
		}
	}

	void Window::cursorPosCallback(GLFWwindow* window, double xpos, double ypos)
	{
		auto* windowInstance = static_cast<Window*>(glfwGetWindowUserPointer(window));

		if (windowInstance->m_cursorPosCallback)
		{
			windowInstance->m_cursorPosCallback(xpos, ypos);
		}
	}
}
