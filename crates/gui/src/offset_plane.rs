// SPDX-License-Identifier: GPL-2.0-only
//! Draft an offset work plane without changing the document until acceptance.
use crate::{GuiCommandState, GuiOffsetPlaneBase, GuiShellAction, dockable_window::DockableWindow};
use imgui::Ui;

const WIDTH: f32 = 280.;

#[derive(Default)]
pub(crate) struct OffsetPlaneDialog {
    document: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
    base: Option<GuiOffsetPlaneBase>,
    offset: String,
    error: Option<String>,
}
impl DockableWindow for OffsetPlaneDialog {}
impl OffsetPlaneDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Offset Plane", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        self.document = document;
        self.dock_state.set_open(document.is_some());
    }
    pub(crate) fn open(&mut self, document: u64, base: Option<GuiOffsetPlaneBase>) {
        self.set_document(Some(document));
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
        self.set_document(Some(document));
        self.base = Some(base);
        self.offset = offset.to_owned();
        self.error = Some(error.to_owned());
    }
    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState) {
        if self.document.is_some()
            && (self.document != state.document
                || state.authoring != crate::GuiAuthoringContext::Part)
        {
            self.set_document(None);
        }
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state);
        let Some(document) = self.document else {
            return;
        };
        let messages = self.validation_messages(state, document);
        let height = properties_height(ui, area.right - area.left, messages);
        crate::widget_abi::fit_viewport_properties(ui, "Offset Plane", area, [WIDTH, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        let mut done = false;
        Self::dockable_window(ui, "Offset Plane", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            done = self.controls(ui, state, actions, document);
        });
        if !open || done {
            self.set_document(None);
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
        ui.separator();
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
        crate::theme::property_label(ui, "Offset");
        let edited = {
            let _numeric = crate::ui_scale::numeric_font(ui);
            let _outline = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
            ui.input_text("##Offset", &mut self.offset).build()
        };
        if edited {
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
        if self.base.is_some()
            && !self.offset.trim().is_empty()
            && self.current_validation(state, document).is_none()
        {
            actions.push(GuiShellAction::ValidateOffsetPlane {
                document,
                offset: self.offset.clone(),
            });
        }
        let _wrap = ui.push_text_wrap_pos();
        for error in self
            .validation_messages(state, document)
            .into_iter()
            .flatten()
        {
            ui.text_colored([1., 0.25, 0.25, 1.], error);
        }
    }

    fn current_validation<'a>(
        &self,
        state: &'a GuiCommandState,
        document: u64,
    ) -> Option<&'a crate::GuiOffsetPlaneValidation> {
        state
            .offset_plane_validation
            .as_ref()
            .filter(|result| result.document == document && result.offset == self.offset)
    }

    fn validation_messages<'a>(
        &'a self,
        state: &'a GuiCommandState,
        document: u64,
    ) -> [Option<&'a str>; 2] {
        let current = self
            .current_validation(state, document)
            .and_then(|result| result.error.as_deref());
        [
            current,
            self.error
                .as_deref()
                .filter(|error| Some(*error) != current),
        ]
    }
}

fn properties_height(ui: &Ui, viewport_width: f32, messages: [Option<&str>; 2]) -> f32 {
    let style = ui.clone_style();
    let width = ((viewport_width - 28.).min(WIDTH) - 2. * style.window_padding[0]).max(1.);
    let diagnostics: f32 = messages
        .into_iter()
        .flatten()
        .map(|message| {
            ui.calc_text_size_with_opts(message, false, width)[1] + style.item_spacing[1]
        })
        .sum();
    3. * ui.frame_height_with_spacing() + 2. * style.window_padding[1] + 28. + diagnostics
}

fn submit_buttons(
    ui: &Ui,
    state: &GuiCommandState,
    request: Option<GuiShellAction>,
    actions: &mut Vec<GuiShellAction>,
) -> bool {
    let accepted = crate::theme::property_accept(
        ui,
        request
            .as_ref()
            .is_some_and(|action| action.dispatchable(state)),
    );
    ui.same_line();
    let cancelled = crate::widget_abi::button(ui, "Cancel", [72., 28.]);
    if accepted && let Some(action) = request {
        actions.push(action);
    }
    accepted || cancelled || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
}

#[cfg(test)]
mod tests {
    #[test]
    fn document_change_closes_before_docking() {
        let mut editor = OffsetPlaneDialog::default();
        editor.open(1, None);
        editor.sync_before_docking(&GuiCommandState::default());
        assert!(editor.document.is_none());
    }

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
