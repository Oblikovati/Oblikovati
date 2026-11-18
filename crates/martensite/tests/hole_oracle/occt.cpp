// SPDX-License-Identifier: GPL-2.0-only
// Generate the per-face Hole oracle with OCCT; see #4093.
#include <BRepAdaptor_Surface.hxx>
#include <BRepAlgoAPI_Cut.hxx>
#include <BRepCheck_Analyzer.hxx>
#include <BRepGProp.hxx>
#include <BRepPrimAPI_MakeBox.hxx>
#include <BRepPrimAPI_MakeCylinder.hxx>
#include <GProp_GProps.hxx>
#include <Standard_Version.hxx>
#include <TopExp_Explorer.hxx>
#include <TopoDS.hxx>
#include <algorithm>
#include <iomanip>
#include <iostream>
#include <stdexcept>
#include <tuple>
#include <vector>

static void sample(const char* label, double depth) {
    const auto box = BRepPrimAPI_MakeBox(1., 1., 1.).Shape();
    const gp_Ax2 axis(gp_Pnt(.5, .5, 1.), gp_Dir(0., 0., -1.));
    const auto drill = BRepPrimAPI_MakeCylinder(axis, .1, depth).Shape();
    BRepAlgoAPI_Cut cut(box, drill);
    if (!cut.IsDone() || !BRepCheck_Analyzer(cut.Shape()).IsValid())
        throw std::runtime_error("expected a valid OCCT Hole result");
    std::vector<std::tuple<int, double, int>> rows;
    for (TopExp_Explorer face(cut.Shape(), TopAbs_FACE); face.More(); face.Next()) {
        const auto entity = TopoDS::Face(face.Current());
        const auto surface = BRepAdaptor_Surface(entity).GetType();
        GProp_GProps properties;
        BRepGProp::SurfaceProperties(entity, properties);
        int loops = 0;
        for (TopExp_Explorer wire(entity, TopAbs_WIRE); wire.More(); wire.Next()) ++loops;
        rows.emplace_back(static_cast<int>(surface), properties.Mass(), loops);
    }
    std::sort(rows.begin(), rows.end());
    for (const auto& [surface, area, loops] : rows)
        std::cout << label << ',' << surface << ',' << area << ',' << loops << '\n';
}

int main() {
    std::cerr << "OCCT " << OCC_VERSION_COMPLETE << '\n';
    std::cout << std::setprecision(17);
    sample("through", 1.);
    sample("blind", .4);
}
