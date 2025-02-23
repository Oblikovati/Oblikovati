#pragma once

namespace Oblikovati::Kernel::CltGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT GraphicsDataset : public Object
	{
		public:
		GraphicsDataset() = default;
		~GraphicsDataset() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsDataset);

		ObjectTypeEnum GetType() override { return kGraphicsDataSetObject; }
		/// <summary>Property returning the unique id of this GraphicsDataSet.</summary>
		virtual int GetId() = 0;
		/// <summary>Property that returns the number of objects in this collection.</summary>
		virtual int Count() = 0;
		/// <summary>Method that removes a coordinate from the set.</summary>
		/// <param name="Index">
		///     <summary>Specifies the index you want to remove from the set. All coordinates above the Index will be shifted down. The coordinate set indices are one-based.</summary>
		/// </param>
		virtual void Remove(int Index);
		/// <summary>Method that deletes the GraphicsDataSet.</summary>
		virtual void Delete() = 0;

	};

	class GraphicsDatasetObject final : public  GraphicsDataset
	{
	public:
		GraphicsDatasetObject() = default;
		~GraphicsDatasetObject() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsDatasetObject);

		int GetId() override { return 0; }
		int Count() override { return 0; }
		void Remove(int Index) override { };
		void Delete() override { };
		
	};
}
