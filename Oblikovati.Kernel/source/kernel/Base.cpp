#include "KernelPCH.h"
#include "Base.h"


namespace Oblikovati {

	void InitializeKernel()
	{
		OBKVT_CORE_TRACE_TAG("Kernel", "Oblikovati {}", OBKVT_VERSION);
		OBKVT_CORE_TRACE_TAG("Kernel", "Initializing...");
	}

	void ShutdownKernel()
	{
		OBKVT_CORE_TRACE_TAG("Kernel", "Shutting down...");

	}

}
