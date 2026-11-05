// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Require a feature-owned numeric formula, including one with failed evaluation.
    /// Example: `parameters.validate_owned_scalar(dimension_id)?` before owner adoption.
    pub fn validate_owned_scalar(&self, id: Id) -> Result<(), ParameterError<Id>> {
        let kind = self.require_record(id)?.state.kind;
        if kind != MODEL_PARAM {
            return Err(ParameterError::Kind {
                kind,
                expected: "a Model scalar parameter",
            });
        }
        self.require_formula(id)
    }

    /// Delete only an unused parameter; for example `parameters.remove(id)`.
    pub fn remove(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        let readers = self.removal_blockers(id, &BTreeSet::new())?;
        if !readers.is_empty() {
            return Err(ParameterError::InUse { id, readers });
        }
        self.remove_for_owner(id)
    }

    /// Query authored consumers even when cycle handling detached evaluation edges.
    /// Example: `parameters.authored_readers(dimension_id)?` before owner deletion.
    pub fn authored_readers(&self, id: Id) -> Result<Vec<Id>, ParameterError<Id>> {
        let name = &self.require_record(id)?.state.name;
        Ok(self
            .order
            .iter()
            .copied()
            .filter(|reader| self.records[reader].references().contains(name))
            .collect())
    }

    /// Remove an owner's parameter and invalidate its former readers; for example `parameters.remove_for_owner(dimension_id)`.
    pub fn remove_for_owner(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        let name = self.require_record(id)?.state.name.clone();
        let readers = self.graph.readers(id)?;
        for reader in &readers {
            self.detach_removed_driver(*reader, id)?;
        }
        self.graph.remove(id)?;
        self.records.remove(&id);
        self.detach_group_memberships(id);
        self.detach_derived_parameter(id);
        self.names.remove(&name);
        self.order.retain(|candidate| *candidate != id);
        self.changed.remove(&id);
        // Remove every stale ID binding before any transitive reader evaluates.
        for reader in &readers {
            let _ = self.bind_expression(*reader);
        }
        for reader in readers {
            self.recompute(reader)?;
        }
        Ok(())
    }

    pub(super) fn removal_blockers(
        &self,
        id: Id,
        removed: &BTreeSet<Id>,
    ) -> Result<Vec<String>, ParameterError<Id>> {
        let record = self.require_record(id)?;
        let mut readers: Vec<String> = self
            .graph
            .readers(id)?
            .iter()
            .filter(|reader| !removed.contains(reader))
            .map(|reader| self.records[reader].state.name.clone())
            .collect();
        if record.state.kind == MODEL_PARAM {
            readers.push("owning feature dimension".into());
        }
        Ok(readers)
    }

    fn detach_removed_driver(&mut self, reader: Id, removed: Id) -> Result<(), ParameterError<Id>> {
        let drivers: Vec<Id> = self
            .graph
            .drivers(reader)?
            .into_iter()
            .filter(|id| *id != removed)
            .collect();
        self.graph.replace_drivers(reader, &drivers)?;
        Ok(())
    }
}
