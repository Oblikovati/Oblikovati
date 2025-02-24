#pragma once
#include "ClientGraphics.h"

namespace Oblikovati::Kernel::CltGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	class ClientGraphics;
	// <summary>
	// The ClientGraphicsCollection object provides access to all the existing objects
	// associated with a graphics container.
	// </summary>
	CONTRACT ClientGraphicsCollection : public Object, public Dictionary<std::string, ClientGraphics*>
	{
	public:
		ClientGraphicsCollection() = default;
		~ClientGraphicsCollection() override = default;

		DISABLE_COPY_AND_MOVE(ClientGraphicsCollection);

		ObjectTypeEnum GetType() override { return kClientGraphicsCollectionObject; }

		virtual ClientGraphics* Add(std::string ClientId) = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class ClientGraphicsCollectionObject final : public  ClientGraphicsCollection
	{
	public:
		ClientGraphicsCollectionObject() = default;
		~ClientGraphicsCollectionObject() override = default;

		DISABLE_COPY_AND_MOVE(ClientGraphicsCollectionObject);

		ClientGraphics* Add(const std::string ClientId) override
		{
			if (ContainsKey(ClientId))
				return operator[](ClientId);

			ClientGraphicsObject clientGraphicsObject;
			ClientGraphicsObject* clientGraphicsObjectPtr = &clientGraphicsObject;
			Dictionary::Add(ClientId, clientGraphicsObjectPtr);
			return  clientGraphicsObjectPtr;
		}
	};
}
