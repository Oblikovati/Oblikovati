#pragma once

#include "../../Math/Matrix.h"
#include "../../Math/Vector.h"

namespace Oblikovati::Kernel::Geometry::_3D 
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	/// <summary>
	/// The Point object. The object is a transient mathematical object and is not displayed graphically.
	///
	/// </summary>
	CONTRACT_STRUCT Point
	{
			Point() = default;
			virtual ~Point() = default;

			DISABLE_COPY_AND_MOVE(Point);

			double GetX() const { return X; }

			double GetY() const { return Y; }

			double GetZ() const { return Z; }
			/// <summary>
			/// Method that sets the data defining this point.
			/// </summary>
			/// <param name="Coords">Input/output Double that specifies the entity's coordinates.</param>
			virtual void PutPointData(double* Coords) = 0;
			/// <summary>
			/// Transform this point by the specified matrix.
			/// </summary>
			/// <param name="Matrix">Input Matrix object that specifies the matrix to transform by.</param>
			virtual void TransformBy(Math::Matrix* Matrix)=0;
			/// <summary>
			/// Translate this point by the specified vector.
			/// </summary>
			/// <param name="Vector">Input Vector object that specifies the vector to translate by.</param>
			virtual void TranslateBy(Math::Vector* Vector) = 0;
			/// <summary>
			/// Determine the distance between this point and the specified point.
			/// </summary>
			/// <param name="Point">Input object that specifies the point.</param>
			/// <returns></returns>
			virtual double DistanceTo(Point* Point)=0;
			/// <summary>
			/// Gets the vector of translation between this point and the specified point.
			/// </summary>
			/// <param name="Point">Input object that specifies the point.</param>
			/// <returns></returns>
			virtual Math::Vector VectorTo(Point* Point)=0;
			/// <summary>
			/// Compares this point for equality with the specified point.
			/// </summary>
			/// <param name="Point">Input Point object that specifies the coordinate to compare.</param>
			/// <param name="Tolerance">Input Double that specifies the tolerance to be used when determining whether the points are equal.</param>
			/// <returns></returns>
			virtual bool IsEqualTo(Point* Point, double Tolerance = 0.0)=0;
		protected:
			double X, Y, Z;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	struct PointObject final : Point
	{
		PointObject() : Point() { }
		~PointObject() override = default;

		DISABLE_COPY_AND_MOVE(PointObject);

		void PutPointData(double* Coords) override { }
		void TransformBy(Math::Matrix* Matrix) override { }
		void TranslateBy(Math::Vector* Vector) override { }
		double DistanceTo(Point* Point) override { return 0; }
		Math::Vector VectorTo(Point* Point) override 
		{
			Math::VectorObject vec;
			return vec;
		}
		bool IsEqualTo(Point* Point, double Tolerance = 0.0) override { return false; }
	};
}
