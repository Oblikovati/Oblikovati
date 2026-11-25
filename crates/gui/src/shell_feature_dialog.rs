// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Shell properties and certified preview requests (#4106).
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiShellAction, GuiShellFeatureRequest};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct ShellFeatureDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    thickness: String,
    attempted: Option<GuiShellFeatureRequest>,
    submitted: Option<GuiShellFeatureRequest>,
}
impl DockableWindow for ShellFeatureDialog {}
impl ShellFeatureDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Shell", &self.dock)
    }
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.thickness = "1 mm".into();
        self.attempted = None;
        self.submitted = None;
    }
    fn request(&self, state: &GuiCommandState) -> Option<GuiShellFeatureRequest> {
        let faces: Vec<_> = state
            .selected_face_appearances
            .iter()
            .map(|(key, _)| key.clone())
            .collect();
        if faces.is_empty() {
            return None;
        }
        Some(GuiShellFeatureRequest {
            document: self.document?,
            generation: state.document_generation?,
            faces,
            thickness: self.thickness.clone(),
        })
    }
    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::PreviewShellFeature(None));
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
                || !state.certified_part_feature_available
                || state.authoring != crate::GuiAuthoringContext::Part)
        {
            self.close(actions);
        }
        if self
            .submitted
            .as_ref()
            .is_some_and(|request| state.document_generation != Some(request.generation))
        {
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
        crate::widget_abi::fit_viewport_properties(ui, "Shell", area, [360., 200.]);
        let mut open = true;
        Self::dockable_window(ui, "Shell", &mut open).build(|| {
            ui.text(format!(
                "{} opening faces selected",
                state.selected_face_appearances.len()
            ));
            ui.input_text("Thickness", &mut self.thickness).build();
            self.refresh_preview(state, actions);
            self.controls(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }
    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|request| {
            GuiShellAction::PreviewShellFeature(Some(request.clone())).dispatchable(state)
        });
        crate::face_draft::refresh_preview(
            &mut self.attempted,
            request,
            actions,
            GuiShellAction::PreviewShellFeature,
        );
    }
    fn controls(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|request| {
            GuiShellAction::CreateShellFeature(request.clone()).dispatchable(state)
        });
        let disabled = ui.begin_disabled(request.is_none());
        if ui.button("OK")
            && let Some(request) = request
        {
            self.submitted = Some(request.clone());
            actions.push(GuiShellAction::CreateShellFeature(request));
        }
        drop(disabled);
        ui.same_line();
        if ui.button("Cancel")
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn shell_dialog_uses_selected_faces_and_cancels_its_preview() {
        let mut state = GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            certified_part_feature_available: true,
            document_generation: Some(3),
            ..Default::default()
        };
        state.selected_face_appearances = vec![("first".into(), false), ("second".into(), false)];
        let mut dialog = ShellFeatureDialog::default();
        dialog.open(&state);
        let request = dialog.request(&state).unwrap();
        assert_eq!(request.faces, ["first", "second"]);
        let mut actions = Vec::new();
        dialog.refresh_preview(&state, &mut actions);
        assert_eq!(
            actions,
            vec![GuiShellAction::PreviewShellFeature(Some(request))]
        );
        dialog.close(&mut actions);
        assert!(!dialog.dock.is_open());
        assert_eq!(
            actions.last(),
            Some(&GuiShellAction::PreviewShellFeature(None))
        );
    }
}
