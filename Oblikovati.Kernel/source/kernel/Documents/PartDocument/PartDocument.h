#pragma once

#include "KernelPCH.h"
#include "../Document.h"
#include "PartComponentDefinition.h"


namespace Oblikovati::Kernel::Docs::PrtDocument
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT PartDocument : public Document
	{
	public:
		PartDocument(){}
		~PartDocument() { }

		virtual PartComponentDefinition* GetComponentDefinition() = 0;
		
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class PartDocumentObject : public PartDocument
	{
	public:
		PartDocumentObject();
		~PartDocumentObject();
		

	private:

	};
}
