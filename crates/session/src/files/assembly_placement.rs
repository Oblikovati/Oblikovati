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
        if name.trim().is_empty() {
            return Err(format!(
                "occurrence name {name:?}: expected a nonempty name"
            ));
        }
        let reference = self.component_reference_path(owner, component)?;
        validate(&transform)?;
        let ground_first = self
            .retained_file(owner)
            .map_err(|error| error.to_string())?
            .package
            .ground_first_component()
            .map_err(|error| error.to_string())?;
        self.with_component_binding(owner, component, &reference, |files| {
            files
                .commands(owner)
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
                                    component: reference.clone(),
                                    transform: transform.cells.to_vec(),
                                    grounded,
                                    ..Default::default()
                                });
                                Ok(id)
                            },
                            project,
                        )
                        .map_err(|error| error.to_string())
                })
        })
    }

    /// Replace one occurrence with a bound component and prepare the typed reply.
    /// Example: `files.replace_occurrence(owner, id, component, |row| Ok(row.clone()))?`.
    pub fn replace_occurrence<T>(
        &mut self,
        owner: u64,
        id: u64,
        component: u64,
        project: impl FnOnce(&AssemblyOccurrenceRecipe) -> Result<T, String>,
    ) -> Result<T, String> {
        let reference = self.component_reference_path(owner, component)?;
        let (ordinal, source) = self.occurrence_for_replacement(owner, id)?;
        if source
            .component_reference()
            .and_then(|name| self.document_reference(owner, name))
            == Some(component)
        {
            return project(source);
        }
        let mut candidate = source.clone();
        candidate.component.clone_from(&reference);
        candidate.virtual_ = false;
        candidate.virtual_part_number.clear();
        candidate.virtual_description.clear();
        candidate.virtual_structure.clear();
        let reply = project(&candidate)?;
        self.with_component_binding(owner, component, &reference, |files| {
            files.commit_replacement(owner, ordinal, candidate)?;
            Ok(reply)
        })
    }

    fn occurrence_for_replacement(
        &self,
        owner: u64,
        id: u64,
    ) -> Result<(usize, &AssemblyOccurrenceRecipe), String> {
        let session = self
            .workspace
            .session(owner)
            .map_err(|cause| cause.to_string())?;
        session
            .snapshot()
            .assembly
            .iter()
            .flatten()
            .enumerate()
            .find(|(_, row)| id != 0 && row.session_id == id)
            .ok_or_else(|| {
                format!("assembly {owner} occurrence {id}: expected a current top-level occurrence")
            })
    }

    fn commit_replacement(
        &mut self,
        owner: u64,
        ordinal: usize,
        candidate: AssemblyOccurrenceRecipe,
    ) -> Result<(), String> {
        self.commands(owner)
            .map_err(|cause| cause.to_string())?
            .run("Replace Component", |snapshot, _| {
                let row = snapshot
                    .assembly
                    .as_mut()
                    .and_then(|rows| rows.get_mut(ordinal))
                    .filter(|row| row.session_id == candidate.session_id)
                    .ok_or_else(|| {
                        format!(
                            "assembly {owner} index {ordinal}: expected occurrence {}",
                            candidate.session_id
                        )
                    })?;
                *row = candidate;
                Ok::<(), String>(())
            })
            .map_err(|cause| cause.to_string())
    }

    fn with_component_binding<T>(
        &mut self,
        owner: u64,
        component: u64,
        reference: &str,
        edit: impl FnOnce(&mut Self) -> Result<T, String>,
    ) -> Result<T, String> {
        let mut proposed = self.document_references.clone();
        let existing = proposed
            .entry(owner)
            .or_default()
            .insert(reference.to_owned(), component);
        if existing.is_some_and(|id| id != component) {
            return Err(format!(
                "component reference {reference:?}: expected the same owned document"
            ));
        }
        self.reference_graph(&proposed)
            .map_err(|error| error.to_string())?;
        let previous = std::mem::replace(&mut self.document_references, proposed);
        let result = edit(self);
        if result.is_err() {
            self.document_references = previous;
        }
        result
    }

    fn component_reference_path(&self, owner: u64, component: u64) -> Result<String, String> {
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
        let path = self.path(component).map_err(|error| error.to_string())?;
        path.to_str()
            .map(str::to_owned)
            .ok_or_else(|| format!("component path {path:?}: expected UTF-8"))
    }
}
