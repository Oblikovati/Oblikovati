// SPDX-License-Identifier: GPL-2.0-only
//! Apply model-state row changes on a private candidate.
use super::{ModelStateMember, ModelStateMembers};
use std::collections::BTreeSet;

/// One resolved row edit, with an optional source for a newly reserved identity.
/// Example: `ModelStateRowEdit { id: 1, name: "Primary".into(), source: None }`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ModelStateRowEdit {
    pub id: u64,
    pub name: String,
    pub source: Option<u64>,
}
impl ModelStateMembers {
    /// Replace rows atomically after the caller reserves every new identity.
    /// Example: `members.replace_rows(&rows)?` keeps existing IDs through a rename.
    pub fn replace_rows(&mut self, rows: &[ModelStateRowEdit]) -> Result<(), String> {
        self.validate_row_edits(rows)?;
        let mut candidate = self.clone();
        for row in rows {
            if let Some(source) = row.source {
                candidate.copy_row_data(source, row);
            }
        }
        candidate.retain_rows(rows);
        candidate.validate()?;
        *self = candidate;
        Ok(())
    }
    fn validate_row_edits(&self, rows: &[ModelStateRowEdit]) -> Result<(), String> {
        if !rows
            .first()
            .is_some_and(|row| row.id == 1 && row.name == "Primary" && row.source.is_none())
        {
            return Err(format!(
                "model-state rows {rows:?}: expected Primary unchanged and first"
            ));
        }
        let mut ids = BTreeSet::new();
        let mut names = BTreeSet::new();
        for row in rows {
            if row.id == 0
                || row.name.trim().is_empty()
                || row.name.contains('\0')
                || !ids.insert(row.id)
                || !names.insert(row.name.as_str())
            {
                return Err(format!(
                    "model-state row {row:?}: expected a unique nonzero ID and unique nonempty name without NUL"
                ));
            }
            let existing = self.members.iter().any(|member| member.id == row.id);
            if existing != row.source.is_none()
                || row.source.is_some_and(|source| {
                    source == row.id
                        || (!self.members.iter().any(|member| member.id == source)
                            && !ids.contains(&source))
                })
            {
                return Err(format!(
                    "model-state row {row:?}: expected an existing member or a new ID with an earlier source"
                ));
            }
        }
        Ok(())
    }
    fn copy_row_data(&mut self, source: u64, row: &ModelStateRowEdit) {
        if let Some(overrides) = self.suppression.get(&source).cloned() {
            self.suppression.insert(row.id, overrides);
        }
        self.property_table.clone_member(source, row.id);
        self.parameter_table.clone_member(source, row.id);
        self.members.push(ModelStateMember {
            id: row.id,
            name: row.name.clone(),
        });
    }
    fn retain_rows(&mut self, rows: &[ModelStateRowEdit]) {
        let ids: BTreeSet<_> = rows.iter().map(|row| row.id).collect();
        self.suppression.retain(|id, _| ids.contains(id));
        for id in self
            .members
            .iter()
            .map(|member| member.id)
            .filter(|id| !ids.contains(id))
            .collect::<Vec<_>>()
        {
            self.property_table.forget_member(id);
            self.parameter_table.forget_member(id);
        }
        self.members = rows
            .iter()
            .map(|row| ModelStateMember {
                id: row.id,
                name: row.name.clone(),
            })
            .collect();
        if !ids.contains(&self.active) {
            self.active = 1;
        }
    }
}
