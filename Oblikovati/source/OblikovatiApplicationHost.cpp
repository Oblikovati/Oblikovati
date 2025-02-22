#include "EntryPoint.h"
#include "OblikovatiKernel.h"

namespace Oblikovati::Host 
{
	class OblikovatiApplicationHost : public Oblikovati::Kernel::ApplicationObject
	{
	public:
		OblikovatiApplicationHost(const Oblikovati::Kernel::ApplicationConfiguration& config)
			: Oblikovati::Kernel::ApplicationObject(config)
		{

		}
		~OblikovatiApplicationHost()
		{

		}

		virtual void OnInit() override
		{

		}

		virtual void* GetApplication() override
		{
			return this;
		}

	};


};

Oblikovati::Kernel::ApplicationObject* Oblikovati::Kernel::CreateApplication(int argc, char** argv)
{
	Oblikovati::Kernel::ApplicationConfiguration config;

	return new Oblikovati::Host::OblikovatiApplicationHost(config);
}

