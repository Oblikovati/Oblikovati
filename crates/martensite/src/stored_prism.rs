// SPDX-License-Identifier: GPL-2.0-only
use crate::{PrismProbeSnapshot, retain_sketch_prism_definition};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
/// Build active stored prism inputs through the same diagnostic kernel path.
/// Example: `retain_stored_sketch_prism(&snapshot, feature_id)?`; this is not the complete body evaluation contract.
pub fn retain_stored_sketch_prism(
    snapshot: &DocumentSnapshot,
    feature_id: FeatureId,
) -> Result<PrismProbeSnapshot, String> {
    let feature = active_prism_entry(snapshot, feature_id)?;
    let PartFeatureDefinition::SketchPrism(definition) = feature.definition() else {
        return Err(format!(
            "part feature {feature_id:?} {:?}: expected stored sketch prism inputs",
            feature.definition()
        ));
    };
    retain_sketch_prism_definition(snapshot, definition)
        .map_err(|error| format!("part feature {feature_id:?}: {error}"))
}
fn active_prism_entry(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<&oblikovati_model::feature::AuthoredFeature<PartFeatureDefinition>, String> {
    let (position, feature) = snapshot
        .work_geometry
        .features()
        .iter()
        .enumerate()
        .find(|(_, feature)| feature.id() == id)
        .ok_or_else(|| format!("part feature {id:?}: expected an existing stored input"))?;
    if position >= snapshot.work_geometry.effective_end() || feature.suppressed() {
        return Err(format!(
            "part feature {id:?} at {position}, suppressed={}: expected active inputs before cutoff {}",
            feature.suppressed(),
            snapshot.work_geometry.effective_end()
        ));
    }
    require_enabled_condition(feature, snapshot)?;
    Ok(feature)
}
fn require_enabled_condition(
    feature: &oblikovati_model::feature::AuthoredFeature<PartFeatureDefinition>,
    snapshot: &DocumentSnapshot,
) -> Result<(), String> {
    let suppress = feature
        .suppression_condition()
        .map(|condition| condition.evaluate(&snapshot.parameters))
        .transpose()
        .map_err(|error| format!("part feature {:?} suppression: {error}", feature.id()))?
        .unwrap_or(false);
    if suppress {
        return Err(format!(
            "part feature {:?} conditional suppression=true: expected enabled inputs",
            feature.id()
        ));
    }
    Ok(())
}
