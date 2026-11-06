// SPDX-License-Identifier: GPL-2.0-only
//! Bounded regions of a sketch's profile curves, from the kernel's general fuse and face builder.
use crate::planar_curve::{
    PlanarGeometry,
    sketch::{SketchCurveFailure, measure_sketch_curve, resolve_sketch_curve},
};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
use rustycad_tkbo::{bop_algo::BuilderFace, brep_algo_api::BRepAlgoAPIBuilderAlgo};
use rustycad_tkbrep::{
    topexp,
    topods::{Edge, Shape, edge, face},
    toptools::IndexedMapOfShape,
};
use rustycad_tkernel::{ncollection::List, standard::Failure};
use rustycad_tkg3d::top_abs::{Orientation, ShapeEnum, State};
use rustycad_tkgeombase::gprop::GProps;
use rustycad_tkmath::gp::{Axis2, Circle3, Direction3, Ellipse3, Plane3, Point2, Point3};
use rustycad_tktopalgo::{
    brep_builder_api::{BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeFace},
    brep_gprop::BRepGProp,
    brep_lib,
    brep_top_adaptor::BRepTopAdaptorFClass2d,
};
use std::fmt;

/// What one bounded region measures, in the sketch's own coordinates and working units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchRegionMeasure {
    pub area: f64,
    pub perimeter: f64,
    pub centroid: [f64; 2],
}
/// A region could not be built; the text names the input and what was expected.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchRegionFailure(String);
impl fmt::Display for SketchRegionFailure {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        output.write_str(&self.0)
    }
}
impl std::error::Error for SketchRegionFailure {}
impl From<Failure> for SketchRegionFailure {
    fn from(cause: Failure) -> Self {
        Self(format!("sketch region kernel operation: {cause}"))
    }
}
impl From<SketchCurveFailure> for SketchRegionFailure {
    fn from(cause: SketchCurveFailure) -> Self {
        Self(format!("sketch region curve: {cause}"))
    }
}
/// tol:native classification and pcurve accuracy, in working lengths.
const CLASSIFICATION_TOLERANCE: f64 = 1e-9;

/// The smallest bounded region holding `point`, if any; open curves bound nothing.
/// Example: `sketch_region_at(&sketch, [1., 1.])?.map(|region| region.area)`.
pub fn sketch_region_at(
    sketch: &Sketch2D,
    point: [f64; 2],
) -> Result<Option<SketchRegionMeasure>, SketchRegionFailure> {
    let mut found: Option<SketchRegionMeasure> = None;
    for area in bounded_faces(sketch)? {
        if !region_holds(&area, point)? {
            continue;
        }
        let measure = measure_region(&area)?;
        if found.is_none_or(|best| measure.area < best.area) {
            found = Some(measure);
        }
    }
    Ok(found)
}
fn bounded_faces(sketch: &Sketch2D) -> Result<Vec<Shape>, SketchRegionFailure> {
    let sources = profile_edges(sketch)?;
    if sources.is_empty() {
        return Ok(Vec::new());
    }
    let split = split_edges(&sources)?;
    let plane = BRepBuilderAPIMakeFace::from_plane(Plane3::default())?.face()?;
    let mut builder = BuilderFace::new();
    builder.set_face(&plane);
    builder.set_shapes(&oriented_edges(&split, &plane)?);
    builder.perform()?;
    if builder.options().has_errors() {
        return Err(SketchRegionFailure(format!(
            "{} profile edges: expected the face builder to complete",
            sources.len()
        )));
    }
    let mut bounded = Vec::new();
    for shape in builder.areas().iter() {
        let classifier = BRepTopAdaptorFClass2d::new(&face(shape)?, CLASSIFICATION_TOLERANCE)?;
        if classifier.perform_infinite_point()? == State::Out {
            bounded.push(shape.clone());
        }
    }
    Ok(bounded)
}
/// Curves meeting at a point split there, so a divided loop yields each of its parts.
fn split_edges(sources: &[Edge]) -> Result<IndexedMapOfShape, SketchRegionFailure> {
    let mut arguments = List::default();
    for source in sources {
        arguments.append(source.clone().into_shape());
    }
    let mut fuse = BRepAlgoAPIBuilderAlgo::new();
    fuse.set_arguments(&arguments);
    fuse.set_non_destructive(true);
    fuse.build()?;
    if fuse.has_errors() {
        return Err(SketchRegionFailure(format!(
            "{} profile edges: expected the general fuse to complete",
            sources.len()
        )));
    }
    let mut split = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(fuse.shape(), ShapeEnum::Edge, &mut split)?;
    Ok(split)
}
fn oriented_edges(
    split: &IndexedMapOfShape,
    plane: &rustycad_tkbrep::topods::Face,
) -> Result<List<Shape>, SketchRegionFailure> {
    let mut inputs = List::default();
    for index in 1..=split.extent() {
        let segment = edge(split.find_key(index)?)?;
        brep_lib::build_pcurve_for_edge_on_plane(&segment, plane)?;
        inputs.append(segment.oriented(Orientation::Forward));
        inputs.append(segment.oriented(Orientation::Reversed));
    }
    Ok(inputs)
}
fn region_holds(area: &Shape, point: [f64; 2]) -> Result<bool, SketchRegionFailure> {
    let classifier = BRepTopAdaptorFClass2d::new(&face(area)?, CLASSIFICATION_TOLERANCE)?;
    Ok(classifier.perform(Point2::new(point[0], point[1]), false)? == State::In)
}
fn measure_region(area: &Shape) -> Result<SketchRegionMeasure, SketchRegionFailure> {
    let mut surface = GProps::default();
    BRepGProp::surface_properties(area, &mut surface, false, false)?;
    let mut boundary = GProps::default();
    BRepGProp::linear_properties(area, &mut boundary, false, false)?;
    let center = surface.centre_of_mass();
    Ok(SketchRegionMeasure {
        area: surface.mass(),
        perimeter: boundary.mass(),
        centroid: [center.x(), center.y()],
    })
}
fn profile_edges(sketch: &Sketch2D) -> Result<Vec<Edge>, SketchRegionFailure> {
    sketch
        .profile_curve_ids()
        .into_iter()
        .map(|entity| profile_edge(sketch, entity))
        .collect()
}
fn profile_edge(sketch: &Sketch2D, entity: SketchEntityId) -> Result<Edge, SketchRegionFailure> {
    let definition = resolve_sketch_curve(sketch, entity)?;
    let measure = measure_sketch_curve(sketch, entity)?;
    let ends = measure.endpoints;
    Ok(match (&definition.geometry, ends) {
        (PlanarGeometry::Line { .. }, Some([first, last])) => {
            BRepBuilderAPIMakeEdge::from_points(model_point(first), model_point(last))?.edge()?
        }
        (PlanarGeometry::Circle { frame, radius }, ends) => {
            let circle = Circle3::new(axis(frame.center)?, *radius)?;
            match ends {
                // The kernel's circle runs counterclockwise; a clockwise arc swaps its ends.
                Some([first, last]) if frame.counterclockwise => {
                    BRepBuilderAPIMakeEdge::from_circ_points(
                        circle,
                        model_point(first),
                        model_point(last),
                    )?
                    .edge()?
                }
                Some([first, last]) => BRepBuilderAPIMakeEdge::from_circ_points(
                    circle,
                    model_point(last),
                    model_point(first),
                )?
                .edge()?,
                None => BRepBuilderAPIMakeEdge::from_circ(circle)?.edge()?,
            }
        }
        (PlanarGeometry::Ellipse { frame, radii }, None) => {
            let axes = Axis2::with_x_direction(
                model_point(frame.center),
                Direction3::new(0., 0., 1.)?,
                Direction3::new(frame.axis[0], frame.axis[1], 0.)?,
            )?;
            BRepBuilderAPIMakeEdge::from_elips(Ellipse3::new(axes, radii[0], radii[1])?)?.edge()?
        }
        (geometry, _) => {
            return Err(SketchRegionFailure(format!(
                "sketch curve {entity:?} {geometry:?}: expected a line, circle, arc or ellipse"
            )));
        }
    })
}
fn axis(center: [f64; 2]) -> Result<Axis2, Failure> {
    Axis2::new(model_point(center), Direction3::new(0., 0., 1.)?)
}
fn model_point(point: [f64; 2]) -> Point3 {
    Point3::new(point[0], point[1], 0.)
}

#[cfg(test)]
#[path = "../tests/sketch_region_unit.rs"]
mod tests;
