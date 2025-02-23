#pragma once

namespace Oblikovati {
	void InitializeKernel();
	void ShutdownKernel();
}

#if !defined(OBKVT_PLATFORM_WINDOWS) && !defined(OBKVT_PLATFORM_LINUX)  && !defined(OBKVT_PLATFORM_MAC)
#error Platform not supported yet!
#endif

#define CONTRACT class
#define CONTRACT_STRUCT struct

#define DISABLE_COPY_AND_MOVE(className) \
    className(className const&) = delete; \
    className& operator=(className const&) = delete; \
    className(className&&) = delete; \
    className& operator=(className&&) = delete
