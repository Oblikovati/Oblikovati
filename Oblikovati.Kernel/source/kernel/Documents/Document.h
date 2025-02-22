#pragma once

#include "../../KernelPCH.h"
#include "ComponentDefinition.h"
#include "../ClientGraphics/GraphicsDataSetsCollection.h"

namespace Oblikovati::Kernel::Docs
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT Document : public Object
	{
	public:
		Document() {}
		~Document() {}
		virtual ComponentDefinition* GetComponentDefinition() = 0;
		virtual CltGraphics::GraphicsDataSetsCollection GetGraphicsDataSetsCollection() = 0;
		virtual ObjectTypeEnum GetType() override { return ObjectTypeEnum::kDocumentObject; }
	};

	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END
}
