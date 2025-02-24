#pragma once

#include "kernel/Geometry/2D/Point2d.h"
#include "kernel/UserInterface/ViewLayoutEnum.h"

namespace Oblikovati::Kernel::CltGraphics
{
	class GraphicsNode;

	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT GraphicsPrimitive : public Object
	{
		public:
		GraphicsPrimitive() = default;
		~GraphicsPrimitive() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsPrimitive);

		ObjectTypeEnum GetType() override { return kGraphicsPrimitiveObject; }
		/// <summary>Property that returns the parent object from whom this object can logically be reached.</summary>
		virtual GraphicsNode* GetParent() { return Parent; }
		/// <summary>Property that returns a Box object which contains the opposing points of a rectangular box that is guaranteed to enclose this object.</summary>
		virtual Math::Box* GetRangeBox() = 0;
		/// <summary>Method that deletes the graphics primitive.</summary>
		virtual void Delete() = 0;
		/// <summary>Method that gets the anchor information of the graphics object. This method returns an error if the 'Anchored' property returns False.</summary>
		/// <param name="Origin">
		///     <summary>Output that specifies the origin of the coordinate system.</summary>
		/// </param>
		/// <param name="Anchor">
		///     <summary>Output that indicates which point is unaffected by the transform behavior.</summary>
		/// </param>
		/// <param name="AnchorRelativeTo">
		///     <summary>Output constant indicating which corner of the view the anchor is relative to.</summary>
		/// </param>
		virtual void GetViewSpaceAnchor(Geometry::_3D::Point* Origin, Geometry::_2D::Point2d* Anchor, UserInterface::ViewLayoutEnum AnchorRelativeTo) = 0;
		/// <summary>Method that anchors the graphics object at the specified point in view space. The Anchored property is set to True.</summary>
		/// <param name="Origin">
		///     <summary>Input that specifies the origin of the coordinate system.</summary>
		/// </param>
		/// <param name="Anchor">
		///     <summary>Input that indicates which point is unaffected by the transform behavior.</summary>
		/// </param>
		/// <param name="AnchorRelativeTo">
		///     <summary>Input constant indicating which corner of the view the anchor is relative to.</summary>
		/// </param>
		/// <returns />
		virtual void SetViewSpaceAnchor(Geometry::_3D::Point* Origin, Geometry::_2D::Point2d* Anchor, UserInterface::ViewLayoutEnum AnchorRelativeTo) = 0;
		/// <summary>Property that indicates whether this graphics primitive is anchored in view space. This property can only be set to False. The Anchored property is automatically set to True by the SetViewSpaceAnchor method.</summary>
		virtual bool GetAnchored() = 0;
		/// <summary>The RemoveViewSpaceAnchor method removes the view space anchor from the object, and sets the Anchored property to false.</summary>
		virtual void RemoveViewSpaceAnchor() = 0;
		/// <summary>Read-only property that returns the Id of the object.</summary>
		virtual int32_t GetId() = 0;
		protected:
			GraphicsNode* Parent;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END
}
