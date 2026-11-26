// SPDX-License-Identifier: GPL-2.0-only
//! Generation-checked edits to authored assembly occurrences.
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'_, O, W, F> {
    /// Set suppression on a top-level occurrence of the identified assembly.
    /// Example: `host.set_occurrence_suppressed(root, &path, true)?;`.
    pub fn set_occurrence_suppressed(
        &self,
        root: u64,
        path: &[(u64, u64, usize)],
        suppressed: bool,
    ) -> Result<(), Error> {
        let mut context = self
            .context
            .try_borrow_mut()
            .map_err(|_| busy("set_occurrence_suppressed"))?;
        set_suppressed(context.files, root, path, suppressed)
    }
}

fn set_suppressed<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    root: u64,
    path: &[(u64, u64, usize)],
    suppressed: bool,
) -> Result<(), Error> {
    let (owner, ordinal, row) = validate_occurrence_path(files, root, path)?;
    if row.suppressed == suppressed {
        return Ok(());
    }
    let mut candidate = row.clone();
    candidate.suppressed = suppressed;
    commit_occurrence(
        files,
        owner,
        ordinal,
        candidate,
        suppression_label(suppressed),
    )
}

fn suppression_label(suppressed: bool) -> &'static str {
    if suppressed {
        "Suppress Component"
    } else {
        "Unsuppress Component"
    }
}

pub(super) fn suppress_wire<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: SuppressOccurrenceArgs,
) -> Result<Vec<u8>, Error> {
    edit_wire(files, args.id, suppression_label(args.suppressed), |row| {
        row.suppressed = args.suppressed;
        Ok(())
    })
}

pub(super) fn ground_wire<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: GroundOccurrenceArgs,
) -> Result<Vec<u8>, Error> {
    let label = if args.grounded {
        "Ground Component"
    } else {
        "Unground Component"
    };
    edit_wire(files, args.id, label, |row| {
        row.grounded = args.grounded;
        Ok(())
    })
}

pub(super) fn transform_wire<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: TransformOccurrenceArgs,
) -> Result<Vec<u8>, Error> {
    oblikovati_martensite::validate_occurrence_placement(&args.transform).map_err(Error)?;
    edit_wire(files, args.id, "Transform Component", |row| {
        if row.placement().map_err(Error)? != args.transform {
            row.transform = args.transform.cells.to_vec();
        }
        Ok(())
    })
}

fn edit_wire<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    id: u64,
    label: &str,
    update: impl FnOnce(&mut AssemblyOccurrenceRecipe) -> Result<(), Error>,
) -> Result<Vec<u8>, Error> {
    let (owner, ordinal, previous) = active_occurrence(files, id)?;
    let mut candidate = previous.clone();
    update(&mut candidate)?;
    let occurrence = crate::file_queries::assembly::occurrence(files, owner, &candidate, 0)?;
    let response = encode(&OccurrenceResult { occurrence })?;
    if candidate != *previous {
        commit_occurrence(files, owner, ordinal, candidate, label)?;
    }
    Ok(response)
}

fn active_occurrence<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    id: u64,
) -> Result<(u64, usize, &AssemblyOccurrenceRecipe), Error> {
    let owner = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("assembly occurrence edit: expected an active assembly".into()))?;
    let session = files
        .workspace()
        .session(owner)
        .map_err(|cause| Error(cause.to_string()))?;
    let records = session
        .snapshot()
        .assembly
        .as_deref()
        .ok_or_else(|| Error(format!("document {owner}: expected an assembly")))?;
    crate::file_queries::assembly::validate_ids(owner, records)?;
    records
        .iter()
        .enumerate()
        .find(|(_, row)| row.session_id == id)
        .map(|(ordinal, row)| (owner, ordinal, row))
        .ok_or_else(|| {
            Error(format!(
                "assembly {owner} occurrence ID {id}: expected a current top-level occurrence"
            ))
        })
}

fn commit_occurrence<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: u64,
    ordinal: usize,
    candidate: AssemblyOccurrenceRecipe,
    label: &str,
) -> Result<(), Error> {
    files
        .commands(owner)
        .map_err(|cause| Error(cause.to_string()))?
        .run(label, |snapshot, _| {
            let row = snapshot
                .assembly
                .as_mut()
                .and_then(|rows| rows.get_mut(ordinal))
                .filter(|row| row.session_id == candidate.session_id)
                .ok_or_else(|| {
                    Error(format!(
                        "assembly {owner} index {ordinal}: expected occurrence {}",
                        candidate.session_id
                    ))
                })?;
            *row = candidate;
            Ok::<(), Error>(())
        })
        .map_err(|cause| Error(cause.to_string()))
}

fn validate_occurrence_path<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &'a FileWorkspace<O, W, F>,
    root: u64,
    path: &[(u64, u64, usize)],
) -> Result<(u64, usize, &'a AssemblyOccurrenceRecipe), Error> {
    let [(owner, generation, ordinal)] = path else {
        return Err(Error(format!(
            "occurrence path {path:?}: expected one top-level occurrence in document {root}; open a nested assembly to edit its components"
        )));
    };
    let (owner, generation, ordinal) = (*owner, *generation, *ordinal);
    if owner != root {
        return Err(Error(format!(
            "occurrence path {path:?}: expected the root document {root}"
        )));
    }
    let session = files
        .workspace()
        .session(owner)
        .map_err(|cause| Error(cause.to_string()))?;
    if session.generation() != generation {
        return Err(Error(format!(
            "occurrence path {path:?}: expected current generation {} for document {owner}",
            session.generation()
        )));
    }
    let row = session.snapshot().assembly.as_ref().and_then(|rows| rows.get(ordinal))
        .ok_or_else(|| Error(format!("occurrence path {path:?}: expected an occurrence at index {ordinal} in assembly {owner}")))?;
    Ok((owner, ordinal, row))
}
