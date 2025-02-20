#include "EntryPoint.h"

class OblikovatiApplication : public Oblikovati::Kernel::Application
{
	public:
	OblikovatiApplication(const Oblikovati::Kernel::ApplicationConfiguration& config)
		: Oblikovati::Kernel::Application(config)
	{

	}
	~OblikovatiApplication()
	{

	}
	virtual void OnInit() override
	{

	}
};

Oblikovati::Kernel::Application* Oblikovati::Kernel::CreateApplication(int argc, char** argv)
{
	Oblikovati::Kernel::ApplicationConfiguration config;
	
	return new OblikovatiApplication(config);
}
