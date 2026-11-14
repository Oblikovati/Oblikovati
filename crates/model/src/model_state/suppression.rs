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
    /// Resolve a member override against the authored feature value. Example: `members.suppressed(id, false)`.
    pub fn suppressed(&self, feature: u64, authored: bool) -> bool {
        self.suppression
            .get(&self.active)
            .and_then(|values| values.get(&feature))
            .copied()
            .unwrap_or(authored)
    }
    /// Apply suppression in the current scope. Example: `members.set_suppressed(id, false, true)`.
    pub fn set_suppressed(&mut self, feature: u64, authored: bool, suppressed: bool) {
        for member in &self.members {
            if self.scope == ModelStateEditScope::AllMembers || member.id == self.active {
                if authored == suppressed {
                    if let Some(overrides) = self.suppression.get_mut(&member.id) {
                        overrides.remove(&feature);
                    }
                } else {
                    self.suppression
                        .entry(member.id)
                        .or_default()
                        .insert(feature, suppressed);
                }
            }
        }
        self.suppression.retain(|_, values| !values.is_empty());
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
    }
    /// Remove overrides for a deleted feature. Example: `members.forget_feature(id)`.
    pub fn forget_feature(&mut self, feature: u64) {
        for values in self.suppression.values_mut() {
            values.remove(&feature);
        }
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
