#pragma once

#include "Object.h"
#include "Documents/Document.h"

namespace Oblikovati::Kernel
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT Application : public Object
	{
		public:
			Application() = default;
			~Application() override = default;

			DISABLE_COPY_AND_MOVE(Application);

			virtual void Run() = 0;
			virtual Docs::Document* GetActiveDocument() = 0;
			virtual void SetActiveDocument(Docs::Document* Document) = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	struct ApplicationConfiguration
	{
		//char* Name = "Oblikovati";
	
	};

	class ApplicationObject : public Application
	{
		public:
			explicit ApplicationObject(const ApplicationConfiguration& Config);
			~ApplicationObject() override;

			DISABLE_COPY_AND_MOVE(ApplicationObject);

			void Run() override;
			Docs::Document* GetActiveDocument() override;
			void SetActiveDocument(Docs::Document* Document) override;
			virtual void OnInit() = 0;
			virtual void OnShutdown();
			ObjectTypeEnum GetType() override
			{
				return kApplicationObject;
			}
		protected:
			Docs::Document* ActiveDocument;
	private: 
		bool m_Running = true;
	};
	
	ApplicationObject* CreateApplication(int Argc, char** Argv);
}
