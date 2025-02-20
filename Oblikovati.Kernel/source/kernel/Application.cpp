#include "kernelpch.h"
#include "Application.h"

namespace Oblikovati {
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
