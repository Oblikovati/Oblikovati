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
    if !case.starts_with("single-") {
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
    if case == "divider" {
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
    let axes = Axis2::new(Point3::new(2., 1.5, 0.), Direction3::new(0., 0., 1.)?)?;
    let segment = match case {
        "circle" | "single-circle" => {
            BRepBuilderAPIMakeEdge::from_circ(Circle3::new(axes, 0.5)?)?.edge()?
        }
        "ellipse" | "single-ellipse" => {
            BRepBuilderAPIMakeEdge::from_elips(Ellipse3::new(axes, 0.7, 0.4)?)?.edge()?
        }
        _ => return Ok(()),
    };
    builder.add(shape, &segment)
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
