// SPDX-License-Identifier: GPL-2.0-only
//! Generation-checked edits to authored assembly occurrences.
use super::*;

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
    let (owner, ordinal, previous) = validate_occurrence_path(files, root, path)?;
    if previous == suppressed {
        return Ok(());
    }
    let label = if suppressed {
        "Suppress Component"
    } else {
        "Unsuppress Component"
    };
    files
        .commands(owner)
        .map_err(|cause| Error(cause.to_string()))?
        .run(label, |snapshot, _| {
            let rows = snapshot
                .assembly
                .as_mut()
                .ok_or_else(|| format!("document {owner}: expected an assembly"))?;
            let row = rows.get_mut(ordinal).ok_or_else(|| {
                format!(
                    "occurrence index {ordinal}: expected a current occurrence in document {owner}"
                )
            })?;
            row.suppressed = suppressed;
            Ok::<(), String>(())
        })
        .map_err(|cause| Error(cause.to_string()))
}

pub(super) fn suppress_wire<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: SuppressOccurrenceArgs,
) -> Result<Vec<u8>, Error> {
    let root = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("assembly.suppress: expected an active assembly".into()))?;
    let session = files
        .workspace()
        .session(root)
        .map_err(|cause| Error(cause.to_string()))?;
    let records = session
        .snapshot()
        .assembly
        .as_ref()
        .ok_or_else(|| Error(format!("document {root}: expected an assembly")))?;
    crate::file_queries::assembly::validate_ids(root, records)?;
    let matches: Vec<_> = records
        .iter()
        .enumerate()
        .filter(|(_, row)| row.session_id == args.id && args.id != 0)
        .collect();
    let [(ordinal, row)] = matches.as_slice() else {
        return Err(Error(format!(
            "assembly occurrence ID {}: expected one current top-level occurrence",
            args.id
        )));
    };
    let mut reply = crate::file_queries::assembly::occurrence(files, root, row, 0)?;
    reply.suppressed = args.suppressed;
    let encoded = encode(&OccurrenceResult { occurrence: reply })?;
    let path = [(root, session.generation(), *ordinal)];
    set_suppressed(files, root, &path, args.suppressed)?;
    Ok(encoded)
}

fn validate_occurrence_path<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    root: u64,
    path: &[(u64, u64, usize)],
) -> Result<(u64, usize, bool), Error> {
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
    Ok((owner, ordinal, row.suppressed))
}
