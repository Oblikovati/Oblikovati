// SPDX-License-Identifier: GPL-2.0-only
//! Create an occurrence and its owned component reference as one operation.
use super::*;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART, Matrix};
use oblikovati_model::{assembly::AssemblyOccurrenceRecipe, history::DocumentSnapshot};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Place a bound component and project its reply before adopting the edit.
    /// Example: `files.place_occurrence(owner, component, "Part:1", matrix, validate, |_, id| Ok(id))?`.
    pub fn place_occurrence<T>(
        &mut self,
        owner: u64,
        component: u64,
        name: &str,
        transform: Matrix,
        validate: impl FnOnce(&Matrix) -> Result<(), String>,
        project: impl FnOnce(&DocumentSnapshot, u64) -> Result<T, String>,
    ) -> Result<T, String> {
        let reference = self.placement_reference(owner, component, name)?;
        validate(&transform)?;
        let ground_first = self
            .retained_file(owner)
            .map_err(|error| error.to_string())?
            .package
            .ground_first_component()
            .map_err(|error| error.to_string())?;
        let previous = self.document_references.clone();
        let mut proposed = previous.clone();
        let existing = proposed
            .entry(owner)
            .or_default()
            .insert(reference.clone(), component);
        if existing.is_some_and(|id| id != component) {
            return Err(format!(
                "component reference {reference:?}: expected the same owned document"
            ));
        }
        self.reference_graph(&proposed)
            .map_err(|error| error.to_string())?;
        self.document_references = proposed;
        let result =
            self.commands(owner)
                .map_err(|error| error.to_string())
                .and_then(|mut commands| {
                    commands
                        .run_projected(
                            "Place component",
                            |snapshot, ids| {
                                let records = snapshot.assembly.as_mut().ok_or_else(|| {
                                    format!("document {owner}: expected an assembly")
                                })?;
                                let id = ids
                                    .allocate_occurrence()
                                    .map_err(|error| error.to_string())?;
                                let grounded = ground_first && records.is_empty();
                                records.push(AssemblyOccurrenceRecipe {
                                    session_id: id,
                                    name: name.into(),
                                    component: reference,
                                    transform: transform.cells.to_vec(),
                                    grounded,
                                    ..Default::default()
                                });
                                Ok(id)
                            },
                            project,
                        )
                        .map_err(|error| error.to_string())
                });
        if result.is_err() {
            self.document_references = previous;
        }
        result
    }

    fn placement_reference(
        &self,
        owner: u64,
        component: u64,
        name: &str,
    ) -> Result<String, String> {
        if self
            .document_type(owner)
            .map_err(|error| error.to_string())?
            != DOCUMENT_ASSEMBLY
        {
            return Err(format!("document {owner}: expected an assembly"));
        }
        let kind = self
            .document_type(component)
            .map_err(|error| error.to_string())?;
        if !matches!(kind, DOCUMENT_PART | DOCUMENT_ASSEMBLY) {
            return Err(format!(
                "component {component} type {kind:?}: expected a part or assembly"
            ));
        }
        if name.trim().is_empty() {
            return Err(format!(
                "occurrence name {name:?}: expected a nonempty name"
            ));
        }
        let path = self.path(component).map_err(|error| error.to_string())?;
        path.to_str()
            .map(str::to_owned)
            .ok_or_else(|| format!("component path {path:?}: expected UTF-8"))
    }
}
