// SPDX-License-Identifier: GPL-2.0-only
//! Detached rigid motions with exact topology history (#4097).
use super::*;
use martensite_tkmath::gp::{Axis1, Axis2, Direction3, Point3, Transform3};
use martensite_tktopalgo::brep_builder_api::BRepBuilderAPITransform;

/// One ordered, resolved rigid operation in model units.
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum RigidMotion {
    Translation([f64; 3]),
    Reflection {
        origin: [f64; 3],
        normal: [f64; 3],
    },
    Rotation {
        origin: [f64; 3],
        direction: [f64; 3],
        radians: f64,
    },
}

impl NamedSolid {
    /// Transform a detached current body; example: `body.rigid_candidate(&history, document, &moves)?`.
    pub fn rigid_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        moves: &[RigidMotion],
    ) -> Result<BlendCandidate, String> {
        let transform = compose(moves)?;
        let achieved = self.achieved_tolerance();
        match self {
            Self::Prism(prism) => {
                let body = prism.body_reference_key(history, document)?.to_vec();
                let input = BlendInputTopology {
                    body: &prism.snapshot.shape,
                    source: prism.blend_sources(history, document)?,
                    source_bodies: vec![body.clone()],
                    body_parent: body.clone(),
                    primary_body: body,
                    inherited_deleted: Vec::new(),
                    inherited_splits: Default::default(),
                    resolution: prism.certification.as_ref().map(|value| value.resolution),
                };
                build_rigid(input, history, document, transform, achieved)
            }
            Self::Derived(body) => body.rigid_candidate(history, document, transform, achieved),
        }
    }
}
pub(super) fn compose(moves: &[RigidMotion]) -> Result<Transform3, String> {
    if moves.is_empty() {
        return Err("rigid motions []: expected at least one operation".into());
    }
    let mut result = Transform3::default();
    for operation in moves {
        result.pre_multiply(motion(*operation)?);
        require_finite_transform(result, *operation)?;
    }
    Ok(result)
}
fn require_finite_transform(result: Transform3, operation: RigidMotion) -> Result<(), String> {
    for row in 1..=3 {
        for column in 1..=4 {
            let value = result
                .value(row, column)
                .map_err(|error| error.to_string())?;
            if !value.is_finite() {
                return Err(format!(
                    "rigid operation {operation:?}, matrix [{row}, {column}] {value}: expected finite composed coefficients"
                ));
            }
        }
    }
    Ok(())
}
fn motion(operation: RigidMotion) -> Result<Transform3, String> {
    match operation {
        RigidMotion::Translation(vector) => {
            finite(vector, "translation")?;
            Ok(Transform3::translation(vector))
        }
        RigidMotion::Reflection { origin, normal } => {
            finite(origin, "reflection origin")?;
            Axis2::new(
                Point3::new(origin[0], origin[1], origin[2]),
                unit_direction(normal, "reflection normal")?,
            )
            .map(Transform3::mirror_plane)
            .map_err(|cause| cause.to_string())
        }
        RigidMotion::Rotation {
            origin,
            direction,
            radians,
        } => {
            finite(origin, "rotation origin")?;
            if !radians.is_finite() {
                return Err(format!(
                    "rotation angle {radians:?}: expected finite radians"
                ));
            }
            let direction = unit_direction(direction, "rotation direction")?;
            Transform3::rotation(
                Axis1::new(Point3::new(origin[0], origin[1], origin[2]), direction),
                radians,
            )
            .map_err(|cause| cause.to_string())
        }
    }
}
fn unit_direction(vector: [f64; 3], label: &str) -> Result<Direction3, String> {
    finite(vector, label)?;
    // Martensite #550: scale the components before the squared-norm calculation.
    let scale = vector.into_iter().map(f64::abs).fold(0., f64::max);
    if scale == 0. {
        return Err(format!("{label} {vector:?}: expected a nonzero direction"));
    }
    let [x, y, z] = vector.map(|value| value / scale);
    Direction3::new(x, y, z)
        .map_err(|cause| format!("{label} {vector:?}: expected a unit direction: {cause}"))
}

fn finite(values: [f64; 3], label: &str) -> Result<(), String> {
    if values.iter().all(|value| value.is_finite()) {
        return Ok(());
    }
    Err(format!(
        "{label} {values:?}: expected three finite coordinates"
    ))
}
pub(super) fn build_rigid(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    transform: Transform3,
    achieved: Option<PrismAchievedTolerance>,
) -> Result<BlendCandidate, String> {
    let mut copier = crate::checked_shape::copy_shape(input.body, "rigid input")
        .map_err(|error| error.to_string())?;
    let copied = copy_named_sources(&mut copier, input.source)?;
    let detached = copier.shape().map_err(|error| error.to_string())?.clone();
    let mut moved = BRepBuilderAPITransform::with_shape(&detached, transform, false, false)
        .map_err(|error| error.to_string())?;
    let shape = moved.shape().map_err(|error| error.to_string())?.clone();
    crate::checked_shape::require_geometry(&shape, "rigid result")
        .map_err(|error| error.to_string())?;
    crate::checked_shape::analytic_properties(&shape, "rigid result")
        .map_err(|error| error.to_string())?;
    let records = moved_history(&moved, copied)?;
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
fn moved_history(
    moved: &BRepBuilderAPITransform,
    sources: Vec<(Vec<u8>, Shape)>,
) -> Result<Vec<BlendHistoryRecord>, String> {
    sources
        .into_iter()
        .map(|(name, source)| {
            let output = moved.modified_shape(&source).map_err(|cause| {
                format!("rigid source {name:?}: expected one transformed entity: {cause}")
            })?;
            Ok(BlendHistoryRecord {
                name,
                source,
                generated: Vec::new(),
                modified: vec![output],
                deleted: false,
                transient: false,
            })
        })
        .collect()
}
