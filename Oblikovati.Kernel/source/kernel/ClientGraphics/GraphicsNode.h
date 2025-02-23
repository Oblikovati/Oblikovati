#pragma once

namespace Oblikovati::Kernel::CltGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT GraphicsNode : public Object
	{
	public:
		GraphicsNode() = default;
		~GraphicsNode() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsNode);

		ObjectTypeEnum GetType() override { return kGraphicsNodeObject; }
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class GraphicsNodeObject final : public GraphicsNode
	{
	public:
		GraphicsNodeObject() = default;
		~GraphicsNodeObject() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsNodeObject);

	};
}
