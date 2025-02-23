#pragma once

#include "../Geometry/3D/Point.h"

namespace Oblikovati::Kernel::Math 
{
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- START
	/// <summary>
	/// The Box object is a mathematical utility object that represents a rectangular box whose 
	/// faces are always parallel to the model XYZ planes. A common use of the Box object is as 
	/// a means of passing the range box information of an entity and interacting with that range box.
	/// </summary>
	CONTRACT_STRUCT Box
	{
		/// <summary>
		/// Get the data defining this Box.
		/// </summary>
		/// <param name="MinPoint">Input Double array that sets the smallest coordinate values for the box.</param>
		/// <param name="MaxPoint">Input Double array that sets the largest coordinate values for the box.</param>
		virtual void GetBoxData(double* MinPoint, double* MaxPoint) = 0;
		/// <summary>
		/// Method that sets the data defining this Box.
		/// </summary>
		/// <param name="MinPoint">Input Double array that sets the smallest coordinate values for the box.</param>
		/// <param name="MaxPoint">Input Double array that sets the largest coordinate values for the box.</param>
		virtual void PutBoxData(double* MinPoint, double* MaxPoint) = 0;
		/// <summary>
		/// Gets the minimum corner of the box.
		/// </summary>
		virtual Geometry::_3D::Point* GetMinPoint()=0;
		/// <summary>
		/// Sets the minimum corner of the box.
		/// </summary>
		virtual void SetMinPoint(Geometry::_3D::Point* Point) = 0;
		/// <summary>
		/// Gets the maximum corner of the box.
		/// </summary>
		virtual Geometry::_3D::Point* GetMaxPoint()=0;
		/// <summary>
		/// Sets the maximum corner of the box.
		/// </summary>
		virtual void SetMaxPoint(Geometry::_3D::Point* Point) = 0;
		/// <summary>
		/// Extends the Box to include the specified point.
		/// </summary>
		/// <param name="Point">Input Point object that specifies the coordinate to include in the box.</param>
		virtual void Extend(Geometry::_3D::Point* Point) = 0;
		/// <summary>
		/// Expands the Box on all sides by the specified distance.
		/// </summary>
		/// <param name="Distance">Input Double that defines the distance by which to expand the box.Input Double that defines the distance by which to expand the box.</param>
		virtual void Expand(double Distance) = 0;
		/// <summary>
		/// Determines whether the specified point is contained within this Box.
		/// </summary>
		/// <param name="Point">Input Point object that specifies the coordinate to check.</param>
		virtual bool Contains(Geometry::_3D::Point* Point) = 0;
		/// <summary>
		/// Determines whether this Box intersects the specified Box.
		/// </summary>
		/// <param name="Box">Input Box object to compare.</param>
		/// <returns></returns>
		virtual bool IsDisjoint(Box* Box) = 0;
		/// <summary>
		/// Creates a copy of this Box object.
		/// </summary>
		/// <returns>New Box object in memory</returns>
		virtual Box* Copy() = 0;
	};
	// DO NOT MODIFY -> INVENTOR API COMPLIANCE <- END

	struct BoxObject final : Box
	{
		void GetBoxData(double* MinPoint, double* MaxPoint) override { }
		void PutBoxData(double* MinPoint, double* MaxPoint) override { }
		Geometry::_3D::Point* GetMinPoint() override
		{
			Geometry::_3D::PointObject point;
			Geometry::_3D::PointObject* pointPtr = &point;
			return pointPtr;  
		}
		void SetMinPoint(Geometry::_3D::Point* Point) override { }
		Geometry::_3D::Point* GetMaxPoint() override
		{
			Geometry::_3D::PointObject point;
			Geometry::_3D::PointObject* pointPtr = &point;
			return pointPtr;
		}
		void SetMaxPoint(Geometry::_3D::Point* Point) override { }
		void Extend(Geometry::_3D::Point* Point) override { }
		void Expand(double Distance) override { }
		bool Contains(Geometry::_3D::Point* Point) override { return false; }
		bool IsDisjoint(Box* Box) override { return false; }
		Box* Copy() override
		{
			BoxObject box;
			BoxObject* boxPtr = &box;
			return boxPtr;
		}
	};

}
