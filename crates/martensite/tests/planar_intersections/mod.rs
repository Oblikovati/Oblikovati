// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn line(origin: [f64; 2], direction: [f64; 2]) -> Handle<dyn Geom2dCurve> {
    let origin = Point2::new(origin[0], origin[1]);
    let direction = Direction2::new(direction[0], direction[1]).unwrap();
    Handle::new(Geom2dLine::from_point(origin, direction)).cast()
}
pub(super) fn ellipse(radii: [f64; 2]) -> Handle<dyn Geom2dCurve> {
    let axis = Axis2d::new(Point2::new(0.0, 0.0), Direction2::new(1.0, 0.0).unwrap());
    let conic = Ellipse2::from_x_axis(axis, radii[0], radii[1], true).unwrap();
    Handle::new(Geom2dEllipse::from_elips2d(conic)).cast()
}
pub(super) fn trim(curve: &Handle<dyn Geom2dCurve>, bounds: [f64; 2]) -> Handle<dyn Geom2dCurve> {
    Handle::new(Geom2dTrimmedCurve::new(curve, bounds[0], bounds[1], true, true).unwrap()).cast()
}
pub(super) fn spline(poles: &[[f64; 2]], weights: Option<&[f64]>) -> Handle<dyn Geom2dCurve> {
    let points: Vec<Point2> = poles.iter().map(|p| Point2::new(p[0], p[1])).collect();
    let points = Array1::from_shared(&points, 1);
    let knots = Array1::from_shared(&[0.0, 1.0], 1);
    let count = i32::try_from(poles.len()).unwrap();
    let multiplicities = [count; 2];
    let mults = Array1::from_shared(&multiplicities, 1);
    let degree = count - 1;
    let curve = spline_from_arrays(&points, &knots, &mults, degree, weights);
    Handle::new(curve).cast()
}
fn spline_from_arrays(
    points: &Array1<'_, Point2>,
    knots: &Array1<'_, f64>,
    mults: &Array1<'_, i32>,
    degree: i32,
    weights: Option<&[f64]>,
) -> Geom2dBSplineCurve {
    match weights {
        Some(weights) => Geom2dBSplineCurve::with_weights(
            points,
            &Array1::from_shared(weights, 1),
            knots,
            mults,
            degree,
            false,
        ),
        None => Geom2dBSplineCurve::new(points, knots, mults, degree, false),
    }
    .unwrap()
}
pub(super) fn circle(radius: f64) -> Handle<dyn Geom2dCurve> {
    let axis = Axis2d::new(Point2::new(0.0, 0.0), Direction2::new(1.0, 0.0).unwrap());
    Handle::new(Geom2dCircle::from_axis(axis, radius, true).unwrap()).cast()
}
pub(super) fn assert_position(actual: Point2, expected: [f64; 2]) {
    assert!(
        (actual.x() - expected[0]).abs() < 1e-8,
        "x: {actual:?} != {expected:?}"
    );
    assert!(
        (actual.y() - expected[1]).abs() < 1e-8,
        "y: {actual:?} != {expected:?}"
    );
}
pub(super) fn assert_points(
    first: &Handle<dyn Geom2dCurve>,
    second: &Handle<dyn Geom2dCurve>,
    expected: &[[f64; 2]],
) {
    assert_ordered_pair(first, second, expected);
    assert_ordered_pair(second, first, expected);
}
fn assert_ordered_pair(
    first: &Handle<dyn Geom2dCurve>,
    second: &Handle<dyn Geom2dCurve>,
    expected: &[[f64; 2]],
) {
    let result = InterCurveCurve::between(first, second, 1e-9).unwrap();
    assert!(
        result.intersector().is_done(),
        "expected completed intersection"
    );
    assert_eq!(result.intersector().nb_segments().unwrap(), 0);
    let count = result.intersector().nb_points().unwrap();
    assert_eq!(usize::try_from(count).unwrap(), expected.len());
    for (hit, position) in sorted_points(&result, count).into_iter().zip(expected) {
        assert_incidence(first, second, hit, *position);
    }
}
fn sorted_points(result: &InterCurveCurve, count: i32) -> Vec<IntersectionPoint> {
    let mut hits: Vec<IntersectionPoint> = (1..=count)
        .map(|index| result.intersector().point(index).unwrap())
        .collect();
    // Kernel result order is not an application identity or an acceptance condition.
    hits.sort_by(|a, b| {
        a.value()
            .x()
            .total_cmp(&b.value().x())
            .then_with(|| a.value().y().total_cmp(&b.value().y()))
    });
    hits
}
pub(super) fn assert_incidence(
    first: &Handle<dyn Geom2dCurve>,
    second: &Handle<dyn Geom2dCurve>,
    hit: IntersectionPoint,
    position: [f64; 2],
) {
    assert_position(hit.value(), position);
    assert_parameter(first, hit.param_on_first(), position);
    assert_parameter(second, hit.param_on_second(), position);
}
fn assert_parameter(curve: &Handle<dyn Geom2dCurve>, parameter: f64, position: [f64; 2]) {
    assert!(
        parameter.is_finite(),
        "expected finite parameter, got {parameter}"
    );
    assert_position(curve.get().unwrap().value(parameter).unwrap(), position);
}
