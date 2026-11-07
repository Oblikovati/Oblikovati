// SPDX-License-Identifier: GPL-2.0-only
//! The model-state branch menu.
use super::*;

#[derive(Debug, Default)]
pub(crate) struct ModelStateEdit {
    pending: Option<(u64, Vec<u64>)>,
    draft: Option<ModelStateDraft>,
}
#[derive(Debug)]
struct ModelStateDraft {
    document: u64,
    id: u64,
    name: String,
    focus: bool,
    error: Option<String>,
}
impl ModelStateEdit {
    fn request(
        &mut self,
        document: u64,
        states: &oblikovati_model::model_state::ModelStateMembers,
    ) {
        self.pending = Some((document, states.list().iter().map(|row| row.id).collect()));
        self.draft = None;
    }
    pub(super) fn observe(
        &mut self,
        document: Option<u64>,
        states: &oblikovati_model::model_state::ModelStateMembers,
    ) {
        if self.draft.as_ref().is_some_and(|draft| {
            Some(draft.document) != document || !states.list().iter().any(|row| row.id == draft.id)
        }) {
            self.draft = None;
        }
        let Some((owner, before)) = self.pending.take() else {
            return;
        };
        if Some(owner) != document {
            return;
        }
        let Some(member) = states.list().iter().find(|row| !before.contains(&row.id)) else {
            return;
        };
        self.draft = Some(ModelStateDraft {
            document: owner,
            id: member.id,
            name: member.name.clone(),
            focus: true,
            error: None,
        });
    }
    pub(super) fn needs_open_branch(&self) -> bool {
        self.draft.is_some() || self.pending.is_some()
    }
    pub(super) fn draw(
        &mut self,
        ui: &Ui,
        document: Option<u64>,
        member: &oblikovati_model::model_state::ModelStateMember,
        states: &oblikovati_model::model_state::ModelStateMembers,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let Some(draft) = &mut self.draft else {
            return false;
        };
        if draft.id != member.id || Some(draft.document) != document {
            return false;
        }
        let _id = ui.push_id(format!("model-state-edit-{}", member.id));
        if std::mem::take(&mut draft.focus) {
            ui.set_keyboard_focus_here();
        }
        let accept = ui
            .input_text("##model-state-name", &mut draft.name)
            .auto_select_all(true)
            .enter_returns_true(true)
            .build();
        let cancel = ui.is_key_pressed_no_repeat(imgui::Key::Escape);
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
                self.draft = None;
            }
        } else if cancel {
            self.draft = None;
        }
        if let Some(message) = self.draft.as_ref().and_then(|row| row.error.as_ref()) {
            ui.text(message);
        }
        true
    }
}

impl BrowserTree<'_> {
    pub(super) fn model_state_menu(
        &self,
        states: &oblikovati_model::model_state::ModelStateMembers,
        view: &mut BrowserView,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(_menu) = self.ui.begin_popup("##model-state-menu") else {
            return;
        };
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
    }
}
