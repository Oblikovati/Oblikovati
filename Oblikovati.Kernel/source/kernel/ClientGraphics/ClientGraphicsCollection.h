#pragma once

#include "ClientGraphics.h"
#include "../../KernelPCH.h"

namespace Oblikovati::Kernel::ClientGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	// <summary>
	// The ClientGraphicsCollection object provides access to all of the existing objects
	// associated with a graphics container.
	// </summary>
	CONTRACT ClientGraphicsCollection : public Object, public List<ClientGraphics>
	{
	public:
		ClientGraphicsCollection() {}
		~ClientGraphicsCollection() {}
		ObjectTypeEnum GetType() override { return ObjectTypeEnum::kClientGraphicsCollectionObject; }
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END
}
