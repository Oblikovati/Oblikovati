#pragma once

#include "GraphicsDataset.h"
#include "GraphicsCoordinateSet.h"
#include "GraphicsIndexSet.h"

namespace Oblikovati::Kernel::CltGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT GraphicsDatasets : public Object, Dictionary<std::string, GraphicsDataset*>
	{
		public:
		GraphicsDatasets() = default;
		~GraphicsDatasets() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsDatasets);

		ObjectTypeEnum GetType() override { return kGraphicsDataSetsObject; }

		/// <summary>Method that creates a new GraphicsCoordinateSet object. You use methods provided by the CoordinateSet object to define the coordinates.</summary>
		/// <param name="DataSetId">
		///     <summary>Input Long that specifies the unique identifier for the GraphicsDataSet object. This must be unique with respect to all other GraphicsDataSet objects within this GraphicsDataSets object.</summary>
		/// </param>
		/// <returns />
		virtual GraphicsCoordinateSet* CreateCoordinateSet(int DataSetId) = 0;
		/// <summary>Method that creates a new GraphicsIndexSet object. You use methods provided by the GraphicsIndexSet object to define the indices.</summary>
		/// <param name="DataSetId">
		///     <summary>Input Long that specifies the unique identifier for the GraphicsDataSet object. This must be unique with respect to all other GraphicsDataSet objects within this GraphicsDataSets object.</summary>
		/// </param>
		/// <returns />
		virtual GraphicsIndexSet* CreateIndexSet(int DataSetId) = 0;
		/// <summary>Method that deletes the GraphicsDataSets object.</summary>
		virtual void Delete() = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class GraphicsDatasetsObject final : public GraphicsDatasets
	{
		GraphicsCoordinateSet* CreateCoordinateSet(int DataSetId) override
		{
			GraphicsCoordinateSetObject graphicsCoordinateSetObject;
			GraphicsCoordinateSetObject* graphicsCoordinateSetObjectPtr = &graphicsCoordinateSetObject;
			return graphicsCoordinateSetObjectPtr;
		}

		GraphicsIndexSet* CreateIndexSet(int DataSetId) override
		{
			GraphicsIndexSetObject graphicsIndexSetObject;
			GraphicsIndexSetObject* graphicsIndexSetObjectPtr = &graphicsIndexSetObject;
			return graphicsIndexSetObjectPtr;
		}

		void Delete() override
		{
			
		}
	};

}
