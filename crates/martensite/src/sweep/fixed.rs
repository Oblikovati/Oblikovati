// SPDX-License-Identifier: GPL-2.0-only
//! Preserve the profile's frame while translating along a straight Sweep path.
use super::{
    Entity, EvaluationFailure, Kind, Phase, PrismResolution, SweepFaceRole, history_failure,
    path::{difference, dot, straight_ends},
    resolved_regions, selected_region, sweep_failure,
};
use martensite_tkbrep::topods::Shape;
use martensite_tkmath::gp::Vector3;
use martensite_tkprim::brep_prim_api::{BRepPrimAPIMakePrism, BRepPrimAPIMakeSweep};
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;
use oblikovati_model::feature::sketch_sweep::ResolvedSketchSweep;

pub(super) fn build(
    input: ResolvedSketchSweep<'_>,
    resolution: PrismResolution,
) -> Result<(Shape, Vec<(SweepFaceRole, Shape)>), EvaluationFailure> {
    let (path, [start, end]) = straight_ends(&input)?;
    let normal = input.profile.plane.normal();
    let offset = difference(start, input.profile.plane.recipe().origin);
    let delta = difference(end, start);
    let distance = dot(offset, normal).abs();
    if !distance.is_finite()
        || distance > resolution.maximum_tolerance
        || dot(delta, normal).abs() <= resolution.maximum_tolerance
        || delta.iter().any(|value| !value.is_finite())
    {
        return Err(Phase::Input.refused(
            Kind::InvalidInput,
            Entity::SketchEntity(path.0),
            "sweep Fixed orientation requires a straight path starting in and leaving the profile plane".into(),
        ));
    }
    let regions = resolved_regions(
        input.profile,
        crate::PrismResolutionPolicy::Fixed(resolution),
    )?;
    let region = selected_region(regions.as_ref(), input.boundary)?;
    let [x, y, z] = delta;
    let mut maker =
        BRepPrimAPIMakePrism::new(region.face().as_ref(), Vector3::new(x, y, z), true, true)
            .map_err(|cause| sweep_failure("expected a fixed-orientation Sweep", cause))?;
    let shape = maker
        .shape()
        .cloned()
        .map_err(|cause| sweep_failure("expected a completed fixed-orientation Sweep", cause))?;
    let cap_error = |cause| history_failure(Entity::Body, cause);
    let mut faces = vec![
        (
            SweepFaceRole::Start,
            maker.first_shape().map_err(cap_error)?,
        ),
        (SweepFaceRole::End, maker.last_shape().map_err(cap_error)?),
    ];
    for (curve, edge) in region.edges() {
        let generated = maker
            .generated(edge.as_ref())
            .map_err(|cause| history_failure(Entity::SketchEntity(curve.0), cause))?;
        let side = generated
            .first()
            .filter(|_| generated.len() == 1)
            .ok_or_else(|| {
                Phase::History.refused(
                    Kind::IncompleteHistory,
                    Entity::SketchEntity(curve.0),
                    "sweep Fixed orientation requires one side from each profile curve".into(),
                )
            })?;
        faces.push((
            SweepFaceRole::Side {
                profile: oblikovati_model::sketch::SketchEntityId(curve.0),
                path,
            },
            side.clone(),
        ));
    }
    Ok((shape, faces))
}
