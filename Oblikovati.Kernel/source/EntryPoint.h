#pragma once

#include "kernel/Application.h"
#include "kernel/OblikovatiAssert.h"

extern Oblikovati::Kernel::ApplicationObject* Oblikovati::Kernel::CreateApplication(int argc, char** argv);

bool g_ApplicationRunning = true;
//We need it global for testing purposes
Oblikovati::Kernel::ApplicationObject* g_Application = nullptr;

namespace Oblikovati 
{
	int Main(int argc, char** argv)
	{
		while (g_ApplicationRunning)
		{
			InitializeKernel();
			g_Application= Kernel::CreateApplication(argc, argv);
			OBKVT_CORE_ASSERT(g_Application, "Client Application is null!");
			g_Application->Run();
			delete g_Application;
			ShutdownKernel();
		}
		return 0;
	}
}
