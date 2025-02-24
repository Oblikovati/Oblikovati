#pragma once
#include "GraphicsCoordinateSet.h"
#include "GraphicsIndexSet.h"
#include "GraphicsPrimitive.h"

namespace Oblikovati::Kernel::CltGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT TriangleStripGraphics : GraphicsPrimitive
	{
		public:
		TriangleStripGraphics() = default;
		~TriangleStripGraphics() override = default;

		DISABLE_COPY_AND_MOVE(TriangleStripGraphics);

		ObjectTypeEnum GetType() override { return kTriangleStripGraphicsObject; }

		/// <summary>Gets the GraphicsCoordinateSet associated with the set.</summary>
		virtual GraphicsCoordinateSet* GetCoordinateSet() = 0;
		/// <summary>Sets the GraphicsCoordinateSet associated with the set.</summary>
		virtual void SetCoordinateSet(GraphicsCoordinateSet* CoordinateSet) = 0;
		/// <summary>Get the GraphicsIndexSet that defines the indices within the coordinate set to use for the triangles of the set.</summary>
		virtual GraphicsIndexSet* GetCoordinateIndexSet() = 0;
		/// <summary>Sets the GraphicsIndexSet that defines the indices within the coordinate set to use for the triangles of the set.</summary>
		virtual void SetCoordinateIndexSet(GraphicsIndexSet* IndexSet) = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class TriangleStripGraphicsObject final : public TriangleStripGraphics
	{
		public:
		TriangleStripGraphicsObject() = default;
		~TriangleStripGraphicsObject() override = default;

		DISABLE_COPY_AND_MOVE(TriangleStripGraphicsObject);
		Math::Box* GetRangeBox() override
		{
			return nullptr;
		}
		void Delete() override
		{
			
		}
		void GetViewSpaceAnchor(Geometry::_3D::Point* Origin, Geometry::_2D::Point2d* Anchor,
			UserInterface::ViewLayoutEnum AnchorRelativeTo) override
		{
			
		}
		void SetViewSpaceAnchor(Geometry::_3D::Point* Origin, Geometry::_2D::Point2d* Anchor,
			UserInterface::ViewLayoutEnum AnchorRelativeTo) override
		{
			
		}
		bool GetAnchored() override
		{
			return false;
		}
		void RemoveViewSpaceAnchor() override
		{
			
		}
		int32_t GetId() override
		{
			return 0;
		}

		GraphicsCoordinateSet* GetCoordinateSet() override
		{
			return nullptr;
		}
		void SetCoordinateSet(GraphicsCoordinateSet* CoordinateSet) override
		{
			
		}
		GraphicsIndexSet* GetCoordinateIndexSet() override
		{
			return nullptr;
		}
		void SetCoordinateIndexSet(GraphicsIndexSet* IndexSet) override
		{
			
		}
	};

}
