// SPDX-License-Identifier: GPL-2.0-only
#include <BRepPrimAPI_MakePrism.hxx>
#include <BRepCheck_Analyzer.hxx>
#include <BRepAdaptor_Surface.hxx>
void prism_face_rows(const std::string& kind,const TopoDS_Shape& region) {
  const auto region_report=report(region);
  const auto solid=BRepPrimAPI_MakePrism(region,gp_Vec(0,0,2),true,true).Shape();
  if(!BRepCheck_Analyzer(solid,true,false,true).IsValid()) throw std::runtime_error(kind+": invalid prism");
  GProp_GProps volume; BRepGProp::VolumeProperties(solid,volume,true,false,false);
  ShapeMap faces; TopExp::MapShapes(solid,TopAbs_FACE,faces);
  for(int index=1;index<=faces.Extent();++index) {
    const auto face=TopoDS::Face(faces.FindKey(index));
    std::cout<<kind<<","<<region_report[0]<<","<<region_report[1]<<","<<region_report[2]<<","<<volume.Mass();
    for(double scalar:report(face)) std::cout<<","<<scalar;
    std::cout<<","<<int(BRepAdaptor_Surface(face).GetType())<<"\n";
  }
}
void curved_prisms() {
  for(const auto& kind:curved_cases) {
    for(const auto& region:region_shapes(fused_shape(kind),kind)) prism_face_rows(kind,region);
  }
}
