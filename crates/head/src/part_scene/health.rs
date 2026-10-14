// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{FeatureError, FeatureHealth, ModelHealthResult};

impl DesktopPartScenes {
    pub(super) fn update_part<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        rebuild: bool,
    ) -> Result<Vec<FeatureError>, String> {
        if rebuild {
            self.adopt_current(files, document)?;
        } else {
            self.ensure_current(files, document)?;
        }
        self.current_feature_errors(files, document)
    }

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

    fn current_feature_errors<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<FeatureError>, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let evaluation = self.current(session)?;
        feature_errors(&session.snapshot().work_geometry, evaluation.health())
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

fn feature_errors(
    features: &oblikovati_model::feature::FeatureSequence<
        oblikovati_model::feature::part::PartFeatureDefinition,
    >,
    records: &[oblikovati_model::feature::FeatureHealthRecord<
        oblikovati_model::feature::part::PartFeatureFailure<
            oblikovati_martensite::EvaluationFailure,
        >,
    >],
) -> Result<Vec<FeatureError>, String> {
    records
        .iter()
        .filter_map(|record| feature_error(features, record).transpose())
        .collect()
}

fn feature_error(
    features: &oblikovati_model::feature::FeatureSequence<
        oblikovati_model::feature::part::PartFeatureDefinition,
    >,
    record: &oblikovati_model::feature::FeatureHealthRecord<
        oblikovati_model::feature::part::PartFeatureFailure<
            oblikovati_martensite::EvaluationFailure,
        >,
    >,
) -> Result<Option<FeatureError>, String> {
    use oblikovati_model::feature::FeatureEvaluationStatus;
    if matches!(
        record.status,
        FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Suppressed
    ) {
        return Ok(None);
    }
    let feature = features.get(record.feature).map_err(|cause| {
        format!(
            "feature error {:?}: expected an authored feature: {cause}",
            record.feature
        )
    })?;
    Ok(Some(FeatureError {
        id: record.feature.0,
        name: feature.name().into(),
        kind: feature_kind(feature.definition()).into(),
        message: status_reason(&record.status),
    }))
}

fn feature_kind(
    definition: &oblikovati_model::feature::part::PartFeatureDefinition,
) -> &'static str {
    use oblikovati_model::feature::part::PartFeatureDefinition;
    match definition {
        PartFeatureDefinition::SketchPrism(_) => "extrude",
        PartFeatureDefinition::DeleteBody(_) => "delete-body",
        PartFeatureDefinition::Axis(_) | PartFeatureDefinition::AxisThroughPoints(_) => "work-axis",
        PartFeatureDefinition::Point(_) | PartFeatureDefinition::PointAtReference(_) => {
            "work-point"
        }
        _ => "work-plane",
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
