#pragma once

namespace Oblikovati::Kernel::CltGraphics
{
	CONTRACT GraphicsNormalSet : public Object
	{
		public:
		GraphicsNormalSet() = default;
		~GraphicsNormalSet() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsNormalSet);

		ObjectTypeEnum GetType() override { return kGraphicsNormalSetObject; }
	};

	class GraphicsNormalSetObject final : public GraphicsNormalSet
	{
		public:
		GraphicsNormalSetObject() = default;
		~GraphicsNormalSetObject() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsNormalSetObject);
	};
}
