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
		Document() = default;
		~Document() override = default;

		DISABLE_COPY_AND_MOVE(Document);

		virtual ComponentDefinition* GetComponentDefinition() = 0;
		virtual CltGraphics::GraphicsDataSetsCollection* GetGraphicsDataSetsCollection() = 0;
		ObjectTypeEnum GetType() override { return kDocumentObject; }
	};

	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END
}
