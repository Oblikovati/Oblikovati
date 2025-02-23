#pragma once

namespace Oblikovati::Kernel::CltGraphics
{
	CONTRACT GraphicsIndexSet : public Object
	{
		public:
		GraphicsIndexSet() = default;
		~GraphicsIndexSet() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsIndexSet);

		ObjectTypeEnum GetType() override { return kGraphicsIndexSetObject; }
	};

	class GraphicsIndexSetObject final : public GraphicsIndexSet
	{
	public:
		GraphicsIndexSetObject() = default;
		~GraphicsIndexSetObject() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsIndexSetObject);
		
	};
}
