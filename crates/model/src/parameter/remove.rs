// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl<Id: Copy + Ord + Debug> ParameterCollection<Id> {
    /// Delete only an unused parameter; for example `parameters.remove(id)`.
    pub fn remove(&mut self, id: Id) -> Result<(), ParameterError<Id>> {
        let record = self.require_record(id)?;
        let mut readers: Vec<String> = self
            .graph
            .readers(id)?
            .iter()
            .map(|reader| self.records[reader].state.name.clone())
            .collect();
        if record.state.kind == MODEL_PARAM {
            readers.push("owning feature dimension".into());
        }
        if !readers.is_empty() {
            return Err(ParameterError::InUse { id, readers });
        }
        self.remove_for_owner(id)
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
