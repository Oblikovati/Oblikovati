#pragma once

#include "../KernelPCH.h"
#include "ObjectType.h"

namespace Oblikovati::Kernel
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	//<summary>
	// Base class for all objects in the kernel.
	//</summary>
	CONTRACT Object
	{
	public:
		Object() {}
		virtual ~Object() {}
		virtual ObjectTypeEnum GetType(void) = 0;
		virtual void* GetApplication() = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END
}
