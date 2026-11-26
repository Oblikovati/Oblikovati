// SPDX-License-Identifier: GPL-2.0-only
//! Lift the authored analytic curves into the path sketch's current frame.
use super::{Entity, EvaluationFailure, Kind, Phase};
use crate::{
    planar_curve::{construct, sketch::resolve_sketch_curve},
    sketch_prism::SketchDirectedCurve,
};
use martensite_tkbrep::{
    brep_tools::WireExplorer,
    topods::{Edge, Wire},
};
use martensite_tkernel::standard::Failure;
use martensite_tkgeomalgo::geom_api::to_3d;
use martensite_tkmath::gp::{Axis3, Direction3, Plane3, Point3};
use martensite_tktopalgo::brep_builder_api::{
    BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeShape, BRepBuilderAPIMakeWire,
};
use oblikovati_model::{
    history::ResolvedDocumentSketch,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};

pub(super) struct NativePath {
    pub wire: Wire,
    pub edges: Vec<(SketchEntityId, Edge)>,
}

pub(super) fn build_path(
    source: ResolvedDocumentSketch<'_>,
    traversal: &[SketchDirectedCurve],
) -> Result<NativePath, EvaluationFailure> {
    let frame = native_frame(source.plane).map_err(|cause| {
        Phase::Placement.native(
            Entity::Body,
            "sweep path: expected a usable sketch frame".into(),
            cause,
        )
    })?;
    let mut maker = BRepBuilderAPIMakeWire::new();
    let mut edges = Vec::with_capacity(traversal.len());
    for member in traversal {
        let edge = path_edge(source.sketch.geometry(), *member, &frame)?;
        edges.push((
            member.curve,
            add_path_edge(&mut maker, member.curve, &edge)?,
        ));
    }
    let wire = maker.wire().map_err(|cause| {
        Phase::Sweep.native(
            Entity::Body,
            "sweep path: expected a completed connected wire".into(),
            cause,
        )
    })?;
    verify_traversal(&wire, &edges)?;
    Ok(NativePath { wire, edges })
}

fn add_path_edge(
    maker: &mut BRepBuilderAPIMakeWire,
    id: SketchEntityId,
    edge: &Edge,
) -> Result<Edge, EvaluationFailure> {
    maker
        .add_edge(edge)
        .map_err(|cause| path_failure(id, cause))?;
    if !maker.is_done() {
        return Err(path_input(
            id,
            format!(
                "sweep path curve {id:?}, native wire status {:?}: expected a connected path",
                maker.error(),
            ),
        ));
    }
    let joined = maker.edge();
    if joined.orientation() != edge.orientation() {
        return Err(path_input(
            id,
            "sweep path: joining reversed the authored direction".into(),
        ));
    }
    Ok(joined.clone())
}

fn verify_traversal(
    wire: &Wire,
    edges: &[(SketchEntityId, Edge)],
) -> Result<(), EvaluationFailure> {
    let mut explorer = WireExplorer::with_wire(wire).map_err(traversal_failure)?;
    for (id, edge) in edges {
        if !explorer.more() || !explorer.current().is_equal(edge) {
            return Err(path_input(
                *id,
                "sweep path: expected wire traversal in authored order and direction".into(),
            ));
        }
        explorer.next().map_err(traversal_failure)?;
    }
    if explorer.more() {
        return Err(Phase::Sweep.refused(
            Kind::InvalidGeometry,
            Entity::Body,
            "sweep path: found unaccounted wire edges".into(),
        ));
    }
    Ok(())
}

fn traversal_failure(cause: Failure) -> EvaluationFailure {
    Phase::Sweep.native(
        Entity::Body,
        "sweep path: expected complete wire traversal".into(),
        cause,
    )
}

fn path_edge(
    sketch: &Sketch2D,
    use_: SketchDirectedCurve,
    frame: &Plane3,
) -> Result<Edge, EvaluationFailure> {
    let id = use_.curve;
    let definition =
        resolve_sketch_curve(sketch, id).map_err(|cause| path_input(id, cause.to_string()))?;
    let planar =
        construct::build_curve(&definition).map_err(|cause| path_input(id, cause.to_string()))?;
    let curve = to_3d(&planar, frame).map_err(|cause| path_failure(id, cause))?;
    let mut edge = BRepBuilderAPIMakeEdge::from_curve(&curve)
        .and_then(|maker| maker.edge())
        .map_err(|cause| path_failure(id, cause))?;
    if use_.reversed {
        edge.reverse();
    }
    Ok(edge)
}

fn native_frame(plane: SketchPlane) -> Result<Plane3, Failure> {
    let [x, y, z] = plane.recipe().origin;
    let [nx, ny, nz] = plane.normal();
    let [xx, xy, xz] = plane.recipe().x_axis;
    Ok(Plane3::from_position(Axis3::new(
        Point3::new(x, y, z),
        Direction3::new(nx, ny, nz)?,
        Direction3::new(xx, xy, xz)?,
    )?))
}

fn path_input(id: SketchEntityId, message: String) -> EvaluationFailure {
    Phase::Input.refused(Kind::InvalidInput, Entity::SketchEntity(id.0), message)
}

fn path_failure(id: SketchEntityId, cause: Failure) -> EvaluationFailure {
    Phase::Sweep.native(
        Entity::SketchEntity(id.0),
        "sweep path: expected a connected analytic edge".into(),
        cause,
    )
}
