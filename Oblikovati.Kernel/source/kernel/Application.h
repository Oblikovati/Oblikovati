#pragma once

#include "Object.h"
#include "Documents/Document.h"
#include "Transients/TransientGeometry.h"
#include "UserInterface/View.h"

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
			virtual UserInterface::View* GetActiveView() = 0;
			virtual Transients::TransientGeometry* GetTransientGeometry() = 0;
			ObjectTypeEnum GetType() override
			{
				return kApplicationObject;
			}
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
			UserInterface::View* GetActiveView() override { return ActiveView; }
			Transients::TransientGeometry* GetTransientGeometry() override { return TransientGeometry; }
		protected:
			Docs::Document* ActiveDocument;
			UserInterface::View* ActiveView;
			Transients::TransientGeometry* TransientGeometry = &m_TransientGeometryObject;
	private: 
		bool m_Running = true;
		Transients::TransientGeometryObject m_TransientGeometryObject;
	};
	
	ApplicationObject* CreateApplication(int Argc, char** Argv);
}
