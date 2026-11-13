// SPDX-License-Identifier: GPL-2.0-only
//! Read assembly occurrence rows from their owning documents.
use super::*;
use oblikovati_gui::GuiBrowserOccurrence;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

pub(super) fn browser_occurrences<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
) -> Result<Vec<GuiBrowserOccurrence>, String> {
    occurrence_rows(files, document, 0)
}

fn occurrence_rows<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    depth: usize,
) -> Result<Vec<GuiBrowserOccurrence>, String> {
    if depth > 128 {
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
        .map(|occurrence| occurrence_row(files, document, occurrence, depth))
        .collect()
}

fn occurrence_row<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    occurrence: &AssemblyOccurrenceRecipe,
    depth: usize,
) -> Result<GuiBrowserOccurrence, String> {
    let reference = (!occurrence.virtual_).then_some(occurrence.component.as_str());
    let target = reference.and_then(|name| files.document_reference(owner, name));
    let error = reference.filter(|_| target.is_none()).map(|name| {
        files
            .document_reference_error(owner, name)
            .unwrap_or("reference is not resolved")
            .to_owned()
    });
    let children = match target {
        Some(id) => occurrence_rows(files, id, depth + 1)?,
        None => Vec::new(),
    };
    Ok(GuiBrowserOccurrence {
        name: occurrence.name.clone(),
        component: occurrence.component.clone(),
        suppressed: occurrence.suppressed,
        virtual_: occurrence.virtual_,
        error,
        children,
    })
}
