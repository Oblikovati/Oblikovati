// SPDX-License-Identifier: GPL-2.0-only
//! Build and check a Loft body from its authored circle sections.
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase, PrismAchievedTolerance, PrismResolution, PrismResolutionPolicy,
};
use martensite_tkbrep::{
    brep::BRepTool,
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
use oblikovati_model::feature::sketch_loft::{
    ResolvedLoftCircle, ResolvedSketchLoft, SketchLoftOutput,
};

pub(crate) struct LoftCandidate {
    pub(crate) shape: Shape,
    pub(crate) history: LoftTopology,
    pub(crate) resolution: PrismResolution,
    pub(crate) achieved: PrismAchievedTolerance,
}

pub(crate) enum LoftTopology {
    Solid { caps: [Shape; 2], wall: Shape },
    Surface { trims: [Shape; 2], wall: Shape },
}

pub(crate) fn build_loft(
    input: &ResolvedSketchLoft,
    policy: PrismResolutionPolicy,
) -> Result<LoftCandidate, EvaluationFailure> {
    let [first, last] = &input.sections;
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
    let solid = input.output == SketchLoftOutput::Solid;
    let mut maker = BRepOffsetAPIThruSections::new(solid, false, resolution.maximum_tolerance);
    // Surface sections are newly created inside this evaluation. Retaining
    // their native edges gives both open trims exact section identities.
    maker.set_mutable_input(!solid);
    for edge in &edges {
        let wire = BRepBuilderAPIMakeWire::from_edge(edge)
            .and_then(|maker| maker.wire())
            .map_err(|cause| native(Phase::Profile, "section wire", cause))?;
        maker.add_wire(&wire);
    }
    let shape = maker
        .shape()
        .cloned()
        .map_err(|cause| native(Phase::Loft, "result body", cause))?;
    require_body_kind(&shape, solid)?;
    crate::checked_shape::require_geometry(&shape, "loft result")?;
    let achieved =
        crate::checked_shape::measured_tolerances(&shape, resolution.maximum_tolerance, "loft")?;
    let first_wall = wall(&mut maker, &edges[0])?;
    let last_wall = wall(&mut maker, &edges[1])?;
    if !first_wall.is_same(&last_wall) {
        return Err(history_failure("both sections must generate the same wall"));
    }
    let history = if solid {
        let caps = [
            maker.first_shape().as_ref().clone(),
            maker.last_shape().as_ref().clone(),
        ];
        verify_faces(&shape, &[&caps[0], &caps[1], &first_wall])?;
        LoftTopology::Solid {
            caps,
            wall: first_wall,
        }
    } else {
        verify_faces(&shape, &[&first_wall])?;
        let trims = surface_trims(&shape, edges)?;
        LoftTopology::Surface {
            trims,
            wall: first_wall,
        }
    };
    Ok(LoftCandidate {
        shape,
        history,
        resolution,
        achieved,
    })
}

fn require_body_kind(shape: &Shape, solid: bool) -> Result<(), EvaluationFailure> {
    let actual = shape
        .shape_type()
        .map_err(|cause| native(Phase::Validation, "body type", cause))?;
    let expected = if solid {
        ShapeEnum::Solid
    } else {
        ShapeEnum::Shell
    };
    if actual != expected {
        return Err(Phase::Validation.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("loft result {actual:?}: expected {expected:?}"),
        ));
    }
    if !solid
        && BRepTool::is_closed(shape)
            .map_err(|cause| native(Phase::Validation, "shell closure", cause))?
    {
        return Err(Phase::Validation.refused(
            Kind::InvalidResult,
            Entity::Body,
            "loft: expected an open shell".into(),
        ));
    }
    Ok(())
}

fn verify_faces(shape: &Shape, sources: &[&Shape]) -> Result<(), EvaluationFailure> {
    let actual = Explorer::new(shape, ShapeEnum::Face, ShapeEnum::Shape)
        .map_err(|cause| native(Phase::History, "result faces", cause))?
        .collect::<Vec<_>>();
    if actual.len() != sources.len()
        || actual
            .iter()
            .any(|face| sources.iter().filter(|source| source.is_same(face)).count() != 1)
    {
        return Err(history_failure(
            "expected history to cover each result face exactly once",
        ));
    }
    Ok(())
}

fn surface_trims(shell: &Shape, original: [Edge; 2]) -> Result<[Shape; 2], EvaluationFailure> {
    let edges = Explorer::new(shell, ShapeEnum::Edge, ShapeEnum::Shape)
        .map_err(|cause| native(Phase::History, "shell edges", cause))?
        .collect::<Vec<_>>();
    let [first, last] = original.map(|edge| edge.as_ref().clone());
    if first.is_same(&last)
        || [&first, &last]
            .into_iter()
            .any(|source| edges.iter().filter(|edge| source.is_same(edge)).count() != 1)
    {
        return Err(history_failure(
            "expected two distinct source edges on the shell boundary",
        ));
    }
    Ok([first, last])
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
