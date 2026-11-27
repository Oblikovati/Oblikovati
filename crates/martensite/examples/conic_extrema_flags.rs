// SPDX-License-Identifier: GPL-2.0-only
//! Read-only SDK reproducer: run with `cargo run -p oblikovati-martensite --example conic_extrema_flags`.
use martensite_tkernel::precision;
use martensite_tkgeombase::extrema::ExtPElC2d;
use martensite_tkmath::{
    elclib,
    gp::{Axis2d, Circle2, Direction2, Ellipse2, Hyperbola2, Parabola2, Point2, Vector2},
};

fn axis(origin: [f64; 2], direction: [f64; 2]) -> Axis2d {
    Axis2d::new(
        Point2::new(origin[0], origin[1]),
        Direction2::new(direction[0], direction[1]).unwrap(),
    )
}
fn report(
    name: &str,
    result: &ExtPElC2d,
    query: Point2,
    derivative: impl Fn(f64) -> (Point2, Vector2, Vector2),
) -> usize {
    let mut mismatches = 0;
    for index in 1..=result.nb_ext().unwrap() {
        let parameter = result.point(index).unwrap().parameter();
        let (position, tangent, second) = derivative(parameter);
        let variation = tangent.square_magnitude() + Vector2::between(query, position).dot(second);
        let stored = result.is_min(index).unwrap();
        let minimum = variation > 0.;
        assert!(
            variation != 0.,
            "case {name}: expected a nondegenerate stationary point"
        );
        mismatches += usize::from(stored != minimum);
        println!(
            "{{\"case\":\"{name}\",\"parameter\":{parameter},\"distance\":{},\"stored_is_min\":{stored},\"second_variation\":{variation},\"expected_is_min\":{minimum}}}",
            result.square_distance(index).unwrap().sqrt()
        );
    }
    mismatches
}
fn circle() -> usize {
    let shape = Circle2::from_x_axis(axis([0.; 2], [1., 0.]), 1., true).unwrap();
    let query = Point2::new(2., 0.);
    let result = ExtPElC2d::with_circle(
        query,
        shape,
        precision::confusion(),
        0.,
        std::f64::consts::TAU,
    )
    .unwrap();
    report("circle", &result, query, |parameter| {
        elclib::d2(parameter, &shape)
    })
}
fn ellipse() -> usize {
    let shape =
        Ellipse2::from_x_axis(axis([2.7574, -6.9211], [0.6, 0.8]), 0.75793, 0.38063, true).unwrap();
    let query = Point2::new(2.9111111111111105, -6.288888888888888);
    let result = ExtPElC2d::with_ellipse(
        query,
        shape,
        precision::confusion(),
        0.,
        std::f64::consts::TAU,
    )
    .unwrap();
    report("rotated-ellipse", &result, query, |parameter| {
        elclib::d2(parameter, &shape)
    })
}
fn hyperbola() -> usize {
    let shape = Hyperbola2::from_x_axis(axis([0.; 2], [1., 0.]), 2., 1., true).unwrap();
    let query = Point2::new(3., 0.);
    let result = ExtPElC2d::with_hyperbola(query, shape, precision::confusion(), -2., 2.).unwrap();
    report("hyperbola", &result, query, |parameter| {
        elclib::d2(parameter, &shape)
    })
}
fn parabola() -> usize {
    let shape = Parabola2::from_mirror_axis(axis([0.; 2], [1., 0.]), 1., true).unwrap();
    let query = Point2::new(3., 0.);
    let result = ExtPElC2d::with_parabola(query, shape, precision::confusion(), -4., 4.).unwrap();
    report("parabola", &result, query, |parameter| {
        elclib::d2(parameter, &shape)
    })
}
fn main() {
    let mismatches = circle() + ellipse() + hyperbola() + parabola();
    assert_eq!(
        mismatches, 0,
        "SDK extrema flags: expected every nondegenerate root to have its geometric minimum classification"
    );
}
