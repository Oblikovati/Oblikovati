// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Shell properties and certified preview requests (#4106).
use crate::dockable_window::DockableWindow;
use crate::shell_panel::{GuiShellV2Action, GuiShellV2Request, ShellV2State};
use crate::{GuiCommandState, GuiShellAction, GuiShellFeatureRequest};
use imgui::Ui;
use oblikovati_model::feature::FeatureId;

#[derive(Default)]
pub(crate) struct ShellFeatureDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    thickness: String,
    attempted: Option<GuiShellFeatureRequest>,
    submitted: Option<GuiShellFeatureRequest>,
    v2: ShellV2State,
}
impl DockableWindow for ShellFeatureDialog {}
impl ShellFeatureDialog {
    pub(crate) fn active(&self) -> bool {
        self.document.is_some()
    }

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
        self.v2.reset_new(state);
    }

    pub(crate) fn open_edit(&mut self, state: &GuiCommandState, feature: FeatureId) {
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        let pick_seq = state.shell_v2_pick.as_ref().map_or(0, |p| p.sequence);
        if let Some(seed) = state
            .browser
            .as_ref()
            .and_then(|browser| browser.shell_v2_edits.get(&feature))
        {
            self.thickness = seed.thickness.clone();
            self.v2.hydrate_edit(feature, seed, pick_seq);
        } else {
            self.thickness = "1 mm".into();
            self.v2.reset_new(state);
            self.v2.feature = Some(feature);
        }
    }

    #[cfg(test)]
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
            if self.attempted.is_some() {
                actions.push(GuiShellAction::PreviewShellFeature(None));
            } else if self.v2.needs_clear() {
                actions.push(GuiShellAction::ShellV2(GuiShellV2Action::Clear));
            }
        }
        self.attempted = None;
        self.submitted = None;
        self.v2.reset_closed();
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
        if self
            .v2
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
        let Some(document) = self.document else {
            return;
        };
        self.v2.sync_pick_mode(actions);
        self.v2.consume_pick(state);
        let height = self.v2.fitted_height(ui);
        crate::widget_abi::fit_viewport_properties(ui, "Shell", area, [340., height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Shell", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            self.v2.draw_body(ui, &mut self.thickness, actions);
            self.refresh_v2_preview(document, state, actions);
            ui.separator();
            self.controls(document, ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    #[cfg(test)]
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

    fn refresh_v2_preview(
        &mut self,
        document: u64,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let request: Option<GuiShellV2Request> = if self.v2.local_error(&self.thickness).is_some() {
            None
        } else {
            self.v2
                .request(document, state, &self.thickness)
                .filter(|req| {
                    GuiShellAction::ShellV2(GuiShellV2Action::Preview(Box::new(req.clone())))
                        .dispatchable(state)
                })
        };
        if self.v2.attempted != request {
            self.v2.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::ShellV2(GuiShellV2Action::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::ShellV2(GuiShellV2Action::ClearPreview)),
            }
        }
    }

    fn controls(
        &mut self,
        document: u64,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let local_ok = self.v2.local_error(&self.thickness).is_none();
        let request = local_ok
            .then(|| self.v2.request(document, state, &self.thickness))
            .flatten()
            .filter(|req| {
                GuiShellAction::ShellV2(GuiShellV2Action::Commit(Box::new(req.clone())))
                    .dispatchable(state)
            });
        if crate::theme::property_accept(ui, request.is_some())
            && let Some(req) = request
        {
            self.v2.submitted = Some(req.clone());
            actions.push(GuiShellAction::ShellV2(GuiShellV2Action::Commit(Box::new(
                req,
            ))));
        }
        ui.same_line_with_spacing(0., 6.);
        if crate::theme::property_cancel(ui)
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
