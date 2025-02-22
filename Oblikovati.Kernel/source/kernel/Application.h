#pragma once

#include "../KernelPCH.h"
#include "Documents/Document.h"

namespace Oblikovati::Kernel
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT Application : public Object
	{
		public:
			Application() {}
			virtual ~Application() {}
			virtual void Run(void) = 0;
			virtual Docs::Document* GetActiveDocument() = 0;
			virtual void SetActiveDocument(Docs::Document* Document) = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	struct ApplicationConfiguration
	{
		//char* Name = "Oblikovati";
	
	};

	class ApplicationObject : Application
	{
		public:
			ApplicationObject(const ApplicationConfiguration& config);
			virtual ~ApplicationObject();
			void Run(void) override;
			virtual Docs::Document* GetActiveDocument() override;
			virtual void SetActiveDocument(Docs::Document* Document) override;
			virtual void OnInit() = 0;
			virtual void OnShutdown();
			virtual Oblikovati::Kernel::ObjectTypeEnum GetType() override
			{
				return Kernel::ObjectTypeEnum::kApplicationObject;
			}
		protected:
			Docs::Document* ActiveDocument;
	private: 
		bool m_Running = true;
	};
	
	ApplicationObject* CreateApplication(int argc, char** argv);
}
