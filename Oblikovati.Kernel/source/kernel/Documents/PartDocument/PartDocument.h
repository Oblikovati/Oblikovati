#pragma once

#include "../Document.h"
#include "PartComponentDefinition.h"


namespace Oblikovati::Kernel::Docs::PrtDocument
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT PartDocument : public Document
	{
	public:
		PartDocument() = default;
		~PartDocument() override = default;

		DISABLE_COPY_AND_MOVE(PartDocument);

		PartComponentDefinition* GetComponentDefinition() override = 0;
		
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class PartDocumentObject : public PartDocument
	{
	public:
		PartDocumentObject() = default;
		~PartDocumentObject() override = default;

		DISABLE_COPY_AND_MOVE(PartDocumentObject);

		PartComponentDefinition* GetComponentDefinition() override
		{
			PartComponentDefinitionObject partComponentDef;
			PartComponentDefinitionObject* partComponentDefPtr = & partComponentDef;
			return  partComponentDefPtr;
		}
	};
}
