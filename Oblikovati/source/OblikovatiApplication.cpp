#include "EntryPoint.h"

class OblikovatiApplication : public Oblikovati::Application
{
	public:
	OblikovatiApplication(const Oblikovati::ApplicationConfiguration& config)
		: Oblikovati::Application(config)
	{

	}
	~OblikovatiApplication()
	{

	}
	virtual void OnInit() override
	{

	}
};

Oblikovati::Application* Oblikovati::CreateApplication(int argc, char** argv)
{
	Oblikovati::ApplicationConfiguration config;
	
	return new OblikovatiApplication(config);
}
