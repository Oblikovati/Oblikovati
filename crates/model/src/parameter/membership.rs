// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::ParameterGroupKey;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Add a membership without affecting other groups; for example `parameters.add_to_group(id, "ratios")`.
    pub fn add_to_group(&mut self, id: Id, key: &str) -> Result<(), ParameterError<Id>> {
        self.require_record(id)?;
        let index = self.group_index(key)?;
        self.groups[index].members.insert(id);
        Ok(())
    }

    /// Remove one membership; for example `parameters.remove_from_group(id, "ratios")`.
    pub fn remove_from_group(&mut self, id: Id, key: &str) -> Result<(), ParameterError<Id>> {
        self.require_record(id)?;
        let index = self.group_index(key)?;
        self.groups[index].members.remove(&id);
        Ok(())
    }

    /// Remove every membership without deleting the parameter; for example `parameters.remove_from_all_groups(id)`.
    pub fn remove_from_all_groups(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        self.require_record(id)?;
        self.detach_group_memberships(id);
        Ok(())
    }

    /// Read a parameter's groups in creation order; for example `parameters.groups_of(id)`.
    pub fn groups_of(&self, id: Id) -> Result<Vec<&ParameterGroupKey>, ParameterError<Id>> {
        self.require_record(id)?;
        Ok(self
            .groups
            .iter()
            .filter(|group| group.members.contains(&id))
            .map(|group| &group.key)
            .collect())
    }

    /// Read members in parameter insertion order; for example `parameters.group_members("ratios")`.
    pub fn group_members(&self, key: &str) -> Result<Vec<Id>, ParameterError<Id>> {
        let index = self.group_index(key)?;
        Ok(self
            .order
            .iter()
            .copied()
            .filter(|id| self.groups[index].members.contains(id))
            .collect())
    }

    pub(super) fn detach_group_memberships(&mut self, id: Id) {
        for group in &mut self.groups {
            group.members.remove(&id);
        }
    }
}
