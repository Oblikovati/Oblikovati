// SPDX-License-Identifier: GPL-2.0-only

//! Dependency tracking ported from Go `model/param/graph.go` at aa8297d1.
//! The owner supplies stable IDs. Graph edits are atomic; authored expressions
//! and parameter health belong to the parameter collection above this layer.

mod error;
mod traversal;

pub use error::DependencyError;
use std::collections::{BTreeMap, BTreeSet};

#[derive(Clone, Debug)]
struct DependencyNode<Id> {
    drivers: BTreeSet<Id>,
    readers: BTreeSet<Id>,
}

/// Acyclic dependencies with insertion-order queries and deterministic evaluation.
#[derive(Clone, Debug)]
pub struct DependencyGraph<Id> {
    nodes: BTreeMap<Id, DependencyNode<Id>>,
    insertion_order: Vec<Id>,
}

impl<Id: Copy + Ord> Default for DependencyGraph<Id> {
    fn default() -> Self {
        Self {
            nodes: BTreeMap::new(),
            insertion_order: Vec::new(),
        }
    }
}

impl<Id: Copy + Ord> DependencyGraph<Id> {
    /// Register an isolated node; existing IDs must not be reused.
    /// ```
    /// use oblikovati_model::dependency::DependencyGraph;
    /// let mut graph = DependencyGraph::default();
    /// graph.insert(42).unwrap();
    /// ```
    pub fn insert(&mut self, id: Id) -> Result<(), DependencyError<Id>> {
        if self.nodes.contains_key(&id) {
            return Err(DependencyError::Duplicate(id));
        }
        self.nodes.insert(
            id,
            DependencyNode {
                drivers: BTreeSet::new(),
                readers: BTreeSet::new(),
            },
        );
        self.insertion_order.push(id);
        Ok(())
    }

    /// Replace all drivers after validation, leaving the graph intact on error.
    /// ```
    /// use oblikovati_model::dependency::DependencyGraph;
    /// let mut graph = DependencyGraph::default();
    /// graph.insert(1).unwrap(); graph.insert(2).unwrap();
    /// graph.replace_drivers(2, &[1]).unwrap();
    /// ```
    pub fn replace_drivers(&mut self, id: Id, drivers: &[Id]) -> Result<(), DependencyError<Id>> {
        self.require_node(id)?;
        let unique: BTreeSet<Id> = drivers.iter().copied().collect();
        self.validate_drivers(id, &unique)?;
        self.detach_drivers(id);
        for driver in &unique {
            self.nodes
                .get_mut(driver)
                .expect("validated driver")
                .readers
                .insert(id);
        }
        self.nodes.get_mut(&id).expect("validated node").drivers = unique;
        Ok(())
    }

    /// Return immediate drivers in insertion order.
    /// Example: after `graph.replace_drivers(2, &[1])`, `graph.drivers(2)` is `[1]`.
    pub fn drivers(&self, id: Id) -> Result<Vec<Id>, DependencyError<Id>> {
        self.require_node(id)?;
        Ok(self.ordered_subset(&self.nodes[&id].drivers))
    }

    /// Return immediate readers in insertion order.
    /// Example: after `graph.replace_drivers(2, &[1])`, `graph.readers(1)` is `[2]`.
    pub fn readers(&self, id: Id) -> Result<Vec<Id>, DependencyError<Id>> {
        self.require_node(id)?;
        Ok(self.ordered_subset(&self.nodes[&id].readers))
    }

    /// Remove an unused node; refusal lists its readers and changes nothing.
    /// Example: `graph.remove(2)` succeeds after node 2 has no readers.
    pub fn remove(&mut self, id: Id) -> Result<(), DependencyError<Id>> {
        let readers = self.readers(id)?;
        if !readers.is_empty() {
            return Err(DependencyError::InUse { id, readers });
        }
        self.detach_drivers(id);
        self.nodes.remove(&id);
        self.insertion_order.retain(|candidate| *candidate != id);
        Ok(())
    }

    fn require_node(&self, id: Id) -> Result<(), DependencyError<Id>> {
        if !self.nodes.contains_key(&id) {
            return Err(DependencyError::Missing(id));
        }
        Ok(())
    }

    fn validate_drivers(&self, id: Id, drivers: &BTreeSet<Id>) -> Result<(), DependencyError<Id>> {
        for driver in drivers {
            self.require_node(*driver)?;
        }
        let affected = self.affected_set(id);
        if let Some(driver) = drivers.intersection(&affected).next() {
            return Err(DependencyError::Cycle {
                id,
                driver: *driver,
            });
        }
        Ok(())
    }

    fn detach_drivers(&mut self, id: Id) {
        let drivers = std::mem::take(&mut self.nodes.get_mut(&id).expect("validated node").drivers);
        for driver in drivers {
            self.nodes
                .get_mut(&driver)
                .expect("registered driver")
                .readers
                .remove(&id);
        }
    }

    fn ordered_subset(&self, members: &BTreeSet<Id>) -> Vec<Id> {
        self.insertion_order
            .iter()
            .copied()
            .filter(|id| members.contains(id))
            .collect()
    }
}
