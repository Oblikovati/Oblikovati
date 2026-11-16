// SPDX-License-Identifier: GPL-2.0-only
//! Dockable fillet draft; the owner holds the temporary preview (#4045).
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiFilletRequest, GuiShellAction};
use imgui::Ui;

#[derive(Default)]
pub(crate) struct FilletDialog {
    document: Option<u64>,
    radius: String,
    attempted: Option<GuiFilletRequest>,
    submitted: Option<GuiFilletRequest>,
}
impl DockableWindow for FilletDialog {}
impl FilletDialog {
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        self.document = state.document;
        self.radius = "2 mm".into();
        self.attempted = None;
        self.submitted = None;
    }
    fn request(&self, state: &GuiCommandState) -> Option<GuiFilletRequest> {
        let document = self.document?;
        let (generation, edges) = state.fillet_selection.as_ref()?;
        Some(GuiFilletRequest {
            document,
            generation: *generation,
            edges: edges.clone(),
            radius: self.radius.clone(),
        })
    }
    fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            actions.push(GuiShellAction::PreviewFillet(None));
        }
        self.attempted = None;
        self.submitted = None;
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
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
        if self.document.is_none() {
            return;
        }
        let display = ui.io().display_size;
        Self::fit_dockable(
            ui,
            "Fillet",
            [(display[0] - 380.).max(0.), 170.],
            [360., 190.],
        );
        let mut open = true;
        Self::dockable_window(ui, "Fillet", &mut open).build(|| {
            ui.text(format!(
                "{} Edge{}",
                state.selected_edge_count,
                if state.selected_edge_count == 1 {
                    ""
                } else {
                    "s"
                }
            ));
            if ui.input_text("Radius", &mut self.radius).build() {
                self.submitted = None;
            }
            self.refresh_preview(state, actions);
            self.controls(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }
    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|request| {
            GuiShellAction::PreviewFillet(Some(request.clone())).dispatchable(state)
        });
        if request == self.attempted {
            return;
        }
        if self.attempted.is_some() {
            actions.push(GuiShellAction::PreviewFillet(None));
        }
        self.attempted = request.clone();
        if let Some(request) = request {
            actions.push(GuiShellAction::PreviewFillet(Some(request)));
        }
    }
    fn controls(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self
            .request(state)
            .filter(|request| GuiShellAction::CreateFillet(request.clone()).dispatchable(state));
        let disabled = ui.begin_disabled(request.is_none());
        if (ui.button("OK")
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Enter)))
            && let Some(request) = request
        {
            self.submitted = Some(request.clone());
            actions.push(GuiShellAction::CreateFillet(request));
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
    fn selected_state() -> GuiCommandState {
        GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            selected_edge_count: 1,
            fillet_available: true,
            fillet_selection: Some((3, vec![vec![1, 2, 3]])),
            ..Default::default()
        }
    }
    #[test]
    fn draft_previews_each_selection_once_and_cancel_clears_it() {
        let mut dialog = FilletDialog::default();
        let mut state = selected_state();
        let mut actions = Vec::new();
        dialog.open(&state);
        dialog.refresh_preview(&state, &mut actions);
        let GuiShellAction::PreviewFillet(Some(request)) = &actions[0] else {
            panic!("expected fillet preview")
        };
        assert_eq!(request.radius, "2 mm");
        assert_eq!(request.edges, vec![vec![1, 2, 3]]);
        actions.clear();
        dialog.refresh_preview(&state, &mut actions);
        assert!(actions.is_empty());
        state.fillet_selection = Some((3, vec![vec![4, 5, 6]]));
        dialog.refresh_preview(&state, &mut actions);
        assert!(matches!(
            actions.as_slice(),
            [
                GuiShellAction::PreviewFillet(None),
                GuiShellAction::PreviewFillet(Some(_))
            ]
        ));
        actions.clear();
        dialog.close(&mut actions);
        assert_eq!(actions, [GuiShellAction::PreviewFillet(None)]);
        assert!(dialog.document.is_none());
    }
}
