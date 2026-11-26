// SPDX-License-Identifier: GPL-2.0-only
//! Analytic draft for a circular section transported along a normal straight path.
use super::path::{difference, dot, straight_ends};
use super::{Entity, EvaluationFailure, Kind, Phase, SweepFaceRole, sweep_failure};
use crate::PrismResolution;
use martensite_tkbrep::topods::Shape;
use martensite_tkmath::gp::{Axis2, Direction3, Point3};
use martensite_tkprim::{brep_prim::BRepPrimOneAxis, brep_prim_api::BRepPrimAPIMakeCone};
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
use oblikovati_model::{feature::sketch_sweep::ResolvedSketchSweep, sketch::SketchEntityId};

pub(super) fn build(
    input: ResolvedSketchSweep<'_>,
    resolution: PrismResolution,
) -> Result<(Shape, Vec<(SweepFaceRole, Shape)>), EvaluationFailure> {
    let [section] = input.boundary.outer.as_slice() else {
        return Err(taper_input("expected one circular profile"));
    };
    if !input.boundary.holes.is_empty() {
        return Err(taper_input("expected a circular profile without holes"));
    }
    let profile = input.profile.sketch.geometry();
    let circle = profile.circle(section.curve).map_err(taper_input)?;
    let center = profile.point(circle.center).map_err(taper_input)?.position;
    let center = input.profile.plane.to_model(center).map_err(taper_input)?;
    let (path, normal, height) = normal_path(&input, center, resolution)?;
    let end_radius = circle.radius + height * input.taper.tan();
    if !end_radius.is_finite() || end_radius <= resolution.maximum_tolerance {
        return Err(taper_input(format!(
            "end radius {end_radius}: expected a circle larger than the model tolerance"
        )));
    }
    let x_axis = input.profile.plane.recipe().x_axis;
    let regions = super::resolved_regions(
        input.profile,
        crate::PrismResolutionPolicy::Fixed(resolution),
    )?;
    super::selected_region(regions.as_ref(), input.boundary)?;
    let axes = || {
        let direction = Direction3::new(normal[0], normal[1], normal[2])?;
        let [x, y, z] = x_axis;
        Axis2::with_x_direction(
            Point3::new(center[0], center[1], center[2]),
            direction,
            Direction3::new(x, y, z)?,
        )
    };
    let mut maker = axes()
        .and_then(|axes| BRepPrimAPIMakeCone::with_axes(axes, circle.radius, end_radius, height))
        .map_err(|cause| sweep_failure("expected an analytic tapered sweep", cause))?;
    let shape = maker
        .shape()
        .cloned()
        .map_err(|cause| sweep_failure("expected a completed tapered sweep", cause))?;
    let faces = cone_faces(&mut maker, section.curve, path)?;
    Ok((shape, faces))
}

fn normal_path(
    input: &ResolvedSketchSweep<'_>,
    profile_center: [f64; 3],
    resolution: PrismResolution,
) -> Result<(SketchEntityId, [f64; 3], f64), EvaluationFailure> {
    let (curve, [start, end]) = straight_ends(input)?;
    let offset = difference(start, profile_center);
    let delta = difference(end, start);
    let normal = input.profile.plane.normal();
    let axial = dot(delta, normal);
    let height = axial.abs();
    let off_axis = std::array::from_fn(|i| delta[i] - axial * normal[i]);
    let drift = dot(off_axis, off_axis).sqrt();
    if height <= resolution.maximum_tolerance
        || drift > height * resolution.angular
        || dot(offset, normal).abs() + drift > resolution.maximum_tolerance
    {
        return Err(taper_input(
            "expected a nonzero path normal to and starting in the profile plane",
        ));
    }
    Ok((curve, normal.map(|v| v * axial.signum()), height))
}

fn cone_faces(
    maker: &mut BRepPrimAPIMakeCone,
    profile: SketchEntityId,
    path: SketchEntityId,
) -> Result<Vec<(SweepFaceRole, Shape)>, EvaluationFailure> {
    let cone = maker.cone();
    let failure = |cause| super::history_failure(Entity::Body, cause);
    Ok(vec![
        (
            SweepFaceRole::Start,
            cone.bottom_face().map_err(failure)?.as_ref().clone(),
        ),
        (
            SweepFaceRole::End,
            cone.top_face().map_err(failure)?.as_ref().clone(),
        ),
        (
            SweepFaceRole::Side { profile, path },
            cone.lateral_face().map_err(failure)?.as_ref().clone(),
        ),
    ])
}

fn taper_input(cause: impl std::fmt::Display) -> EvaluationFailure {
    Phase::Input.refused(
        Kind::InvalidInput,
        Entity::Body,
        format!("sweep taper: {cause}"),
    )
}
