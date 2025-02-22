#include "EntryPoint.h"
#include "OblikovatiKernel.h"

namespace Oblikovati::Host 
{
	class OblikovatiApplicationTestHost : public Oblikovati::Kernel::ApplicationObject
	{
	public:
		OblikovatiApplicationTestHost(const Oblikovati::Kernel::ApplicationConfiguration& config)
			: Oblikovati::Kernel::ApplicationObject(config)
		{

		}
		~OblikovatiApplicationTestHost()
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

	return new Oblikovati::Host::OblikovatiApplicationTestHost(config);
}

