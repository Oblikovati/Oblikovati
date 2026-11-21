// SPDX-License-Identifier: GPL-2.0-only
//! Member identities use the permanent recipe reservation stream; Primary is built in.
use super::{DocumentEditHistory, DocumentSnapshot, HistoryError, RecipeReservations};

pub(super) fn authored_ids(snapshot: &DocumentSnapshot) -> Vec<u64> {
    snapshot
        .model_states
        .iter()
        .flat_map(|members| members.list())
        .filter(|member| member.id != 1)
        .map(|member| member.id)
        .collect()
}
impl DocumentSnapshot {
    pub(super) fn validate_model_state_transition(
        &self,
        before: &Self,
    ) -> Result<(), HistoryError> {
        if before.model_states.is_some() && self.model_states.is_none() {
            return Err(HistoryError::ModelStateMembers(
                "model-state content None: expected retained part members".into(),
            ));
        }
        self.validate_model_state_members()?;
        let before_table = before
            .model_states
            .as_ref()
            .map(|members| members.parameter_table());
        let after_table = self
            .model_states
            .as_ref()
            .map(|members| members.parameter_table());
        if before_table != after_table || self.parameter_names_changed(before) {
            self.validate_member_parameter_expressions()?;
        }
        Ok(())
    }

    fn parameter_names_changed(&self, before: &Self) -> bool {
        let ids = self.parameters.ids();
        if ids != before.parameters.ids() {
            return true;
        }
        ids.iter().any(|id| {
            self.parameters.get(*id).map(|state| &state.name)
                != before.parameters.get(*id).map(|state| &state.name)
        })
    }

    pub(super) fn validate_model_state_members(&self) -> Result<(), HistoryError> {
        let Some(members) = &self.model_states else {
            return Ok(());
        };
        members
            .validate()
            .map_err(HistoryError::ModelStateMembers)?;
        self.validate_model_state_property_ownership(members)?;
        self.validate_model_state_property_projection(members)?;
        self.validate_model_state_parameters(members)?;
        for feature in members.referenced_features() {
            self.work_geometry
                .get(crate::feature::FeatureId(feature))
                .map_err(|_| {
                    HistoryError::ModelStateMembers(format!(
                        "model-state feature {feature}: expected an existing feature"
                    ))
                })?;
        }
        Ok(())
    }
    fn validate_model_state_parameters(
        &self,
        members: &crate::model_state::ModelStateMembers,
    ) -> Result<(), HistoryError> {
        let active = members
            .active()
            .ok_or_else(|| {
                HistoryError::ModelStateMembers(
                    "model-state parameter projection: expected an active member".into(),
                )
            })?
            .id;
        for base in members.parameter_table().bases() {
            let parameter = self.parameters.get(base.parameter).ok_or_else(|| {
                HistoryError::ModelStateMembers(format!(
                    "model-state parameter {}: expected an existing parameter",
                    base.parameter
                ))
            })?;
            let expected = members.parameter_table().expression(active, base.parameter);
            if expected != Some(parameter.source.as_str()) {
                return Err(HistoryError::ModelStateMembers(format!(
                    "model-state parameter {} expression {:?}: expected the active member expression {:?}",
                    base.parameter, parameter.source, expected
                )));
            }
        }
        Ok(())
    }
    fn validate_model_state_property_projection(
        &self,
        members: &crate::model_state::ModelStateMembers,
    ) -> Result<(), HistoryError> {
        let active = members
            .active()
            .ok_or_else(|| {
                HistoryError::ModelStateMembers(
                    "model-state property projection: expected an active member".into(),
                )
            })?
            .id;
        for base in members.property_table().bases() {
            let key = &base.property;
            let expected = members.property_table().effective_value(active, key);
            let actual = self.properties.get(&key.set, &key.name);
            let matches = match (expected, actual) {
                (None, None) => true,
                (Some(expected), Some(actual)) => {
                    crate::metadata_value::same(&expected.value, &actual.value)
                }
                _ => false,
            };
            if !matches {
                return Err(HistoryError::ModelStateMembers(format!(
                    "model-state property {key:?}: expected the active member value"
                )));
            }
        }
        Ok(())
    }
    fn validate_model_state_property_ownership(
        &self,
        members: &crate::model_state::ModelStateMembers,
    ) -> Result<(), HistoryError> {
        for base in members.property_table().bases() {
            let key = &base.property;
            let owned = self
                .properties
                .get(&key.set, &key.name)
                .is_some_and(|property| !property.from_parameter.is_empty());
            let exposed = key.set == crate::property::USER_DEFINED
                && self
                    .parameters
                    .find(&key.name)
                    .and_then(|id| self.parameters.get(id))
                    .is_some_and(|parameter| parameter.annotations.exposed_as_property);
            if owned || exposed {
                return Err(HistoryError::ModelStateMembers(format!(
                    "model-state property {key:?}: expected a property not owned by a parameter"
                )));
            }
        }
        Ok(())
    }
}
impl DocumentEditHistory {
    pub(super) fn validate_model_state_ids(
        &self,
        candidate: &DocumentSnapshot,
        before: u64,
    ) -> Result<(), HistoryError> {
        candidate.validate_model_state_members()?;
        super::sketch_ids::require_reserved(
            authored_ids(candidate),
            authored_ids(&self.current),
            before,
            self.high_water(),
            HistoryError::UnreservedIdentity,
        )
    }
}
impl RecipeReservations<'_> {
    /// Reserve an authored member identity that undo cannot reuse. Example: `ids.allocate_model_state()?`.
    pub fn allocate_model_state(&mut self) -> Result<u64, crate::identity::RecipeIdError> {
        let reserved = self.allocate()?;
        if reserved == 1 {
            return self.allocate();
        }
        Ok(reserved)
    }
}
