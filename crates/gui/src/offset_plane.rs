// SPDX-License-Identifier: GPL-2.0-only
//! Draft an offset work plane without changing the document until acceptance.
use crate::{GuiCommandState, GuiOffsetPlaneBase, GuiShellAction, dockable_window::DockableWindow};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct OffsetPlaneDialog {
    document: Option<u64>,
    base: Option<GuiOffsetPlaneBase>,
    offset: String,
    error: Option<String>,
}
impl DockableWindow for OffsetPlaneDialog {}
impl OffsetPlaneDialog {
    pub(crate) fn open(&mut self, document: u64, base: Option<GuiOffsetPlaneBase>) {
        self.document = Some(document);
        self.base = base;
        self.offset = "10 mm".into();
        self.error = None;
    }
    pub(crate) fn active(&self) -> bool {
        self.document.is_some()
    }
    pub(crate) fn choose(&mut self, base: GuiOffsetPlaneBase) {
        if self.base == Some(base) {
            return;
        }
        self.base = Some(base);
        self.error = None;
    }
    pub(crate) fn reject(
        &mut self,
        document: u64,
        base: GuiOffsetPlaneBase,
        offset: &str,
        error: &str,
    ) {
        self.document = Some(document);
        self.base = Some(base);
        self.offset = offset.to_owned();
        self.error = Some(error.to_owned());
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = self.document else {
            return;
        };
        if state.document != Some(document) || state.authoring != crate::GuiAuthoringContext::Part {
            self.document = None;
            return;
        }
        let mut open = true;
        let mut done = false;
        Self::dockable_window(ui, "Offset Plane", &mut open)
            .size([340., 210.], imgui::Condition::FirstUseEver)
            .build(|| done = self.controls(ui, state, actions, document));
        if !open || done {
            self.document = None;
        }
    }
    fn controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        document: u64,
    ) -> bool {
        self.offset_field(ui, state, actions, document);
        let request = self.base.map(|base| GuiShellAction::CreateOffsetPlane {
            document,
            base,
            offset: self.offset.clone(),
        });
        submit_buttons(ui, state, request, actions)
    }
    fn offset_field(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        document: u64,
    ) {
        ui.text(if self.base.is_some() {
            "1 Plane"
        } else {
            "Select Plane"
        });
        let disabled = ui.begin_disabled(self.base.is_none());
        if ui.input_text("Offset", &mut self.offset).build() {
            self.error = None;
        }
        drop(disabled);
        self.validation_message(ui, state, actions, document);
    }
    fn validation_message(
        &self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
        document: u64,
    ) {
        let validation = state
            .offset_plane_validation
            .as_ref()
            .filter(|result| result.document == document && result.offset == self.offset);
        if self.base.is_some() && !self.offset.trim().is_empty() && validation.is_none() {
            actions.push(GuiShellAction::ValidateOffsetPlane {
                document,
                offset: self.offset.clone(),
            });
        }
        if let Some(error) = validation.and_then(|result| result.error.as_ref()) {
            ui.text_colored([1., 0.25, 0.25, 1.], error);
        }
        if let Some(error) = &self.error {
            ui.text_colored([1., 0.25, 0.25, 1.], error);
        }
    }
}

fn submit_buttons(
    ui: &Ui,
    state: &GuiCommandState,
    request: Option<GuiShellAction>,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let disabled = ui.begin_disabled(
        !request
            .as_ref()
            .is_some_and(|action| action.dispatchable(state)),
    );
    let accepted = crate::widget_abi::button(ui, "OK", [0., 0.]);
    drop(disabled);
    ui.same_line();
    let cancelled = crate::widget_abi::button(ui, "Cancel", [0., 0.]);
    if accepted && let Some(action) = request {
        actions.push(action);
    }
    accepted || cancelled || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_rejected_draft_survives_the_same_selection_and_resets_on_a_new_one() {
        let mut draft = OffsetPlaneDialog::default();
        let xz = GuiOffsetPlaneBase::Origin(crate::GuiOriginPlaneKind::Xz);
        let xy = GuiOffsetPlaneBase::Origin(crate::GuiOriginPlaneKind::Xy);
        draft.open(7, Some(xz));
        draft.reject(7, xz, "0 mm", "expected a nonzero length");
        draft.choose(xz);
        assert_eq!(draft.offset, "0 mm");
        assert_eq!(draft.error.as_deref(), Some("expected a nonzero length"));
        draft.choose(xy);
        assert!(draft.error.is_none());
        assert_eq!(draft.base, Some(xy));
    }

    #[test]
    fn creation_requires_validation_for_the_same_document_and_expression() {
        let action = GuiShellAction::CreateOffsetPlane {
            document: 7,
            base: GuiOffsetPlaneBase::Origin(crate::GuiOriginPlaneKind::Xz),
            offset: "50 mm".into(),
        };
        let mut state = GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            ..Default::default()
        };
        assert!(!action.dispatchable(&state));
        state.offset_plane_validation = Some(crate::GuiOffsetPlaneValidation {
            generation: 1,
            document: 7,
            offset: "0 mm".into(),
            error: None,
        });
        assert!(!action.dispatchable(&state));
        let validation = state.offset_plane_validation.as_mut().unwrap();
        validation.offset = "50 mm".into();
        validation.error = Some("invalid".into());
        assert!(!action.dispatchable(&state));
        state.offset_plane_validation.as_mut().unwrap().error = None;
        assert!(action.dispatchable(&state));
    }
}
