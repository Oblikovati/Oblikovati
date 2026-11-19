// SPDX-License-Identifier: GPL-2.0-only
//! Resolve client draft row identities through the document's monotonic allocator.
use super::*;
use oblikovati_model::model_state::ModelStateRowEdit;
use std::collections::{BTreeMap, BTreeSet};

pub(super) struct ResolvedRows {
    pub edits: Vec<ModelStateRowEdit>,
    pub references: Vec<ModelStateTableRowRef>,
    pub created: Vec<ModelStateTableCreatedRow>,
    keys: BTreeMap<String, u64>,
}
impl ResolvedRows {
    pub(super) fn id(&self, reference: &ModelStateTableRowRef) -> Result<u64, Error> {
        match (reference.member_id, reference.draft_key.as_str()) {
            (id, "") if id != 0 && self.edits.iter().any(|row| row.id == id) => Ok(id),
            (0, key) if !key.is_empty() => self.keys.get(key).copied().ok_or_else(|| {
                Error(format!(
                    "model-state row reference {reference:?}: expected a known draft row"
                ))
            }),
            _ => Err(Error(format!(
                "model-state row reference {reference:?}: expected exactly one existing member ID or draft key"
            ))),
        }
    }
}

pub(super) fn resolve(
    rows: Vec<ModelStateTableRow>,
    members: &ModelStateMembers,
    ids: &mut RecipeReservations<'_>,
) -> Result<ResolvedRows, Error> {
    let mut result = ResolvedRows {
        edits: Vec::new(),
        references: Vec::new(),
        created: Vec::new(),
        keys: BTreeMap::new(),
    };
    let mut existing = BTreeSet::new();
    for row in rows {
        let (id, source) = resolve_row(&row, members, ids, &mut result, &mut existing)?;
        result.references.push(row.ref_.clone());
        result.edits.push(ModelStateRowEdit {
            id,
            name: row.name,
            source,
        });
    }
    Ok(result)
}
fn resolve_row(
    row: &ModelStateTableRow,
    members: &ModelStateMembers,
    ids: &mut RecipeReservations<'_>,
    result: &mut ResolvedRows,
    existing: &mut BTreeSet<u64>,
) -> Result<(u64, Option<u64>), Error> {
    let reference = &row.ref_;
    if reference.member_id != 0 && reference.draft_key.is_empty() {
        if row.source.is_some()
            || !existing.insert(reference.member_id)
            || !members
                .list()
                .iter()
                .any(|member| member.id == reference.member_id)
        {
            return Err(Error(format!(
                "model-state row {row:?}: expected a unique existing member without a source"
            )));
        }
        return Ok((reference.member_id, None));
    }
    if reference.member_id != 0
        || reference.draft_key.trim().is_empty()
        || reference.draft_key.contains('\0')
        || result.keys.contains_key(&reference.draft_key)
    {
        return Err(Error(format!(
            "model-state row {row:?}: expected a unique nonempty draft key"
        )));
    }
    let source = resolve_source(row.source.as_deref(), members, result)?;
    let id = ids
        .allocate_model_state()
        .map_err(|cause| Error(cause.to_string()))?;
    result.keys.insert(reference.draft_key.clone(), id);
    result.created.push(ModelStateTableCreatedRow {
        draft_key: reference.draft_key.clone(),
        member_id: id,
    });
    Ok((id, Some(source)))
}
fn resolve_source(
    source: Option<&ModelStateTableRowRef>,
    members: &ModelStateMembers,
    resolved: &ResolvedRows,
) -> Result<u64, Error> {
    let source = source.ok_or_else(|| {
        Error("new model-state row source None: expected an existing or earlier draft row".into())
    })?;
    if source.member_id != 0
        && source.draft_key.is_empty()
        && members
            .list()
            .iter()
            .any(|member| member.id == source.member_id)
    {
        return Ok(source.member_id);
    }
    if source.member_id == 0 && !source.draft_key.is_empty() {
        if let Some(id) = resolved.keys.get(&source.draft_key) {
            return Ok(*id);
        }
    }
    Err(Error(format!(
        "new model-state row source {source:?}: expected an existing or earlier draft row"
    )))
}
