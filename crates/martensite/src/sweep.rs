// SPDX-License-Identifier: GPL-2.0-only
//! Build a solid from a current sketch region and directed planar path.
mod path;
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase, PrismAchievedTolerance, PrismResolution, PrismResolutionPolicy,
    ReleaseProbeProperties,
    sketch_prism::regions::{region_boundary, resolved_regions},
};
use martensite_tkbo::profile_extrusion::{PlanarRegion, PlanarRegionSet};
use martensite_tkbrep::{brep::BRepTool, topexp::Explorer, topods::Shape};
use martensite_tkernel::standard::Failure;
use martensite_tkg3d::top_abs::ShapeEnum;
use martensite_tkoffset::brep_offset_api::BRepOffsetAPIMakePipe;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeSweep;
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
use oblikovati_model::{
    feature::{sketch_prism::SketchPrismSelection, sketch_sweep::ResolvedSketchSweep},
    sketch::SketchEntityId,
};

/// Source roles from the native sweep, independent of result enumeration.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum SweepFaceRole {
    Start,
    End,
    Side {
        profile: SketchEntityId,
        path: SketchEntityId,
    },
}

/// A privately constructed solid with verified face lineage.
pub struct SweepCandidate {
    pub(crate) shape: Shape,
    pub(crate) faces: Vec<(SweepFaceRole, Shape)>,
    pub(crate) resolution: PrismResolution,
    surface_error: f64,
    achieved: PrismAchievedTolerance,
}

impl SweepCandidate {
    /// Integrate the current solid in working units. Example: `candidate.properties()?.volume`.
    pub fn properties(&self) -> Result<ReleaseProbeProperties, EvaluationFailure> {
        crate::checked_shape::analytic_properties(&self.shape, "sweep result")
    }

    /// List the validated construction roles. Example: `candidate.face_roles().count()`.
    pub fn face_roles(&self) -> impl Iterator<Item = SweepFaceRole> + '_ {
        self.faces.iter().map(|(role, _)| *role)
    }

    /// Report the kernel's surface approximation estimate. Example: `candidate.reported_surface_error()`.
    pub fn reported_surface_error(&self) -> f64 {
        self.surface_error
    }

    /// Read measured B-rep tolerances. Example: `candidate.achieved_tolerance().edge`.
    pub fn achieved_tolerance(&self) -> PrismAchievedTolerance {
        self.achieved
    }
}

/// Build a checked solid with the released path builder. Example: `build_sweep(input, resolution)?`.
pub fn build_sweep(
    input: ResolvedSketchSweep<'_>,
    policy: PrismResolutionPolicy,
) -> Result<SweepCandidate, EvaluationFailure> {
    let resolution = policy.for_sketch(&input.profile)?;
    let regions = resolved_regions(input.profile, PrismResolutionPolicy::Fixed(resolution))?;
    let region = selected_region(regions.as_ref(), input.boundary)?;
    let path = path::build_path(input.path, input.traversal)?;
    let mut maker =
        BRepOffsetAPIMakePipe::new(&path.wire, region.face().as_ref()).map_err(|cause| {
            sweep_failure("expected a sweep from the supplied region and path", cause)
        })?;
    let shape = require_solid(&mut maker)?;
    let surface_error = maker.error_on_surface();
    require_surface_error(surface_error, resolution.maximum_tolerance)?;
    let achieved = measured_tolerances(&shape, resolution.maximum_tolerance)?;
    let faces = face_history(&mut maker, region, &path)?;
    verify_faces(&shape, &faces)?;
    Ok(SweepCandidate {
        shape,
        faces,
        resolution,
        surface_error,
        achieved,
    })
}

fn measured_tolerances(
    shape: &Shape,
    maximum: f64,
) -> Result<PrismAchievedTolerance, EvaluationFailure> {
    let measure = |kind| {
        let tolerance = BRepTool::max_tolerance(shape, kind).map_err(|cause| {
            Phase::Validation.native(Entity::Body, format!("sweep {kind:?} tolerance"), cause)
        })?;
        if !tolerance.is_finite() || !(0.0..=maximum).contains(&tolerance) {
            return Err(Phase::Validation.refused(
                Kind::ToleranceExceeded,
                Entity::Body,
                format!(
                    "sweep {kind:?} tolerance {tolerance}: expected a finite value in 0..={maximum}"
                ),
            ));
        }
        Ok(tolerance)
    };
    Ok(PrismAchievedTolerance {
        vertex: measure(ShapeEnum::Vertex)?,
        edge: measure(ShapeEnum::Edge)?,
        face: measure(ShapeEnum::Face)?,
    })
}

fn selected_region<'a>(
    regions: Option<&'a PlanarRegionSet>,
    selection: &SketchPrismSelection,
) -> Result<&'a PlanarRegion, EvaluationFailure> {
    let mut matches = regions
        .into_iter()
        .flat_map(|regions| regions.regions())
        .filter(|region| matches_selection(region, selection));
    if let (Some(region), None) = (matches.next(), matches.next()) {
        return Ok(region);
    }
    Err(Phase::Profile.refused(
        Kind::InvalidInput,
        Entity::Body,
        format!("sweep profile {selection:?}: expected one current region with the saved boundary"),
    ))
}

fn matches_selection(region: &PlanarRegion, selection: &SketchPrismSelection) -> bool {
    let boundary = region_boundary(region);
    same_loop(&boundary.outer, &selection.outer)
        && boundary.holes.len() == selection.holes.len()
        && selection.holes.iter().all(|expected| {
            boundary
                .holes
                .iter()
                .any(|current| same_loop(current, expected))
        })
}

fn same_loop(
    current: &[crate::sketch_prism::SketchRegionCurveUse],
    saved: &[crate::sketch_prism::SketchDirectedCurve],
) -> bool {
    current.len() == saved.len()
        && current.first().is_some_and(|first| {
            let Some(start) = saved.iter().position(|item| item.curve == first.curve) else {
                return false;
            };
            current
                .iter()
                .zip(saved.iter().cycle().skip(start))
                .all(|(left, right)| left.curve == right.curve && left.reversed == right.reversed)
        })
}

fn require_solid(maker: &mut BRepOffsetAPIMakePipe) -> Result<Shape, EvaluationFailure> {
    let shape = maker
        .shape()
        .cloned()
        .map_err(|cause| sweep_failure("expected a completed sweep", cause))?;
    let kind = shape
        .shape_type()
        .map_err(|cause| sweep_failure("expected body type", cause))?;
    if kind != ShapeEnum::Solid {
        return Err(Phase::Sweep.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("sweep result {kind:?}: expected a solid"),
        ));
    }
    crate::checked_shape::require_geometry(&shape, "sweep result")?;
    Ok(shape)
}

fn require_surface_error(actual: f64, maximum: f64) -> Result<(), EvaluationFailure> {
    if actual.is_finite() && (0.0..=maximum).contains(&actual) {
        return Ok(());
    }
    Err(Phase::Validation.refused(
        Kind::ToleranceExceeded,
        Entity::Body,
        format!("sweep surface estimate {actual}: expected a finite error in 0..={maximum}"),
    ))
}

fn face_history(
    maker: &mut BRepOffsetAPIMakePipe,
    region: &PlanarRegion,
    path: &path::NativePath,
) -> Result<Vec<(SweepFaceRole, Shape)>, EvaluationFailure> {
    let start = maker
        .first_shape()
        .map_err(|cause| history_failure(Entity::Body, cause))?;
    let end = maker
        .last_shape()
        .map_err(|cause| history_failure(Entity::Body, cause))?;
    let mut faces = vec![(SweepFaceRole::Start, start), (SweepFaceRole::End, end)];
    for (path_id, spine) in &path.edges {
        for (curve, profile) in region.edges() {
            let side = maker
                .generated_by(spine.as_ref(), profile.as_ref())
                .map_err(|cause| history_failure(Entity::SketchEntity(curve.0), cause))?;
            faces.push((
                SweepFaceRole::Side {
                    profile: SketchEntityId(curve.0),
                    path: *path_id,
                },
                side,
            ));
        }
    }
    Ok(faces)
}

fn verify_faces(
    shape: &Shape,
    sources: &[(SweepFaceRole, Shape)],
) -> Result<(), EvaluationFailure> {
    let actual = Explorer::new(shape, ShapeEnum::Face, ShapeEnum::Shape)
        .map_err(|cause| history_failure(Entity::Body, cause))?
        .collect::<Vec<_>>();
    if actual.len() == sources.len()
        && actual.iter().all(|face| {
            sources
                .iter()
                .filter(|(_, candidate)| candidate.is_same(face))
                .count()
                == 1
        })
    {
        return Ok(());
    }
    Err(Phase::History.refused(
        Kind::IncompleteHistory,
        Entity::Body,
        format!(
            "sweep faces {}, source roles {}: expected one unique role per face",
            actual.len(),
            sources.len()
        ),
    ))
}

fn sweep_failure(message: &str, cause: Failure) -> EvaluationFailure {
    Phase::Sweep.native(Entity::Body, format!("sweep: {message}"), cause)
}

fn history_failure(entity: Entity, cause: Failure) -> EvaluationFailure {
    Phase::History.native(
        entity,
        "sweep: expected source-to-result face history".into(),
        cause,
    )
}
