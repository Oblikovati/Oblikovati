// SPDX-License-Identifier: GPL-2.0-only
//! Exact planar-profile revolutions backed by Martensite construction history.
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase, FaceProbe, PlanarProfileLoop, ReleaseProbeProperties,
    TopologySummary,
};
use martensite_tkbrep::{topexp, topods::Shape, toptools::IndexedMapOfShape};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::{Axis1, Direction3, Point3};
use martensite_tkprim::brep_prim_api::{BRepPrimAPIMakeRevol, BRepPrimAPIMakeSweep};
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
use oblikovati_model::sketch::SketchPlane;
use std::f64::consts::TAU;

/// One planar profile revolution in working units with an axis on its sketch plane.
#[derive(Clone, Debug, PartialEq)]
pub struct RevolveProbeDefinition {
    pub outer: PlanarProfileLoop,
    pub holes: Vec<PlanarProfileLoop>,
    pub axis: [[f64; 2]; 2],
    pub angle: f64,
}

/// Semantic face provenance derived from the Revolve construction history.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum RevolveFaceRole {
    Start,
    End,
    ProfileEdge(u64),
}

/// Analytic result of one exact profile revolution.
#[derive(Clone, Debug, PartialEq)]
pub struct RevolveProbeReport {
    pub properties: ReleaseProbeProperties,
    pub topology: TopologySummary,
    pub faces: Vec<FaceProbe<RevolveFaceRole>>,
}

/// Check one XY-plane revolution; example: `probe_planar_revolve(&definition)?`.
pub fn probe_planar_revolve(
    definition: &RevolveProbeDefinition,
) -> Result<RevolveProbeReport, String> {
    probe_revolve_on_plane(definition, SketchPlane::xy()).map_err(|cause| cause.to_string())
}

/// Check one revolution on a validated sketch plane; example: `probe_revolve_on_plane(&definition, plane)?`.
pub fn probe_revolve_on_plane(
    definition: &RevolveProbeDefinition,
    plane: SketchPlane,
) -> Result<RevolveProbeReport, EvaluationFailure> {
    validate_definition(definition)?;
    crate::prism::input::validate_profile(&definition.outer, &definition.holes)?;
    let profile =
        crate::prism::profile::build_profile(&definition.outer, &definition.holes, plane)?;
    let axis = model_axis(plane, definition.axis)?;
    let mut maker = construct_revolution(&profile.face.into_shape(), axis, definition.angle)?;
    let shape = completed_revolution(&mut maker, definition.angle)?;
    crate::checked_shape::require_geometry(&shape, "revolve result")?;
    let lineage = construction_lineage(&mut maker, &profile.edges, &shape)?;
    checked_report(&shape, &lineage)
}

fn validate_definition(definition: &RevolveProbeDefinition) -> Result<(), EvaluationFailure> {
    let axis = definition.axis;
    if axis.iter().flatten().any(|value| !value.is_finite()) || axis[0] == axis[1] {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            format!("revolve axis {axis:?}: expected two distinct finite sketch-plane positions"),
        ));
    }
    let angle = definition.angle;
    if !angle.is_finite() || !(0. < angle && angle < TAU) {
        return Err(Phase::Input.refused(Kind::InvalidInput, Entity::Body, format!(
            "revolve angle {angle:?}: expected a finite positive partial angle below one turn {TAU:?}; full turns require complete source-edge history"
        )));
    }
    Ok(())
}

fn model_axis(plane: SketchPlane, endpoints: [[f64; 2]; 2]) -> Result<Axis1, EvaluationFailure> {
    let origin = model_point(plane, endpoints[0])?;
    let end = model_point(plane, endpoints[1])?;
    let direction = Direction3::new(
        end.x() - origin.x(),
        end.y() - origin.y(),
        end.z() - origin.z(),
    )
    .map_err(|cause| {
        Phase::Placement.native(
            Entity::Body,
            format!("revolve axis {endpoints:?}: expected a usable model-space direction"),
            cause,
        )
    })?;
    Ok(Axis1::new(origin, direction))
}

fn model_point(plane: SketchPlane, point: [f64; 2]) -> Result<Point3, EvaluationFailure> {
    let [x, y, z] = plane
        .to_model(point)
        .map_err(|cause| Phase::Placement.sketch(Entity::Body, cause))?;
    Ok(Point3::new(x, y, z))
}

fn construct_revolution(
    profile: &Shape,
    axis: Axis1,
    angle: f64,
) -> Result<BRepPrimAPIMakeRevol, EvaluationFailure> {
    let maker = BRepPrimAPIMakeRevol::new(profile, axis, angle, false);
    maker.map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            format!("revolve angle {angle:?}: expected constructed solid"),
            cause,
        )
    })
}

fn completed_revolution(
    maker: &mut BRepPrimAPIMakeRevol,
    angle: f64,
) -> Result<Shape, EvaluationFailure> {
    require_completed(maker, angle)?;
    maker.shape().cloned().map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            "revolve result: expected completed shape".into(),
            cause,
        )
    })
}

fn require_completed(maker: &BRepPrimAPIMakeRevol, angle: f64) -> Result<(), EvaluationFailure> {
    if maker.is_done() {
        return Ok(());
    }
    Err(Phase::Prism.refused(
        Kind::InvalidGeometry,
        Entity::Body,
        format!(
            "revolve angle {angle:?}: expected a profile that does not cross the rotation axis"
        ),
    ))
}

fn construction_lineage(
    maker: &mut BRepPrimAPIMakeRevol,
    edges: &[(u64, martensite_tkbrep::topods::Edge)],
    shape: &Shape,
) -> Result<Vec<(RevolveFaceRole, Shape)>, EvaluationFailure> {
    let mut lineage = cap_lineage(maker)?;
    for (key, edge) in edges {
        lineage.push((
            RevolveFaceRole::ProfileEdge(*key),
            generated_face(maker, *key, edge)?,
        ));
    }
    require_face_coverage(shape, &lineage)?;
    Ok(lineage)
}

fn cap_lineage(
    maker: &mut BRepPrimAPIMakeRevol,
) -> Result<Vec<(RevolveFaceRole, Shape)>, EvaluationFailure> {
    Ok(vec![
        (
            RevolveFaceRole::Start,
            cap_face(maker.first_shape(), "revolve start")?,
        ),
        (
            RevolveFaceRole::End,
            cap_face(maker.last_shape(), "revolve end")?,
        ),
    ])
}

fn cap_face(
    result: Result<Shape, martensite_tkernel::standard::Failure>,
    context: &str,
) -> Result<Shape, EvaluationFailure> {
    let shape = result.map_err(|cause| {
        Phase::History.native(
            Entity::Body,
            format!("{context}: expected cap history"),
            cause,
        )
    })?;
    require_face(shape, context)
}

fn generated_face(
    maker: &mut BRepPrimAPIMakeRevol,
    key: u64,
    edge: &martensite_tkbrep::topods::Edge,
) -> Result<Shape, EvaluationFailure> {
    let shape = single_generated_face(maker, key, edge)?;
    require_face(shape, "revolve side")
}

fn single_generated_face(
    maker: &mut BRepPrimAPIMakeRevol,
    key: u64,
    edge: &martensite_tkbrep::topods::Edge,
) -> Result<Shape, EvaluationFailure> {
    let generated = generated_face_shape(maker, key, edge)?;
    require_one_generated_face(key, generated)
}

fn generated_face_shape(
    maker: &mut BRepPrimAPIMakeRevol,
    key: u64,
    edge: &martensite_tkbrep::topods::Edge,
) -> Result<(usize, Option<Shape>), EvaluationFailure> {
    let generated = maker.generated(edge).map_err(|cause| {
        Phase::History.native(
            Entity::Edge(key),
            format!("revolve edge {key}: expected generated face history"),
            cause,
        )
    })?;
    Ok((generated.len(), generated.first().cloned()))
}

fn require_one_generated_face(
    key: u64,
    generated: (usize, Option<Shape>),
) -> Result<Shape, EvaluationFailure> {
    if let (1, Some(shape)) = generated {
        return Ok(shape);
    }
    Err(Phase::History.refused(
        Kind::IncompleteHistory,
        Entity::Edge(key),
        format!(
            "revolve edge {key} generated {} shapes: expected exactly one side face",
            generated.0
        ),
    ))
}

fn require_face(shape: Shape, context: &str) -> Result<Shape, EvaluationFailure> {
    martensite_tkbrep::topods::face(&shape).map_err(|cause| {
        Phase::History.native(
            Entity::Body,
            format!("{context}: expected a native face"),
            cause,
        )
    })?;
    Ok(shape)
}

fn require_face_coverage(
    shape: &Shape,
    lineage: &[(RevolveFaceRole, Shape)],
) -> Result<(), EvaluationFailure> {
    let faces = result_faces(shape)?;
    require_face_count(faces.extent(), lineage.len())?;
    for index in 1..=faces.extent() {
        require_unique_face_role(&faces, index, lineage)?;
    }
    Ok(())
}

fn require_unique_face_role(
    faces: &IndexedMapOfShape,
    index: usize,
    lineage: &[(RevolveFaceRole, Shape)],
) -> Result<(), EvaluationFailure> {
    let face = faces
        .find_key(index)
        .map_err(|cause| Phase::History.native(Entity::Body, String::new(), cause))?;
    let count = lineage
        .iter()
        .filter(|(_, candidate)| candidate.is_same(face))
        .count();
    if count == 1 {
        return Ok(());
    }
    Err(Phase::History.refused(
        Kind::IncompleteHistory,
        Entity::FaceIndex(index),
        format!(
            "revolve face {index} history count={count}: expected exactly one construction role"
        ),
    ))
}

fn result_faces(shape: &Shape) -> Result<IndexedMapOfShape, EvaluationFailure> {
    let mut faces = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, ShapeEnum::Face, &mut faces)
        .map_err(|cause| Phase::History.native(Entity::Body, String::new(), cause))?;
    Ok(faces)
}

fn require_face_count(faces: usize, lineage: usize) -> Result<(), EvaluationFailure> {
    if faces == lineage {
        return Ok(());
    }
    Err(Phase::History.refused(
        Kind::IncompleteHistory,
        Entity::Body,
        format!("revolve faces={faces}, lineage={lineage}: expected complete face coverage"),
    ))
}

fn checked_report(
    shape: &Shape,
    lineage: &[(RevolveFaceRole, Shape)],
) -> Result<RevolveProbeReport, EvaluationFailure> {
    let faces = lineage
        .iter()
        .map(|(role, face)| face_probe(*role, face))
        .collect::<Result<Vec<_>, _>>()?;
    Ok(RevolveProbeReport {
        properties: crate::checked_shape::analytic_properties(shape, "revolve result")?,
        topology: crate::prism::report::topology_summary(shape)?,
        faces,
    })
}

fn face_probe(
    role: RevolveFaceRole,
    face: &Shape,
) -> Result<FaceProbe<RevolveFaceRole>, EvaluationFailure> {
    let (area, centroid, loops) =
        crate::prism::report::face_measurement(face, &format!("revolve face {role:?}"))?;
    Ok(FaceProbe {
        role,
        area,
        centroid,
        loops,
    })
}
