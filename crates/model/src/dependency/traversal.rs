// SPDX-License-Identifier: GPL-2.0-only

use super::{DependencyError, DependencyGraph};
use std::collections::{BTreeMap, BTreeSet};

impl<Id: Copy + Ord> DependencyGraph<Id> {
    /// Order the changed node and all its readers, with each driver first.
    /// Insertion order breaks ties. Independent branches are excluded (#1414).
    /// ```
    /// use oblikovati_model::dependency::DependencyGraph;
    /// let mut graph = DependencyGraph::default();
    /// graph.insert(1).unwrap(); graph.insert(2).unwrap();
    /// graph.replace_drivers(2, &[1]).unwrap();
    /// assert_eq!(graph.recompute_order(1).unwrap(), vec![1, 2]);
    /// ```
    pub fn recompute_order(&self, changed: Id) -> Result<Vec<Id>, DependencyError<Id>> {
        self.require_node(changed)?;
        Ok(self.order_subset(&self.affected_set(changed)))
    }

    /// Order every node with drivers first; useful after ownership changes.
    /// Example: `let order = graph.recompute_all();` includes isolated documents.
    pub fn recompute_all(&self) -> Vec<Id> {
        self.order_subset(&self.nodes.keys().copied().collect())
    }

    fn order_subset(&self, affected: &BTreeSet<Id>) -> Vec<Id> {
        let ranks = self.insertion_ranks();
        let mut pending = self.pending_counts(affected);
        let mut ready = Self::ready_nodes(&pending, &ranks);
        let mut ordered = Vec::with_capacity(affected.len());
        while let Some((_, id)) = ready.pop_first() {
            ordered.push(id);
            self.release_readers(id, &ranks, &mut pending, &mut ready);
        }
        ordered
    }

    fn insertion_ranks(&self) -> BTreeMap<Id, usize> {
        self.insertion_order
            .iter()
            .copied()
            .enumerate()
            .map(|(rank, id)| (id, rank))
            .collect()
    }

    fn ready_nodes(
        pending: &BTreeMap<Id, usize>,
        ranks: &BTreeMap<Id, usize>,
    ) -> BTreeSet<(usize, Id)> {
        pending
            .iter()
            .filter(|(_, count)| **count == 0)
            .map(|(id, _)| (ranks[id], *id))
            .collect()
    }

    // Iteration avoids exhausting the stack on imported, deeply chained recipes.
    pub(super) fn affected_set(&self, changed: Id) -> BTreeSet<Id> {
        let mut affected = BTreeSet::new();
        let mut remaining = vec![changed];
        while let Some(id) = remaining.pop() {
            if affected.insert(id) {
                remaining.extend(self.nodes[&id].readers.iter().copied());
            }
        }
        affected
    }

    fn pending_counts(&self, affected: &BTreeSet<Id>) -> BTreeMap<Id, usize> {
        affected
            .iter()
            .map(|id| (*id, self.nodes[id].drivers.intersection(affected).count()))
            .collect()
    }

    fn release_readers(
        &self,
        id: Id,
        ranks: &BTreeMap<Id, usize>,
        pending: &mut BTreeMap<Id, usize>,
        ready: &mut BTreeSet<(usize, Id)>,
    ) {
        for reader in &self.nodes[&id].readers {
            let Some(count) = pending.get_mut(reader) else {
                continue;
            };
            *count -= 1;
            if *count == 0 {
                ready.insert((ranks[reader], *reader));
            }
        }
    }
}
