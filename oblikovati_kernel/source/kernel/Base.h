#pragma once

namespace Oblikovati {
	void InitializeKernel();
	void ShutdownKernel();
}

#if !defined(OBKVT_PLATFORM_WINDOWS) && !defined(OBKVT_PLATFORM_LINUX)  && !defined(OBKVT_PLATFORM_MAC)
#error Unknown platform.
#endif
