// SPDX-License-Identifier: GPL-2.0-only
//! Build certified helical and spiral Coil solids from a resolved sketch profile and axis.
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase, PrismAchievedTolerance, PrismResolution, PrismResolutionPolicy,
    sketch_prism::regions::{region_boundary, resolved_regions},
};
use martensite_tkbo::profile_extrusion::{PlanarRegion, PlanarRegionSet};
use martensite_tkbrep::{
    brep_tools::outer_wire,
    topexp::Explorer,
    topods::{Shape, Wire},
};
use martensite_tkernel::{
    message::ProgressRange,
    ncollection::{Array1, HArray1},
    standard::{Failure, Handle},
};
use martensite_tkg3d::{geom::GeomCurve, top_abs::ShapeEnum};
use martensite_tkgeomalgo::geom_api::Interpolate;
use martensite_tkgeombase::gprop::GProps;
use martensite_tkmath::gp::Point3;
use martensite_tkoffset::brep_offset_api::BRepOffsetAPIMakePipeShell;
use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeSweep;
use martensite_tktopalgo::{
    brep_builder_api::{BRepBuilderAPIMakeEdge, BRepBuilderAPIMakeShape, BRepBuilderAPIMakeWire},
    brep_gprop::BRepGProp,
};
use oblikovati_model::feature::{
    sketch_coil::{CoilHandedness, ResolvedSketchCoil},
    sketch_prism::SketchPrismSelection,
};
use std::f64::consts::TAU;

/// Source roles from a native Coil sweep, independent of result enumeration.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum CoilFaceRole {
    Start,
    End,
    Side(u64),
}

/// A privately constructed Coil solid with verified face lineage.
pub struct CoilCandidate {
    pub(crate) shape: Shape,
    pub(crate) faces: Vec<(CoilFaceRole, Shape)>,
    pub(crate) resolution: PrismResolution,
    achieved: PrismAchievedTolerance,
}

impl CoilCandidate {
    /// Read measured B-rep tolerances. Example: `candidate.achieved_tolerance().edge`.
    pub fn achieved_tolerance(&self) -> PrismAchievedTolerance {
        self.achieved
    }
}

struct CoilRailSpec {
    axis_origin: [f64; 3],
    axis_direction: [f64; 3],
    pitch: f64,
    revolutions: f64,
    taper: f64,
    spiral: bool,
    handedness: CoilHandedness,
    start_end: Option<(f64, f64)>,
    end_end: Option<(f64, f64)>,
}

/// Build a checked helical or spiral solid from a resolved Coil recipe.
/// Example: `build_coil(input, policy)?`.
pub fn build_coil(
    input: ResolvedSketchCoil<'_>,
    policy: PrismResolutionPolicy,
) -> Result<CoilCandidate, EvaluationFailure> {
    let spec = CoilRailSpec {
        axis_origin: input.axis_origin,
        axis_direction: input.axis_direction,
        pitch: input.pitch,
        revolutions: input.revolutions,
        taper: input.taper,
        spiral: input.spiral,
        handedness: input.handedness,
        start_end: input.start_end,
        end_end: input.end_end,
    };
    let resolution = policy.for_sketch(&input.sketch)?;
    let regions = resolved_regions(input.sketch, PrismResolutionPolicy::Fixed(resolution))?;
    let region = selected_region(regions.as_ref(), input.selection)?;
    let centroid = profile_centroid(region)?;
    let spine = build_spine_wire(&spec, centroid, resolution.maximum_tolerance)?;
    let profile_wire = outer_wire(region.face()).map_err(|cause| {
        Phase::Profile.native(
            Entity::Body,
            "coil: expected closed outer profile wire".into(),
            cause,
        )
    })?;
    let mut pipe = BRepOffsetAPIMakePipeShell::new(&spine)
        .map_err(|cause| coil_failure("expected pipe shell builder", cause))?;
    pipe.set_mode_frenet(false)
        .map_err(|cause| coil_failure("expected corrected Frenet trihedron", cause))?;
    pipe.add(profile_wire.as_ref(), false, false)
        .map_err(|cause| coil_failure("expected profile section addition", cause))?;
    pipe.build(&ProgressRange::default())
        .map_err(|cause| coil_failure("expected completed coil sweep", cause))?;
    if !pipe.is_done() {
        return Err(Phase::Sweep.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("coil sweep did not complete: {:?}", pipe.get_status()),
        ));
    }
    let closed = pipe
        .make_solid()
        .map_err(|cause| coil_failure("expected closed coil solid", cause))?;
    if !closed {
        return Err(Phase::Sweep.refused(
            Kind::InvalidResult,
            Entity::Body,
            "coil sweep could not close into a solid".into(),
        ));
    }
    let shape = pipe
        .shape()
        .cloned()
        .map_err(|cause| coil_failure("expected coil solid shape", cause))?;
    require_solid(&shape)?;
    let achieved =
        crate::checked_shape::measured_tolerances(&shape, resolution.maximum_tolerance, "coil")?;
    let faces = face_history(&mut pipe, region)?;
    verify_faces(&shape, &faces)?;
    Ok(CoilCandidate {
        shape,
        faces,
        resolution,
        achieved,
    })
}

fn profile_centroid(region: &PlanarRegion) -> Result<[f64; 3], EvaluationFailure> {
    let mut props = GProps::default();
    BRepGProp::surface_properties(region.face().as_ref(), &mut props, false, false)
        .map_err(|cause| coil_failure("expected profile area properties", cause))?;
    let p = props.centre_of_mass();
    Ok([p.x(), p.y(), p.z()])
}

fn build_spine_wire(
    spec: &CoilRailSpec,
    centroid: [f64; 3],
    tolerance: f64,
) -> Result<Wire, EvaluationFailure> {
    let a = spec.axis_direction;
    let rel = [
        centroid[0] - spec.axis_origin[0],
        centroid[1] - spec.axis_origin[1],
        centroid[2] - spec.axis_origin[2],
    ];
    let along = rel[0] * a[0] + rel[1] * a[1] + rel[2] * a[2];
    let foot = [
        spec.axis_origin[0] + along * a[0],
        spec.axis_origin[1] + along * a[1],
        spec.axis_origin[2] + along * a[2],
    ];
    let radial = [
        centroid[0] - foot[0],
        centroid[1] - foot[1],
        centroid[2] - foot[2],
    ];
    let r0 = (radial[0] * radial[0] + radial[1] * radial[1] + radial[2] * radial[2]).sqrt();
    if !r0.is_finite() || r0 <= tolerance {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            "coil profile centroid lies on the coil axis".into(),
        ));
    }
    let u = [radial[0] / r0, radial[1] / r0, radial[2] / r0];
    let v = [
        a[1] * u[2] - a[2] * u[1],
        a[2] * u[0] - a[0] * u[2],
        a[0] * u[1] - a[1] * u[0],
    ];
    let sense = match spec.handedness {
        CoilHandedness::Right => 1.0,
        CoilHandedness::Left => -1.0,
    };
    let points = sample_rail_points(spec, (foot, u, v, a), (r0, sense, tolerance))?;
    let n = i32::try_from(points.len()).map_err(|_| {
        Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            "coil spine sample overflow".into(),
        )
    })?;
    let mut arr = Array1::with_bounds(1, n).map_err(|cause| {
        Phase::Input.refused(
            Kind::InvalidInput,
            Entity::Body,
            format!("coil spine point array: {cause:?}"),
        )
    })?;
    for (index, pt) in points.into_iter().enumerate() {
        arr[(index + 1) as i32] = pt;
    }
    let mut interp = Interpolate::new(&Handle::new(HArray1::from_owned(arr)), false, 1e-7)
        .map_err(|cause| coil_failure("expected spine spline interpolator", cause))?;
    interp
        .perform()
        .map_err(|cause| coil_failure("expected spine spline interpolation", cause))?;
    let curve: Handle<dyn GeomCurve> = interp
        .curve()
        .map_err(|cause| coil_failure("expected interpolated coil curve", cause))?
        .clone()
        .upcast();
    let edge = BRepBuilderAPIMakeEdge::from_curve(&curve)
        .and_then(|maker| maker.edge())
        .map_err(|cause| coil_failure("expected coil spine edge", cause))?;
    BRepBuilderAPIMakeWire::from_edge(&edge)
        .and_then(|maker| maker.wire())
        .map_err(|cause| coil_failure("expected coil spine wire", cause))
}

fn sample_rail_points(
    spec: &CoilRailSpec,
    (foot, u, v, a): ([f64; 3], [f64; 3], [f64; 3], [f64; 3]),
    (r0, sense, tolerance): (f64, f64, f64),
) -> Result<Vec<Point3>, EvaluationFailure> {
    let (start_flat, start_trans) = spec
        .start_end
        .map(|(f, t)| (f / TAU, t / TAU))
        .unwrap_or((0.0, 0.0));
    let (end_flat, end_trans) = spec
        .end_end
        .map(|(f, t)| (f / TAU, t / TAU))
        .unwrap_or((0.0, 0.0));
    let body_start = start_flat + start_trans;
    let body_end = body_start + spec.revolutions;
    let end_trans_end = body_end + end_trans;
    let total_turns = end_trans_end + end_flat;
    let steps = ((total_turns * 12.0).ceil() as usize).clamp(12, 96);
    let pitch_at = |t: f64| -> f64 {
        if t <= start_flat {
            0.0
        } else if t < body_start && start_trans > 0.0 {
            spec.pitch * ((t - start_flat) / start_trans)
        } else if t <= body_end {
            spec.pitch
        } else if t < end_trans_end && end_trans > 0.0 {
            spec.pitch * ((end_trans_end - t) / end_trans)
        } else {
            0.0
        }
    };
    let dt = total_turns / steps as f64;
    let mut advance = 0.0;
    let mut points = Vec::with_capacity(steps + 1);
    for i in 0..=steps {
        let t = i as f64 * dt;
        if i > 0 {
            advance += 0.5 * (pitch_at((i - 1) as f64 * dt) + pitch_at(t)) * dt;
        }
        let theta = sense * TAU * t;
        let (rise, radial_delta) = if spec.spiral {
            (0.0, advance)
        } else {
            (advance, advance * spec.taper.tan())
        };
        let r = r0 + radial_delta;
        if !r.is_finite() || r <= tolerance {
            return Err(Phase::Input.refused(
                Kind::InvalidInput,
                Entity::Body,
                format!("coil taper collapses radius at turn {t:.2}"),
            ));
        }
        let (c, s) = (theta.cos(), theta.sin());
        points.push(Point3::new(
            foot[0] + r * (c * u[0] + s * v[0]) + rise * a[0],
            foot[1] + r * (c * u[1] + s * v[1]) + rise * a[1],
            foot[2] + r * (c * u[2] + s * v[2]) + rise * a[2],
        ));
    }
    Ok(points)
}

fn selected_region<'a>(
    regions: Option<&'a PlanarRegionSet>,
    selection: &SketchPrismSelection,
) -> Result<&'a PlanarRegion, EvaluationFailure> {
    let mut matches = regions
        .into_iter()
        .flat_map(|set| set.regions())
        .filter(|region| matches_selection(region, selection));
    if let (Some(region), None) = (matches.next(), matches.next()) {
        return Ok(region);
    }
    Err(Phase::Profile.refused(
        Kind::InvalidInput,
        Entity::Body,
        format!("coil profile {selection:?}: expected one matching sketch region"),
    ))
}

fn matches_selection(region: &PlanarRegion, selection: &SketchPrismSelection) -> bool {
    let boundary = region_boundary(region);
    same_loop(&boundary.outer, &selection.outer)
        && boundary.holes.len() == selection.holes.len()
        && selection
            .holes
            .iter()
            .all(|expected| boundary.holes.iter().any(|cur| same_loop(cur, expected)))
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
                .all(|(l, r)| l.curve == r.curve && l.reversed == r.reversed)
        })
}

fn require_solid(shape: &Shape) -> Result<(), EvaluationFailure> {
    let kind = shape
        .shape_type()
        .map_err(|cause| coil_failure("expected body type", cause))?;
    if kind != ShapeEnum::Solid {
        return Err(Phase::Sweep.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("coil result {kind:?}: expected a solid"),
        ));
    }
    crate::checked_shape::require_geometry(shape, "coil result")
}

fn face_history(
    pipe: &mut BRepOffsetAPIMakePipeShell,
    region: &PlanarRegion,
) -> Result<Vec<(CoilFaceRole, Shape)>, EvaluationFailure> {
    let start = pipe
        .first_shape()
        .map_err(|cause| history_failure(Entity::Body, cause))?;
    let end = pipe
        .last_shape()
        .map_err(|cause| history_failure(Entity::Body, cause))?;
    let mut faces = vec![(CoilFaceRole::Start, start), (CoilFaceRole::End, end)];
    for (curve, edge) in region.edges() {
        let generated = pipe
            .generated(edge.as_ref())
            .map_err(|cause| history_failure(Entity::SketchEntity(curve.0), cause))?;
        let side = generated
            .first()
            .filter(|_| generated.len() == 1)
            .ok_or_else(|| {
                Phase::History.refused(
                    Kind::IncompleteHistory,
                    Entity::SketchEntity(curve.0),
                    "coil: expected one side face per profile edge".into(),
                )
            })?;
        faces.push((CoilFaceRole::Side(curve.0), side.clone()));
    }
    Ok(faces)
}

fn verify_faces(shape: &Shape, sources: &[(CoilFaceRole, Shape)]) -> Result<(), EvaluationFailure> {
    let actual: Vec<_> = Explorer::new(shape, ShapeEnum::Face, ShapeEnum::Shape)
        .map_err(|cause| history_failure(Entity::Body, cause))?
        .collect();
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
            "coil faces {}, source roles {}: expected one unique role per face",
            actual.len(),
            sources.len()
        ),
    ))
}

fn coil_failure(message: &str, cause: Failure) -> EvaluationFailure {
    Phase::Sweep.native(Entity::Body, format!("coil: {message}"), cause)
}

fn history_failure(entity: Entity, cause: Failure) -> EvaluationFailure {
    Phase::History.native(
        entity,
        "coil: expected source-to-result face history".into(),
        cause,
    )
}
