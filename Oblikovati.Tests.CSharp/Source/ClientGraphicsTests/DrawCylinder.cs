using System;
using System.Reflection.Metadata;
using Oblikovati.API;

public class CustomTriangles
{
	private Oblikovati.API.Application inventorApp;

	public CustomTriangles(Oblikovati.API.Application app)
	{
		inventorApp = app;
	}

	public void DrawCustomTriangles()
	{
		var oDoc = inventorApp.ActiveDocument;

		// Set a reference to component definition of the active document.
		// This assumes that a part or assembly document is active.
		ComponentDefinition oCompDef = inventorApp.ActiveDocument.ComponentDefinition;

		// Check to see if the test graphics data object already exists.
		// If it does clean up by removing all associated of the client graphics
		// from the document. If it doesn't create it and draw a line.
		GraphicsDataSets oGraphicsData = null;
		try
		{
			oGraphicsData = oDoc.GraphicsDataSetsCollection["SampleGraphicsID"];

			// An existing client graphics object was successfully obtained so clean up.
			oGraphicsData.Delete();
			oCompDef.ClientGraphicsCollection["SampleGraphicsID"].Delete();

			// update the display to see the results.
			inventorApp.ActiveView.Update();
		}
		catch
		{
			// Set a reference to the transient geometry object for user later.
			TransientGeometry oTransGeom = inventorApp.TransientGeometry;

			// Create a graphics data set object. This object contains all of the
			// information used to define the graphics.
			GraphicsDataSets oDataSets = oDoc.GraphicsDataSetsCollection.Add("SampleGraphicsID");

			// Create a coordinate set.
			GraphicsCoordinateSet oCoordSet = oDataSets.CreateCoordinateSet(1);

			// Get the number of sides for the cylinder.
			int iSideCount = Convert.ToInt32(Microsoft.VisualBasic.Interaction.InputBox("Enter number of sides for the cylinder. (Must be greater than 2.)", "Cylinder Tolerance", "25"));

			double dRadius = 3;
			double dHeight = 7;

			// Create an array containing points to used in drawing the cylinder.
			// The points could also be directly defined in a coordinate set, but
			// defining them in array and then setting the coordinates using the
			// array is more efficient for a large set of points.
			double[] adPointCoords = new double[((iSideCount + 1) * 2) * 3];
			double dAngle = 0;

			// Define the points for the outline of the first end of the cylinder.
			for (int i = 0; i < iSideCount; i++)
			{
				adPointCoords[i * 3] = dRadius * Math.Cos(dAngle);
				adPointCoords[i * 3 + 1] = dRadius * Math.Sin(dAngle);
				adPointCoords[i * 3 + 2] = 0;

				// Increment the angle for the next point
				dAngle = dAngle + ((2 * Math.PI) / iSideCount);
			}

			// Define the points for the outline of the other end of the cylinder.
			dAngle = 0;
			for (int i = iSideCount; i < (iSideCount * 2); i++)
			{
				adPointCoords[i * 3] = dRadius * Math.Cos(dAngle);
				adPointCoords[i * 3 + 1] = dRadius * Math.Sin(dAngle);
				adPointCoords[i * 3 + 2] = dHeight;

				// Increment the angle for the next point
				dAngle = dAngle + ((2 * Math.PI) / iSideCount);
			}

			// Define coordinate at the center of the first end of the cylinder.
			adPointCoords[(iSideCount * 2) * 3] = 0;
			adPointCoords[(iSideCount * 2) * 3 + 1] = 0;
			adPointCoords[(iSideCount * 2) * 3 + 2] = 0;

			// Define coordinate at the center of the other end of the cylinder.
			adPointCoords[(iSideCount * 2) * 3 + 3] = 0;
			adPointCoords[(iSideCount * 2) * 3 + 4] = 0;
			adPointCoords[(iSideCount * 2) * 3 + 5] = dHeight;

			// Assign the points to the coordinate set.
			oCoordSet.PutCoordinates(ref adPointCoords);

			// Create an index set to use for the triangle fan for
			// cap of the first end of the cylinder. This will
			// serve as in index look-up into the coordinate set.
			GraphicsIndexSet oCap1Index = oDataSets.CreateIndexSet(1);

			// Set the index values. This could also be done by setting the
			// value in an array and then using the array to set the values
			// of the index set. Using an array is more efficient, but this
			// is used to here to demonstrate the IndexSet object's Add method.
			oCap1Index.Add(1, iSideCount * 2 + 1);
			oCap1Index.Add(2, 1);
			oCap1Index.Add(3, 2);
			for (int i = 4; i <= iSideCount + 1; i++)
			{
				oCap1Index.Add(i, i - 1);
			}
			oCap1Index.Add(iSideCount + 2, 1);

			// Create an index set to use for the triangle fan of the other cylinder cap.
			GraphicsIndexSet oCap2Index = oDataSets.CreateIndexSet(2);

			// Set the index values.
			oCap2Index.Add(1, iSideCount * 2 + 2);
			oCap2Index.Add(2, iSideCount + 1);
			oCap2Index.Add(3, iSideCount + 2);
			for (int i = 4; i <= iSideCount + 1; i++)
			{
				oCap2Index.Add(i, i + iSideCount - 1);
			}
			oCap2Index.Add(iSideCount + 2, iSideCount + 1);

			// Create an index set to use for the triangle strip that will
			// define the sides of the cylinder.
			GraphicsIndexSet oCylinderIndex = oDataSets.CreateIndexSet(3);

			// Define the index values in an array.
			int[] iIndexValues = new int[(iSideCount + 1) * 2];
			for (int i = 0; i < iSideCount; i++)
			{
				iIndexValues[i * 2] = i + 1;
				iIndexValues[i * 2 + 1] = i + iSideCount + 1;
			}
			iIndexValues[((iSideCount + 1) * 2) - 2] = 1;
			iIndexValues[((iSideCount + 1) * 2) - 1] = iSideCount + 1;

			// Define the index values in the index set using the array.
			oCylinderIndex.PutIndices(ref iIndexValues);

			// Create the ClientGraphics object.
			ClientGraphics oClientGraphics = oCompDef.ClientGraphicsCollection.Add("SampleGraphicsID");

			// Create a new graphics node within the client graphics objects.
			GraphicsNode oCylinderNode = oClientGraphics.AddNode(1);

			// Create a triangle fan for cap 1 of the cylinder.
			TriangleFanGraphics oCap1TriangleFan = oCylinderNode.AddTriangleFanGraphics();

			// Set the coordinates and index set for the cap.
			oCap1TriangleFan.CoordinateSet = oCoordSet;
			oCap1TriangleFan.CoordinateIndexSet = oCap1Index;

			// Create a triangle fan for cap 2 of the cylinder.
			TriangleFanGraphics oCap2TriangleFan = oCylinderNode.AddTriangleFanGraphics();

			// Set the coordinates and index set for the cap.
			oCap2TriangleFan.CoordinateSet = oCoordSet;
			oCap2TriangleFan.CoordinateIndexSet = oCap2Index;

			// Create a triangle string for the sides of the cylinder.
			TriangleStripGraphics oCylinderStrip = oCylinderNode.AddTriangleStripGraphics();

			// Set the coordinates and index set for the cap.
			oCylinderStrip.CoordinateSet = oCoordSet;
			oCylinderStrip.CoordinateIndexSet = oCylinderIndex;

			Asset oAppearance = oDoc.AppearanceAssets[1];

			oCylinderNode.Appearance = oAppearance;

			// update the display to see the results.
			inventorApp.ActiveView.Update();

			// Define a normal for cap 1.
			GraphicsNormalSet oCap1Normals = oDataSets.CreateNormalSet(1);
			oCap1Normals.Add(1, oTransGeom.CreateUnitVector(0, 0, -1));

			// Assign the normals to the cap.
			oCap1TriangleFan.NormalSet = oCap1Normals;

			// Define a normal for cap 2.
			GraphicsNormalSet oCap2Normals = oDataSets.CreateNormalSet(2);
			oCap2Normals.Add(1, oTransGeom.CreateUnitVector(0, 0, 1));

			// Assign the normals to the cap.
			oCap2TriangleFan.NormalSet = oCap2Normals;

			// Create an array that contains the normals for each vertex of the cylinder.
			double[] adNormals = new double[((iSideCount + 1) * 2) * 3];
			dAngle = 0;
			for (int i = 0; i <= iSideCount; i++)
			{
				double dX = Math.Cos(dAngle);
				double dY = Math.Sin(dAngle);
				adNormals[i * 6] = dX;
				adNormals[i * 6 + 1] = dY;
				adNormals[i * 6 + 2] = 0;
				adNormals[i * 6 + 3] = dX;
				adNormals[i * 6 + 4] = dY;
				adNormals[i * 6 + 5] = 0;

				// Increment the angle for the next normal.
				dAngle = dAngle + ((2 * Math.PI) / iSideCount);
			}

			// Create and set the normal set for the cylinder.
			GraphicsNormalSet oCylinderNormals = oDataSets.CreateNormalSet(3);
			oCylinderNormals.PutNormals(ref adNormals);

			// Assign the normals to the cylinder.
			oCylinderStrip.NormalSet = oCylinderNormals;

			// update the display to see the results.
			inventorApp.ActiveView.Update();
		}
	}
}
