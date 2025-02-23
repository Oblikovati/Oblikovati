#pragma once

namespace Oblikovati::Kernel::CltGraphics
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	CONTRACT GraphicsCoordinateSet : public Object
	{
		public:
		GraphicsCoordinateSet() = default;
		~GraphicsCoordinateSet() override = default;

		DISABLE_COPY_AND_MOVE(GraphicsCoordinateSet);

		ObjectTypeEnum GetType() override { return kGraphicsCoordinateSetObject; }

		/// <summary>Method that sets all of the coordinates of the set. This will replace all existing coordinates currently defined for the set.</summary>
		/// <param name="Coordinates">
		///     <summary>Input/output array of Doubles that contains the x-y-z value of the coordinates. The array is a single dimension array containing sequential x, y, z values.</summary>
		/// </param>
		/// <returns />
		virtual void PutCoordinates(double* Coordinates) = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	class GraphicsCoordinateSetObject final : public GraphicsCoordinateSet
	{
	public:
		void PutCoordinates(double* Coordinates) override
		{
			
		}
		
	};
}
