// SPDX-License-Identifier: GPL-2.0-only
//! Build certified solid and surface revolutions from a resolved sketch region and axis.
use super::{
    RevolveFaceRole, cap_lineage, model_axis, require_completed, require_face,
    require_face_coverage,
};
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase, PrismAchievedTolerance, PrismResolution, PrismResolutionPolicy,
    sketch_prism::regions::resolved_regions,
};
use martensite_tkbo::profile_extrusion::{PlanarRegion, PlanarRegionSet};
use martensite_tkbrep::{brep_tools::outer_wire, topods::Shape};
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::{Axis1, Transform3};
use martensite_tkprim::{
    brep_prim_api::{BRepPrimAPIMakeRevol, BRepPrimAPIMakeSweep},
    brep_sweep::BRepSweepRevol,
};
use martensite_tktopalgo::{
    brep_builder_api::{BRepBuilderAPIMakeShape, BRepBuilderAPITransform},
    brep_lib,
};
use oblikovati_model::feature::{
    sketch_prism::SketchPrismSelection,
    sketch_revolve::{ResolvedSketchRevolve, SketchRevolveDirection},
};
use std::f64::consts::TAU;

const FULL_TURN_EPSILON: f64 = 1e-9;

/// A privately constructed revolved solid with verified face lineage.
pub struct RevolveCandidate {
    pub(crate) shape: Shape,
    pub(crate) faces: Vec<(RevolveFaceRole, Shape)>,
    pub(crate) resolution: PrismResolution,
    achieved: PrismAchievedTolerance,
}

/// A privately constructed revolved surface shell with verified face and trim lineage.
pub struct SurfaceRevolveCandidate {
    pub(crate) shell: Shape,
    pub(crate) faces: Vec<(RevolveFaceRole, Shape)>,
    pub(crate) start_trim: Option<Shape>,
    pub(crate) end_trim: Option<Shape>,
    pub(crate) resolution: PrismResolution,
    achieved: PrismAchievedTolerance,
}

impl RevolveCandidate {
    /// Read measured B-rep tolerances. Example: `candidate.achieved_tolerance().edge`.
    pub fn achieved_tolerance(&self) -> PrismAchievedTolerance {
        self.achieved
    }
}

impl SurfaceRevolveCandidate {
    /// Read measured B-rep tolerances. Example: `candidate.achieved_tolerance().edge`.
    pub fn achieved_tolerance(&self) -> PrismAchievedTolerance {
        self.achieved
    }
}

struct RevolveSpan {
    axis: Axis1,
    start_offset: f64,
    total_angle: f64,
    full: bool,
}

/// Build a checked solid revolution from a resolved sketch profile, axis, and extent.
/// Example: `build_revolve(input, policy)?`.
pub fn build_revolve(
    input: ResolvedSketchRevolve<'_>,
    policy: PrismResolutionPolicy,
) -> Result<RevolveCandidate, EvaluationFailure> {
    let plane = input.sketch.plane;
    let resolution = policy.for_sketch(&input.sketch)?;
    let regions = resolved_regions(input.sketch, PrismResolutionPolicy::Fixed(resolution))?;
    let region = selected_region(regions.as_ref(), input.selection)?;
    let span = resolve_span(
        plane,
        input.axis,
        input.angle,
        input.second_angle,
        input.full,
        input.direction,
    )?;
    let edges: Vec<_> = region
        .edges()
        .iter()
        .map(|(id, edge)| (id.0, edge.as_ref().clone()))
        .collect();
    let (profile, edges) =
        rotate_source(region.face().as_ref(), &edges, span.axis, span.start_offset)?;
    let (shape, faces) = if span.full {
        build_full_shape(&profile, &edges, span.axis, true)?
    } else {
        build_partial_solid(&profile, &edges, span.axis, span.total_angle)?
    };
    checked_candidate(shape, faces, resolution)
}

/// Build a checked surface revolution shell from the profile's outer wire.
/// Example: `build_surface_revolve(input, policy)?`.
pub fn build_surface_revolve(
    input: ResolvedSketchRevolve<'_>,
    policy: PrismResolutionPolicy,
) -> Result<SurfaceRevolveCandidate, EvaluationFailure> {
    let plane = input.sketch.plane;
    let resolution = policy.for_sketch(&input.sketch)?;
    let regions = resolved_regions(input.sketch, PrismResolutionPolicy::Fixed(resolution))?;
    let region = selected_region(regions.as_ref(), input.selection)?;
    let span = resolve_span(
        plane,
        input.axis,
        input.angle,
        input.second_angle,
        input.full,
        input.direction,
    )?;
    let wire = outer_wire(region.face()).map_err(|cause| {
        Phase::Profile.native(
            Entity::Body,
            "surface revolve: expected closed profile wire".into(),
            cause,
        )
    })?;
    let edges: Vec<_> = region
        .edges()
        .iter()
        .map(|(id, edge)| (id.0, edge.as_ref().clone()))
        .collect();
    let (profile, edges) = rotate_source(wire.as_ref(), &edges, span.axis, span.start_offset)?;
    let (shell, faces, start_trim, end_trim) = if span.full {
        let (shape, faces) = build_full_shape(&profile, &edges, span.axis, false)?;
        (shape, faces, None, None)
    } else {
        build_partial_surface(&profile, &edges, span.axis, span.total_angle)?
    };
    let kind = shell.shape_type().map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            "surface revolve: expected shell type".into(),
            cause,
        )
    })?;
    if !matches!(kind, ShapeEnum::Shell | ShapeEnum::Face) {
        return Err(Phase::Prism.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("surface revolve result {kind:?}: expected a shell"),
        ));
    }
    crate::checked_shape::require_geometry(&shell, "surface revolve")?;
    let achieved = crate::checked_shape::measured_tolerances(
        &shell,
        resolution.maximum_tolerance,
        "surface revolve",
    )?;
    Ok(SurfaceRevolveCandidate {
        shell,
        faces,
        start_trim,
        end_trim,
        resolution,
        achieved,
    })
}

fn resolve_span(
    plane: oblikovati_model::sketch::SketchPlane,
    endpoints: [[f64; 2]; 2],
    angle: f64,
    second_angle: Option<f64>,
    full: bool,
    direction: SketchRevolveDirection,
) -> Result<RevolveSpan, EvaluationFailure> {
    if endpoints.iter().flatten().any(|value| !value.is_finite()) || endpoints[0] == endpoints[1] {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            format!("revolve axis {endpoints:?}: expected two distinct finite sketch positions"),
        ));
    }
    let axis = model_axis(plane, endpoints)?;
    if full {
        return Ok(RevolveSpan {
            axis,
            start_offset: 0.,
            total_angle: TAU,
            full: true,
        });
    }
    if !angle.is_finite() || !(0. < angle && angle <= TAU + FULL_TURN_EPSILON) {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            format!("revolve angle {angle:?}: expected a finite angle in (0, 2π]"),
        ));
    }
    match direction {
        SketchRevolveDirection::Positive => Ok(span_from_total(axis, 0., angle)),
        SketchRevolveDirection::Negative => Ok(span_from_total(axis.reversed(), 0., angle)),
        SketchRevolveDirection::Symmetric => Ok(span_from_total(axis, -0.5 * angle, angle)),
        SketchRevolveDirection::Asymmetric => {
            let second = second_angle.ok_or_else(|| {
                Phase::Input.refused(
                    Kind::InvalidInput,
                    Entity::Body,
                    "revolve asymmetric: expected second angle".into(),
                )
            })?;
            if !second.is_finite() || second <= 0. {
                return Err(Phase::Input.refused(
                    Kind::InvalidInput,
                    Entity::Body,
                    format!("revolve second angle {second:?}: expected a positive finite angle"),
                ));
            }
            let total = angle + second;
            if total > TAU + FULL_TURN_EPSILON {
                return Err(Phase::Input.refused(
                    Kind::InvalidInput,
                    Entity::Body,
                    format!("revolve asymmetric total {total:?}: expected combined angle <= 2π"),
                ));
            }
            Ok(span_from_total(axis, -second, total))
        }
    }
}

fn span_from_total(axis: Axis1, start_offset: f64, total_angle: f64) -> RevolveSpan {
    let full = (total_angle - TAU).abs() <= FULL_TURN_EPSILON;
    RevolveSpan {
        axis,
        start_offset: if full { 0. } else { start_offset },
        total_angle: if full { TAU } else { total_angle },
        full,
    }
}

fn rotate_source(
    source: &Shape,
    edges: &[(u64, Shape)],
    axis: Axis1,
    offset: f64,
) -> Result<(Shape, Vec<(u64, Shape)>), EvaluationFailure> {
    if offset.abs() <= FULL_TURN_EPSILON {
        return Ok((source.clone(), edges.to_vec()));
    }
    let mut transform = Transform3::default();
    transform.set_rotation(axis, offset).map_err(|cause| {
        Phase::Placement.native(Entity::Body, "revolve offset rotation".into(), cause)
    })?;
    let mut moved =
        BRepBuilderAPITransform::with_shape(source, transform, true, false).map_err(|cause| {
            Phase::Placement.native(Entity::Body, "revolve profile rotation".into(), cause)
        })?;
    let rotated = moved
        .shape()
        .map_err(|cause| Phase::Placement.native(Entity::Body, "rotated profile".into(), cause))?
        .clone();
    let mut rotated_edges = Vec::with_capacity(edges.len());
    for (id, edge) in edges {
        let mapped = moved.modified_shape(edge).map_err(|cause| {
            Phase::History.native(
                Entity::Edge(*id),
                format!("revolve rotated edge {id}: expected transformed edge"),
                cause,
            )
        })?;
        rotated_edges.push((*id, mapped));
    }
    Ok((rotated, rotated_edges))
}

fn build_full_shape(
    profile: &Shape,
    edges: &[(u64, Shape)],
    axis: Axis1,
    is_solid: bool,
) -> Result<(Shape, Vec<(RevolveFaceRole, Shape)>), EvaluationFailure> {
    let validator = BRepPrimAPIMakeRevol::full(profile, axis, false).map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            "revolve full: expected valid revolution".into(),
            cause,
        )
    })?;
    require_completed(&validator, TAU)?;
    let mut sweep = BRepSweepRevol::full(profile, axis, false).map_err(|cause| {
        Phase::Prism.native(Entity::Body, "revolve full: expected sweep".into(), cause)
    })?;
    let shape = sweep.shape().map_err(|cause| {
        Phase::Prism.native(Entity::Body, "revolve full: expected shape".into(), cause)
    })?;
    brep_lib::update_inner_tolerances(&shape).map_err(|cause| {
        Phase::Validation.native(
            Entity::Body,
            "revolve full: expected updated tolerances".into(),
            cause,
        )
    })?;
    let mut faces = Vec::new();
    for (curve, edge) in edges {
        let generated = sweep.shape_of(edge).map_err(|cause| {
            Phase::History.native(
                Entity::Edge(*curve),
                format!("revolve edge {curve}: expected sweep history"),
                cause,
            )
        })?;
        if generated.is_null() {
            continue;
        }
        if !is_solid && generated.shape_type().ok() != Some(ShapeEnum::Face) {
            continue;
        }
        faces.push((
            RevolveFaceRole::ProfileEdge(*curve),
            require_face(generated, "revolve full side")?,
        ));
    }
    require_face_coverage(&shape, &faces)?;
    Ok((shape, faces))
}

fn build_partial_solid(
    profile: &Shape,
    edges: &[(u64, Shape)],
    axis: Axis1,
    angle: f64,
) -> Result<(Shape, Vec<(RevolveFaceRole, Shape)>), EvaluationFailure> {
    let mut maker = BRepPrimAPIMakeRevol::new(profile, axis, angle, false).map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            format!("revolve angle {angle:?}: expected constructed solid"),
            cause,
        )
    })?;
    require_completed(&maker, angle)?;
    let shape = maker.shape().cloned().map_err(|cause| {
        Phase::Prism.native(Entity::Body, "revolve result: expected shape".into(), cause)
    })?;
    let mut faces = cap_lineage(&mut maker)?;
    collect_partial_side_faces(&mut maker, edges, &mut faces)?;
    require_face_coverage(&shape, &faces)?;
    Ok((shape, faces))
}

type PartialSurfaceParts = (
    Shape,
    Vec<(RevolveFaceRole, Shape)>,
    Option<Shape>,
    Option<Shape>,
);

fn build_partial_surface(
    profile: &Shape,
    edges: &[(u64, Shape)],
    axis: Axis1,
    angle: f64,
) -> Result<PartialSurfaceParts, EvaluationFailure> {
    let mut maker = BRepPrimAPIMakeRevol::new(profile, axis, angle, false).map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            format!("surface revolve angle {angle:?}: expected constructed shell"),
            cause,
        )
    })?;
    require_completed(&maker, angle)?;
    let shell = maker.shape().cloned().map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            "surface revolve: expected shell".into(),
            cause,
        )
    })?;
    let mut faces = Vec::new();
    collect_partial_side_faces(&mut maker, edges, &mut faces)?;
    require_face_coverage(&shell, &faces)?;
    let start = maker.first_shape().map_err(|cause| {
        Phase::History.native(Entity::Body, "surface revolve start trim".into(), cause)
    })?;
    let end = maker.last_shape().map_err(|cause| {
        Phase::History.native(Entity::Body, "surface revolve end trim".into(), cause)
    })?;
    Ok((shell, faces, Some(start), Some(end)))
}

fn collect_partial_side_faces(
    maker: &mut BRepPrimAPIMakeRevol,
    edges: &[(u64, Shape)],
    faces: &mut Vec<(RevolveFaceRole, Shape)>,
) -> Result<(), EvaluationFailure> {
    for (curve, edge) in edges {
        let generated = maker.generated(edge).map_err(|cause| {
            Phase::History.native(
                Entity::Edge(*curve),
                format!("revolve edge {curve}: expected generated face"),
                cause,
            )
        })?;
        let face_list: Vec<_> = generated
            .iter()
            .filter(|shape| shape.shape_type().ok() == Some(ShapeEnum::Face))
            .cloned()
            .collect();
        match face_list.len() {
            0 => continue,
            1 => faces.push((
                RevolveFaceRole::ProfileEdge(*curve),
                require_face(face_list[0].clone(), "revolve side")?,
            )),
            count => {
                return Err(Phase::History.refused(
                    Kind::IncompleteHistory,
                    Entity::Edge(*curve),
                    format!("revolve edge {curve} generated {count} faces: expected 0 or 1"),
                ));
            }
        }
    }
    Ok(())
}

fn checked_candidate(
    shape: Shape,
    faces: Vec<(RevolveFaceRole, Shape)>,
    resolution: PrismResolution,
) -> Result<RevolveCandidate, EvaluationFailure> {
    let kind = shape.shape_type().map_err(|cause| {
        Phase::Prism.native(
            Entity::Body,
            "revolve result: expected body type".into(),
            cause,
        )
    })?;
    if kind != ShapeEnum::Solid {
        return Err(Phase::Prism.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("revolve result {kind:?}: expected a solid"),
        ));
    }
    crate::checked_shape::require_geometry(&shape, "revolve result")?;
    let achieved =
        crate::checked_shape::measured_tolerances(&shape, resolution.maximum_tolerance, "revolve")?;
    Ok(RevolveCandidate {
        shape,
        faces,
        resolution,
        achieved,
    })
}

fn selected_region<'a>(
    regions: Option<&'a PlanarRegionSet>,
    selection: &SketchPrismSelection,
) -> Result<&'a PlanarRegion, EvaluationFailure> {
    let mut matches = regions
        .into_iter()
        .flat_map(|regions| regions.regions())
        .filter(|region| super::matches_selection(region, selection));
    if let (Some(region), None) = (matches.next(), matches.next()) {
        return Ok(region);
    }
    Err(Phase::Profile.refused(
        Kind::InvalidInput,
        Entity::Body,
        format!(
            "revolve profile {selection:?}: expected one current region with the saved boundary"
        ),
    ))
}
