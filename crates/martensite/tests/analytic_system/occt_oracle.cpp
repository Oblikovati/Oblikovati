// SPDX-License-Identifier: GPL-2.0-only
#include <BRepPrimAPI_MakeBox.hxx>
#include <BRepGProp.hxx>
#include <GProp_GProps.hxx>
#include <GProp_PrincipalProps.hxx>
#include <Standard_Version.hxx>
#include <algorithm>
#include <array>
#include <iomanip>
#include <iostream>
static void sample(double scale, std::array<double,3> offset) {
    auto first=BRepPrimAPI_MakeBox(4*scale,3*scale,2*scale).Shape();
    auto second=BRepPrimAPI_MakeBox(gp_Pnt(offset[0]*scale,offset[1]*scale,offset[2]*scale),4*scale,3*scale,2*scale).Shape();
    GProp_GProps volume, component, area, second_area;
    BRepGProp::VolumeProperties(first,volume,true,false,false);
    BRepGProp::VolumeProperties(second,component,true,false,false); volume.Add(component);
    BRepGProp::SurfaceProperties(first,area,false,false);
    BRepGProp::SurfaceProperties(second,second_area,false,false);
    const auto center=volume.CentreOfMass(); const auto tensor=volume.MatrixOfInertia();
    std::array<double,3> moments; volume.PrincipalProperties().Moments(moments[0],moments[1],moments[2]);
    std::sort(moments.begin(),moments.end());
    std::cout<<scale<<','<<offset[0]<<','<<offset[1]<<','<<offset[2]<<','<<volume.Mass()<<','<<area.Mass()+second_area.Mass()<<','<<center.X()<<','<<center.Y()<<','<<center.Z();
    for(auto pair:std::array<std::array<int,2>,6>{{{1,1},{2,2},{3,3},{1,2},{2,3},{3,1}}}) std::cout<<','<<tensor.Value(pair[0],pair[1]);
    for(double moment:moments) std::cout<<','<<moment;
    std::cout<<'\n';
}
int main() {
    std::cerr<<OCC_VERSION_COMPLETE<<'\n'; std::cout<<std::setprecision(17);
    for(double scale:{0.01,1.,100.})
        for(auto offset:std::array<std::array<double,3>,3>{{{10,0,0},{10,-20,30},{0,0,0}}}) sample(scale,offset);
}
