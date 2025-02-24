#pragma once
#include "Asset.h"

namespace Oblikovati::Kernel::MaterialsAndAppearances
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	/// <summary>The Asset object represents an appearance, material, or physical asset within a library.</summary>
	CONTRACT AssetsEnumerator : public Object, public List<Asset*>
	{
		public:
		AssetsEnumerator() = default;
		~AssetsEnumerator() override = default;

		DISABLE_COPY_AND_MOVE(AssetsEnumerator);

		ObjectTypeEnum GetType() override { return kAssetsEnumeratorObject; }
	};

	class AssetsEnumeratorObject final : public AssetsEnumerator
	{
		public:
			AssetsEnumeratorObject() = default;
		~AssetsEnumeratorObject() override = default;

		DISABLE_COPY_AND_MOVE(AssetsEnumeratorObject);
		
	};
}
