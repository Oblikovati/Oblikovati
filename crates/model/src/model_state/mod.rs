// SPDX-License-Identifier: GPL-2.0-only
//! Owned model-state members; the built-in Primary member has a stable identity.
use serde::{Deserialize, Serialize};

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
}

impl Default for ModelStateMembers {
    fn default() -> Self {
        Self {
            members: vec![ModelStateMember {
                id: 1,
                name: "Primary".into(),
            }],
            active: 1,
        }
    }
}

impl ModelStateMembers {
    /// Inspect members in creation order. Example: `members.list().len()`.
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
        seen.activate(self.active)
    }

    /// Create an inactive member with an externally reserved identity. Example: `members.insert(id, "Variant")?`.
    pub fn insert(&mut self, id: u64, name: &str) -> Result<(), String> {
        self.require_unique_name(name, None)?;
        if id == 0 || self.members.iter().any(|member| member.id == id) {
            return Err(format!(
                "model-state ID {id}: expected nonzero unused identity"
            ));
        }
        self.members.push(ModelStateMember {
            id,
            name: name.into(),
        });
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
        self.insert(reserved, name)
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
        if self.active == id {
            self.active = 1;
        }
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
            || self
                .members
                .iter()
                .any(|member| member.name == name && Some(member.id) != except)
        {
            return Err(format!(
                "model-state name {name:?}: expected a nonempty unique name"
            ));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests;
