#pragma once

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
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class ClientGraphicsCollectionObject final : public  ClientGraphicsCollection
	{
	public:
		ClientGraphicsCollectionObject() = default;
		~ClientGraphicsCollectionObject() override = default;

		DISABLE_COPY_AND_MOVE(ClientGraphicsCollectionObject);

	};
}
