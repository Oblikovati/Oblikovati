#pragma once
#include "kernel/ClientGraphics/ClientGraphicsCollection.h"

namespace Oblikovati::Kernel::Docs
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT ComponentDefinition : public Object
	{
	public:
		ComponentDefinition() = default;
		~ComponentDefinition() override = default;

		DISABLE_COPY_AND_MOVE(ComponentDefinition);

		virtual CltGraphics::ClientGraphicsCollection* GetClientGraphicsCollection() { return  ClientGraphicsCollection;  }

		protected:
			CltGraphics::ClientGraphicsCollection* ClientGraphicsCollection;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END
}
