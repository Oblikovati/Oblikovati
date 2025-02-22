#include "gtest/gtest.h"
#include "OblikovatiKernel.h"

extern Oblikovati::Kernel::ApplicationObject* g_Application;

namespace Oblikovati::Tests::ClientGraphics
{

// REFERENCE: AUTODESK INVENTOR SDK
// LICENSED UNDER: Creative Commons Attribution-NonCommercial-ShareAlike 3.0 Unported License.
// https://help.autodesk.com/view/INVNTOR/2025/ENU/?guid=GraphicsNode_AddTriangleFanGraphics_Sample
// 	Public Sub DrawCustomTriangles()
//     Dim oDoc As Document
//     Set oDoc = ThisApplication.ActiveDocument
// 
//     ' Set a reference to component definition of the active document.
//     ' This assumes that a part or assembly document is active.
//     Dim oCompDef As ComponentDefinition
//     Set oCompDef = ThisApplication.ActiveDocument.ComponentDefinition
//     
//     ' Check to see if the test graphics data object already exists.
//     ' If it does clean up by removing all associated of the client graphics
//     ' from the document.  If it doesn't create it and draw a line.
//     On Error Resume Next
//     Dim oGraphicsData As GraphicsDataSets
//     Set oGraphicsData = oDoc.GraphicsDataSetsCollection.Item("SampleGraphicsID")
//     If Err.Number = 0 Then
//         On Error GoTo 0
//         ' An existing client graphics object was successfully obtained so clean up.
//         oGraphicsData.Delete
//         oCompDef.ClientGraphicsCollection.Item("SampleGraphicsID").Delete
//     
//         ' update the display to see the results.
//         ThisApplication.ActiveView.Update
//     Else
//         Err.Clear
//         On Error GoTo 0
//         
//         ' Set a reference to the transient geometry object for user later.
//         Dim oTransGeom As TransientGeometry
//         Set oTransGeom = ThisApplication.TransientGeometry
//         
//         ' Create a graphics data set object.  This object contains all of the
//         ' information used to define the graphics.
//         Dim oDataSets As GraphicsDataSets
//         Set oDataSets = oDoc.GraphicsDataSetsCollection.Add("SampleGraphicsID")
//         
//         ' Create a coordinate set.
//         Dim oCoordSet As GraphicsCoordinateSet
//         Set oCoordSet = oDataSets.CreateCoordinateSet(1)
// 
//         ' Get the number of sides for the cylinder.
//         Dim iSideCount As Long
//         iSideCount = InputBox("Enter number of sides for the cylinder.  (Must be greater than 2.)", "Cylinder Tolerance", 25)
//         
//         Dim dRadius As Double
//         Dim dHeight As Double
//         dRadius = 3
//         dHeight = 7
//         
//         ' Create an array containing points to used in drawing the cylinder.
//         ' The points could also be directly defined in a coordinate set, but
//         ' defining them in array and then setting the coordinates using the
//         ' array is more efficient for a large set of points.
//         Dim adPointCoords() As Double
//         ReDim adPointCoords(1 To ((iSideCount + 1) * 2) * 3) As Double
//         Dim i As Long
//         Dim dAngle As Double
//         
//         ' Define the points for the outline of the first end of the cylinder.
//         dAngle = 0
//         For i = 0 To iSideCount - 1
//             adPointCoords(i * 3 + 1) = dRadius * Cos(dAngle)
//             adPointCoords(i * 3 + 2) = dRadius * Sin(dAngle)
//             adPointCoords(i * 3 + 3) = 0
//             
//             ' Increment the angle for the next point
//             dAngle = dAngle + ((2 * 3.14159265358979) / iSideCount)
//         Next
//         
//         ' Define the points for the outline of the other end of the cylinder.
//         dAngle = 0
//         For i = iSideCount To (iSideCount * 2) - 1
//             adPointCoords(i * 3 + 1) = dRadius * Cos(dAngle)
//             adPointCoords(i * 3 + 2) = dRadius * Sin(dAngle)
//             adPointCoords(i * 3 + 3) = dHeight
//             
//             ' Increment the angle for the next point
//             dAngle = dAngle + ((2 * 3.14159265358979) / iSideCount)
//         Next
//         
//         ' Define coordinate at the center of the first end of the cylinder.
//         adPointCoords((iSideCount * 2) * 3 + 1) = 0
//         adPointCoords((iSideCount * 2) * 3 + 2) = 0
//         adPointCoords((iSideCount * 2) * 3 + 3) = 0
//         
//         ' Define coordinate at the center of the other end of the cylinder.
//         adPointCoords((iSideCount * 2) * 3 + 4) = 0
//         adPointCoords((iSideCount * 2) * 3 + 5) = 0
//         adPointCoords((iSideCount * 2) * 3 + 6) = dHeight
// 
//         ' Assign the points to the coordinate set.
//         Call oCoordSet.PutCoordinates(adPointCoords)
//         
//         ' Create an index set to use for the triangle fan for
//         ' cap of the first end of the cylinder.  This will
//         ' serve as in index look-up into the coordinate set.
//         Dim oCap1Index As GraphicsIndexSet
//         Set oCap1Index = oDataSets.CreateIndexSet(1)
//         
//         ' Set the index values.  This could also be done by setting the
//         ' value in an array and then using the array to set the values
//         ' of the index set.  Using an array is more efficient, but this
//         ' is used to here to demonstrate the IndexSet object's Add method.
//         Call oCap1Index.Add(1, iSideCount * 2 + 1)
//         Call oCap1Index.Add(2, 1)
//         Call oCap1Index.Add(3, 2)
//         For i = 4 To iSideCount + 1
//             Call oCap1Index.Add(i, i - 1)
//         Next
//         Call oCap1Index.Add(iSideCount + 2, 1)
// 
//         ' Create an index set to use for the triangle fan of the other cylinder cap.
//         Dim oCap2Index As GraphicsIndexSet
//         Set oCap2Index = oDataSets.CreateIndexSet(2)
//         
//         ' Set the index values.
//         Call oCap2Index.Add(1, iSideCount * 2 + 2)
//         Call oCap2Index.Add(2, iSideCount + 1)
//         Call oCap2Index.Add(3, iSideCount + 2)
//         For i = 4 To iSideCount + 1
//             Call oCap2Index.Add(i, i + iSideCount - 1)
//         Next
//         Call oCap2Index.Add(iSideCount + 2, iSideCount + 1)
// 
//         ' Create an index set to use for the triangle strip that will
//         ' define the sides of the cylinder.
//         Dim oCylinderIndex As GraphicsIndexSet
//         Set oCylinderIndex = oDataSets.CreateIndexSet(3)
// 
//         ' Define the index values in an array.
//         Dim iIndexValues() As Long
//         ReDim iIndexValues(1 To (iSideCount + 1) * 2) As Long
//         For i = 0 To iSideCount - 1
//             iIndexValues(i * 2 + 1) = i + 1
//             iIndexValues(i * 2 + 2) = i + iSideCount + 1
//         Next
//         iIndexValues(((iSideCount + 1) * 2) - 1) = 1
//         iIndexValues((iSideCount + 1) * 2) = iSideCount + 1
// 
//         ' Define the index values in the index set using the array.
//         Call oCylinderIndex.PutIndices(iIndexValues)
// 
//         ' Create the ClientGraphics object.
//         Dim oClientGraphics As ClientGraphics
//         Set oClientGraphics = oCompDef.ClientGraphicsCollection.Add("SampleGraphicsID")
//        
//         ' Create a new graphics node within the client graphics objects.
//         Dim oCylinderNode As GraphicsNode
//         Set oCylinderNode = oClientGraphics.AddNode(1)
// 
//         ' Create a triangle fan for cap 1 of the cylinder.
//         Dim oCap1TriangleFan As TriangleFanGraphics
//         Set oCap1TriangleFan = oCylinderNode.AddTriangleFanGraphics
// 
//         ' Set the coordinates and index set for the cap.
//         oCap1TriangleFan.CoordinateSet = oCoordSet
//         oCap1TriangleFan.CoordinateIndexSet = oCap1Index
// 
//         ' Create a triangle fan for cap 2 of the cylinder.
//         Dim oCap2TriangleFan As TriangleFanGraphics
//         Set oCap2TriangleFan = oCylinderNode.AddTriangleFanGraphics
// 
//         ' Set the coordinates and index set for the cap.
//         oCap2TriangleFan.CoordinateSet = oCoordSet
//         oCap2TriangleFan.CoordinateIndexSet = oCap2Index
// 
//         ' Create a triangle string for the sides of the cylinder.
//         Dim oCylinderStrip As TriangleStripGraphics
//         Set oCylinderStrip = oCylinderNode.AddTriangleStripGraphics
// 
//         ' Set the coordinates and index set for the cap.
//         oCylinderStrip.CoordinateSet = oCoordSet
//         oCylinderStrip.CoordinateIndexSet = oCylinderIndex
//         
//         Dim oAppearance As Asset
//         Set oAppearance = oDoc.AppearanceAssets(1)
// 
//         oCylinderNode.Appearance = oAppearance
// 
//         ' update the display to see the results.
//         ThisApplication.ActiveView.Update
//         
//         ' Define a normal for cap 1.
//         Dim oCap1Normals As GraphicsNormalSet
//         Set oCap1Normals = oDataSets.CreateNormalSet(1)
//         Call oCap1Normals.Add(1, oTransGeom.CreateUnitVector(0, 0, -1))
//         
//         ' Assign the normals to the cap.
//         oCap1TriangleFan.NormalSet = oCap1Normals
//         
//         ' Define a normal for cap 2.
//         Dim oCap2Normals As GraphicsNormalSet
//         Set oCap2Normals = oDataSets.CreateNormalSet(2)
//         Call oCap2Normals.Add(1, oTransGeom.CreateUnitVector(0, 0, 1))
//    
//         ' Assign the normals to the cap.
//         oCap2TriangleFan.NormalSet = oCap2Normals
//         
//         ' Create an array that contains the normals for each vertex of the cylinder.
//         Dim adNormals() As Double
//         ReDim adNormals(1 To ((iSideCount + 1) * 2) * 3) As Double
//         dAngle = 0
//         For i = 0 To iSideCount
//             Dim dX As Double
//             Dim dY As Double
//             dX = Cos(dAngle)
//             dY = Sin(dAngle)
//             adNormals(i * 6 + 1) = dX
//             adNormals(i * 6 + 2) = dY
//             adNormals(i * 6 + 3) = 0
//             adNormals(i * 6 + 4) = dX
//             adNormals(i * 6 + 5) = dY
//             adNormals(i * 6 + 6) = 0
// 
//             ' Increment the angle for the next normal.
//             dAngle = dAngle + ((2 * 3.14159265358979) / iSideCount)
//         Next
//         
//         ' Create and set the normal set for the cylinder.
//         Dim oCylinderNormals As GraphicsNormalSet
//         Set oCylinderNormals = oDataSets.CreateNormalSet(3)
//         Call oCylinderNormals.PutNormals(adNormals)
//         
//         ' Assign the normals to the cylinder.
//         oCylinderStrip.NormalSet = oCylinderNormals
//         
//         ' update the display to see the results.
//         ThisApplication.ActiveView.Update
//     End If
// End Sub
// END OF REFERENCE
 
/// <summary>
/// This sample demonstrates the creation of client graphics triangles using triange fans and strips. 
/// It does this by drawing a cylinder. 
/// The end caps are triangle fans and the cylinder is made from a triangle strip.
/// </summary>

TEST(ClientGraphicsTests, DrawCylinder)
{
	//auto thisApplication = g_Application;

	// TODO: We have to create the host instance on a separate thread
	// The test should dispatch the desired state and some way get a capture from the framebuffer
	// and compare with a known good previous capture.

	// TODO: Create and Set active document as a PartDocument

	//auto doc = thisApplication->GetActiveDocument();

	//auto componentDefinition =  doc->GetComponentDefinition();

	//auto graphicsData = doc->GetGraphicsDataSetsCollection()["SampleGraphicsID"]

	EXPECT_STRNE("hello", "cylinder");
	EXPECT_EQ(7 * 6, 42);
}

};
