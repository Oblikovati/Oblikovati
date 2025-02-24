#pragma once

namespace Oblikovati::Kernel::MaterialsAndAppearances
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	/// <summary>The Asset object represents an appearance, material, or physical asset within a library.</summary>
	CONTRACT Asset : public Object
	{
		public:
		Asset() = default;
		~Asset() override = default;

		DISABLE_COPY_AND_MOVE(Asset);

		ObjectTypeEnum GetType() override { return kAssetObject; }
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class AssetObject final : public Asset
	{
		public:
		AssetObject() = default;
		~AssetObject() override = default;

		DISABLE_COPY_AND_MOVE(AssetObject);


	};
}
