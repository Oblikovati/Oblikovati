// SPDX-License-Identifier: GPL-2.0-only
#include <BRepAlgoAPI_BuilderAlgo.hxx>
#include <BOPAlgo_BuilderFace.hxx>
#include <BOPAlgo_Builder.hxx>
#include <BOPAlgo_Alerts.hxx>
#include <BRepBuilderAPI_MakeEdge.hxx>
#include <BRepBuilderAPI_MakeFace.hxx>
#include <BRepLib.hxx>
#include <BRepTopAdaptor_FClass2d.hxx>
#include <BRepGProp.hxx>
#include <GProp_GProps.hxx>
#include <TopExp.hxx>
#include <TopoDS.hxx>
#include <NCollection_IndexedMap.hxx>
#include <TopTools_ShapeMapHasher.hxx>
#include <gp_Circ.hxx>
#include <gp_Pln.hxx>
#include <gp_Elips.hxx>
#include <array>
#include <vector>
#include <algorithm>
#include <iomanip>
#include <iostream>
#include <stdexcept>
using ShapeMap = NCollection_IndexedMap<TopoDS_Shape, TopTools_ShapeMapHasher>;
using ShapeList = NCollection_List<TopoDS_Shape>;
using Point = std::array<double,2>;
using Row = std::array<double,6>;
void append_segment(ShapeList& inputs, Point start, Point end) {
  inputs.Append(BRepBuilderAPI_MakeEdge(gp_Pnt(start[0],start[1],0),
                                      gp_Pnt(end[0],end[1],0)).Edge());
}
void append_loop(ShapeList& inputs, const std::vector<Point>& points) {
  for (size_t i=0;i<points.size();++i) append_segment(inputs,points[i],points[(i+1)%points.size()]);
}
void append_conics(ShapeList& inputs,const std::string& kind) {
  const gp_Ax2 axes(gp_Pnt(2,1.5,0),gp_Dir(0,0,1));
  if (kind=="circle" || kind=="single-circle" || kind=="circle-divider") inputs.Append(BRepBuilderAPI_MakeEdge(gp_Circ(axes,0.5)).Edge());
  if (kind=="ellipse" || kind=="single-ellipse") inputs.Append(BRepBuilderAPI_MakeEdge(gp_Elips(axes,0.7,0.4)).Edge());
  const gp_Ax2 boundary(gp_Pnt(4,1.5,0),gp_Dir(0,0,1));
  if (kind=="circle-boundary") inputs.Append(BRepBuilderAPI_MakeEdge(gp_Circ(boundary,1)).Edge());
  if (kind=="ellipse-boundary") inputs.Append(BRepBuilderAPI_MakeEdge(gp_Elips(boundary,1,0.6)).Edge());
  if (kind=="overlap-circles") {
    for(double x:{1.5,2.5}) inputs.Append(BRepBuilderAPI_MakeEdge(gp_Circ(gp_Ax2(gp_Pnt(x,1.5,0),gp_Dir(0,0,1)),1)).Edge());
  }
  if (kind=="circle-tangent") inputs.Append(BRepBuilderAPI_MakeEdge(gp_Circ(gp_Ax2(gp_Pnt(5,1.5,0),gp_Dir(0,0,1)),1)).Edge());
  if (kind.rfind("pair-",0)==0) {
    const double distance = kind=="pair-separated" ? 2.0001 : kind=="pair-tangent" ? 2. : 1.9999;
    for(double x:{0.,distance}) inputs.Append(BRepBuilderAPI_MakeEdge(gp_Circ(gp_Ax2(gp_Pnt(x,0,0),gp_Dir(0,0,1)),1)).Edge());
  }
}
ShapeList arguments(const std::string& kind) {
  ShapeList inputs;
  if (kind.rfind("single-",0)!=0 && kind.rfind("pair-",0)!=0) append_loop(inputs,{{0,0},{4,0},{4,3},{0,3}});
  if (kind=="nested") append_loop(inputs,{{1,1},{3,1},{3,2},{1,2}});
  if (kind=="divider" || kind=="circle-divider") append_segment(inputs,{2,0},{2,3});
  if (kind=="crossing") append_segment(inputs,{0,0},{4,3});
  append_conics(inputs,kind);
  return inputs;
}
ShapeList oriented_edges(const TopoDS_Shape& split, const TopoDS_Face& plane) {
  ShapeMap edges;
  TopExp::MapShapes(split,TopAbs_EDGE,edges);
  ShapeList inputs;
  for (int i=1;i<=edges.Extent();++i) {
    const auto edge=TopoDS::Edge(edges.FindKey(i));
    BRepLib::BuildPCurveForEdgeOnPlane(edge,plane);
    inputs.Append(edge.Oriented(TopAbs_FORWARD));
    inputs.Append(edge.Oriented(TopAbs_REVERSED));
  }
  return inputs;
}
Row report(const TopoDS_Shape& shape) {
  GProp_GProps props;
  BRepGProp::SurfaceProperties(shape,props,false,false);
  const auto center=props.CentreOfMass();
  ShapeMap wires,edges;
  TopExp::MapShapes(shape,TopAbs_WIRE,wires);
  TopExp::MapShapes(shape,TopAbs_EDGE,edges);
  return {props.Mass(),center.X(),center.Y(),center.Z(),double(wires.Extent()),double(edges.Extent())};
}
ShapeList region_shapes(const TopoDS_Shape& shape,const std::string& kind) {
  const auto plane=BRepBuilderAPI_MakeFace(gp_Pln()).Face();
  BOPAlgo_BuilderFace builder;
  builder.SetFace(plane); builder.SetShapes(oriented_edges(shape,plane)); builder.Perform();
  if (builder.HasErrors()) throw std::runtime_error(kind+": face builder failed");
  ShapeList bounded;
  for(const auto& region:builder.Areas()) {
    if(BRepTopAdaptor_FClass2d(TopoDS::Face(region),1e-9).PerformInfinitePoint()==TopAbs_OUT)
      bounded.Append(region);
  }
  return bounded;
}
std::vector<Row> regions_from_shape(const TopoDS_Shape& shape,const std::string& kind) {
  std::vector<Row> rows;
  for(const auto& region:region_shapes(shape,kind)) rows.push_back(report(region));
  std::sort(rows.begin(),rows.end());
  return rows;
}
TopoDS_Shape fused_shape(const std::string& kind) {
  BRepAlgoAPI_BuilderAlgo fuse;
  fuse.SetArguments(arguments(kind)); fuse.SetNonDestructive(true); fuse.Build();
  if(fuse.HasErrors()) throw std::runtime_error(kind+": general fuse failed");
  return fuse.Shape();
}
std::vector<Row> regions(const std::string& kind) {
  return regions_from_shape(fused_shape(kind),kind);
}

void single_curve_declines() {
  for (const auto kind:{"single-circle","single-ellipse"}) {
    BRepAlgoAPI_BuilderAlgo fuse;
    fuse.SetArguments(arguments(kind));
    fuse.SetNonDestructive(true);
    fuse.Build();
    if (!fuse.HasError(STANDARD_TYPE(BOPAlgo_AlertTooFewArguments)))
      throw std::runtime_error(std::string(kind)+": expected too-few-arguments decline");
    std::cout<<kind<<",BOPAlgo_AlertTooFewArguments\n";
  }
}
// Diagnostic subclass only: retain General Fuse, change its input cardinality contract.
class NonemptyBuilder : public BOPAlgo_Builder {
  void CheckData() override {
    if (Arguments().IsEmpty()) { AddError(new BOPAlgo_AlertTooFewArguments); return; }
    CheckFiller();
  }
};
void nonempty_regions() {
  for(const auto kind:{"rectangle","nested","divider","crossing","circle","ellipse","single-circle","single-ellipse"}) {
    NonemptyBuilder builder; builder.SetArguments(arguments(kind)); builder.SetNonDestructive(true); builder.Perform();
    if(builder.HasErrors()) throw std::runtime_error(std::string(kind)+": nonempty General Fuse failed");
    for(const auto& row:regions_from_shape(builder.Shape(),kind)) {
      std::cout<<kind; for(double scalar:row) std::cout<<","<<scalar; std::cout<<"\n";
    }
  }
}
const std::vector<std::string> curved_cases={"circle-boundary","ellipse-boundary","circle-divider","overlap-circles","circle-tangent","pair-separated","pair-tangent","pair-near-overlap"};
#include "curved_prism_oracle.hpp"
void curved_regions() {
  for(const auto& kind:curved_cases) {
    for(const auto& row:regions(kind)) {
      std::cout<<kind; for(double scalar:row) std::cout<<","<<scalar; std::cout<<"\n";
    }
  }
}
int main(int argc, char** argv) {
  std::cout<<std::setprecision(17);
  if (argc==2 && std::string(argv[1])=="--nonempty") { nonempty_regions(); return 0; }
  if (argc==2 && std::string(argv[1])=="--curved") { curved_regions(); return 0; }
  if (argc==2 && std::string(argv[1])=="--curved-prisms") { curved_prisms(); return 0; }
  if (argc>1) { single_curve_declines(); return 0; }
  for (const auto kind:{"rectangle","nested","divider","crossing","circle","ellipse"}) {
    for (const auto& row:regions(kind)) {
      std::cout<<kind;
      for (double scalar:row) std::cout<<","<<scalar;
      std::cout<<"\n";
    }
  }
}
