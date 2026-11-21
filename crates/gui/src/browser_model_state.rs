// SPDX-License-Identifier: GPL-2.0-only
//! The model-state branch menu.
use super::*;

#[derive(Debug, Default)]
pub(crate) struct ModelStateEdit {
    core: browser_inline_edit::BrowserInlineEdit,
}
impl ModelStateEdit {
    fn request(
        &mut self,
        document: u64,
        states: &oblikovati_model::model_state::ModelStateMembers,
    ) {
        self.core
            .request(document, states.list().iter().map(|row| row.id));
    }
    pub(super) fn observe(
        &mut self,
        document: Option<u64>,
        states: &oblikovati_model::model_state::ModelStateMembers,
    ) {
        self.core.observe(document, || {
            states
                .list()
                .iter()
                .map(|row| (row.id, row.name.clone()))
                .collect()
        });
    }
    pub(super) fn needs_open_branch(&self) -> bool {
        self.core.needs_open_branch()
    }
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        document: Option<u64>,
        member: &oblikovati_model::model_state::ModelStateMember,
        states: &oblikovati_model::model_state::ModelStateMembers,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let Some((accept, cancel)) = self.core.input(
            ui,
            document,
            member.id,
            "model-state-edit",
            "##model-state-name",
        ) else {
            return false;
        };
        let Some(draft) = self.core.draft.as_mut() else {
            return false;
        };
        if accept {
            let invalid = draft.name.trim().is_empty()
                || draft.name.contains('\0')
                || states
                    .list()
                    .iter()
                    .any(|row| row.id != draft.id && row.name == draft.name);
            if invalid {
                draft.error = Some("Enter a unique, nonempty name".into());
            } else {
                if draft.name != member.name {
                    actions.push(GuiShellAction::BrowserChoice {
                        document: draft.document,
                        choice: crate::GuiBrowserChoice::RenameModelState {
                            id: draft.id,
                            name: draft.name.clone(),
                        },
                    });
                }
                self.core.draft = None;
            }
        } else if cancel {
            self.core.draft = None;
        }
        if let Some(message) = self.core.draft.as_ref().and_then(|row| row.error.as_ref()) {
            ui.text(message);
        }
        true
    }
}

impl BrowserTree<'_> {
    pub(super) fn model_states(&self, view: &mut BrowserView, actions: &mut Vec<GuiShellAction>) {
        let Some(states) = &self.browser.model_states else {
            return;
        };
        view.model_state_edit.observe(self.state.document, states);
        let node = if view.model_state_edit.needs_open_branch() {
            self.ui
                .tree_node_config("Model State")
                .opened(true, imgui::Condition::Always)
                .push()
        } else {
            self.branch("Model State", false)
        };
        if self.ui.is_item_hovered() && self.ui.is_mouse_clicked(MouseButton::Right) {
            self.ui.open_popup("##model-state-menu");
        }
        if let Some(active) = states.active() {
            self.ui.same_line_with_spacing(0., 16.);
            self.ui.text(format!("<{}>", active.name));
        }
        self.ui.same_line_with_spacing(0., 8.);
        let all =
            states.edit_scope() == oblikovati_model::model_state::ModelStateEditScope::AllMembers;
        if self.ui.small_button(if all {
            "All##state-scope"
        } else {
            "1##state-scope"
        }) {
            actions.push(GuiShellAction::BrowserChoice {
                document: self.state.document.unwrap_or(0),
                choice: crate::GuiBrowserChoice::ToggleModelStateEditScope,
            });
        }
        if self.ui.is_item_hovered() {
            self.ui
                .tooltip_text(if all { "All Members" } else { "Single Member" });
        }
        self.model_state_menu(states, view, actions);
        let Some(_states) = node else {
            return;
        };
        for member in states.list() {
            if view
                .model_state_edit
                .draw(self.ui, self.state.document, member, states, actions)
            {
                continue;
            }
            let active = states.active().is_some_and(|row| row.id == member.id);
            let label = if active {
                format!("{} (Active)", member.name)
            } else {
                member.name.clone()
            };
            let _id = self.ui.push_id(format!("model-state-{}", member.id));
            let position = self.ui.cursor_screen_pos();
            crate::widget_abi::selectable(self.ui, "##model-state", active, true);
            if active {
                active_state_mark(self.ui, position);
            }
            crate::widget_abi::literal_text(
                self.ui,
                [
                    position[0] + 16. * self.ui.current_font_size() / 13.,
                    position[1],
                ],
                self.ui.style_color(StyleColor::Text),
                &label,
            );
            if !active
                && self.ui.is_item_hovered()
                && self.ui.is_mouse_double_clicked(MouseButton::Left)
            {
                actions.push(GuiShellAction::BrowserChoice {
                    document: self.state.document.unwrap_or(0),
                    choice: crate::GuiBrowserChoice::ActivateModelState(member.id),
                });
            }
        }
    }
    pub(super) fn model_state_menu(
        &self,
        states: &oblikovati_model::model_state::ModelStateMembers,
        view: &mut BrowserView,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(_menu) = self.ui.begin_popup("##model-state-menu") else {
            return;
        };
        if self.ui.menu_item("Edit Table") {
            actions.push(GuiShellAction::OpenModelStateTable);
        }
        self.ui.menu("New Model State", || {
            if self.ui.menu_item("New Model State") {
                view.model_state_edit
                    .request(self.state.document.unwrap_or(0), states);
                actions.push(GuiShellAction::BrowserChoice {
                    document: self.state.document.unwrap_or(0),
                    choice: crate::GuiBrowserChoice::CreateModelState,
                });
            }
            if let Some(active) = states.active() {
                let label = format!("Duplicate {}", active.name);
                let cursor = self.ui.cursor_pos();
                let width = self.ui.calc_text_size(&label)[0];
                crate::widget_abi::dummy(self.ui, [width, 0.]);
                crate::widget_abi::cursor_position(self.ui, cursor);
                let position = self.ui.cursor_screen_pos();
                let chosen = crate::widget_abi::selectable(
                    self.ui,
                    "##duplicate-active-model-state",
                    false,
                    false,
                );
                crate::widget_abi::literal_text(
                    self.ui,
                    position,
                    self.ui.style_color(StyleColor::Text),
                    &label,
                );
                if chosen {
                    view.model_state_edit
                        .request(self.state.document.unwrap_or(0), states);
                    actions.push(GuiShellAction::BrowserChoice {
                        document: self.state.document.unwrap_or(0),
                        choice: crate::GuiBrowserChoice::CopyModelState(active.id),
                    });
                }
            }
        });
        // Members have no descendant rows, so both commands leave the direct rows in place.
        self.ui.menu_item("Expand All Children");
        self.ui.menu_item("Collapse All Children");
    }
}

pub(super) fn active_state_mark(ui: &Ui, position: [f32; 2]) {
    let [x, y] = position;
    let scale = ui.current_font_size() / 13.;
    let color = ui.style_color(StyleColor::CheckMark);
    let draw = ui.get_window_draw_list();
    draw.add_line(
        [x + 2. * scale, y + 7. * scale],
        [x + 5. * scale, y + 10. * scale],
        color,
    )
    .thickness(2. * scale)
    .build();
    draw.add_line(
        [x + 5. * scale, y + 10. * scale],
        [x + 12. * scale, y + 2. * scale],
        color,
    )
    .thickness(2. * scale)
    .build();
}
