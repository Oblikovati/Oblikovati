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

		virtual void Add(uint32_t a, uint32_t b) = 0;
		virtual void PutIndices(double* Indices) = 0;
	};

	class GraphicsIndexSetObject final : public GraphicsIndexSet
	{
	public:
		GraphicsIndexSetObject() = default;
		~GraphicsIndexSetObject() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsIndexSetObject);

		void Add(uint32_t a, uint32_t b) override
		{
			
		}

		void PutIndices(double* Indices) override
		{
			
		}
	};
}
