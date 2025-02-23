#pragma once

namespace Oblikovati::Kernel::Transients
{
	CONTRACT TransientGeometry : public Object
	{
		ObjectTypeEnum GetType() override { return kTransientGeometryObject; }
	};

	class TransientGeometryObject final : public TransientGeometry
	{

		
	};
}
