#include "kernelpch.h"
#include "Application.h"

namespace Oblikovati::Kernel {
	Application::Application(const ApplicationConfiguration& config)
	{

	}
	void Application::Run(void)
	{
		OnInit();
	}

	Application::~Application()
	{

	}
}
