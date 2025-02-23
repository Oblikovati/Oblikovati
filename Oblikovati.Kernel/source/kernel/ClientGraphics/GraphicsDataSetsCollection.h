#pragma once

#include "GraphicsDataSets.h"

namespace Oblikovati::Kernel::CltGraphics
{

	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT GraphicsDataSetsCollection : public Object, public Dictionary<std::string, GraphicsDatasets*>
	{
	public:
		GraphicsDataSetsCollection() = default;
		~GraphicsDataSetsCollection() override = default;
		virtual GraphicsDatasets* Add(std::string Id) = 0;

		DISABLE_COPY_AND_MOVE(GraphicsDataSetsCollection);


	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class GraphicsDataSetsCollectionObject : public  GraphicsDataSetsCollection
	{
	public:
		GraphicsDatasets* Add(const std::string Id) override
		{
			GraphicsDatasetsObject gDatasetObj;
			GraphicsDatasetsObject* gDatasetObjPrt = &gDatasetObj;
			Dictionary::Add(Id, gDatasetObjPrt);
			return gDatasetObjPrt;
		}
	};
}
