// SPDX-License-Identifier: GPL-2.0-only
//! Check staged model-state names before sending a table edit (#4119).
use super::*;
use oblikovati_api::wire::ModelStateTableRow;

pub(super) enum NamePrompt {
    Empty,
    Duplicate(String),
    CannotRemove,
}
pub(super) fn invalid(rows: &[ModelStateTableRow], index: usize) -> bool {
    let name = &rows[index].name;
    name.trim().is_empty()
        || name.contains('\0')
        || rows
            .iter()
            .enumerate()
            .any(|(other, row)| other != index && row.name == *name)
}
fn duplicate(rows: &[ModelStateTableRow]) -> Option<String> {
    rows.iter()
        .enumerate()
        .find(|(index, row)| !row.name.trim().is_empty() && invalid(rows, *index))
        .map(|(_, row)| row.name.clone())
}
fn remove_duplicates(view: &mut ModelStateTableView) -> bool {
    let mut candidate = view.clone();
    while let Some(name) = candidate.rows.as_deref().and_then(duplicate) {
        let rows = candidate.rows.as_deref().unwrap_or_default();
        let Some(index) = rows.iter().enumerate().rev().find_map(|(index, row)| {
            (row.name == name
                && row.ref_.member_id != candidate.active_member_id
                && crate::model_state_table_rows::can_delete(&candidate, index))
            .then_some(index)
        }) else {
            return false;
        };
        if !crate::model_state_table_rows::delete(&mut candidate, index) {
            return false;
        }
    }
    *view = candidate;
    true
}
impl ModelStateTable {
    pub(super) fn name_problem(&self) -> Option<NamePrompt> {
        let rows = self.draft.as_ref()?.rows.as_deref()?;
        if rows
            .iter()
            .any(|row| row.name.trim().is_empty() || row.name.contains('\0'))
        {
            return Some(NamePrompt::Empty);
        }
        duplicate(rows).map(NamePrompt::Duplicate)
    }
    pub(super) fn name_modal(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        if self.name_prompt.is_some() {
            ui.open_popup("Model state names");
        }
        let Some(_popup) = ui
            .modal_popup_config("Model state names")
            .always_auto_resize(true)
            .begin_popup()
        else {
            return;
        };
        match self.name_prompt.as_ref() {
            Some(NamePrompt::Empty) => {
                ui.text("Could not create: the member name is empty.");
                if ui.button("Cancel") {
                    self.name_prompt = None;
                    self.close();
                    ui.close_current_popup();
                }
            }
            Some(NamePrompt::Duplicate(name)) => {
                ui.text(format!(
                    "Duplicate name {name:?}. Duplicates will be removed if you continue."
                ));
                if ui.button("Yes") {
                    let removed = self.draft.as_mut().is_some_and(remove_duplicates);
                    if removed {
                        self.name_prompt = None;
                        self.accept_after_verification = self.draft.as_ref().map(request);
                        self.verify(actions);
                        ui.close_current_popup();
                    } else {
                        self.name_prompt = Some(NamePrompt::CannotRemove);
                    }
                }
                ui.same_line();
                if ui.button("No") {
                    self.name_prompt = None;
                    ui.close_current_popup();
                }
            }
            Some(NamePrompt::CannotRemove) => {
                ui.text(
                    "Cannot remove a duplicate that is active or is the source of another row.",
                );
                if ui.button("OK") {
                    self.name_prompt = None;
                    ui.close_current_popup();
                }
            }
            None => {}
        }
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn identifies_empty_and_all_duplicate_names() {
        let rows = vec![
            ModelStateTableRow {
                name: "A".into(),
                ..Default::default()
            },
            ModelStateTableRow {
                name: "A".into(),
                ..Default::default()
            },
            ModelStateTableRow::default(),
        ];
        assert!((0..3).all(|index| invalid(&rows, index)));
    }
    #[test]
    fn duplicate_removal_is_staged_and_keeps_existing_member_identity() {
        let row = |id, name: &str| ModelStateTableRow {
            ref_: ModelStateTableRowRef {
                member_id: id,
                ..Default::default()
            },
            name: name.into(),
            source: None,
        };
        let mut view = ModelStateTableView {
            active_member_id: 1,
            rows: Some(vec![row(1, "Primary"), row(2, "Same"), row(3, "Same")]),
            ..Default::default()
        };
        assert!(remove_duplicates(&mut view));
        assert_eq!(
            view.rows
                .unwrap()
                .iter()
                .map(|row| row.ref_.member_id)
                .collect::<Vec<_>>(),
            [1, 2]
        );
    }
    #[test]
    fn duplicate_removal_does_not_delete_the_active_member() {
        let row = |id, name: &str| ModelStateTableRow {
            ref_: ModelStateTableRowRef {
                member_id: id,
                ..Default::default()
            },
            name: name.into(),
            source: None,
        };
        let mut view = ModelStateTableView {
            active_member_id: 3,
            rows: Some(vec![row(1, "Primary"), row(2, "Same"), row(3, "Same")]),
            ..Default::default()
        };
        assert!(remove_duplicates(&mut view));
        assert_eq!(
            view.rows
                .unwrap()
                .iter()
                .map(|row| row.ref_.member_id)
                .collect::<Vec<_>>(),
            [1, 3]
        );
    }
}
