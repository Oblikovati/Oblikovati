// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::GuiParameterGroup;
use imgui::{TreeNodeFlags, Ui};

/// Inline controls for creating, renaming, and removing document parameter groups.
#[derive(Default)]
pub(super) struct ParameterGroupManager {
    internal_name: String,
    create_display_name: String,
    client_id: String,
    selected: Option<String>,
    edit_display_name: String,
    delete_parameters: bool,
    pending: Option<GuiParameterGroupAction>,
}

impl ParameterGroupManager {
    pub(super) fn synchronize(&mut self, snapshot: Option<&GuiParameterSnapshot>) {
        self.retire_foreign_pending(snapshot);
        if self.selected.as_ref().is_none_or(|key| {
            !snapshot.is_some_and(|view| view.groups.iter().any(|group| &group.key == key))
        }) {
            self.selected = None;
            self.delete_parameters = false;
        }
    }

    pub(super) fn acknowledge(
        &mut self,
        action: &GuiParameterGroupAction,
        result: Result<(), String>,
    ) -> Option<String> {
        if self.pending.as_ref() != Some(action) {
            return None;
        }
        self.pending = None;
        if result.is_ok() {
            self.apply_success(action);
            return None;
        }
        result.err()
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
                self.edit_display_name.clone_from(&group.display_name);
                self.delete_parameters = false;
            }
            ui.same_line();
            ui.text_disabled(format!("{} member(s)", group.members.len()));
            self.draw_members(ui, snapshot, group, actions);
        }
    }

    fn draw_members(
        &mut self,
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
            let clicked = ui.small_button(format!(
                "Remove {}##group:{}:member:{member}",
                row.detail.parameter_info.name, group.key
            ));
            self.queue(
                actions,
                GuiParameterGroupAction::RemoveMember {
                    document: snapshot.document,
                    internal_name: group.key.clone(),
                    parameter: *member,
                },
                clicked,
            );
        }
    }

    fn draw_create(&mut self, ui: &Ui, document: u64, actions: &mut Vec<GuiParameterAction>) {
        ui.separator();
        ui.text("New group");
        ui.input_text("Internal name##new-group", &mut self.internal_name)
            .build();
        ui.input_text("Display name##new-group", &mut self.create_display_name)
            .build();
        ui.input_text("Client ID##new-group", &mut self.client_id)
            .build();
        self.queue(
            actions,
            GuiParameterGroupAction::Create {
                document,
                internal_name: self.internal_name.trim().into(),
                display_name: self.create_display_name.trim().into(),
                client_id: self.client_id.trim().into(),
            },
            ui.button("Create Group"),
        );
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
        ui.input_text("Display name##edit-group", &mut self.edit_display_name)
            .build();
        self.queue(
            actions,
            GuiParameterGroupAction::Rename {
                document,
                internal_name: key.into(),
                display_name: self.edit_display_name.trim().into(),
            },
            ui.button("Rename Group"),
        );
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
        self.queue(
            actions,
            GuiParameterGroupAction::Delete {
                document,
                internal_name: key,
                delete_parameters: self.delete_parameters,
            },
            ui.button("Delete Group"),
        );
    }

    fn queue(
        &mut self,
        actions: &mut Vec<GuiParameterAction>,
        action: GuiParameterGroupAction,
        clicked: bool,
    ) {
        if !clicked || self.pending.is_some() {
            return;
        }
        self.pending = Some(action.clone());
        actions.push(GuiParameterAction::GroupManagement(action));
    }

    fn apply_success(&mut self, action: &GuiParameterGroupAction) {
        match action {
            GuiParameterGroupAction::Create { .. } => self.clear_create(),
            GuiParameterGroupAction::Delete { internal_name, .. }
                if self.selected.as_deref() == Some(internal_name) =>
            {
                self.selected = None
            }
            _ => {}
        }
    }

    fn clear_create(&mut self) {
        self.internal_name.clear();
        self.create_display_name.clear();
        self.client_id.clear();
    }

    fn retire_foreign_pending(&mut self, snapshot: Option<&GuiParameterSnapshot>) {
        if self
            .pending
            .as_ref()
            .is_some_and(|action| snapshot.is_none_or(|view| view.document != action.document()))
        {
            self.pending = None;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_action() -> GuiParameterGroupAction {
        GuiParameterGroupAction::Create {
            document: 7,
            internal_name: "ratios".into(),
            display_name: "Ratios".into(),
            client_id: "addin".into(),
        }
    }

    #[test]
    fn successful_create_clears_only_the_matching_draft() {
        let action = create_action();
        let mut manager = ParameterGroupManager {
            internal_name: "ratios".into(),
            create_display_name: "Ratios".into(),
            client_id: "addin".into(),
            pending: Some(action.clone()),
            ..Default::default()
        };
        assert_eq!(manager.acknowledge(&action, Ok(())), None);
        assert!(manager.internal_name.is_empty());
        assert!(manager.create_display_name.is_empty());
        assert!(manager.client_id.is_empty());
        assert!(manager.pending.is_none());
    }

    #[test]
    fn foreign_result_and_document_change_retire_no_valid_draft() {
        let action = create_action();
        let mut manager = ParameterGroupManager {
            pending: Some(action.clone()),
            ..Default::default()
        };
        let foreign = GuiParameterGroupAction::Create {
            document: 7,
            internal_name: "other".into(),
            display_name: "Other".into(),
            client_id: String::new(),
        };
        assert_eq!(manager.acknowledge(&foreign, Err("ignored".into())), None);
        assert_eq!(manager.pending, Some(action));
        manager.synchronize(None);
        assert!(manager.pending.is_none());
    }
}
