// SPDX-License-Identifier: GPL-2.0-only
//! Project owned assembly occurrences to the public wire contract.
use super::*;
use oblikovati_model::assembly::AssemblyOccurrenceRecipe;

pub(super) fn occurrences<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<OccurrencesResult, Error> {
    let document = files
        .workspace()
        .active_document()
        .ok_or_else(|| Error("assembly.occurrences: expected an active assembly".into()))?;
    Ok(OccurrencesResult {
        occurrences: Some(rows(files, document, 0)?),
    })
}

pub(crate) fn rows<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    depth: usize,
) -> Result<Vec<OccurrenceInfo>, Error> {
    project_rows(files, document, depth, true)
}

fn project_rows<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    depth: usize,
    require_resolution: bool,
) -> Result<Vec<OccurrenceInfo>, Error> {
    if depth > 128 {
        return Err(Error(format!(
            "assembly document {document}: expected at most 128 nested levels"
        )));
    }
    let session = files
        .workspace()
        .session(document)
        .map_err(|error| Error(error.to_string()))?;
    let records = session
        .snapshot()
        .assembly
        .as_ref()
        .ok_or_else(|| Error(format!("document {document}: expected an assembly")))?;
    validate_ids(document, records)?;
    records
        .iter()
        .map(|record| project_occurrence(files, document, record, depth, require_resolution))
        .collect()
}

pub(crate) fn validate_ids(
    document: u64,
    records: &[AssemblyOccurrenceRecipe],
) -> Result<(), Error> {
    let mut ids = std::collections::BTreeSet::new();
    for record in records {
        if record.session_id == 0 || !ids.insert(record.session_id) {
            return Err(Error(format!(
                "assembly {document} occurrence ID {}: expected a unique assigned session ID",
                record.session_id
            )));
        }
    }
    Ok(())
}

pub(crate) fn occurrence<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    record: &AssemblyOccurrenceRecipe,
    depth: usize,
) -> Result<OccurrenceInfo, Error> {
    project_occurrence(files, owner, record, depth, true)
}

pub(crate) fn after_removal<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    removed: u64,
    records: &[AssemblyOccurrenceRecipe],
) -> Result<OccurrencesResult, Error> {
    let occurrences = records
        .iter()
        .filter(|row| row.session_id != removed)
        .map(|row| project_occurrence(files, owner, row, 0, false))
        .collect::<Result<Vec<_>, _>>()?;
    Ok(OccurrencesResult {
        occurrences: Some(occurrences),
    })
}

fn project_occurrence<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    record: &AssemblyOccurrenceRecipe,
    depth: usize,
    require_resolution: bool,
) -> Result<OccurrenceInfo, Error> {
    let Some(name) = record.component_reference() else {
        return occurrence_info(record, None);
    };
    let Some(target) = files.document_reference(owner, name) else {
        if !require_resolution {
            return occurrence_info(record, None);
        }
        let reason = files
            .document_reference_error(owner, name)
            .unwrap_or("reference is not resolved");
        return Err(Error(format!(
            "assembly {owner} occurrence {:?} component {name:?}: expected a resolved reference: {reason}",
            record.name
        )));
    };
    let session = files
        .workspace()
        .session(target)
        .map_err(|error| Error(error.to_string()))?;
    let children = if session.snapshot().assembly.is_some() {
        Some(project_rows(files, target, depth + 1, require_resolution)?)
    } else {
        None
    };
    occurrence_info(record, children)
}

pub(crate) fn occurrence_info(
    record: &AssemblyOccurrenceRecipe,
    children: Option<Vec<OccurrenceInfo>>,
) -> Result<OccurrenceInfo, Error> {
    Ok(OccurrenceInfo {
        id: record.session_id,
        name: record.name.clone(),
        transform: record.placement().map_err(Error)?,
        suppressed: record.suppressed,
        grounded: record.grounded,
        adaptive: record.adaptive,
        flexible: record.flexible,
        substitute: record.substitute,
        visible: true,
        enabled: true,
        virtual_: record.virtual_,
        part_number: record.virtual_part_number.clone(),
        children,
        ..Default::default()
    })
}
