// SPDX-License-Identifier: GPL-2.0-only
//! Discover selectable columns with their inherited source values.
use super::*;
use oblikovati_api::types::{MODEL_PARAM, TABLE_PARAM, USER_PARAM};

pub(super) fn properties(
    snapshot: &DocumentSnapshot,
    members: &ModelStateMembers,
) -> Vec<ModelStateTablePropertyChoice> {
    super::property_view::available(members, &snapshot.properties)
        .into_iter()
        .map(|property| {
            let base = super::property_view::base(members, &snapshot.properties, &property);
            ModelStateTablePropertyChoice { property, base }
        })
        .collect()
}
pub(super) fn features(snapshot: &DocumentSnapshot) -> Vec<ModelStateTableFeatureChoice> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .map(|feature| ModelStateTableFeatureChoice {
            id: feature.id().0,
            name: feature.name().into(),
            base_suppressed: feature.suppressed(),
        })
        .collect()
}
pub(super) fn parameters(snapshot: &DocumentSnapshot) -> Vec<ModelStateTableParameterChoice> {
    snapshot
        .parameters
        .ids()
        .iter()
        .filter_map(|id| {
            let parameter = snapshot.parameters.get(*id)?;
            if !matches!(parameter.kind, USER_PARAM | MODEL_PARAM | TABLE_PARAM) {
                return None;
            }
            let quantity = parameter.current_value()?;
            let feature_id = snapshot
                .work_geometry
                .features()
                .iter()
                .find(|feature| {
                    feature
                        .owned_scalars()
                        .iter()
                        .any(|binding| binding.parameter == *id)
                })
                .map_or(0, |feature| feature.id().0);
            let base_expression = snapshot
                .model_states
                .as_ref()
                .and_then(|members| {
                    members
                        .parameter_table()
                        .bases()
                        .iter()
                        .find(|base| base.parameter == *id)
                })
                .map_or_else(|| parameter.source.clone(), |base| base.expression.clone());
            Some(ModelStateTableParameterChoice {
                id: *id,
                feature_id,
                name: parameter.name.clone(),
                unit: snapshot
                    .parameters
                    .units()
                    .preferred_name(quantity.dimension)
                    .into(),
                base_expression,
            })
        })
        .collect()
}
