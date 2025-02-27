#pragma once

#define GLFW_INCLUDE_VULKAN
#include <GLFW/glfw3.h>

#include <string>
#include <functional>

namespace Oblikovati::Kernel::VulkanRenderer
{

	class Window
	{
	public:
		Window(int width, int height, const std::string& title);
		~Window();

		DISABLE_COPY_AND_MOVE(Window);

		bool shouldClose() const;
		void pollEvents();

		GLFWwindow* getGLFWWindow() const { return m_window; }
		int getWidth() const { return m_width; }
		int getHeight() const { return m_height; }
		bool wasResized() const { return m_resized; }

		void resetResizedFlag() { m_resized = false; }

		using KeyCallback = std::function<void(int key, int scancode, int action, int mods)>;
		using MouseButtonCallback = std::function<void(int button, int action, int mods)>;
		using CursorPosCallback = std::function<void(double xpos, double ypos)>;

		void setKeyCallback(KeyCallback callback) { m_keyCallback = std::move(callback); }
		void setMouseButtonCallback(MouseButtonCallback callback) { m_mouseButtonCallback = std::move(callback); }
		void setCursorPosCallback(CursorPosCallback callback) { m_cursorPosCallback = std::move(callback); }

		VkSurfaceKHR createSurface(VkInstance instance);

	private:
		static void framebufferResizeCallback(GLFWwindow* window, int width, int height);
		static void keyCallback(GLFWwindow* window, int key, int scancode, int action, int mods);
		static void mouseButtonCallback(GLFWwindow* window, int button, int action, int mods);
		static void cursorPosCallback(GLFWwindow* window, double xpos, double ypos);

		GLFWwindow* m_window;
		int m_width;
		int m_height;
		std::string m_title;
		bool m_resized = false;

		KeyCallback m_keyCallback;
		MouseButtonCallback m_mouseButtonCallback;
		CursorPosCallback m_cursorPosCallback;
	};
}
