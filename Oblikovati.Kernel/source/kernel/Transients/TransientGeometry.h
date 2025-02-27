#pragma once
#include "kernel/Math/UnitVector.h"

namespace Oblikovati::Kernel::Transients
{
	CONTRACT TransientGeometry : public Object
	{
		public:
		ObjectTypeEnum GetType() override { return kTransientGeometryObject; }

		virtual Math::UnitVector* CreateUnitVector(float X, float Y, float Z) = 0;
	};

	class TransientGeometryObject final : public TransientGeometry
	{

		Math::UnitVector* CreateUnitVector(float X, float Y, float Z) override
		{
			return nullptr;
		}
	};
}
