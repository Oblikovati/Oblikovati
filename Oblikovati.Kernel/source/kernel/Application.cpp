#include "kernelpch.h"
#include "Application.h"
#include "Documents/Document.h"
#include "Renderer/Model.h"

extern bool g_ApplicationRunning;

namespace Oblikovati::Kernel {
	ApplicationObject::ApplicationObject(const ApplicationConfiguration& Config)
	{
		ActiveDocument = nullptr;
		m_settings = Config;
	}
	void ApplicationObject::Run(void)
	{
		OnInit();
		while (m_Running)
		{

		}
		OnShutdown();
	}

	void ApplicationObject::OnShutdown()
	{
		g_ApplicationRunning = false;
	}

	Docs::Document* ApplicationObject::GetActiveDocument()
	{
		return nullptr;
	}

	void ApplicationObject::SetActiveDocument(Docs::Document* Document)
	{

	}


	bool ApplicationObject::loadModel(const std::string& path)
	{
		try
		{
			auto model = std::make_unique<VulkanRenderer::Model>();
			if (model->loadFromFile(path))
			{
				m_models.push_back(std::move(model));
				//Logger::info("Model loaded successfully: " + path);
				return true;
			}
			else
			{
				//Logger::error("Failed to load model: " + path);
				return false;
			}
		}
		catch (const std::exception& e)
		{
			//Logger::error("Exception while loading model: " + std::string(e.what()));
			return false;
		}
	}

	void ApplicationObject::init()
	{
		// Create window
		m_window = std::make_unique<VulkanRenderer::Window>(m_settings.Width, m_settings.Height, m_settings.Title);

		// Create renderer
		m_renderer = std::make_unique<VulkanRenderer::Renderer>(m_window.get(), m_settings.EnableValidation);

		m_initialized = true;
		m_Running = true;

		//Logger::info("Application initialized successfully");
	}

	bool ApplicationObject::renderSingleFrame()
	{
		if (!m_initialized)
		{
			init();
		}

		try
		{
			m_renderer->renderFrame(m_models);
			return true;
		}
		catch (const std::exception& e)
		{
			//Logger::error("Exception during frame rendering: " + std::string(e.what()));
			return false;
		}
	}


	ApplicationObject::~ApplicationObject() = default;

}
