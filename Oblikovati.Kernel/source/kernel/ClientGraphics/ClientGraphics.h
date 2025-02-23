#pragma once

#include "ClientGraphicsCollection.h"
#include "../Math/Box.h"
#include "GraphicsNode.h"
#include "GraphicsSelectabilityEnum.h"
#include "GraphicsVisibilityEnum.h"


namespace Oblikovati::Kernel::CltGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	/// <summary>
	/// The ClientGraphics object is a collection of objects.
	/// </summary>
	CONTRACT ClientGraphics : Object, Dictionary<std::string,GraphicsNode*>
	{
	public:
		ClientGraphics() = default;
		~ClientGraphics() override = default;

		DISABLE_COPY_AND_MOVE(ClientGraphics);

		ObjectTypeEnum GetType() override { return kClientGraphicsObject; }

		virtual ClientGraphicsCollection* GetParent() =0;

		virtual GraphicsNode* GetItemById(int32_t NodeId) = 0;
		/// <summary>
		/// Property that returns the ID of this ClientGraphics object.
		/// </summary>
		/// <returns></returns>
		virtual std::string GetClientId() = 0;
		/// <summary>
		/// >Property that returns a Box object which contains the opposing points of a 
		/// rectangular box that is guaranteed to enclose this object.
		/// </summary>
		/// <returns></returns>
		virtual Math::Box* GetRangeBox() = 0;
		/// <summary>
		/// Input Long that specifies the identifier for the newly created entity.
		/// </summary>
		/// <param name="NodeId"> This id needs to be unique with respect to all other objects in this ClientGraphics object.</param>
		/// <returns>GraphicsNode</returns>
		virtual GraphicsNode* AddNode(int32_t NodeId) = 0;
		/// <summary>
		/// Property that gets whether all of the GraphicsNode objects within the ClientGraphics object are selectable. 
		/// When getting this property valid values are kAllGraphicsSelectable, kNoGraphicsSelectable, and kSomeGraphicsSelectable. 
		/// </summary>
		/// <returns></returns>
		virtual GraphicsSelectabilityEnum GetSelectable() = 0;
		/// <summary>
		/// Property that sets whether all of the GraphicsNode objects within the ClientGraphics object are selectable. 
		/// When setting this property kAllGraphicsSelectable and kNoGraphicsSelectable are valid.
		/// </summary>
		/// <returns></returns>
		virtual void SetSelectable(GraphicsSelectabilityEnum Selectability) = 0;
		/// <summary>
		/// Property that gets  whether all of the GraphicsNode objects within the ClientGraphics object are visible. 
		/// When getting this property valid values are kAllGraphicsVisible, kNoGraphicsVisible, and kSomeGraphicsVisible. 
		/// </summary>
		/// <returns></returns>
		virtual GraphicsVisibilityEnum GetVisible() = 0;
		/// <summary>
		/// Property thatsets whether all of the GraphicsNode objects within the ClientGraphics object are visible. 
		/// When setting this property kAllGraphicsVisible and kNoGraphicsVisible are valid.
		/// </summary>
		/// <returns></returns>
		virtual void SetVisible(GraphicsVisibilityEnum Visibility) = 0;
		/// <summary>
		/// Method that deletes this ClientGraphics object. This will delete all of its associated GraphicsNode objects too.
		/// </summary>
		virtual void Delete() = 0;
		/// <summary>
		/// >Read-only property that returns whether the creation of this ClientGraphics object and all its associated data is non-transacting.
		/// </summary>
		/// <returns></returns>
		virtual bool GetNonTransacting() = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class ClientGraphicsObject final : public ClientGraphics
	{
	public:
		ClientGraphicsObject() = default;
		~ClientGraphicsObject() override = default;

		DISABLE_COPY_AND_MOVE(ClientGraphicsObject);

		GraphicsNode* GetItemById(int32_t NodeId) override { return nullptr; }
		std::string GetClientId() override { return ""; }
		Math::Box* GetRangeBox() override 
		{
			Math::BoxObject box;
			Math::BoxObject* boxPtr = &box;
			return boxPtr;
		}

		ClientGraphicsCollection* GetParent() override;
		GraphicsNode* AddNode(int32_t NodeId) override;
		GraphicsSelectabilityEnum GetSelectable() override;
		void SetSelectable(GraphicsSelectabilityEnum Selectability) override;
		GraphicsVisibilityEnum GetVisible() override;
		void SetVisible(GraphicsVisibilityEnum Visibility) override;
		void Delete() override;
		bool GetNonTransacting() override;
	};
}
