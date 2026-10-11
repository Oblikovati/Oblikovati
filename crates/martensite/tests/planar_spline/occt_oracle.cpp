// SPDX-License-Identifier: GPL-2.0-only
#include <Geom2d_BSplineCurve.hxx>
#include <Geom2d_OffsetCurve.hxx>
#include <NCollection_Array1.hxx>
#include <Standard_Version.hxx>
#include <iomanip>
#include <iostream>

static Handle(Geom2d_Curve) parent(int kind, double scale) {
    const int count = kind == 2 ? 4 : 3;
    NCollection_Array1<gp_Pnt2d> poles(1, count);
    poles(1) = gp_Pnt2d(0, 0); poles(2) = gp_Pnt2d(scale, 2 * scale);
    poles(3) = gp_Pnt2d(3 * scale, 0);
    if (kind == 2) poles(4) = gp_Pnt2d(scale, -scale);
    NCollection_Array1<double> knots(1, kind == 2 ? 5 : 2);
    NCollection_Array1<int> mults(1, knots.Length());
    for (int i = 1; i <= knots.Length(); ++i) { knots(i) = i - 1; mults(i) = kind == 2 ? 1 : 3; }
    if (kind != 1) return new Geom2d_BSplineCurve(poles, knots, mults, 2, kind == 2);
    NCollection_Array1<double> weights(1, count);
    weights(1) = 1; weights(2) = 0.5; weights(3) = 1;
    return new Geom2d_BSplineCurve(poles, weights, knots, mults, 2, false);
}
static void samples(int kind, double scale, double left) {
    auto basis = parent(kind, scale);
    Handle(Geom2d_Curve) curve = left == 0 ? basis : Handle(Geom2d_Curve)(new Geom2d_OffsetCurve(basis, -left));
    for (double fraction : {0.0, 0.25, 0.5, 0.75, 1.0}) {
        double u = basis->FirstParameter() + fraction * (basis->LastParameter() - basis->FirstParameter());
        auto sample = curve->EvalD1(u);
        std::cout << kind << ',' << scale << ',' << left << ',' << u << ','
                  << sample.Point.X() << ',' << sample.Point.Y() << ',' << sample.D1.X() << ',' << sample.D1.Y() << '\n';
    }
}
int main() {
    std::cerr << OCC_VERSION_COMPLETE << '\n';
    std::cout << std::setprecision(17);
    for (int kind : {0, 1, 2}) for (double scale : {1e-6, 1.0, 1e6})
        for (double left : {-0.1 * scale, 0.0, 0.1 * scale}) samples(kind, scale, left);
}
