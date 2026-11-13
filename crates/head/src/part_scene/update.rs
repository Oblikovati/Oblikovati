// SPDX-License-Identifier: GPL-2.0-only
//! Replay a part and project its health through the public document API.
use super::*;
use oblikovati_api::wire::FeatureError;
use oblikovati_model::feature::{FeatureEvaluationStatus, part::PartFeatureDefinition};

impl DesktopPartScenes {
    pub(super) fn requires_update_for_api<
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
            .map_err(|error| error.to_string())?;
        if session.snapshot().work_geometry.features().is_empty() {
            return Ok(false);
        }
        Ok(self.slot.current(session).is_err())
    }
    pub(super) fn recompute_for_api<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<FeatureError>, String> {
        if self
            .slot
            .latest()
            .is_some_and(|view| view.document != document)
        {
            self.slot.clear();
        }
        self.adopt_current(files, document)?;
        self.invalidate_presentation();
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        self.slot
            .current(session)
            .map_err(|error| error.to_string())?
            .health()
            .iter()
            .filter_map(|record| feature_error(session.snapshot(), record).transpose())
            .collect()
    }
}

fn feature_error(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    record: &FeatureHealthRecord<PartFeatureFailure<oblikovati_martensite::EvaluationFailure>>,
) -> Result<Option<FeatureError>, String> {
    let message = match &record.status {
        FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Suppressed => return Ok(None),
        status => format!("{status:?}"),
    };
    let feature = snapshot
        .work_geometry
        .get(record.feature)
        .map_err(|error| error.to_string())?;
    Ok(Some(FeatureError {
        id: record.feature.0,
        name: feature.name().into(),
        kind: feature_kind(feature.definition()).into(),
        message,
    }))
}

fn feature_kind(definition: &PartFeatureDefinition) -> &'static str {
    use PartFeatureDefinition::*;
    match definition {
        SketchPrism(_) => "extrude",
        DeleteBody(_) => "delete-body",
        Fixed(_) => "fixed-frame",
        Offset { .. } => "plane-offset",
        ParallelThroughPoint(_) => "plane-point",
        ThroughThreePoints(_) => "three-points",
        LineThroughPoint(_) => "line-point",
        NormalToAxis(_) => "normal-to-curve",
        AtAngleAboutAxis(_) => "line-plane-angle",
        ThroughTwoAxes(_) => "two-lines",
        Axis(_) => "line",
        Point(_) => "position",
        PointAtReference(_) => "point-reference",
        AxisThroughPoints(_) => "two-points",
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn feature_kinds_use_the_go_wire_names() {
        let prism = PartFeatureDefinition::SketchPrism(Box::new(
            crate::scene_fixture::selected_prism_definition(),
        ));
        let fixed = PartFeatureDefinition::Fixed(oblikovati_model::sketch::SketchPlane::xy());
        assert_eq!(feature_kind(&prism), "extrude");
        assert_eq!(feature_kind(&fixed), "fixed-frame");
    }
}
