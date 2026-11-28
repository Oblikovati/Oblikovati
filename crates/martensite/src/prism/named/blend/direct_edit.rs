// SPDX-License-Identifier: GPL-2.0-only
//! Detached Direct Edit B-rep transformations with exact topology history (#3918).
use super::*;
use martensite_tkmath::gp::{Axis1, Direction3, GeneralTransform3, Point3, Transform3};
use martensite_tktopalgo::brep_builder_api::{BRepBuilderAPIGTransform, BRepBuilderAPITransform};
use oblikovati_model::feature::direct_edit::{
    DirectEditOperationKind, DirectEditScaleMode, ResolvedDirectEdit,
};

const SCALE_EPSILON: f64 = 1e-12;
const LINEAR_EPSILON: f64 = 1e-12;
const ANGLE_EPSILON: f64 = 1e-12;

impl NamedSolid {
    /// Apply ordered Direct Edit operations to a detached current solid (#3918).
    /// Example: `body.direct_edit_candidate(&history, document, &resolved)?`.
    pub fn direct_edit_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        input: &ResolvedDirectEdit,
    ) -> Result<BlendCandidate, String> {
        let achieved = self.achieved_tolerance();
        match self {
            Self::Prism(prism) => {
                let body = prism.body_reference_key(history, document)?.to_vec();
                let topology = BlendInputTopology {
                    body: &prism.snapshot.shape,
                    source: prism.blend_sources(history, document)?,
                    source_bodies: vec![body.clone()],
                    body_parent: body.clone(),
                    primary_body: body,
                    inherited_deleted: Vec::new(),
                    inherited_splits: Default::default(),
                    resolution: prism.certification.as_ref().map(|value| value.resolution),
                };
                build_direct_edit(topology, history, document, input, achieved)
            }
            Self::Derived(body) => body.direct_edit_candidate(history, document, input, achieved),
        }
    }
}

pub(super) fn build_direct_edit(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    resolved: &ResolvedDirectEdit,
    achieved: Option<PrismAchievedTolerance>,
) -> Result<BlendCandidate, String> {
    if resolved.operations.is_empty() {
        return Err("direct edit operations []: expected at least one sub-operation".into());
    }
    let mut copier = crate::checked_shape::copy_shape(input.body, "direct edit input")
        .map_err(|cause| cause.to_string())?;
    let originals = copy_named_sources(&mut copier, input.source)?;
    let mut shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let mut tracked: Vec<Shape> = originals.iter().map(|(_, s)| s.clone()).collect();

    for op in &resolved.operations {
        (shape, tracked) = apply_operation(&shape, &tracked, op)?;
    }
    crate::checked_shape::require_geometry(&shape, "direct edit result")
        .map_err(|cause| cause.to_string())?;
    crate::checked_shape::analytic_properties(&shape, "direct edit result")
        .map_err(|cause| cause.to_string())?;

    let records = originals
        .into_iter()
        .zip(tracked)
        .map(|((name, source), output)| BlendHistoryRecord {
            name,
            source,
            generated: Vec::new(),
            modified: vec![output],
            deleted: false,
            transient: false,
        })
        .collect();

    Ok(BlendCandidate {
        resolution: input.resolution,
        achieved_tolerance: achieved,
        shape,
        history: records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        inherited_splits: input.inherited_splits,
        document,
        scope: history.evaluation_scope(),
    })
}

fn apply_operation(
    shape: &Shape,
    tracked: &[Shape],
    op: &DirectEditOperationKind,
) -> Result<(Shape, Vec<Shape>), String> {
    match op {
        DirectEditOperationKind::Scale {
            mode,
            uniform,
            factors,
        } => {
            let [sx, sy, sz] = match mode {
                DirectEditScaleMode::Uniform => [*uniform, *uniform, *uniform],
                DirectEditScaleMode::NonUniform => *factors,
            };
            apply_scale(shape, tracked, [sx, sy, sz])
        }
        DirectEditOperationKind::Move { translation, .. } => {
            if translation.iter().all(|c| c.abs() <= LINEAR_EPSILON) {
                return Ok((shape.clone(), tracked.to_vec()));
            }
            let trsf = Transform3::translation(*translation);
            apply_rigid_transform(shape, tracked, trsf, false)
        }
        DirectEditOperationKind::Rotate {
            origin,
            axis,
            angle_deg,
            ..
        } => {
            if angle_deg.abs() <= ANGLE_EPSILON {
                return Ok((shape.clone(), tracked.to_vec()));
            }
            let dir = unit_direction(*axis)?;
            let trsf = Transform3::rotation(
                Axis1::new(Point3::new(origin[0], origin[1], origin[2]), dir),
                angle_deg.to_radians(),
            )
            .map_err(|cause| cause.to_string())?;
            apply_rigid_transform(shape, tracked, trsf, false)
        }
        DirectEditOperationKind::Size { offset, .. } => {
            if offset.abs() <= LINEAR_EPSILON {
                return Ok((shape.clone(), tracked.to_vec()));
            }
            let center = super::bounds::shape_bounds_center(shape)?;
            let bounds = shape_extent(shape)?;
            let max_dim = bounds.into_iter().fold(0.0_f64, f64::max).max(1e-6);
            let factor = ((max_dim + 2.0 * offset) / max_dim).max(0.05);
            let mut trsf = Transform3::default();
            trsf.set_scale(Point3::new(center[0], center[1], center[2]), factor)
                .map_err(|cause| cause.to_string())?;
            apply_rigid_transform(shape, tracked, trsf, true)
        }
        DirectEditOperationKind::Delete { .. } => Ok((shape.clone(), tracked.to_vec())),
    }
}

fn apply_scale(
    shape: &Shape,
    tracked: &[Shape],
    [sx, sy, sz]: [f64; 3],
) -> Result<(Shape, Vec<Shape>), String> {
    for (axis, value) in [("X", sx), ("Y", sy), ("Z", sz)] {
        if !value.is_finite() || value <= 0.0 {
            return Err(format!(
                "direct edit scale {axis} {value:?}: expected a finite positive factor"
            ));
        }
    }
    let uniform_like = (sx - sy).abs() <= SCALE_EPSILON && (sy - sz).abs() <= SCALE_EPSILON;
    if uniform_like && (sx - 1.0).abs() <= SCALE_EPSILON {
        return Ok((shape.clone(), tracked.to_vec()));
    }
    let center = super::bounds::shape_bounds_center(shape)?;
    if uniform_like {
        let mut trsf = Transform3::default();
        trsf.set_scale(Point3::new(center[0], center[1], center[2]), sx)
            .map_err(|cause| cause.to_string())?;
        return apply_rigid_transform(shape, tracked, trsf, true);
    }
    let mut gtrsf = GeneralTransform3::default();
    gtrsf.set_value(1, 1, sx).map_err(|c| c.to_string())?;
    gtrsf.set_value(2, 2, sy).map_err(|c| c.to_string())?;
    gtrsf.set_value(3, 3, sz).map_err(|c| c.to_string())?;
    gtrsf
        .set_value(1, 4, (1.0 - sx) * center[0])
        .map_err(|c| c.to_string())?;
    gtrsf
        .set_value(2, 4, (1.0 - sy) * center[1])
        .map_err(|c| c.to_string())?;
    gtrsf
        .set_value(3, 4, (1.0 - sz) * center[2])
        .map_err(|c| c.to_string())?;
    let mut deformed =
        BRepBuilderAPIGTransform::with_shape(shape, gtrsf, false).map_err(|c| c.to_string())?;
    let next_shape = deformed.shape().map_err(|c| c.to_string())?.clone();
    let next_tracked = tracked
        .iter()
        .map(|item| {
            deformed
                .modified_shape(item)
                .map_err(|cause| format!("direct edit non-uniform scale history: {cause}"))
        })
        .collect::<Result<Vec<_>, _>>()?;
    Ok((next_shape, next_tracked))
}

fn apply_rigid_transform(
    shape: &Shape,
    tracked: &[Shape],
    trsf: Transform3,
    copy_geom: bool,
) -> Result<(Shape, Vec<Shape>), String> {
    let mut moved = BRepBuilderAPITransform::with_shape(shape, trsf, copy_geom, false)
        .map_err(|cause| cause.to_string())?;
    let next_shape = moved.shape().map_err(|cause| cause.to_string())?.clone();
    let next_tracked = tracked
        .iter()
        .map(|item| {
            moved
                .modified_shape(item)
                .map_err(|cause| format!("direct edit transform history: {cause}"))
        })
        .collect::<Result<Vec<_>, _>>()?;
    Ok((next_shape, next_tracked))
}

fn unit_direction(vector: [f64; 3]) -> Result<Direction3, String> {
    let scale = vector.into_iter().map(f64::abs).fold(0.0, f64::max);
    if scale == 0.0 || !scale.is_finite() {
        return Err(format!(
            "direct edit axis {vector:?}: expected a finite nonzero direction"
        ));
    }
    let [x, y, z] = vector.map(|v| v / scale);
    Direction3::new(x, y, z).map_err(|cause| cause.to_string())
}

fn shape_extent(shape: &Shape) -> Result<[f64; 3], String> {
    use martensite_tkmath::bnd::BoundingBox3;
    use martensite_tktopalgo::brep_bnd_lib::BRepBndLib;
    let mut bbox = BoundingBox3::default();
    BRepBndLib::add_optimal(shape, &mut bbox, false, false).map_err(|cause| cause.to_string())?;
    let [xmin, ymin, zmin, xmax, ymax, zmax] = bbox.get().map_err(|cause| cause.to_string())?;
    Ok([xmax - xmin, ymax - ymin, zmax - zmin])
}
