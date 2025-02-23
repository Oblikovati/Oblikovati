#pragma once

#include "../ComponentDefinition.h"

namespace Oblikovati::Kernel::Docs::PrtDocument
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT PartComponentDefinition : public ComponentDefinition
	{
	public:
		PartComponentDefinition() = default;
		~PartComponentDefinition() override = default;

		DISABLE_COPY_AND_MOVE(PartComponentDefinition);

		ObjectTypeEnum GetType() override { return kPartComponentDefinitionObject; }
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class PartComponentDefinitionObject final : public PartComponentDefinition
	{
	public:
		PartComponentDefinitionObject() = default;
		~PartComponentDefinitionObject() override = default;

		DISABLE_COPY_AND_MOVE(PartComponentDefinitionObject);
	};
}
