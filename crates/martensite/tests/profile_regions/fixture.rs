// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkbrep::brep::BRepBuilder;
use rustycad_tkmath::gp::{Axis2, Circle3, Direction3, Ellipse3, Point3};
use rustycad_tktopalgo::brep_builder_api::BRepBuilderAPIMakeEdge;
/// Build fresh analytic edges, including separate coincident endpoints.
/// Example: `let input = build_region_fixture("divider")?;`.
pub(super) fn build_region_fixture(case: &str) -> Result<Shape, Failure> {
    let mut shape = rustycad_tkbrep::topods::Compound::default();
    let builder = BRepBuilder::default();
    builder.make_compound(&mut shape);
    let mut shape = shape.into_shape();
    append_region_conic(&builder, &mut shape, case)?;
    if !case.starts_with("single-") && !case.starts_with("pair-") {
        append_region_lines(&builder, &mut shape, case)?;
    }
    Ok(shape)
}
fn append_region_lines(
    builder: &BRepBuilder,
    shape: &mut Shape,
    case: &str,
) -> Result<(), Failure> {
    append_region_loop(builder, shape, &[[0., 0.], [4., 0.], [4., 3.], [0., 3.]])?;
    if case == "nested" {
        append_region_loop(builder, shape, &[[1., 1.], [3., 1.], [3., 2.], [1., 2.]])?;
    }
    if case == "divider" || case == "circle-divider" {
        append_region_segment(builder, shape, [2., 0.], [2., 3.])?;
    }
    if case == "crossing" {
        append_region_segment(builder, shape, [0., 0.], [4., 3.])?;
    }
    Ok(())
}
fn append_region_conic(
    builder: &BRepBuilder,
    shape: &mut Shape,
    case: &str,
) -> Result<(), Failure> {
    for [x, y, major, minor] in region_conic_parameters(case) {
        let axes = Axis2::new(Point3::new(x, y, 0.), Direction3::new(0., 0., 1.)?)?;
        let segment = if major == minor {
            BRepBuilderAPIMakeEdge::from_circ(Circle3::new(axes, major)?)?.edge()?
        } else {
            BRepBuilderAPIMakeEdge::from_elips(Ellipse3::new(axes, major, minor)?)?.edge()?
        };
        builder.add(shape, &segment)?;
    }
    Ok(())
}
fn region_conic_parameters(case: &str) -> Vec<[f64; 4]> {
    match case {
        "circle" | "single-circle" | "circle-divider" => vec![[2., 1.5, 0.5, 0.5]],
        "ellipse" | "single-ellipse" => vec![[2., 1.5, 0.7, 0.4]],
        "circle-tangent" => vec![[5., 1.5, 1., 1.]],
        "pair-separated" => vec![[0., 0., 1., 1.], [2.0001, 0., 1., 1.]],
        "pair-tangent" => vec![[0., 0., 1., 1.], [2., 0., 1., 1.]],
        "pair-near-overlap" => vec![[0., 0., 1., 1.], [1.9999, 0., 1., 1.]],
        "circle-boundary" => vec![[4., 1.5, 1., 1.]],
        "ellipse-boundary" => vec![[4., 1.5, 1., 0.6]],
        "overlap-circles" => vec![[1.5, 1.5, 1., 1.], [2.5, 1.5, 1., 1.]],
        _ => Vec::new(),
    }
}

fn append_region_loop(
    builder: &BRepBuilder,
    shape: &mut Shape,
    points: &[[f64; 2]],
) -> Result<(), Failure> {
    for index in 0..points.len() {
        append_region_segment(
            builder,
            shape,
            points[index],
            points[(index + 1) % points.len()],
        )?;
    }
    Ok(())
}
fn append_region_segment(
    builder: &BRepBuilder,
    shape: &mut Shape,
    start: [f64; 2],
    end: [f64; 2],
) -> Result<(), Failure> {
    let a = Point3::new(start[0], start[1], 0.);
    let b = Point3::new(end[0], end[1], 0.);
    let segment = BRepBuilderAPIMakeEdge::from_points(a, b)?.edge()?;
    builder.add(shape, &segment)
}
