#pragma once

#include "Base.h"
#include "ObjectType.h"

namespace Oblikovati::Kernel
{
	class Application;
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	//<summary>
	// Base class for all objects in the kernel.
	//</summary>
	CONTRACT Object
	{
	public:
		Object() = default;
		virtual ~Object() = default;
		
		DISABLE_COPY_AND_MOVE(Object);

		virtual ObjectTypeEnum GetType() = 0;
		virtual Application* GetApplication() { return Application; }
		protected:
			Application* Application;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END
}
