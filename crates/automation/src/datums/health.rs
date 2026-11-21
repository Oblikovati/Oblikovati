// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn reason(
    evaluation: &WorkGeometryEvaluation,
    id: FeatureId,
    cause: Option<&WorkGeometryError>,
) -> Result<String, Error> {
    let Some(cause) = cause else {
        return Ok(String::new());
    };
    let first = unique_receipt(evaluation, id)?;
    match first {
        None => Ok(cause.to_string()),
        Some(record) => failed_receipt(id, &record.status),
    }
}
fn failed_receipt(
    id: FeatureId,
    status: &FeatureEvaluationStatus<WorkGeometryError>,
) -> Result<String, Error> {
    match status {
        FeatureEvaluationStatus::Failed(cause) => Ok(cause.to_string()),
        FeatureEvaluationStatus::ConditionFailed(cause) => Ok(cause.to_string()),
        FeatureEvaluationStatus::Suppressed | FeatureEvaluationStatus::Blocked { .. } => Ok(
            format!("datum {id:?} replay {status:?}: expected available current geometry"),
        ),
        FeatureEvaluationStatus::Healthy => Err(Error(format!(
            "datum {id:?} healthy receipt has no geometry: expected a matching current result"
        ))),
    }
}

fn unique_receipt(
    evaluation: &WorkGeometryEvaluation,
    id: FeatureId,
) -> Result<Option<&oblikovati_model::feature::FeatureHealthRecord<WorkGeometryError>>, Error> {
    let mut records = evaluation
        .health
        .iter()
        .filter(|record| record.feature == id);
    let first = records.next();
    if records.next().is_some() {
        return Err(Error(format!(
            "datum {id:?}: expected at most one current health receipt"
        )));
    }
    Ok(first)
}
