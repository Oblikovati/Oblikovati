// SPDX-License-Identifier: GPL-2.0-only
// Standalone native capture; see implementation-circumcircle.md.
#include <gce_MakeCirc2d.hxx>
#include <gp_Circ2d.hxx>
#include <Standard_Version.hxx>
#include <array>
#include <iomanip>
#include <iostream>
using Triple = std::array<std::array<double, 2>, 3>;
static void emit_case(const Triple& points) {
    gce_MakeCirc2d maker(gp_Pnt2d(points[0][0], points[0][1]),
                        gp_Pnt2d(points[1][0], points[1][1]),
                        gp_Pnt2d(points[2][0], points[2][1]));
    if (!maker.IsDone()) { throw "expected completed ordinary oracle construction"; }
    const auto circle = maker.Value();
    std::cout << "{\"points\":[";
    for (std::size_t i = 0; i < points.size(); ++i) {
        if (i != 0) { std::cout << ','; }
        std::cout << '[' << points[i][0] << ',' << points[i][1] << ']';
    }
    std::cout << "],\"center\":[" << circle.Location().X() << ','
              << circle.Location().Y() << "],\"radius\":" << circle.Radius() << '}';
}
int main() {
    const std::array<Triple, 4> cases = {{
        {{{2, 0}, {0, 2}, {-2, 0}}},
        {{{0.1, 0.2}, {2.3, -1.7}, {4.4, 5.6}}},
        {{{-4, 2}, {0, -1}, {7, 3}}},
        {{{1e8 + 2, -1e8}, {1e8, -1e8 + 2}, {1e8 - 2, -1e8}}}
    }};
    std::cout << std::setprecision(17) << "{\"version\":\"" << OCC_VERSION_COMPLETE << "\",\"cases\":[";
    for (std::size_t i = 0; i < cases.size(); ++i) {
        if (i != 0) { std::cout << ','; }
        emit_case(cases[i]);
    }
    std::cout << "]}\n";
}
