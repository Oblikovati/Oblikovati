// SPDX-License-Identifier: GPL-2.0-only
#include <BRepBuilderAPI_MakePolygon.hxx>
#include <BRepBuilderAPI_MakeFace.hxx>
#include <BRepPrimAPI_MakePrism.hxx>
#include <BRepGProp.hxx>
#include <GProp_GProps.hxx>
#include <GProp_PrincipalProps.hxx>
#include <algorithm>
#include <TopExp.hxx>
#include <NCollection_IndexedMap.hxx>
#include <TopTools_ShapeMapHasher.hxx>
#include <Standard_Version.hxx>
#include <iomanip>
#include <iostream>
#include <vector>
#include <array>
#include <string>
struct Profile { TopoDS_Face face; std::vector<TopoDS_Edge> edges; };
static TopoDS_Wire loop(const std::vector<gp_Pnt>& vertices, std::vector<TopoDS_Edge>& edges) {
    BRepBuilderAPI_MakePolygon maker;
    maker.Add(vertices[0]);
    for (size_t i=1; i<vertices.size(); ++i) { maker.Add(vertices[i]); edges.push_back(maker.Edge()); }
    maker.Close(); edges.push_back(maker.Edge());
    return maker.Wire();
}
struct Frame { std::array<double,3> origin, x, y, normal; };
static Frame frame(int index) {
    switch(index) {
    case 0: return {{0,0,0},{1,0,0},{0,1,0},{0,0,1}};
    case 1: return {{10,-20,30},{1,0,0},{0,1,0},{0,0,1}};
    case 2: return {{10,-20,30},{0,1,0},{0,0,1},{1,0,0}};
    case 3: return {{10,-20,30},{0.6,0.8,0},{0,0,1},{0.8,-0.6,0}};
    case 4: return {{10,-20,30},{1,0,0},{0,-1,0},{0,0,-1}};
    case 5: return {{10,-20,30},{1+5e-10,0,0},{5e-10,1,0},{0,0,1}};
    default: throw std::runtime_error("expected frame index 0..5");
    }
}
static gp_Pnt placed_point(const gp_Pnt& point, int index) {
    const auto f=frame(index); std::array<double,3> xyz;
    for(size_t i=0;i<3;++i) xyz[i]=(f.origin[i]+point.X()*f.x[i])+point.Y()*f.y[i];
    return {xyz[0],xyz[1],xyz[2]};
}
static gp_Vec placed_vector(const gp_Vec& vector, int index) {
    const auto f=frame(index); std::array<double,3> xyz;
    for(size_t i=0;i<3;++i) xyz[i]=(vector.X()*f.x[i]+vector.Y()*f.y[i])+vector.Z()*f.normal[i];
    return {xyz[0],xyz[1],xyz[2]};
}
static Profile profile(int kind, double scale, int placement) {
    std::vector<gp_Pnt> points = kind==2
        ? std::vector<gp_Pnt>{{0,0,0},{3,0,0},{3,1,0},{1,1,0},{1,3,0},{0,3,0}}
        : std::vector<gp_Pnt>{{0,0,0},{4,0,0},{4,3,0},{0,3,0}};
    for (auto& point:points) point=placed_point(gp_Pnt(point.XYZ()*scale),placement);
    Profile result;
    BRepBuilderAPI_MakeFace maker(loop(points,result.edges), true);
    if (kind==1) {
        std::vector<gp_Pnt> hole{{scale,scale,0},{scale,2*scale,0},{3*scale,2*scale,0},{3*scale,scale,0}};
        for(auto& point:hole) point=placed_point(point,placement);
        maker.Add(loop(hole,result.edges));
    }
    result.face=maker.Face(); return result;
}
static void face_row(int kind, double scale, const gp_Vec& vector, int role, int key,
                     const TopoDS_Shape& face, const GProp_GProps& volume, int placement, bool placed) {
    GProp_GProps area; BRepGProp::SurfaceProperties(face,area,false,false);
    NCollection_IndexedMap<TopoDS_Shape,TopTools_ShapeMapHasher> loops;
    TopExp::MapShapes(face,TopAbs_WIRE,loops);
    const auto point=area.CentreOfMass(); const auto center=volume.CentreOfMass();
    if(placed) std::cout<<placement<<',';
    std::cout<<kind<<','<<scale<<','<<vector.X()<<','<<vector.Y()<<','<<vector.Z()<<','
        <<role<<','<<key<<','<<area.Mass()<<','<<point.X()<<','<<point.Y()<<','<<point.Z()<<','
        <<loops.Extent()<<','<<volume.Mass()<<','<<center.X()<<','<<center.Y()<<','<<center.Z()<<'\n';
}
static void mass_row(int kind, double scale, const gp_Vec& vector, const TopoDS_Shape& shape, const GProp_GProps& volume) {
    GProp_GProps area; BRepGProp::SurfaceProperties(shape,area,false,false);
    const auto center=volume.CentreOfMass(); const auto tensor=volume.MatrixOfInertia();
    const auto principal=volume.PrincipalProperties(); std::array<double,3> moments;
    principal.Moments(moments[0],moments[1],moments[2]); std::sort(moments.begin(),moments.end());
    std::cout<<kind<<','<<scale<<','<<vector.X()<<','<<vector.Y()<<','<<vector.Z()<<','
        <<volume.Mass()<<','<<area.Mass()<<','<<center.X()<<','<<center.Y()<<','<<center.Z();
    for(auto pair:std::vector<std::array<int,2>>{{1,1},{2,2},{3,3},{1,2},{2,3},{3,1}})
        std::cout<<','<<tensor.Value(pair[0],pair[1]);
    for(auto moment:moments) std::cout<<','<<moment;
    std::cout<<'\n';
}
static void sample(int kind, double scale, gp_Vec vector, int placement, bool placed, bool mass) {
    vector*=scale; auto input=profile(kind,scale,placement);
    BRepPrimAPI_MakePrism maker(input.face,placed_vector(vector,placement),true,true);
    GProp_GProps volume; BRepGProp::VolumeProperties(maker.Shape(),volume,true,false,false);
    if(mass) { mass_row(kind,scale,vector,maker.Shape(),volume); return; }
    face_row(kind,scale,vector,0,0,maker.FirstShape(),volume,placement,placed);
    face_row(kind,scale,vector,1,0,maker.LastShape(),volume,placement,placed);
    for (size_t i=0; i<input.edges.size(); ++i) {
        const auto& generated=maker.Generated(input.edges[i]);
        if (generated.Extent()!=1) throw std::runtime_error("expected one generated side face");
        face_row(kind,scale,vector,2,static_cast<int>(i)+1,generated.First(),volume,placement,placed);
    }
}
int main(int argc, char** argv) {
    const bool placed=argc==2 && std::string(argv[1])=="--placed";
    const bool mass=argc==2 && std::string(argv[1])=="--mass";
    if(argc!=1 && !placed && !mass) throw std::runtime_error("expected no argument, --placed, or --mass");
    std::cerr<<OCC_VERSION_COMPLETE<<'\n'; std::cout<<std::setprecision(17);
    for (int placement=placed?1:0; placement<=(placed?5:0); ++placement)
    for (int kind:{0,1,2}) for (double scale:{0.01,1.,100.})
        for (gp_Vec vector:{gp_Vec(0,0,2),gp_Vec(0,0,-2),gp_Vec(1,2,3)}) sample(kind,scale,vector,placement,placed,mass);
}
