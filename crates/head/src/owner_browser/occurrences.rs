// SPDX-License-Identifier: GPL-2.0-only
//! Read assembly occurrence rows from their owning documents.
use super::*;
use oblikovati_gui::GuiBrowserOccurrence;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

pub(in super::super) fn contains_occurrence(
    rows: &[GuiBrowserOccurrence],
    path: &[(u64, u64, usize)],
) -> bool {
    rows.iter()
        .any(|row| row.path == path || contains_occurrence(&row.children, path))
}

pub(super) fn browser_occurrences<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
) -> Result<Vec<GuiBrowserOccurrence>, String> {
    occurrence_rows(files, document, &[])
}

fn occurrence_rows<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    parent: &[(u64, u64, usize)],
) -> Result<Vec<GuiBrowserOccurrence>, String> {
    if parent.len() > 128 {
        return Err(format!(
            "assembly document {document}: expected at most 128 nested levels"
        ));
    }
    let session = files
        .workspace()
        .session(document)
        .map_err(|error| error.to_string())?;
    let Some(occurrences) = &session.snapshot().assembly else {
        return Ok(Vec::new());
    };
    occurrences
        .iter()
        .enumerate()
        .map(|(index, occurrence)| {
            occurrence_row(
                files,
                document,
                occurrence,
                parent,
                (session.generation(), index),
            )
        })
        .collect()
}

fn occurrence_row<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    occurrence: &AssemblyOccurrenceRecipe,
    parent: &[(u64, u64, usize)],
    identity: (u64, usize),
) -> Result<GuiBrowserOccurrence, String> {
    let mut path = parent.to_vec();
    path.push((owner, identity.0, identity.1));
    let reference = (!occurrence.virtual_).then_some(occurrence.component.as_str());
    let target = reference.and_then(|name| files.document_reference(owner, name));
    let deferred = reference.is_some()
        && files
            .express_assembly_view(owner)
            .map_err(|e| e.to_string())?
            .is_some();
    let error = reference
        .filter(|_| target.is_none() && !deferred)
        .map(|name| {
            files
                .document_reference_error(owner, name)
                .unwrap_or("reference is not resolved")
                .to_owned()
        });
    let children = match target {
        Some(id) => occurrence_rows(files, id, &path)?,
        None => Vec::new(),
    };
    Ok(GuiBrowserOccurrence {
        path,
        name: occurrence.name.clone(),
        component: occurrence.component.clone(),
        suppressed: occurrence.suppressed,
        virtual_: occurrence.virtual_,
        deferred,
        error,
        children,
    })
}
