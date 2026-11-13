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
        let (owner, ordinal, previous) = validate_occurrence_path(context.files, root, path)?;
        if previous == suppressed {
            return Ok(());
        }
        let label = if suppressed {
            "Suppress Component"
        } else {
            "Unsuppress Component"
        };
        context.files.commands(owner).map_err(|cause| Error(cause.to_string()))?
            .run(label, |snapshot, _| {
                let rows = snapshot.assembly.as_mut().ok_or_else(|| format!("document {owner}: expected an assembly"))?;
                let row = rows.get_mut(ordinal).ok_or_else(|| format!("occurrence index {ordinal}: expected a current occurrence in document {owner}"))?;
                row.suppressed = suppressed;
                Ok::<(), String>(())
            }).map_err(|cause| Error(cause.to_string()))
    }
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
