#include "EntryPoint.h"
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
		~OblikovatiApplicationTestHost()
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
	constexpr  ApplicationConfiguration config;

	return new Host::OblikovatiApplicationTestHost(config);
}
