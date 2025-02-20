#pragma once

#include "kernel/Application.h"
#include "kernel/Assert.h"

extern Oblikovati::Application* Oblikovati::CreateApplication(int argc, char** argv);
bool g_ApplicationRunning = true;

namespace Oblikovati {

	int Main(int argc, char** argv)
	{
		while (g_ApplicationRunning)
		{

			InitializeKernel();
			Application* app = CreateApplication(argc, argv);
			OBKVT_CORE_ASSERT(app, "Client Application is null!");
			app->Run();
			delete app;
			ShutdownKernel();
		}
		return 0;
	}

}

#if OBKVT_RELEASE && OBKVT_PLATFORM_WINDOWS

int APIENTRY WinMain(HINSTANCE hInst, HINSTANCE hInstPrev, PSTR cmdline, int cmdshow)
{
	return Oblikovati::Main(__argc, __argv);
}

#else

int main(int argc, char** argv)
{
	return Oblikovati::Main(argc, argv);
}

#endif
