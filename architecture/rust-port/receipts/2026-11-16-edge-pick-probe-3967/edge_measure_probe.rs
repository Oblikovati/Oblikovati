// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkbrep::topods::Shape;
use martensite_tkmath::gp::Point3;
use martensite_tktopalgo::{
    brep_builder_api::BRepBuilderAPIMakeEdge, brep_extrema::BRepExtremaDistShapeShape,
};

fn segment(a: [f64; 3], b: [f64; 3]) -> Shape {
    BRepBuilderAPIMakeEdge::from_points(
        Point3::new(a[0], a[1], a[2]),
        Point3::new(b[0], b[1], b[2]),
    )
    .unwrap()
    .edge()
    .unwrap()
    .into_shape()
}

#[test]
fn analytic_segment_distance_covers_interior_and_endpoint() {
    let sight = segment([0., 0., 0.], [10., 0., 0.]);
    for (a, b, want) in [
        ([5., -1., 0.], [5., 1., 0.], 0.),
        ([5., 2., 0.], [5., 3., 0.], 2.),
        ([-3., 1., 0.], [-3., 2., 0.], 10_f64.sqrt()),
    ] {
        let edge = segment(a, b);
        let result =
            BRepExtremaDistShapeShape::with_shapes(&sight, &edge, Default::default()).unwrap();
        assert!(result.is_done());
        assert!(
            (result.value().unwrap() - want).abs() < 1e-8,
            "{a:?} {b:?}: got {:?}, expected {want}",
            result.value()
        );
    }
}

#[test]
fn analytic_circle_ray_distance_at_non_seam_point() {
    use martensite_tkernel::standard::Handle;
    use martensite_tkg3d::geom::{GeomCircle, GeomCurve};
    use martensite_tkmath::gp::{Axis2, Direction3};
    let circle: Handle<dyn GeomCurve> = Handle::new(
        GeomCircle::new(
            Axis2::new(
                Point3::new(0., 0., 0.),
                Direction3::new(0., 0., 1.).unwrap(),
            )
            .unwrap(),
            1.,
        )
        .unwrap(),
    )
    .cast();
    let edge = BRepBuilderAPIMakeEdge::from_curve(&circle)
        .unwrap()
        .edge()
        .unwrap()
        .into_shape();
    let ray = segment([0., 1., -2.], [0., 1., 2.]);
    let result = BRepExtremaDistShapeShape::with_shapes(&ray, &edge, Default::default()).unwrap();
    assert!(result.is_done());
    assert!(
        result.value().unwrap() < 1e-8,
        "distance {:?}: expected zero",
        result.value()
    );
}

#[test]
fn analytic_spline_ray_distance_at_interior_point() {
    use martensite_tkernel::{ncollection::Array1, standard::Handle};
    use martensite_tkg3d::geom::{GeomBSplineCurve, GeomCurve};
    let mut poles = Array1::with_bounds(1, 4).unwrap();
    poles[1] = Point3::new(0., 0., 0.);
    poles[2] = Point3::new(0., 1., 0.);
    poles[3] = Point3::new(1., 1., 0.);
    poles[4] = Point3::new(1., 0., 0.);
    let mut knots = Array1::with_bounds(1, 2).unwrap();
    knots[1] = 0.;
    knots[2] = 1.;
    let mut mults = Array1::with_bounds(1, 2).unwrap();
    mults[1] = 4;
    mults[2] = 4;
    let curve: Handle<dyn GeomCurve> =
        Handle::new(GeomBSplineCurve::new(&poles, &knots, &mults, 3, false).unwrap()).cast();
    let edge = BRepBuilderAPIMakeEdge::from_curve(&curve)
        .unwrap()
        .edge()
        .unwrap()
        .into_shape();
    let ray = segment([0.5, 0.75, -2.], [0.5, 0.75, 2.]);
    let result = BRepExtremaDistShapeShape::with_shapes(&ray, &edge, Default::default()).unwrap();
    assert!(result.is_done());
    assert!(
        result.value().unwrap() < 1e-8,
        "distance {:?}: expected zero",
        result.value()
    );
}

#[test]
fn analytic_circle_ray_reports_both_zero_distance_intersections() {
    use martensite_tkernel::standard::Handle;
    use martensite_tkg3d::geom::{GeomCircle, GeomCurve};
    use martensite_tkmath::gp::{Axis2, Direction3};
    let curve: Handle<dyn GeomCurve> = Handle::new(
        GeomCircle::new(
            Axis2::new(
                Point3::new(0., 0., 0.),
                Direction3::new(0., 0., 1.).unwrap(),
            )
            .unwrap(),
            1.,
        )
        .unwrap(),
    )
    .cast();
    let edge = BRepBuilderAPIMakeEdge::from_curve(&curve)
        .unwrap()
        .edge()
        .unwrap()
        .into_shape();
    let ray = segment([-3., 0., 0.], [3., 0., 0.]);
    let result = BRepExtremaDistShapeShape::with_shapes(&ray, &edge, Default::default()).unwrap();
    assert!(result.is_done());
    let mut xs: Vec<_> = (1..=result.nb_solution())
        .map(|i| result.point_on_shape1(i).unwrap().x())
        .collect();
    xs.sort_by(f64::total_cmp);
    assert!(
        xs.iter().any(|x| (*x + 1.).abs() < 1e-8) && xs.iter().any(|x| (*x - 1.).abs() < 1e-8),
        "intersection x values {xs:?}: expected -1 and 1"
    );
}

#[test]
fn global_minimum_does_not_identify_first_hit_within_pick_radius() {
    let ray = segment([0., 0., 0.], [0., 0., 20.]);
    let edge = segment([0.05, 0., 1.], [0., 0., 10.]);
    let result = BRepExtremaDistShapeShape::with_shapes(&ray, &edge, Default::default()).unwrap();
    assert!(result.is_done());
    assert!(result.value().unwrap() < 1e-8);
    let nearest = result.point_on_shape1(1).unwrap();
    assert!(
        (nearest.z() - 10.).abs() < 1e-8,
        "closest point {nearest:?}: expected z=10"
    );
    assert!(
        0.05 < 0.1,
        "edge endpoint at z=1 is inside the 0.1 pick radius"
    );
}
