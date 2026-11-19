// SPDX-License-Identifier: GPL-2.0-only
//! Per-member feature suppression and the scope of new edits.
use super::ModelStateMembers;
use serde::{Deserialize, Serialize};

/// Whether a feature edit changes one member or every member.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub enum ModelStateEditScope {
    #[default]
    SingleMember,
    AllMembers,
}
impl ModelStateMembers {
    /// Read the current edit scope. Example: `members.edit_scope()`.
    pub fn edit_scope(&self) -> ModelStateEditScope {
        self.scope
    }
    /// Change the scope of subsequent edits. Example: `members.set_edit_scope(ModelStateEditScope::AllMembers)`.
    pub fn set_edit_scope(&mut self, scope: ModelStateEditScope) {
        self.scope = scope;
    }
    /// Resolve the active member's feature state; example: `members.suppressed(id, false)`.
    pub fn suppressed(&self, feature: u64, authored: bool) -> bool {
        self.suppression
            .get(&self.active)
            .and_then(|values| values.get(&feature))
            .copied()
            .unwrap_or(authored)
    }
    /// Resolve an existing member's feature state; example: `members.suppressed_for(1, id, false)?`.
    pub fn suppressed_for(
        &self,
        member: u64,
        feature: u64,
        authored: bool,
    ) -> Result<bool, String> {
        self.validate_suppression_target(member, feature)?;
        Ok(self
            .suppression
            .get(&member)
            .and_then(|values| values.get(&feature))
            .copied()
            .unwrap_or(authored))
    }
    /// Read an authored override, if present; example: `members.suppression_override(1, id)?`.
    pub fn suppression_override(&self, member: u64, feature: u64) -> Result<Option<bool>, String> {
        self.validate_suppression_target(member, feature)?;
        Ok(self
            .suppression
            .get(&member)
            .and_then(|values| values.get(&feature))
            .copied())
    }
    /// Change one existing member without changing the active member; example: `members.set_suppressed_for(1, id, false, true)?`.
    pub fn set_suppressed_for(
        &mut self,
        member: u64,
        feature: u64,
        authored: bool,
        suppressed: bool,
    ) -> Result<(), String> {
        self.validate_suppression_target(member, feature)?;
        self.change_suppression(member, feature, authored, suppressed);
        Ok(())
    }
    fn validate_suppression_target(&self, member: u64, feature: u64) -> Result<(), String> {
        if feature == 0 || !self.members.iter().any(|row| row.id == member) {
            return Err(format!(
                "model-state suppression member {member} feature {feature}: expected an existing member and nonzero feature ID"
            ));
        }
        Ok(())
    }
    fn change_suppression(&mut self, member: u64, feature: u64, authored: bool, suppressed: bool) {
        if authored == suppressed {
            if let Some(overrides) = self.suppression.get_mut(&member) {
                overrides.remove(&feature);
            }
        } else {
            self.suppression
                .entry(member)
                .or_default()
                .insert(feature, suppressed);
        }
        self.suppression.retain(|_, values| !values.is_empty());
    }
    /// Apply suppression in the current scope; example: `members.set_suppressed(id, false, true)`.
    pub fn set_suppressed(&mut self, feature: u64, authored: bool, suppressed: bool) {
        let selected: Vec<u64> = self
            .members
            .iter()
            .filter(|member| {
                self.scope == ModelStateEditScope::AllMembers || member.id == self.active
            })
            .map(|member| member.id)
            .collect();
        for member in selected {
            self.change_suppression(member, feature, authored, suppressed);
        }
    }
    /// A feature created in one member is suppressed in the other members.
    /// Example: `members.register_feature(id)` after authoring the feature.
    pub fn register_feature(&mut self, feature: u64) {
        if self.scope == ModelStateEditScope::AllMembers {
            return;
        }
        for member in &self.members {
            if member.id != self.active {
                self.suppression
                    .entry(member.id)
                    .or_default()
                    .insert(feature, true);
            }
        }
    }
    /// Read referenced feature identities for document validation. Example: `members.referenced_features()`.
    pub fn referenced_features(&self) -> impl Iterator<Item = u64> + '_ {
        self.suppression
            .values()
            .flat_map(|values| values.keys().copied())
            .chain(self.table_columns.iter().filter_map(|column| match column {
                super::ModelStateTableColumn::Suppression(feature) => Some(*feature),
                _ => None,
            }))
    }
    /// Remove overrides for a deleted feature. Example: `members.forget_feature(id)`.
    pub fn forget_feature(&mut self, feature: u64) {
        for values in self.suppression.values_mut() {
            values.remove(&feature);
        }
        self.table_columns.retain(|column| !matches!(column, super::ModelStateTableColumn::Suppression(id) if *id == feature));
        self.suppression.retain(|_, values| !values.is_empty());
    }
    pub(super) fn validate_suppression(&self) -> Result<(), String> {
        for (member, values) in &self.suppression {
            if !self.members.iter().any(|item| item.id == *member) || values.contains_key(&0) {
                return Err(format!(
                    "model-state suppression {member:?}: expected an existing member and nonzero feature IDs"
                ));
            }
        }
        Ok(())
    }
}
