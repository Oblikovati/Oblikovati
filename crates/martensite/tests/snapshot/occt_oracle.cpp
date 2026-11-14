// SPDX-License-Identifier: GPL-2.0-only
#include <BRepBuilderAPI_Copy.hxx>
#include <BRepPrimAPI_MakeBox.hxx>
#include <BRepGProp.hxx>
#include <GProp_GProps.hxx>
#include <Standard_Version.hxx>
#include <TopExp.hxx>
#include <NCollection_IndexedMap.hxx>
#include <TopTools_ShapeMapHasher.hxx>
#include <gp_Trsf.hxx>
#include <iomanip>
#include <iostream>
static int count(const TopoDS_Shape& shape, TopAbs_ShapeEnum kind) {
    NCollection_IndexedMap<TopoDS_Shape, TopTools_ShapeMapHasher> mapped;
    TopExp::MapShapes(shape, kind, mapped);
    return mapped.Extent();
}
static void sample(double scale, bool reversed) {
    auto source = BRepPrimAPI_MakeBox(2*scale, 3*scale, 4*scale).Shape();
    gp_Trsf pose; pose.SetTranslation(gp_Vec(10*scale, 20*scale, 30*scale));
    source.Move(TopLoc_Location(pose));
    if (reversed) source.Reverse();
    auto copied = BRepBuilderAPI_Copy(source, true, true).Shape();
    GProp_GProps props; BRepGProp::VolumeProperties(copied, props, true, false, false);
    const auto center = props.CentreOfMass();
    std::cout << scale << ',' << reversed << ',' << props.Mass() << ','
              << center.X() << ',' << center.Y() << ',' << center.Z() << ','
              << count(copied, TopAbs_FACE) << ',' << count(copied, TopAbs_EDGE) << ','
              << count(copied, TopAbs_VERTEX) << '\n';
}
int main() {
    std::cerr << OCC_VERSION_COMPLETE << '\n';
    std::cout << std::setprecision(17);
    for (double scale : {0.01, 1.0, 100.0}) for (bool reversed : {false, true}) sample(scale, reversed);
}
