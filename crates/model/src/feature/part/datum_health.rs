// SPDX-License-Identifier: GPL-2.0-only
//! Datum health projection from part-feature evaluation records.
use super::PartFeatureFailure;
use crate::feature::{
    FeatureEvaluationStatus, FeatureHealthRecord, work_geometry::WorkGeometryError,
};

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
