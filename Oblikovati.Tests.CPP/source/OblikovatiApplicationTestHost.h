#pragma once

#include "OblikovatiKernel.h"

namespace Oblikovati::Host 
{
	class OblikovatiApplicationTestHost : public Kernel::ApplicationObject
	{
	public:
		OblikovatiApplicationTestHost(const Kernel::ApplicationConfiguration& Config)
			: ApplicationObject(Config)
		{

		}
		~OblikovatiApplicationTestHost() override
		{

		}

		DISABLE_COPY_AND_MOVE(OblikovatiApplicationTestHost);


		void OnInit() override
		{

		}

		Application* GetApplication() override
		{
			return this;
		}

	};
};

Oblikovati::Kernel::ApplicationObject* Oblikovati::Kernel::CreateApplication(int argc, char** argv)
{
	ApplicationConfiguration config;

	config.Width = 800;
	config.Height = 600;
	config.Title = "Renderer Test";

	return new Host::OblikovatiApplicationTestHost(config);
}
