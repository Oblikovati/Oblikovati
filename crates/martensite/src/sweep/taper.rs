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
    let (section, center, radius) = circular_section(&input, "taper")?;
    let (path, ends) = straight_ends(&input)?;
    let (normal, height) = normal_path(
        ends,
        input.profile.plane.normal(),
        center,
        resolution,
        "taper",
    )?;
    let end_radius = radius + height * input.taper.tan();
    if !end_radius.is_finite() || end_radius <= resolution.maximum_tolerance {
        return Err(angle_input(
            "taper",
            format!("end radius {end_radius}: expected a circle larger than the model tolerance"),
        ));
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
        .and_then(|axes| BRepPrimAPIMakeCone::with_axes(axes, radius, end_radius, height))
        .map_err(|cause| sweep_failure("expected an analytic tapered sweep", cause))?;
    let shape = maker
        .shape()
        .cloned()
        .map_err(|cause| sweep_failure("expected a completed tapered sweep", cause))?;
    let faces = cone_faces(&mut maker, section, path)?;
    Ok((shape, faces))
}

pub(super) fn require_coaxial_twist(
    input: &ResolvedSketchSweep<'_>,
    resolution: PrismResolution,
) -> Result<(), EvaluationFailure> {
    let (_, center, _) = circular_section(input, "twist")?;
    let (_, ends) = straight_ends(input)?;
    // At either endpoint the straight axis must pass through the transported center. The
    // resulting circular sections are invariant under rotation about this axis.
    let normal = input.profile.plane.normal();
    normal_path(ends, normal, center, resolution, "twist")?;
    for endpoint in ends {
        let offset = difference(endpoint, center);
        let along = dot(offset, normal);
        let radial = std::array::from_fn(|i| offset[i] - along * normal[i]);
        if dot(radial, radial).sqrt() > resolution.maximum_tolerance {
            return Err(angle_input(
                "twist",
                "expected a straight path through the circle center",
            ));
        }
    }
    Ok(())
}

fn circular_section(
    input: &ResolvedSketchSweep<'_>,
    option: &str,
) -> Result<(SketchEntityId, [f64; 3], f64), EvaluationFailure> {
    let ([section], []) = (
        input.boundary.outer.as_slice(),
        input.boundary.holes.as_slice(),
    ) else {
        return Err(angle_input(
            option,
            "expected one circular profile without holes",
        ));
    };
    let profile = input.profile.sketch.geometry();
    let circle = profile
        .circle(section.curve)
        .map_err(|cause| angle_input(option, cause))?;
    let point = profile
        .point(circle.center)
        .map_err(|cause| angle_input(option, cause))?;
    let center = input
        .profile
        .plane
        .to_model(point.position)
        .map_err(|cause| angle_input(option, cause))?;
    Ok((section.curve, center, circle.radius))
}

fn normal_path(
    [start, end]: [[f64; 3]; 2],
    normal: [f64; 3],
    profile_center: [f64; 3],
    resolution: PrismResolution,
    option: &str,
) -> Result<([f64; 3], f64), EvaluationFailure> {
    let offset = difference(start, profile_center);
    let delta = difference(end, start);
    let axial = dot(delta, normal);
    let height = axial.abs();
    let off_axis = std::array::from_fn(|i| delta[i] - axial * normal[i]);
    let drift = dot(off_axis, off_axis).sqrt();
    if height <= resolution.maximum_tolerance
        || drift > height * resolution.angular
        || dot(offset, normal).abs() + drift > resolution.maximum_tolerance
    {
        return Err(angle_input(
            option,
            "expected a nonzero path normal to and starting in the profile plane",
        ));
    }
    Ok((normal.map(|v| v * axial.signum()), height))
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

fn angle_input(option: &str, cause: impl std::fmt::Display) -> EvaluationFailure {
    Phase::Input.refused(
        Kind::InvalidInput,
        Entity::Body,
        format!("sweep {option}: {cause}"),
    )
}
