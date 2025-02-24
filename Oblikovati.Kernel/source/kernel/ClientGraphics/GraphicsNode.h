#pragma once
#include "GraphicsPrimitive.h"
#include "TriangleFanGraphics.h"
#include "TriangleStripGraphics.h"
#include "kernel/MaterialsAndAppearances/Asset.h"

namespace Oblikovati::Kernel::CltGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT GraphicsNode : public Object, public Dictionary<uint32_t, GraphicsPrimitive*>
	{
	public:
		GraphicsNode() = default;
		~GraphicsNode() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsNode);

		ObjectTypeEnum GetType() override { return kGraphicsNodeObject; }

		/// <summary>Method that creates a new TriangleFanGraphics graphic primitive.</summary>
		virtual TriangleFanGraphics* AddTriangleFanGraphics() = 0;
		/// <summary>Method that creates a new TriangleStripGraphics graphic primitive.</summary>
		/// <returns />
		virtual TriangleStripGraphics* AddTriangleStripGraphics() = 0;

		/// <summary>Gets the appearance asset associated with the graphics node.</summary>
		virtual MaterialsAndAppearances::Asset* GetAppearance() = 0;
		/// <summary>Sets the appearance asset associated with the graphics node.</summary>
		virtual void SetAppearance(MaterialsAndAppearances::Asset* Appearance) = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class GraphicsNodeObject final : public GraphicsNode
	{
	public:
		GraphicsNodeObject() = default;
		~GraphicsNodeObject() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsNodeObject);

		TriangleFanGraphics* AddTriangleFanGraphics() override
		{
			TriangleFanGraphicsObject triangleFanGraphicsObject;
			TriangleFanGraphicsObject* triangleFanGraphicsObjectPtr = &triangleFanGraphicsObject;
			return triangleFanGraphicsObjectPtr;
		}

		TriangleStripGraphics* AddTriangleStripGraphics() override
		{
			return nullptr;
		}
		MaterialsAndAppearances::Asset* GetAppearance() override
		{
			return nullptr;
		}
		void SetAppearance(MaterialsAndAppearances::Asset* Appearance) override
		{
			
		}
	};
}
