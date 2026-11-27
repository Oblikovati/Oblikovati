// SPDX-License-Identifier: GPL-2.0-only
//! Author additional occurrences of an already owned component definition.
use super::*;
use crate::file_queries::assembly::{occurrence, occurrence_info, validate_ids};
use oblikovati_model::{
    assembly::AssemblyOccurrenceRecipe,
    history::{DocumentSnapshot, RecipeReservations},
};

pub(super) fn place_one<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: PlaceByDefinitionArgs,
) -> Result<Vec<u8>, Error> {
    let placement = BatchPlacement {
        name: args.name,
        transform: args.transform,
    };
    place(
        files,
        args.source,
        vec![placement],
        "Place component",
        |mut rows| {
            let occurrence = rows
                .pop()
                .ok_or_else(|| Error("expected one new occurrence".into()))?;
            encode(&OccurrenceResult { occurrence })
        },
    )
}

pub(super) fn place_batch<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: PlaceByDefinitionBatchArgs,
) -> Result<Vec<u8>, Error> {
    place(
        files,
        args.source,
        args.placements.unwrap_or_default(),
        "Place components",
        |rows| {
            encode(&PlaceByDefinitionBatchResult {
                occurrences: Some(rows),
            })
        },
    )
}

fn place<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    source: u64,
    placements: Vec<BatchPlacement>,
    label: &str,
    project: impl FnOnce(Vec<OccurrenceInfo>) -> Result<Vec<u8>, Error>,
) -> Result<Vec<u8>, Error> {
    let owner = files.workspace().active_document().ok_or_else(|| {
        Error("assembly placement by definition: expected an active assembly".into())
    })?;
    let session = files
        .workspace()
        .session(owner)
        .map_err(|cause| Error(cause.to_string()))?;
    let rows = session
        .snapshot()
        .assembly
        .as_deref()
        .ok_or_else(|| Error(format!("document {owner}: expected an assembly")))?;
    let definition = source_definition(owner, rows, source)?;
    if placements.is_empty() {
        return project(Vec::new());
    }
    let children = occurrence(files, owner, definition, 0)?.children;
    let drafts = placements
        .into_iter()
        .map(|item| new_recipe(definition, item))
        .collect::<Result<Vec<_>, _>>()?;
    commit(files, owner, label, drafts, children, project)
}

fn source_definition(
    owner: u64,
    rows: &[AssemblyOccurrenceRecipe],
    source: u64,
) -> Result<&AssemblyOccurrenceRecipe, Error> {
    validate_ids(owner, rows)?;
    rows.iter()
        .find(|row| row.session_id == source)
        .ok_or_else(|| {
            Error(format!(
                "assembly {owner} source occurrence {source}: expected a top-level occurrence"
            ))
        })
}

fn new_recipe(
    source: &AssemblyOccurrenceRecipe,
    item: BatchPlacement,
) -> Result<AssemblyOccurrenceRecipe, Error> {
    if item.name.trim().is_empty() {
        return Err(Error(format!(
            "occurrence name {:?}: expected a nonempty name",
            item.name
        )));
    }
    oblikovati_martensite::validate_occurrence_placement(&item.transform).map_err(Error)?;
    Ok(AssemblyOccurrenceRecipe {
        name: item.name,
        component: source.component.clone(),
        transform: item.transform.cells.to_vec(),
        virtual_: source.virtual_,
        virtual_part_number: source.virtual_part_number.clone(),
        virtual_description: source.virtual_description.clone(),
        virtual_structure: source.virtual_structure.clone(),
        ..Default::default()
    })
}

fn commit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: u64,
    label: &str,
    drafts: Vec<AssemblyOccurrenceRecipe>,
    children: Option<Vec<OccurrenceInfo>>,
    project: impl FnOnce(Vec<OccurrenceInfo>) -> Result<Vec<u8>, Error>,
) -> Result<Vec<u8>, Error> {
    files
        .commands(owner)
        .map_err(|cause| Error(cause.to_string()))?
        .run_projected(
            label,
            |snapshot, ids| append(snapshot, ids, owner, drafts),
            |snapshot, first| {
                let added = snapshot
                    .assembly
                    .as_ref()
                    .and_then(|rows| rows.get(first..))
                    .ok_or_else(|| {
                        Error(format!(
                            "assembly {owner}: expected new occurrences at index {first}"
                        ))
                    })?;
                let rows = added
                    .iter()
                    .map(|row| occurrence_info(row, children.clone()))
                    .collect::<Result<Vec<_>, _>>()?;
                project(rows)
            },
        )
        .map_err(|cause| Error(cause.to_string()))
}

fn append(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    owner: u64,
    drafts: Vec<AssemblyOccurrenceRecipe>,
) -> Result<usize, Error> {
    let rows = snapshot
        .assembly
        .as_mut()
        .ok_or_else(|| Error(format!("document {owner}: expected an assembly")))?;
    let first = rows.len();
    for mut draft in drafts {
        draft.session_id = ids
            .allocate_occurrence()
            .map_err(|cause| Error(cause.to_string()))?;
        rows.push(draft);
    }
    Ok(first)
}
