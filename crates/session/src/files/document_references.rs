// SPDX-License-Identifier: GPL-2.0-only
//! Retain shared referenced documents for drawings and assemblies.
use super::*;
use oblikovati_model::dependency::DependencyGraph;
use std::collections::BTreeSet;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Attach an authored reference without creating cycles; example: `files.attach_document_reference(owner, name, model, true)?`.
    pub fn attach_document_reference(
        &mut self,
        owner: u64,
        name: String,
        model: u64,
        background: bool,
    ) -> Result<(), FileWorkspaceError> {
        self.retained_file(owner)?;
        self.retained_file(model)?;
        if name.is_empty() {
            return Err(FileWorkspaceError::ReferenceGraph(format!(
                "document {owner} reference {name:?}: expected a nonempty reference"
            )));
        }
        let kind = self.document_type(model)?;
        if !matches!(
            kind,
            oblikovati_api::types::DOCUMENT_PART | oblikovati_api::types::DOCUMENT_ASSEMBLY
        ) {
            return Err(FileWorkspaceError::ReferenceGraph(format!(
                "document {owner} reference {name:?} targets document {model} of type {kind:?}: expected a part or assembly"
            )));
        }
        if !self.reference_is_authored(owner, &name) {
            return Err(FileWorkspaceError::ReferenceGraph(format!(
                "document {owner} reference {name:?}: expected an authored reference"
            )));
        }
        let mut proposed = self.document_references.clone();
        let previous = proposed
            .entry(owner)
            .or_default()
            .insert(name.clone(), model);
        self.reference_graph(&proposed)?;
        self.document_references = proposed;
        if let Some(errors) = self.document_reference_errors.get_mut(&owner) {
            errors.remove(&name);
        }
        if background {
            self.background_reference_models.insert(model);
        }
        if let Some(previous) = previous {
            self.release_unused_reference_model(previous);
        }
        Ok(())
    }
    /// Attach the sole model reference of a drawing; example: `files.attach_drawing_reference(id, name, model, true)?`.
    pub fn attach_drawing_reference(
        &mut self,
        drawing: u64,
        name: String,
        model: u64,
        background: bool,
    ) -> Result<(), FileWorkspaceError> {
        self.attach_document_reference(drawing, name.clone(), model, background)?;
        self.document_reference_errors.remove(&drawing);
        let stale: Vec<_> = self
            .document_references
            .get(&drawing)
            .into_iter()
            .flat_map(|refs| refs.keys())
            .filter(|key| **key != name)
            .cloned()
            .collect();
        for key in stale {
            self.detach_document_reference(drawing, &key);
        }
        Ok(())
    }
    /// Find a resolved authored reference; example: `files.document_reference(owner, name)`.
    pub fn document_reference(&self, owner: u64, name: &str) -> Option<u64> {
        self.reference_is_authored(owner, name)
            .then(|| self.document_references.get(&owner)?.get(name).copied())
            .flatten()
            .filter(|model| self.retained.contains_key(model))
    }
    /// Retain a newly opened background component while it has owners.
    /// Example: `files.keep_background_reference(owner, name, component)?` after placement.
    pub fn keep_background_reference(
        &mut self,
        owner: u64,
        name: &str,
        model: u64,
    ) -> Result<(), FileWorkspaceError> {
        if self.document_reference(owner, name) != Some(model) {
            return Err(FileWorkspaceError::ReferenceGraph(format!(
                "document {owner} reference {name:?} target {model}: expected an owned reference"
            )));
        }
        self.background_reference_models.insert(model);
        Ok(())
    }
    /// Transfer a referenced hidden model to the graph; example: `files.keep_referenced_background_model(model)?`.
    pub fn keep_referenced_background_model(
        &mut self,
        model: u64,
    ) -> Result<(), FileWorkspaceError> {
        self.retained_file(model)?;
        if !self.has_reference_dependents(model) {
            return Err(FileWorkspaceError::ReferenceGraph(format!(
                "background model {model}: expected an authored dependent"
            )));
        }
        self.background_reference_models.insert(model);
        Ok(())
    }
    pub(super) fn owned_assembly_references(&self, owner: u64) -> BTreeSet<String> {
        self.document_references
            .get(&owner)
            .into_iter()
            .flat_map(|refs| refs.keys())
            .filter(|name| {
                let path = Path::new(name);
                path.is_absolute()
                    && self
                        .document_reference(owner, name)
                        .and_then(|target| self.path(target).ok())
                        == Some(path)
            })
            .cloned()
            .collect()
    }
    /// Find a drawing model; example: `files.drawing_reference(drawing, name)`.
    pub fn drawing_reference(&self, drawing: u64, name: &str) -> Option<u64> {
        self.document_reference(drawing, name)
    }
    /// Check whether another open document retains this model; example: `files.has_reference_dependents(model)`.
    pub fn has_reference_dependents(&self, model: u64) -> bool {
        self.document_references.iter().any(|(&owner, refs)| {
            refs.iter()
                .any(|(name, &target)| target == model && self.reference_is_authored(owner, name))
        })
    }
    /// Order owners before their referenced models; example: `files.document_close_order()?`.
    pub fn document_close_order(&self) -> Result<Vec<u64>, FileWorkspaceError> {
        let graph = self.reference_graph(&self.document_references)?;
        reference_close_order(&graph, self.workspace.documents())
    }
    fn reference_is_authored(&self, owner: u64, name: &str) -> bool {
        let Ok(session) = self.workspace.session(owner) else {
            return false;
        };
        let snapshot = session.snapshot();
        snapshot
            .drawing
            .as_ref()
            .is_some_and(|drawing| drawing.model_reference() == name)
            || snapshot.assembly.as_ref().is_some_and(|records| {
                records
                    .iter()
                    .any(|record| record.component_reference() == Some(name))
            })
    }
    pub(super) fn reference_graph(
        &self,
        bindings: &BTreeMap<u64, BTreeMap<String, u64>>,
    ) -> Result<DependencyGraph<u64>, FileWorkspaceError> {
        let mut graph = DependencyGraph::default();
        for &id in self.workspace.documents() {
            graph.insert(id).map_err(reference_graph_error)?;
        }
        for (&owner, refs) in bindings {
            let drivers: Vec<_> = refs
                .iter()
                .filter(|(_, target)| self.retained.contains_key(target))
                .map(|(_, &target)| target)
                .collect();
            graph
                .replace_drivers(owner, &drivers)
                .map_err(reference_graph_error)?;
        }
        Ok(graph)
    }
    /// Record one unresolved authored reference; example: `files.reject_document_reference(id, name, reason)?`.
    pub fn reject_document_reference(
        &mut self,
        owner: u64,
        name: String,
        cause: String,
    ) -> Result<(), FileWorkspaceError> {
        self.retained_file(owner)?;
        if name.is_empty() || !self.reference_is_authored(owner, &name) {
            return Err(FileWorkspaceError::ReferenceGraph(format!(
                "document {owner} reference {name:?}: expected a nonempty authored reference"
            )));
        }
        self.detach_document_reference(owner, &name);
        self.store_reference_error(owner, name, cause);
        Ok(())
    }
    /// Read the current failure for an authored reference; example: `files.document_reference_error(id, name)`.
    pub fn document_reference_error(&self, owner: u64, name: &str) -> Option<&str> {
        if !self.reference_is_authored(owner, name) {
            return None;
        }
        self.document_reference_errors
            .get(&owner)?
            .get(name)
            .map(String::as_str)
    }
    /// Remove bindings no longer in the recipe; example: `files.reconcile_document_references(id)?` after an edit.
    pub fn reconcile_document_references(&mut self, owner: u64) -> Result<(), FileWorkspaceError> {
        self.retained_file(owner)?;
        let stale: Vec<String> = self
            .document_references
            .get(&owner)
            .into_iter()
            .flat_map(|references| references.keys())
            .filter(|name| !self.reference_is_authored(owner, name))
            .cloned()
            .collect();
        for name in stale {
            self.detach_document_reference(owner, &name);
        }
        let stale_errors: Vec<String> = self
            .document_reference_errors
            .get(&owner)
            .into_iter()
            .flat_map(|errors| errors.keys())
            .filter(|name| !self.reference_is_authored(owner, name))
            .cloned()
            .collect();
        if let Some(errors) = self.document_reference_errors.get_mut(&owner) {
            for name in stale_errors {
                errors.remove(&name);
            }
        }
        Ok(())
    }
    fn store_reference_error(&mut self, owner: u64, name: String, cause: String) {
        self.document_reference_errors
            .entry(owner)
            .or_default()
            .insert(name, cause);
    }
    /// Record a failed drawing model load; example: `files.reject_drawing_reference(id, name, cause)`.
    pub fn reject_drawing_reference(&mut self, drawing: u64, name: String, cause: String) {
        self.clear_drawing_reference(drawing);
        self.store_reference_error(drawing, name, cause);
    }
    /// Read the last failure for a drawing reference; example: `files.drawing_reference_error(id, name)`.
    pub fn drawing_reference_error(&self, drawing: u64, name: &str) -> Option<&str> {
        self.document_reference_error(drawing, name)
    }
    /// Release a drawing's old model; example: `files.clear_drawing_reference(id)`.
    pub fn clear_drawing_reference(&mut self, drawing: u64) {
        self.document_reference_errors.remove(&drawing);
        self.release_document_references(drawing);
    }
    fn detach_document_reference(&mut self, owner: u64, name: &str) {
        let old = self
            .document_references
            .get_mut(&owner)
            .and_then(|refs| refs.remove(name));
        if let Some(old) = old {
            self.release_unused_reference_model(old);
        }
    }
    pub(super) fn release_document_references(&mut self, owner: u64) {
        self.document_reference_errors.remove(&owner);
        let Some(refs) = self.document_references.remove(&owner) else {
            return;
        };
        for model in refs.into_values().collect::<BTreeSet<_>>() {
            self.release_unused_reference_model(model);
        }
    }
    fn release_unused_reference_model(&mut self, model: u64) {
        if self.has_reference_dependents(model)
            || !self.background_reference_models.contains(&model)
        {
            return;
        }
        let can_release = self
            .document_info(model)
            .is_ok_and(|info| !info.visible && !info.dirty);
        if can_release && self.workspace.close_discarding_changes(model).is_ok() {
            self.background_reference_models.remove(&model);
            self.release_file(model);
        }
    }
}
fn reference_graph_error(
    error: oblikovati_model::dependency::DependencyError<u64>,
) -> FileWorkspaceError {
    FileWorkspaceError::ReferenceGraph(error.to_string())
}
fn reference_close_order(
    graph: &DependencyGraph<u64>,
    documents: &[u64],
) -> Result<Vec<u64>, FileWorkspaceError> {
    let ranks: BTreeMap<_, _> = documents
        .iter()
        .enumerate()
        .map(|(rank, &id)| (id, rank))
        .collect();
    let mut pending = BTreeMap::new();
    let mut ready = BTreeSet::new();
    for &id in documents {
        let count = graph.readers(id).map_err(reference_graph_error)?.len();
        pending.insert(id, count);
        if count == 0 {
            ready.insert((ranks[&id], id));
        }
    }
    let mut order = Vec::with_capacity(documents.len());
    while let Some((_, id)) = ready.pop_first() {
        order.push(id);
        release_close_drivers(graph, id, &ranks, &mut pending, &mut ready)?;
    }
    if order.len() != documents.len() {
        return Err(FileWorkspaceError::ReferenceGraph(format!(
            "documents {documents:?}: expected an acyclic close order"
        )));
    }
    Ok(order)
}
fn release_close_drivers(
    graph: &DependencyGraph<u64>,
    id: u64,
    ranks: &BTreeMap<u64, usize>,
    pending: &mut BTreeMap<u64, usize>,
    ready: &mut BTreeSet<(usize, u64)>,
) -> Result<(), FileWorkspaceError> {
    for driver in graph.drivers(id).map_err(reference_graph_error)? {
        let count = pending
            .get_mut(&driver)
            .expect("registered reference model");
        *count -= 1;
        if *count == 0 {
            ready.insert((ranks[&driver], driver));
        }
    }
    Ok(())
}
