// SPDX-License-Identifier: GPL-2.0-only
//! Sweep a certified closed contour as a wire to produce an uncapped surface.
use super::{
    BRepOffsetAPIMakePipe, BRepTool, Entity, EvaluationFailure, Explorer, Kind, Phase,
    PrismAchievedTolerance, PrismResolution, PrismResolutionPolicy, ResolvedSketchSweep, Shape,
    ShapeEnum, SweepFaceRole, history_failure, measured_tolerances, path, require_surface_error,
    resolved_regions, selected_region, side_history, sweep_failure, verify_faces,
};
use martensite_tkbrep::{brep_tools::outer_wire, topods};
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeSweep;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
use oblikovati_model::feature::sketch_sweep::SketchSweepOrientation;

/// Checked surface topology and the history reported for its faces and trim wires.
pub struct SurfaceSweepCandidate {
    shell: Shape,
    faces: Vec<(SweepFaceRole, Shape)>,
    trims: [Shape; 2],
    achieved: PrismAchievedTolerance,
    pub(crate) resolution: PrismResolution,
}

impl SurfaceSweepCandidate {
    /// Borrow the uncapped shell. Example: `candidate.shell()` when binding named topology.
    pub fn shell(&self) -> &Shape {
        &self.shell
    }

    /// See the sketch curve and path segment that generated each face. Example: `candidate.faces()`.
    pub fn faces(&self) -> &[(SweepFaceRole, Shape)] {
        &self.faces
    }

    /// Borrow the start and end trim wires. Example: `let [start, end] = candidate.trims();`.
    pub fn trims(&self) -> &[Shape; 2] {
        &self.trims
    }

    /// Read measured B-rep tolerances. Example: `candidate.achieved_tolerance().edge`.
    pub fn achieved_tolerance(&self) -> PrismAchievedTolerance {
        self.achieved
    }
}

/// Build an uncapped Follow Path sweep. Example: `build_surface_sweep(input, policy)?`.
pub fn build_surface_sweep(
    input: ResolvedSketchSweep<'_>,
    policy: PrismResolutionPolicy,
) -> Result<SurfaceSweepCandidate, EvaluationFailure> {
    require_surface_inputs(&input)?;
    let resolution = policy.for_sketch(&input.profile)?;
    let regions = resolved_regions(input.profile, PrismResolutionPolicy::Fixed(resolution))?;
    let region = selected_region(regions.as_ref(), input.boundary)?;
    let profile = outer_wire(region.face())
        .map_err(|cause| sweep_failure("surface requires a closed profile wire", cause))?;
    let path = path::build_path(input.path, input.traversal)?;
    let mut maker = BRepOffsetAPIMakePipe::new(&path.wire, profile.as_ref())
        .map_err(|cause| sweep_failure("expected a sweep of the profile wire", cause))?;
    let shell = maker
        .shape()
        .cloned()
        .map_err(|cause| sweep_failure("expected a completed surface", cause))?;
    require_open_shell(&shell)?;
    require_surface_error(maker.error_on_surface(), resolution.maximum_tolerance)?;
    let achieved = measured_tolerances(&shell, resolution.maximum_tolerance)?;
    let faces = side_history(&mut maker, region, &path)?;
    verify_faces(&shell, &faces)?;
    let trims = trim_history(&mut maker, &shell)?;
    Ok(SurfaceSweepCandidate {
        shell,
        faces,
        trims,
        achieved,
        resolution,
    })
}

fn require_surface_inputs(input: &ResolvedSketchSweep<'_>) -> Result<(), EvaluationFailure> {
    if input.orientation == SketchSweepOrientation::FollowPath
        && input.taper == 0.
        && input.twist == 0.
        && input.boundary.holes.is_empty()
    {
        return Ok(());
    }
    Err(Phase::Sweep.refused(
        Kind::Unsupported,
        Entity::Body,
        "surface Sweep requires a single contour in Follow Path orientation with zero taper and twist".into(),
    ))
}

fn require_open_shell(shape: &Shape) -> Result<(), EvaluationFailure> {
    let shell =
        topods::shell(shape).map_err(|cause| sweep_failure("expected a surface shell", cause))?;
    let closed = BRepTool::is_closed(&shell)
        .map_err(|cause| sweep_failure("expected to check shell closure", cause))?;
    if closed {
        return Err(Phase::Validation.refused(
            Kind::InvalidResult,
            Entity::Body,
            "surface Sweep returned a closed shell".into(),
        ));
    }
    crate::checked_shape::require_geometry(shape, "surface Sweep")
}

fn trim_history(
    maker: &mut BRepOffsetAPIMakePipe,
    shell: &Shape,
) -> Result<[Shape; 2], EvaluationFailure> {
    let failure = |cause| history_failure(Entity::Body, cause);
    let start = maker.first_shape().map_err(failure)?;
    let end = maker.last_shape().map_err(failure)?;
    let shell_edges = Explorer::new(shell, ShapeEnum::Edge, ShapeEnum::Shape)
        .map_err(failure)?
        .collect::<Vec<_>>();
    verify_trim_edges(&start, &end, &shell_edges)?;
    Ok([start, end])
}

fn verify_trim_edges(start: &Shape, end: &Shape, edges: &[Shape]) -> Result<(), EvaluationFailure> {
    let mut trims = trim_edges(start)?;
    trims.extend(trim_edges(end)?);
    // A seam belongs to its face twice. A free edge must occur in exactly one trim wire.
    let once = |edge: &Shape| edges.iter().filter(|other| other.is_same(edge)).count() == 1;
    if trims.iter().all(once)
        && edges
            .iter()
            .filter(|edge| once(edge))
            .all(|edge| trims.iter().filter(|trim| trim.is_same(edge)).count() == 1)
    {
        return Ok(());
    }
    Err(trim_failure(
        "expected each free shell edge in exactly one starting or ending trim",
    ))
}

fn trim_edges(wire: &Shape) -> Result<Vec<Shape>, EvaluationFailure> {
    let fail = |cause| history_failure(Entity::Body, cause);
    let _wire = topods::wire(wire).map_err(fail)?;
    let edges = Explorer::new(wire, ShapeEnum::Edge, ShapeEnum::Shape)
        .map_err(fail)?
        .collect::<Vec<_>>();
    if edges.is_empty() {
        return Err(trim_failure("expected nonempty starting and ending trims"));
    }
    Ok(edges)
}

fn trim_failure(message: &str) -> EvaluationFailure {
    Phase::History.refused(
        Kind::IncompleteHistory,
        Entity::Body,
        format!("surface Sweep: {message}"),
    )
}
