// SPDX-License-Identifier: GPL-2.0-only
//! Owned model-state members; the built-in Primary member has a stable identity.
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
mod parameter_table;
mod property_table;
mod row_batch;
mod suppression;
mod table_columns;
pub use parameter_table::{
    ModelStateParameterBase, ModelStateParameterTable, ModelStateParameterValue,
};
pub use property_table::{
    ModelStatePropertyBase, ModelStatePropertyTable, ModelStatePropertyValue,
};
pub use row_batch::ModelStateRowEdit;
pub use suppression::ModelStateEditScope;
pub use table_columns::ModelStateTableColumn;

/// A named document member. Example: `members.active().map(|member| member.name.as_str())`.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelStateMember {
    pub id: u64,
    pub name: String,
}

/// Ordered members and active selection, copied by document history.
#[derive(Clone, Debug, PartialEq, Eq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelStateMembers {
    members: Vec<ModelStateMember>,
    active: u64,
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    suppression: BTreeMap<u64, BTreeMap<u64, bool>>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    table_columns: Vec<ModelStateTableColumn>,
    #[serde(default)]
    scope: ModelStateEditScope,
    #[serde(default, skip_serializing_if = "property_table_is_empty")]
    property_table: ModelStatePropertyTable,
    #[serde(default, skip_serializing_if = "ModelStateParameterTable::is_empty")]
    parameter_table: ModelStateParameterTable,
}

impl Default for ModelStateMembers {
    fn default() -> Self {
        Self {
            members: vec![ModelStateMember {
                id: 1,
                name: "Primary".into(),
            }],
            active: 1,
            suppression: BTreeMap::new(),
            table_columns: Vec::new(),
            scope: ModelStateEditScope::SingleMember,
            property_table: ModelStatePropertyTable::default(),
            parameter_table: ModelStateParameterTable::default(),
        }
    }
}

impl ModelStateMembers {
    /// Inspect members in table order. Example: `members.list().len()`.
    pub fn list(&self) -> &[ModelStateMember] {
        &self.members
    }

    /// Inspect the active member. Example: `members.active().map(|member| member.id)`.
    pub fn active(&self) -> Option<&ModelStateMember> {
        self.members.iter().find(|member| member.id == self.active)
    }

    /// Check restored content before adoption. Example: `members.validate()?`.
    pub fn validate(&self) -> Result<(), String> {
        if self.members.first()
            != Some(&ModelStateMember {
                id: 1,
                name: "Primary".into(),
            })
        {
            return Err(format!(
                "members {:?}: expected immutable Primary with ID 1 first",
                self.members
            ));
        }
        let mut seen = Self::default();
        for member in self.members.iter().skip(1) {
            seen.insert(member.id, &member.name)?;
        }
        seen.activate(self.active)?;
        self.validate_suppression()?;
        self.validate_table_columns(&self.table_columns)?;
        self.property_table.validate(&self.members)?;
        self.parameter_table.validate(&self.members)
    }

    /// Create an inactive member with an externally reserved identity. Example: `members.insert(id, "Variant")?`.
    pub fn insert(&mut self, id: u64, name: &str) -> Result<(), String> {
        self.require_unique_name(name, None)?;
        if id == 0 || self.members.iter().any(|member| member.id == id) {
            return Err(format!(
                "model-state ID {id}: expected nonzero unused identity"
            ));
        }
        let inherited = self.suppression.get(&self.active).cloned();
        self.property_table.clone_member(self.active, id);
        self.parameter_table.clone_member(self.active, id);
        self.members.push(ModelStateMember {
            id,
            name: name.into(),
        });
        if let Some(overrides) = inherited {
            self.suppression.insert(id, overrides);
        }
        Ok(())
    }

    /// Rename an authored member without changing identity. Example: `members.rename(id, "Large")?`.
    pub fn rename(&mut self, id: u64, name: &str) -> Result<(), String> {
        if id == 1 {
            return Err("model-state ID 1: expected an authored member for rename".into());
        }
        self.require_unique_name(name, Some(id))?;
        let member = self
            .members
            .iter_mut()
            .find(|member| member.id == id)
            .ok_or_else(|| format!("model-state ID {id}: expected an existing member"))?;
        member.name = name.into();
        Ok(())
    }

    /// Activate an existing member. Example: `members.activate(1)?`.
    pub fn activate(&mut self, id: u64) -> Result<(), String> {
        if !self.members.iter().any(|member| member.id == id) {
            return Err(format!("model-state ID {id}: expected an existing member"));
        }
        self.active = id;
        Ok(())
    }

    /// Copy a member without changing active selection. Example: `members.copy(source, reserved, "Copy")?`.
    pub fn copy(&mut self, source: u64, reserved: u64, name: &str) -> Result<(), String> {
        if !self.members.iter().any(|member| member.id == source) {
            return Err(format!(
                "model-state source {source}: expected an existing member"
            ));
        }
        let overrides = self.suppression.get(&source).cloned();
        self.insert(reserved, name)?;
        self.property_table.clone_member(source, reserved);
        self.parameter_table.clone_member(source, reserved);
        self.suppression.remove(&reserved);
        if let Some(overrides) = overrides {
            self.suppression.insert(reserved, overrides);
        }
        Ok(())
    }

    /// Delete an authored member, returning selection to Primary when required.
    /// Example: `members.delete(id)?` after checking document edit scope.
    pub fn delete(&mut self, id: u64) -> Result<(), String> {
        if id == 1 {
            return Err("model-state ID 1: expected an authored member for delete".into());
        }
        let index = self
            .members
            .iter()
            .position(|member| member.id == id)
            .ok_or_else(|| format!("model-state ID {id}: expected an existing member"))?;
        self.members.remove(index);
        self.suppression.remove(&id);
        self.property_table.forget_member(id);
        self.parameter_table.forget_member(id);
        if self.active == id {
            self.active = 1;
        }
        Ok(())
    }

    /// Reorder every member without changing identities; example: `members.reorder(&[1, 3, 2])?`.
    pub fn reorder(&mut self, order: &[u64]) -> Result<(), String> {
        if order.len() != self.members.len() || order.first() != Some(&1) {
            return Err(format!(
                "model-state order {order:?}: expected Primary first and every member exactly once"
            ));
        }
        let mut reordered = Vec::with_capacity(order.len());
        for id in order {
            let member = self
                .members
                .iter()
                .find(|member| member.id == *id)
                .ok_or_else(|| {
                    format!("model-state order {order:?}: expected existing unique members")
                })?;
            if reordered
                .iter()
                .any(|previous: &ModelStateMember| previous.id == *id)
            {
                return Err(format!(
                    "model-state order {order:?}: expected existing unique members"
                ));
            }
            reordered.push(member.clone());
        }
        self.members = reordered;
        Ok(())
    }

    /// Inspect retained parameter expressions; example: `members.parameter_table()`.
    pub fn parameter_table(&self) -> &ModelStateParameterTable {
        &self.parameter_table
    }

    /// Adopt validated parameter expressions; example: `members.set_parameter_table(candidate)?`.
    pub fn set_parameter_table(
        &mut self,
        candidate: ModelStateParameterTable,
    ) -> Result<(), String> {
        candidate.validate(&self.members)?;
        self.parameter_table = candidate;
        Ok(())
    }

    /// Inspect the property table; example: `members.property_table()`.
    pub fn property_table(&self) -> &ModelStatePropertyTable {
        &self.property_table
    }

    /// Adopt a validated property table; example: `members.set_property_table(candidate)?`.
    pub fn set_property_table(&mut self, candidate: ModelStatePropertyTable) -> Result<(), String> {
        candidate.validate(&self.members)?;
        self.property_table = candidate;
        Ok(())
    }

    /// Choose the first unused generated name. Example: `members.generated_name("Model State")`.
    pub fn generated_name(&self, prefix: &str) -> String {
        let mut index = 1usize;
        let mut name = format!("{prefix}{index}");
        while self.members.iter().any(|member| member.name == name) {
            index += 1;
            name = format!("{prefix}{index}");
        }
        name
    }

    fn require_unique_name(&self, name: &str, except: Option<u64>) -> Result<(), String> {
        if name.trim().is_empty()
            || name.contains('\0')
            || self
                .members
                .iter()
                .any(|member| member.name == name && Some(member.id) != except)
        {
            return Err(format!(
                "model-state name {name:?}: expected a nonempty unique name without NUL"
            ));
        }
        Ok(())
    }
}

fn property_table_is_empty(table: &ModelStatePropertyTable) -> bool {
    table.values().is_empty() && table.bases().is_empty()
}

#[cfg(test)]
mod property_table_tests;
#[cfg(test)]
mod tests;

#[cfg(test)]
mod parameter_table_tests;
#[cfg(test)]
mod row_batch_tests;
#[cfg(test)]
mod suppression_table_tests;
#[cfg(test)]
mod table_columns_tests;
