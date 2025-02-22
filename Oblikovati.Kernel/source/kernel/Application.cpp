#include "kernelpch.h"
#include "Application.h"
#include "Documents/Document.h"

extern bool g_ApplicationRunning;

namespace Oblikovati::Kernel {
	ApplicationObject::ApplicationObject(const ApplicationConfiguration& config)
	{

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

	Docs::Document* ApplicationObject::GetActiveDocument(void)
	{
		return nullptr;
	}

	void ApplicationObject::SetActiveDocument(Docs::Document* Document)
	{

	}

	ApplicationObject::~ApplicationObject()
	{

	}

}
