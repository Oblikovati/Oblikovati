// SPDX-License-Identifier: GPL-2.0-only
//! Part feature failure types and datum-health filtering.
use crate::feature::{
    FeatureEvaluationStatus, FeatureHealthRecord, sketch_prism::SketchPrismError,
    work_geometry::WorkGeometryError,
};

/// Decline a named-face query when a diagnostic evaluator cannot resolve it.
/// Example: `unavailable_planar_face(&key)` reports the offending key.
pub fn unavailable_planar_face(key: &[u8]) -> String {
    format!("face key {key:?}: expected a named planar-face evaluator")
}

/// One blend operation selected before construction.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PartBlendKind {
    Fillet,
    Chamfer,
}

impl PartBlendKind {
    /// Decline an unavailable operation; example: `kind.unsupported()`.
    pub fn unsupported<Failure>(self) -> PartFeatureFailure<Failure> {
        let kind = match self {
            Self::Fillet => "fillet",
            Self::Chamfer => "chamfer",
        };
        PartFeatureFailure::Unsupported(format!(
            "{kind}: expected an available named-edge evaluator"
        ))
    }
}

/// Current ordered diagnostics; these errors are not stored in authored recipes.
#[derive(Clone, Debug, PartialEq)]
pub enum PartFeatureFailure<Failure> {
    Datum(Box<WorkGeometryError>),
    Metadata(Box<WorkGeometryError>),
    Inputs(SketchPrismError),
    SweepInput(Box<crate::feature::sketch_sweep::SketchSweepError>),
    RevolveInput(Box<crate::feature::sketch_revolve::SketchRevolveError>),
    CoilInput(String),
    EmbossInput(String),
    RibInput(String),
    DecalInput(String),
    DerivedPartInput(String),
    ImportedModelInput(String),
    UnwrapInput(String),
    LoftInput(String),
    Kernel(Failure),
    BodyReference(String),
    FilletInput(String),
    ChamferInput(String),
    HoleInput(String),
    DraftInput(String),
    ThreadInput(String),
    MarkInput(String),
    ShellInput(String),
    MoveBodyInput(String),
    Unsupported(String),
}

pub(super) fn datum_health<Failure>(
    record: &FeatureHealthRecord<PartFeatureFailure<Failure>>,
) -> Option<FeatureHealthRecord<WorkGeometryError>> {
    let status = match &record.status {
        FeatureEvaluationStatus::Healthy => FeatureEvaluationStatus::Healthy,
        FeatureEvaluationStatus::Suppressed => FeatureEvaluationStatus::Suppressed,
        FeatureEvaluationStatus::ConditionFailed(error) => {
            FeatureEvaluationStatus::ConditionFailed(error.clone())
        }
        FeatureEvaluationStatus::Blocked { drivers } => FeatureEvaluationStatus::Blocked {
            drivers: drivers.clone(),
        },
        FeatureEvaluationStatus::Failed(
            PartFeatureFailure::Datum(error) | PartFeatureFailure::Metadata(error),
        ) => FeatureEvaluationStatus::Failed((**error).clone()),
        FeatureEvaluationStatus::Failed(_) => return None,
    };
    Some(FeatureHealthRecord {
        feature: record.feature,
        status,
    })
}
