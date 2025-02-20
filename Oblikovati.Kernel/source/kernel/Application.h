#pragma once

#include "Base.h"

namespace Oblikovati
{
	class IApplication
	{
		public:
			IApplication() {}
			virtual ~IApplication() {}
			virtual void Run(void) = 0;

	};

	struct ApplicationConfiguration
	{
		//char* Name = "Oblikovati";
	
	};

	class Application : IApplication
	{
		public:
			Application(const ApplicationConfiguration& config);
			virtual ~Application();
			void Run(void) override;
			virtual void OnInit() {}
	};
	
	Application* CreateApplication(int argc, char** argv);
}
