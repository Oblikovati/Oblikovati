// SPDX-License-Identifier: GPL-2.0-only
#include <Geom_Ellipse.hxx>
#include <gp_Ax2.hxx>
#include <gp_Dir.hxx>
#include <gp_Pnt.hxx>
#include <Standard_Version.hxx>
#include <array>
#include <iomanip>
#include <iostream>
void sample(double scale, std::array<double, 2> axis, std::array<double, 2> radii, double angle) {
    gp_Ax2 frame(gp_Pnt(2 * scale, -3 * scale, 0), gp_Dir(0, 0, 1), gp_Dir(axis[0], axis[1], 0));
    Geom_Ellipse ellipse(frame, radii[0] * scale, radii[1] * scale);
    auto result = ellipse.EvalD1(angle);
    std::cout << 2 * scale << ',' << -3 * scale << ',' << axis[0] << ',' << axis[1] << ',';
    std::cout << radii[0] * scale << ',' << radii[1] * scale << ',' << angle << ',';
    std::cout << result.Point.X() << ',' << result.Point.Y() << ',' << result.D1.X() << ',' << result.D1.Y() << '\n';
}
int main() {
    std::cerr << OCC_VERSION_COMPLETE << '\n';
    std::cout << std::setprecision(17);
    const std::array<double, 3> scales{1e-6, 1.0, 1e6};
    const std::array<std::array<double, 2>, 3> axes{{{1, 0}, {0, 1}, {3, 4}}};
    const std::array<std::array<double, 2>, 3> radii{{{3, 1}, {1, 1}, {1, 1e-8}}};
    const std::array<double, 4> angles{0.0, 0.37, 1.5707963267948966, 3.141592653589793};
    for (std::size_t row = 0; row < 108; ++row) {
        sample(scales[row / 36], axes[(row / 12) % 3], radii[(row / 4) % 3], angles[row % 4]);
    }
}
