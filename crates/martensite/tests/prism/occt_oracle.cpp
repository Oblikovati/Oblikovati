// SPDX-License-Identifier: GPL-2.0-only
#include <BRepBuilderAPI_MakePolygon.hxx>
#include <BRepBuilderAPI_MakeFace.hxx>
#include <BRepPrimAPI_MakePrism.hxx>
#include <BRepGProp.hxx>
#include <GProp_GProps.hxx>
#include <TopExp.hxx>
#include <NCollection_IndexedMap.hxx>
#include <TopTools_ShapeMapHasher.hxx>
#include <Standard_Version.hxx>
#include <iomanip>
#include <iostream>
#include <vector>
struct Profile { TopoDS_Face face; std::vector<TopoDS_Edge> edges; };
static TopoDS_Wire loop(const std::vector<gp_Pnt>& vertices, std::vector<TopoDS_Edge>& edges) {
    BRepBuilderAPI_MakePolygon maker;
    maker.Add(vertices[0]);
    for (size_t i=1; i<vertices.size(); ++i) { maker.Add(vertices[i]); edges.push_back(maker.Edge()); }
    maker.Close(); edges.push_back(maker.Edge());
    return maker.Wire();
}
static Profile profile(int kind, double scale) {
    std::vector<gp_Pnt> points = kind==2
        ? std::vector<gp_Pnt>{{0,0,0},{3,0,0},{3,1,0},{1,1,0},{1,3,0},{0,3,0}}
        : std::vector<gp_Pnt>{{0,0,0},{4,0,0},{4,3,0},{0,3,0}};
    for (auto& point:points) point.SetXYZ(point.XYZ()*scale);
    Profile result;
    BRepBuilderAPI_MakeFace maker(loop(points,result.edges), true);
    if (kind==1) {
        std::vector<gp_Pnt> hole{{scale,scale,0},{scale,2*scale,0},{3*scale,2*scale,0},{3*scale,scale,0}};
        maker.Add(loop(hole,result.edges));
    }
    result.face=maker.Face(); return result;
}
static void face_row(int kind, double scale, const gp_Vec& vector, int role, int key,
                     const TopoDS_Shape& face, const GProp_GProps& volume) {
    GProp_GProps area; BRepGProp::SurfaceProperties(face,area,false,false);
    NCollection_IndexedMap<TopoDS_Shape,TopTools_ShapeMapHasher> loops;
    TopExp::MapShapes(face,TopAbs_WIRE,loops);
    const auto point=area.CentreOfMass(); const auto center=volume.CentreOfMass();
    std::cout<<kind<<','<<scale<<','<<vector.X()<<','<<vector.Y()<<','<<vector.Z()<<','
        <<role<<','<<key<<','<<area.Mass()<<','<<point.X()<<','<<point.Y()<<','<<point.Z()<<','
        <<loops.Extent()<<','<<volume.Mass()<<','<<center.X()<<','<<center.Y()<<','<<center.Z()<<'\n';
}
static void sample(int kind, double scale, gp_Vec vector) {
    vector*=scale; auto input=profile(kind,scale);
    BRepPrimAPI_MakePrism maker(input.face,vector,true,true);
    GProp_GProps volume; BRepGProp::VolumeProperties(maker.Shape(),volume,true,false,false);
    face_row(kind,scale,vector,0,0,maker.FirstShape(),volume);
    face_row(kind,scale,vector,1,0,maker.LastShape(),volume);
    for (size_t i=0; i<input.edges.size(); ++i) {
        const auto& generated=maker.Generated(input.edges[i]);
        if (generated.Extent()!=1) throw std::runtime_error("expected one generated side face");
        face_row(kind,scale,vector,2,static_cast<int>(i)+1,generated.First(),volume);
    }
}
int main() {
    std::cerr<<OCC_VERSION_COMPLETE<<'\n'; std::cout<<std::setprecision(17);
    for (int kind:{0,1,2}) for (double scale:{0.01,1.,100.})
        for (gp_Vec vector:{gp_Vec(0,0,2),gp_Vec(0,0,-2),gp_Vec(1,2,3)}) sample(kind,scale,vector);
}
