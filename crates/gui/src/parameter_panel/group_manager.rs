// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::GuiParameterGroup;
use imgui::{TreeNodeFlags, Ui};

/// Inline controls for creating, renaming, and removing document parameter groups.
#[derive(Default)]
pub(super) struct ParameterGroupManager {
    internal_name: String,
    display_name: String,
    client_id: String,
    selected: Option<String>,
    delete_parameters: bool,
}

impl ParameterGroupManager {
    pub(super) fn synchronize(&mut self, snapshot: Option<&GuiParameterSnapshot>) {
        if self.selected.as_ref().is_none_or(|key| {
            !snapshot.is_some_and(|view| view.groups.iter().any(|group| &group.key == key))
        }) {
            self.selected = None;
            self.delete_parameters = false;
        }
    }

    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        snapshot: &GuiParameterSnapshot,
        actions: &mut Vec<GuiParameterAction>,
    ) {
        if ui.collapsing_header("Parameter Groups", TreeNodeFlags::empty()) {
            self.draw_groups(ui, snapshot, actions);
            self.draw_create(ui, snapshot.document, actions);
            self.draw_selected(ui, snapshot.document, actions);
        }
    }

    fn draw_groups(
        &mut self,
        ui: &Ui,
        snapshot: &GuiParameterSnapshot,
        actions: &mut Vec<GuiParameterAction>,
    ) {
        for group in &snapshot.groups {
            ui.text(format!("{} ({})", group.display_name, group.key));
            ui.same_line();
            if ui.small_button(format!("Edit##group:{}", group.key)) {
                self.selected = Some(group.key.clone());
                self.display_name.clone_from(&group.display_name);
                self.delete_parameters = false;
            }
            ui.same_line();
            ui.text_disabled(format!("{} member(s)", group.members.len()));
            self.draw_members(ui, snapshot, group, actions);
        }
    }

    fn draw_members(
        &self,
        ui: &Ui,
        snapshot: &GuiParameterSnapshot,
        group: &GuiParameterGroup,
        actions: &mut Vec<GuiParameterAction>,
    ) {
        for member in &group.members {
            let Some(row) = snapshot.rows.iter().find(|row| row.parameter == *member) else {
                continue;
            };
            ui.same_line();
            if ui.small_button(format!(
                "Remove {}##group:{}:member:{member}",
                row.detail.parameter_info.name, group.key
            )) {
                actions.push(GuiParameterAction::GroupManagement(
                    GuiParameterGroupAction::RemoveMember {
                        document: snapshot.document,
                        internal_name: group.key.clone(),
                        parameter: *member,
                    },
                ));
            }
        }
    }

    fn draw_create(&mut self, ui: &Ui, document: u64, actions: &mut Vec<GuiParameterAction>) {
        ui.separator();
        ui.text("New group");
        ui.input_text("Internal name##new-group", &mut self.internal_name)
            .build();
        ui.input_text("Display name##new-group", &mut self.display_name)
            .build();
        ui.input_text("Client ID##new-group", &mut self.client_id)
            .build();
        if ui.button("Create Group") {
            actions.push(GuiParameterAction::GroupManagement(
                GuiParameterGroupAction::Create {
                    document,
                    internal_name: self.internal_name.trim().into(),
                    display_name: self.display_name.trim().into(),
                    client_id: self.client_id.trim().into(),
                },
            ));
        }
    }

    fn draw_selected(&mut self, ui: &Ui, document: u64, actions: &mut Vec<GuiParameterAction>) {
        let Some(key) = self.selected.clone() else {
            return;
        };
        self.draw_rename(ui, document, &key, actions);
        self.draw_delete(ui, document, key, actions);
    }

    fn draw_rename(
        &mut self,
        ui: &Ui,
        document: u64,
        key: &str,
        actions: &mut Vec<GuiParameterAction>,
    ) {
        ui.separator();
        ui.text(format!("Edit {key}"));
        ui.input_text("Display name##edit-group", &mut self.display_name)
            .build();
        if ui.button("Rename Group") {
            actions.push(GuiParameterAction::GroupManagement(
                GuiParameterGroupAction::Rename {
                    document,
                    internal_name: key.into(),
                    display_name: self.display_name.trim().into(),
                },
            ));
        }
    }

    fn draw_delete(
        &mut self,
        ui: &Ui,
        document: u64,
        key: String,
        actions: &mut Vec<GuiParameterAction>,
    ) {
        ui.checkbox("Delete member parameters", &mut self.delete_parameters);
        ui.same_line();
        if ui.button("Delete Group") {
            actions.push(GuiParameterAction::GroupManagement(
                GuiParameterGroupAction::Delete {
                    document,
                    internal_name: key,
                    delete_parameters: self.delete_parameters,
                },
            ));
        }
    }
}
