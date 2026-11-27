// SPDX-License-Identifier: GPL-2.0-only
//! Dockable straight-hole draft and its private preview (#4093).
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiHoleRequest, GuiShellAction};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct HoleDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    diameter: String,
    depth: String,
    attempted: Option<GuiHoleRequest>,
    submitted: Option<GuiHoleRequest>,
}
impl DockableWindow for HoleDialog {}
impl HoleDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Hole", &self.dock)
    }
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.diameter = "2 mm".into();
        self.depth.clear();
        self.attempted = None;
        self.submitted = None;
    }
    fn request(&self, state: &GuiCommandState) -> Option<GuiHoleRequest> {
        let [(face, _)] = state.selected_face_appearances.as_slice() else {
            return None;
        };
        Some(GuiHoleRequest {
            document: self.document?,
            generation: state.fillet_selection.as_ref()?.0,
            face: face.clone(),
            diameter: self.diameter.clone(),
            depth: self.depth.clone(),
        })
    }
    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::PreviewHole(None));
        }
        self.attempted = None;
        self.submitted = None;
    }
    pub(crate) fn sync_before_docking(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_some()
            && (self.document != state.document
                || !state.fillet_available
                || state.authoring != crate::GuiAuthoringContext::Part)
        {
            self.close(actions);
        }
        if self.submitted.as_ref().is_some_and(|request| {
            state
                .fillet_selection
                .as_ref()
                .is_none_or(|(generation, _)| *generation != request.generation)
        }) {
            self.close(actions);
        }
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        let height =
            4. * ui.frame_height_with_spacing() + 2. * ui.clone_style().window_padding[1] + 28.;
        crate::widget_abi::fit_viewport_properties(ui, "Hole", area, [280., height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Hole", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            self.draw_properties(ui, state);
            self.refresh_preview(state, actions);
            ui.separator();
            self.controls(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }
    fn draw_properties(&mut self, ui: &Ui, state: &GuiCommandState) {
        ui.text(if self.request(state).is_some() {
            "1 Face"
        } else {
            "Select a planar face"
        });
        hole_value(ui, "Diameter", &mut self.diameter, "");
        hole_value(ui, "Depth", &mut self.depth, "Through all");
    }
    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|request| {
            GuiShellAction::PreviewHole(Some(request.clone())).dispatchable(state)
        });
        crate::face_draft::refresh_preview(
            &mut self.attempted,
            request,
            actions,
            GuiShellAction::PreviewHole,
        );
    }
    fn controls(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self
            .request(state)
            .filter(|request| GuiShellAction::CreateHole(request.clone()).dispatchable(state));
        if crate::theme::property_accept(ui, request.is_some())
            && let Some(request) = request
        {
            self.submitted = Some(request.clone());
            actions.push(GuiShellAction::CreateHole(request));
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel", [72., 28.])
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
    }
}

fn hole_value(ui: &Ui, label: &str, value: &mut String, hint: &str) {
    crate::theme::property_label(ui, label);
    let _id = ui.push_id(label);
    let _numeric = crate::ui_scale::numeric_font(ui);
    let _outline = ui.push_style_var(imgui::StyleVar::FrameBorderSize(1.));
    ui.input_text("##value", value).hint(hint).build();
}

#[cfg(test)]
mod tests {
    use super::*;

    fn part() -> GuiCommandState {
        GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            fillet_available: true,
            fillet_selection: Some((3, Vec::new())),
            ..Default::default()
        }
    }
    #[test]
    fn opening_and_cancelling_an_empty_part_keeps_no_draft() {
        let state = part();
        assert!(GuiShellAction::OpenHole.enabled(&state));
        let mut dialog = HoleDialog::default();
        dialog.open(&state);
        assert!(dialog.dock.is_open());
        assert!(dialog.request(&state).is_none());
        let mut actions = Vec::new();
        dialog.close(&mut actions);
        assert!(!dialog.dock.is_open());
        assert_eq!(actions, vec![GuiShellAction::PreviewHole(None)]);
    }
    #[test]
    fn selected_face_request_is_bound_to_the_document_generation() {
        let mut state = part();
        state.selected_face_appearances.push(("face".into(), false));
        let mut dialog = HoleDialog::default();
        dialog.open(&state);
        let request = dialog.request(&state).unwrap();
        assert_eq!(request.generation, 3);
        state.fillet_selection = Some((4, Vec::new()));
        assert!(!GuiShellAction::CreateHole(request).enabled(&state));
    }
}
