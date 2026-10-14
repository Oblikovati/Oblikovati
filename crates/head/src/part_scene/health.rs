// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{FeatureHealth, ModelHealthResult};

impl DesktopPartScenes {
    pub(super) fn part_requires_update<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<bool, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.snapshot().work_geometry.features().is_empty() {
            return Ok(false);
        }
        Ok(self.current(session).is_err())
    }

    pub(super) fn current_model_health<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<ModelHealthResult, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let evaluation = self.current(session)?;
        let entries = health_entries(&session.snapshot().work_geometry, evaluation.health())?;
        aggregate(entries)
    }
}

fn health_entries(
    features: &oblikovati_model::feature::FeatureSequence<
        oblikovati_model::feature::part::PartFeatureDefinition,
    >,
    records: &[oblikovati_model::feature::FeatureHealthRecord<
        oblikovati_model::feature::part::PartFeatureFailure<
            oblikovati_martensite::EvaluationFailure,
        >,
    >],
) -> Result<Vec<FeatureHealth>, String> {
    records
        .iter()
        .filter_map(|record| health_entry(features, record).transpose())
        .collect()
}

fn health_entry(
    features: &oblikovati_model::feature::FeatureSequence<
        oblikovati_model::feature::part::PartFeatureDefinition,
    >,
    record: &oblikovati_model::feature::FeatureHealthRecord<
        oblikovati_model::feature::part::PartFeatureFailure<
            oblikovati_martensite::EvaluationFailure,
        >,
    >,
) -> Result<Option<FeatureHealth>, String> {
    let feature = features.get(record.feature).map_err(|cause| {
        format!(
            "feature health {:?}: expected an authored feature: {cause}",
            record.feature
        )
    })?;
    match &record.status {
        oblikovati_model::feature::FeatureEvaluationStatus::Healthy => Ok(None),
        oblikovati_model::feature::FeatureEvaluationStatus::Suppressed => Ok(Some(FeatureHealth {
            name: feature.name().into(),
            status: "suppressed".into(),
            reason: String::new(),
        })),
        status => Ok(Some(FeatureHealth {
            name: feature.name().into(),
            status: "sick".into(),
            reason: status_reason(status),
        })),
    }
}

fn status_reason(
    status: &oblikovati_model::feature::FeatureEvaluationStatus<
        oblikovati_model::feature::part::PartFeatureFailure<
            oblikovati_martensite::EvaluationFailure,
        >,
    >,
) -> String {
    use oblikovati_model::feature::{FeatureEvaluationStatus, part::PartFeatureFailure};
    match status {
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Datum(cause))
        | FeatureEvaluationStatus::Failed(PartFeatureFailure::Metadata(cause)) => cause.to_string(),
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Inputs(cause)) => cause.to_string(),
        FeatureEvaluationStatus::Failed(PartFeatureFailure::Kernel(cause)) => cause.to_string(),
        FeatureEvaluationStatus::Failed(PartFeatureFailure::BodyReference(cause)) => cause.clone(),
        FeatureEvaluationStatus::ConditionFailed(cause) => cause.to_string(),
        FeatureEvaluationStatus::Blocked { drivers } => {
            format!("blocked by unavailable features {drivers:?}")
        }
        FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Suppressed => String::new(),
    }
}

fn aggregate(unhealthy: Vec<FeatureHealth>) -> Result<ModelHealthResult, String> {
    let sick_count = sick_count(&unhealthy)?;
    Ok(ModelHealthResult {
        overall: if sick_count == 0 { "ok" } else { "sick" }.into(),
        sick_count,
        unhealthy: (!unhealthy.is_empty()).then_some(unhealthy),
    })
}

fn sick_count(entries: &[FeatureHealth]) -> Result<i64, String> {
    let count = entries
        .iter()
        .filter(|entry| entry.status == "sick")
        .count();
    i64::try_from(count).map_err(|cause| format!("sick feature count {count}: {cause}"))
}
