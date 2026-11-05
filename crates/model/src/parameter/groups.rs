// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::ParameterGroupKey;

/// Read-only group projection. Membership is addressed by stable parameter IDs.
#[derive(Clone, Debug, PartialEq)]
pub struct ParameterGroup<Id> {
    pub key: ParameterGroupKey,
    pub display_name: String,
    pub client_id: String,
    pub(super) members: BTreeSet<Id>,
}

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Read groups in creation order; for example `parameters.groups()`.
    pub fn groups(&self) -> &[ParameterGroup<Id>] {
        &self.groups
    }

    /// Read a group by its immutable key; for example `parameters.group("gears:ratios")`.
    pub fn group(&self, key: &str) -> Option<&ParameterGroup<Id>> {
        self.groups.iter().find(|group| group.key.0 == key)
    }

    /// Create an empty group; for example `parameters.add_group("ratios", "Gear ratios", "addin")`.
    pub fn add_group(
        &mut self,
        key: &str,
        display: &str,
        client: &str,
    ) -> Result<(), ParameterError<Id>> {
        if key.is_empty() || self.group(key).is_some() {
            return Err(group_error(key, key, "a unique nonempty internal key"));
        }
        self.groups.push(ParameterGroup {
            key: ParameterGroupKey(key.to_owned().into()),
            display_name: if display.is_empty() {
                key.into()
            } else {
                display.into()
            },
            client_id: client.into(),
            members: BTreeSet::new(),
        });
        Ok(())
    }

    /// Rename presentation without changing identity; for example `parameters.set_group_display_name("ratios", "Ratios")`.
    pub fn set_group_display_name(
        &mut self,
        key: &str,
        display: &str,
    ) -> Result<(), ParameterError<Id>> {
        let index = self.group_index(key)?;
        // #1612: enforce this on the aggregate, independent of UI or wire callers.
        if display.is_empty() {
            return Err(group_error(key, display, "a nonempty display name"));
        }
        self.groups[index].display_name = display.into();
        Ok(())
    }

    /// Remove a group, optionally deleting its members; for example `parameters.delete_group("ratios", false)`.
    pub fn delete_group(
        &mut self,
        key: &str,
        delete_parameters: bool,
    ) -> Result<(), ParameterError<Id>> {
        let index = self.group_index(key)?;
        let members = self.group_members(key)?;
        if delete_parameters {
            self.validate_group_cascade(&members)?;
            for id in members {
                self.remove_for_owner(id)?;
            }
        }
        self.groups.remove(index);
        Ok(())
    }

    pub(super) fn group_index(&self, key: &str) -> Result<usize, ParameterError<Id>> {
        self.groups
            .iter()
            .position(|group| group.key.0 == key)
            .ok_or_else(|| group_error(key, key, "an existing group key"))
    }

    fn validate_group_cascade(&self, members: &[Id]) -> Result<(), ParameterError<Id>> {
        let removed: BTreeSet<Id> = members.iter().copied().collect();
        for &id in members {
            let readers = self.removal_blockers(id, &removed)?;
            if !readers.is_empty() {
                return Err(ParameterError::InUse { id, readers });
            }
        }
        Ok(())
    }
}

fn group_error<Id>(key: &str, found: &str, expected: &'static str) -> ParameterError<Id> {
    ParameterError::Group {
        key: key.into(),
        found: found.into(),
        expected,
    }
}
