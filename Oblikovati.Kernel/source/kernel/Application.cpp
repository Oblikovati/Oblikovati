#include "kernelpch.h"
#include "Application.h"
#include "Documents/Document.h"

extern bool g_ApplicationRunning;

namespace Oblikovati::Kernel {
	ApplicationObject::ApplicationObject(const ApplicationConfiguration& Config)
	{
		ActiveDocument = nullptr;
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

	ApplicationObject::~ApplicationObject() = default;

}
