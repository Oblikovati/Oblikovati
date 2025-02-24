#pragma once

#include "../Document.h"
#include "PartComponentDefinition.h"
#include "kernel/MaterialsAndAppearances/AssetsEnumerator.h"


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

		virtual MaterialsAndAppearances::AssetsEnumerator* GetAppearanceAssets() = 0;

		virtual MaterialsAndAppearances::AssetsEnumerator* GetMaterialAssets() = 0;
		
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class PartDocumentObject final : public PartDocument
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

		CltGraphics::GraphicsDataSetsCollection* GetGraphicsDataSetsCollection() override
		{
			return nullptr;
		}
		MaterialsAndAppearances::AssetsEnumerator* GetAppearanceAssets() override
		{
			return nullptr;
		}
		MaterialsAndAppearances::AssetsEnumerator* GetMaterialAssets() override
		{
			return nullptr;
		}
	};
}
