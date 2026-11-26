// SPDX-License-Identifier: GPL-2.0-only
//! Build and check an independent Loft solid from its authored circles.
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase, PrismAchievedTolerance, PrismResolution, PrismResolutionPolicy,
};
use martensite_tkbrep::{
    topexp::Explorer,
    topods::{Edge, Shape},
};
use martensite_tkernel::standard::Failure;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkmath::gp::{Axis2, Circle3, Direction3, Point3};
use martensite_tkoffset::brep_offset_api::BRepOffsetAPIThruSections;
use martensite_tktopalgo::brep_builder_api::{
    BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeShape, BRepBuilderAPIMakeWire,
};
use oblikovati_model::feature::sketch_loft::ResolvedLoftCircle;

pub(crate) struct LoftCandidate {
    pub(crate) shape: Shape,
    pub(crate) faces: [Shape; 3],
    pub(crate) resolution: PrismResolution,
    pub(crate) achieved: PrismAchievedTolerance,
}

pub(crate) fn build_loft(
    sections: &[ResolvedLoftCircle; 2],
    policy: PrismResolutionPolicy,
) -> Result<LoftCandidate, EvaluationFailure> {
    let [first, last] = sections;
    let separation = first
        .center
        .into_iter()
        .zip(last.center)
        .map(|(a, b)| a - b)
        .fold(0.0_f64, f64::hypot);
    let size = separation.max(2. * first.radius.max(last.radius));
    let resolution = policy.for_size(size).map_err(|cause| {
        Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            format!("loft resolution: {cause}"),
        )
    })?;
    let edges = [circle(first)?, circle(last)?];
    let mut maker = BRepOffsetAPIThruSections::new(true, false, resolution.maximum_tolerance);
    maker.set_mutable_input(false);
    for edge in &edges {
        let wire = BRepBuilderAPIMakeWire::from_edge(edge)
            .and_then(|maker| maker.wire())
            .map_err(|cause| native(Phase::Profile, "section wire", cause))?;
        maker.add_wire(&wire);
    }
    let shape = maker
        .shape()
        .cloned()
        .map_err(|cause| native(Phase::Loft, "solid", cause))?;
    if shape
        .shape_type()
        .map_err(|cause| native(Phase::Validation, "body type", cause))?
        != ShapeEnum::Solid
    {
        return Err(Phase::Validation.refused(
            Kind::InvalidResult,
            Entity::Body,
            "loft: expected one solid".into(),
        ));
    }
    crate::checked_shape::require_geometry(&shape, "loft result")?;
    let achieved =
        crate::checked_shape::measured_tolerances(&shape, resolution.maximum_tolerance, "loft")?;
    let first_wall = wall(&mut maker, &edges[0])?;
    let last_wall = wall(&mut maker, &edges[1])?;
    if !first_wall.is_same(&last_wall) {
        return Err(history_failure("both sections must generate the same wall"));
    }
    let faces = [
        maker.first_shape().as_ref().clone(),
        maker.last_shape().as_ref().clone(),
        first_wall,
    ];
    let actual = Explorer::new(&shape, ShapeEnum::Face, ShapeEnum::Shape)
        .map_err(|cause| native(Phase::History, "result faces", cause))?
        .collect::<Vec<_>>();
    if actual.len() != faces.len()
        || actual
            .iter()
            .any(|face| faces.iter().filter(|source| source.is_same(face)).count() != 1)
    {
        return Err(history_failure(
            "expected two distinct caps and a wall covering every face",
        ));
    }
    Ok(LoftCandidate {
        shape,
        faces,
        resolution,
        achieved,
    })
}

fn circle(section: &ResolvedLoftCircle) -> Result<Edge, EvaluationFailure> {
    let [x, y, z] = section.center;
    let [nx, ny, nz] = section.normal;
    let [xx, xy, xz] = section.x_direction;
    let create = || -> Result<Edge, Failure> {
        let frame = Axis2::with_x_direction(
            Point3::new(x, y, z),
            Direction3::new(nx, ny, nz)?,
            Direction3::new(xx, xy, xz)?,
        )?;
        BRepBuilderAPIMakeEdge::from_circ(Circle3::new(frame, section.radius)?)?.edge()
    };
    create().map_err(|cause| {
        Phase::Profile.native(
            Entity::SketchEntity(section.source.circle.0),
            "loft section circle".into(),
            cause,
        )
    })
}

fn wall(maker: &mut BRepOffsetAPIThruSections, section: &Edge) -> Result<Shape, EvaluationFailure> {
    let generated = maker
        .generated(section.as_ref())
        .map_err(|cause| native(Phase::History, "generated wall", cause))?;
    generated
        .first()
        .filter(|_| generated.len() == 1)
        .cloned()
        .ok_or_else(|| history_failure("expected one generated face per circular section"))
}

fn history_failure(context: &str) -> EvaluationFailure {
    Phase::History.refused(
        Kind::IncompleteHistory,
        Entity::Body,
        format!("loft: {context}"),
    )
}

fn native(phase: Phase, context: &str, cause: Failure) -> EvaluationFailure {
    phase.native(Entity::Body, format!("loft: {context}"), cause)
}
