// SPDX-License-Identifier: GPL-2.0-only
use super::RibbonCustomization;
use imgui::Ui;
use oblikovati_api::wire::BindingInfo;

impl RibbonCustomization {
    pub(super) fn sync_keyboard(&mut self, commands: &crate::GuiCommandState) {
        let Some(current) = &commands.keymap else {
            return;
        };
        if self.keyboard_draft.is_none() {
            self.keyboard_draft = Some(current.clone());
            self.keyboard_base = Some(current.export());
            return;
        }
        if !self.keyboard_changed() {
            self.keyboard_draft = Some(current.clone());
            self.keyboard_base = Some(current.export());
            self.keyboard_conflict = false;
            return;
        }
        if self.keyboard_changed()
            && self
                .keyboard_draft
                .as_ref()
                .is_some_and(|draft| draft.export() == current.export())
        {
            self.keyboard_base = Some(current.export());
            if self.keyboard_close_pending {
                self.open = false;
            }
            self.keyboard_close_pending = false;
            self.keyboard_conflict = false;
            return;
        }
        self.keyboard_conflict = self
            .keyboard_base
            .as_ref()
            .is_some_and(|base| *base != current.export());
    }
    pub(super) fn keyboard_changed(&self) -> bool {
        self.keyboard_draft
            .as_ref()
            .zip(self.keyboard_base.as_ref())
            .is_some_and(|(draft, base)| draft.export() != *base)
    }
    pub(super) fn submit_keyboard(&self, actions: &mut Vec<crate::GuiShellAction>) -> bool {
        if !self.keyboard_changed() || self.keyboard_conflict {
            return false;
        }
        if let Some(draft) = &self.keyboard_draft {
            actions.push(crate::GuiShellAction::SetKeymap(draft.export()));
            return true;
        }
        false
    }
    pub(super) fn keyboard_catalog(&mut self, ui: &Ui) {
        let Some(draft) = &self.keyboard_draft else {
            ui.text("Keyboard shortcuts are unavailable.");
            return;
        };
        let bindings = draft.catalog();
        ui.text("Keyboard shortcuts");
        ui.set_next_item_width(130.);
        ui.combo_simple_string(
            "Categories",
            &mut self.keyboard_category,
            &["All", "Application", "View", "Sketch"],
        );
        ui.same_line();
        ui.set_next_item_width(130.);
        ui.combo_simple_string(
            "Command type",
            &mut self.keyboard_kind,
            &["All", "Built-in", "Command"],
        );
        ui.set_next_item_width(260.);
        ui.input_text("Search commands", &mut self.keyboard_search)
            .build();
        ui.same_line();
        if ui.small_button("Clear") {
            self.keyboard_search.clear();
        }
        ui.child_window("##keyboard-list")
            .size([0., 190.])
            .border(true)
            .build(|| {
                ui.columns(4, "##keyboard-columns", true);
                for title in ["Keys", "Command Name", "Type", "Category"] {
                    ui.text(title);
                    ui.next_column();
                }
                ui.separator();
                for binding in filtered(
                    &bindings,
                    &self.keyboard_search,
                    self.keyboard_category,
                    self.keyboard_kind,
                ) {
                    let label = format!("{}##{}", binding.chord, binding.action_id);
                    let selected_key = ui
                        .selectable_config(label)
                        .selected(self.keyboard_selected.as_deref() == Some(&binding.action_id))
                        .build();
                    ui.next_column();
                    let selected_name = ui
                        .selectable_config(format!(
                            "{}##name-{}",
                            binding.display_name, binding.action_id
                        ))
                        .selected(self.keyboard_selected.as_deref() == Some(&binding.action_id))
                        .build();
                    if selected_key || selected_name {
                        self.keyboard_selected = Some(binding.action_id.clone());
                        self.keyboard_chord = binding.chord.clone();
                        self.keyboard_alias = binding.alias.clone();
                        self.keyboard_error = None;
                    }
                    ui.next_column();
                    for value in [&binding.kind, &category(binding)] {
                        ui.text(value);
                        ui.next_column();
                    }
                }
                ui.columns(1, "##keyboard-columns-end", false);
            });
        if ui.button("Reset All Keys") {
            if let Some(draft) = &mut self.keyboard_draft {
                draft.reset_all_chords();
            }
            if let Some(id) = self.keyboard_selected.clone() {
                self.refresh_keyboard_fields(&id);
            }
            self.keyboard_close_pending = false;
        }
        ui.same_line();
        if ui.button("Copy to Clipboard") {
            ui.set_clipboard_text(clipboard_table(&bindings));
        }
        self.keyboard_editor(ui);
    }
    fn keyboard_editor(&mut self, ui: &Ui) {
        let Some(id) = self.keyboard_selected.clone() else {
            return;
        };
        ui.input_text("Shortcut", &mut self.keyboard_chord).build();
        let alias_available = self
            .keyboard_draft
            .as_ref()
            .is_some_and(|draft| draft.supports_sketch_alias(&id));
        let disabled = ui.begin_disabled(!alias_available);
        ui.input_text("Alias", &mut self.keyboard_alias).build();
        drop(disabled);
        if alias_available {
            ui.text_disabled("Aliases run when the sketch is ready.");
        } else {
            ui.text_disabled("An alias is not available for this action.");
        }
        if ui.button("Set binding") {
            let result = self.stage_binding(&id);
            self.keyboard_error = result.err();
            self.keyboard_close_pending = false;
        }
        ui.same_line();
        if ui.button("Reset binding") {
            self.keyboard_error = self
                .keyboard_draft
                .as_mut()
                .and_then(|draft| draft.reset(&id).err());
            if self.keyboard_error.is_none() {
                self.refresh_keyboard_fields(&id);
            }
            self.keyboard_close_pending = false;
        }
        if let Some(error) = &self.keyboard_error {
            ui.text_wrapped(error);
        }
    }
    fn stage_binding(&mut self, id: &str) -> Result<(), String> {
        let Some(draft) = &mut self.keyboard_draft else {
            return Err("keyboard draft: expected an active keymap".into());
        };
        let mut candidate = draft.clone();
        candidate.set_chord(id, &self.keyboard_chord)?;
        if candidate.supports_sketch_alias(id) {
            candidate.set_alias(id, &self.keyboard_alias)?;
        }
        *draft = candidate;
        Ok(())
    }
    fn refresh_keyboard_fields(&mut self, id: &str) {
        let binding = self.keyboard_draft.as_ref().and_then(|draft| {
            draft
                .catalog()
                .into_iter()
                .find(|binding| binding.action_id == id)
        });
        if let Some(binding) = binding {
            self.keyboard_chord = binding.chord;
            self.keyboard_alias = binding.alias;
        }
    }
}
fn category(binding: &BindingInfo) -> String {
    if binding.action_id.starts_with("view.") {
        "View".into()
    } else if binding.action_id.starts_with("sketch-")
        || binding.action_id.starts_with("constraint-")
        || binding.action_id == "dimension"
    {
        "Sketch".into()
    } else {
        "Application".into()
    }
}
fn filtered<'a>(
    bindings: &'a [BindingInfo],
    query: &str,
    selected_category: usize,
    selected_kind: usize,
) -> Vec<&'a BindingInfo> {
    let query = query.trim().to_lowercase();
    let mut result: Vec<_> = bindings
        .iter()
        .filter(|binding| {
            let matches_text = binding.display_name.to_lowercase().contains(&query)
                || binding.alias.to_lowercase().contains(&query);
            let matches_category = match selected_category {
                1 => category(binding) == "Application",
                2 => category(binding) == "View",
                3 => category(binding) == "Sketch",
                _ => true,
            };
            let matches_kind = match selected_kind {
                1 => binding.kind == "builtin",
                2 => binding.kind == "command",
                _ => true,
            };
            matches_text && matches_category && matches_kind
        })
        .collect();
    result.sort_by_key(|binding| binding.display_name.to_lowercase());
    result
}
fn clipboard_table(bindings: &[BindingInfo]) -> String {
    let mut rows = vec!["Keys\tCommand Name\tType\tCategory".to_owned()];
    rows.extend(bindings.iter().map(|binding| {
        format!(
            "{}\t{}\t{}\t{}",
            binding.chord,
            binding.display_name,
            binding.kind,
            category(binding)
        )
    }));
    rows.join("\n")
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn search_uses_actual_names_and_aliases_in_alphabetical_order() {
        let bindings = vec![
            BindingInfo {
                display_name: "Undo".into(),
                alias: "Reverse".into(),
                ..Default::default()
            },
            BindingInfo {
                display_name: "Redo".into(),
                ..Default::default()
            },
        ];
        assert_eq!(filtered(&bindings, "", 0, 0)[0].display_name, "Redo");
        assert_eq!(filtered(&bindings, "REV", 0, 0)[0].display_name, "Undo");
        assert!(filtered(&bindings, "pattern", 0, 0).is_empty());
    }
    #[test]
    fn filters_and_clipboard_use_real_catalog_values() {
        let bindings = oblikovati_model::keymap::Keymap::desktop_defaults().catalog();
        assert_eq!(filtered(&bindings, "home", 2, 1).len(), 1);
        assert!(filtered(&bindings, "home", 1, 1).is_empty());
        assert_eq!(filtered(&bindings, "rectangle", 3, 2).len(), 4);
        assert!(clipboard_table(&bindings).contains("Ctrl+Z\tUndo\tbuiltin\tApplication"));
    }
    #[test]
    fn binding_edits_are_staged_and_acknowledged_by_owner_state() {
        let original = oblikovati_model::keymap::Keymap::desktop_defaults();
        let mut dialog = RibbonCustomization {
            open: true,
            ..Default::default()
        };
        let mut state = crate::GuiCommandState {
            keymap: Some(original.clone()),
            ..Default::default()
        };
        dialog.sync_keyboard(&state);
        dialog.keyboard_chord = "Ctrl+U".into();
        dialog.keyboard_alias = "undo".into();
        dialog.stage_binding("edit.undo").unwrap();
        assert!(dialog.keyboard_changed());
        assert_eq!(
            original.resolve_chord(&oblikovati_model::keymap::parse_chord("Ctrl+Z").unwrap()),
            Some("edit.undo")
        );
        let mut actions = Vec::new();
        assert!(dialog.submit_keyboard(&mut actions));
        let crate::GuiShellAction::SetKeymap(delta) = &actions[0] else {
            panic!("expected keymap action")
        };
        let mut applied = original;
        applied.restore_saved(delta).unwrap();
        state.keymap = Some(applied);
        dialog.keyboard_close_pending = true;
        dialog.sync_keyboard(&state);
        assert!(!dialog.keyboard_changed());
        assert!(!dialog.open);
    }
    #[test]
    fn shortcut_edit_preserves_a_saved_alias() {
        let mut original = oblikovati_model::keymap::Keymap::desktop_defaults();
        original.set_alias("view.pan", "move").unwrap();
        let mut dialog = RibbonCustomization {
            keyboard_draft: Some(original),
            keyboard_chord: "Ctrl+U".into(),
            keyboard_alias: "different".into(),
            ..Default::default()
        };
        dialog.stage_binding("view.pan").unwrap();
        assert_eq!(
            dialog.keyboard_draft.unwrap().resolve_alias("move"),
            Some("view.pan")
        );
    }
    #[test]
    fn concurrent_keymap_change_refuses_to_submit_stale_draft() {
        let original = oblikovati_model::keymap::Keymap::desktop_defaults();
        let mut state = crate::GuiCommandState {
            keymap: Some(original.clone()),
            ..Default::default()
        };
        let mut dialog = RibbonCustomization {
            open: true,
            ..Default::default()
        };
        dialog.sync_keyboard(&state);
        dialog.keyboard_chord = "Ctrl+U".into();
        dialog.stage_binding("edit.undo").unwrap();
        let mut external = original;
        external.set_chord("edit.redo", "Ctrl+R").unwrap();
        state.keymap = Some(external);
        dialog.sync_keyboard(&state);
        assert!(dialog.keyboard_conflict);
        let mut actions = Vec::new();
        assert!(!dialog.submit_keyboard(&mut actions));
        assert!(actions.is_empty());
    }
}
