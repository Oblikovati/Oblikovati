// SPDX-License-Identifier: GPL-2.0-only
//! Dockable body motion draft with a private preview (#4097).
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiMoveBodyRequest, GuiShellAction};
use imgui::Ui;
use oblikovati_api::wire::featureargs::{MoveBody, MoveBodyOp};

#[derive(Default)]
pub(crate) struct MoveBodyDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    body: Option<String>,
    offset: [String; 3],
    attempted: Option<GuiMoveBodyRequest>,
    submitted: Option<GuiMoveBodyRequest>,
}
impl DockableWindow for MoveBodyDialog {}
impl MoveBodyDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Move Bodies", &self.dock)
    }
    pub(crate) fn open(&mut self, state: &GuiCommandState) {
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.body = state.selected_body_key.clone().or_else(|| {
            state
                .browser
                .as_ref()?
                .bodies
                .first()
                .map(|body| body.key.clone())
        });
        self.offset = ["0 mm".into(), "0 mm".into(), "0 mm".into()];
        self.attempted = None;
        self.submitted = None;
    }
    fn request(&self, state: &GuiCommandState) -> Option<GuiMoveBodyRequest> {
        let body = state
            .browser
            .as_ref()?
            .bodies
            .iter()
            .find(|body| Some(&body.key) == self.body.as_ref())?;
        Some(GuiMoveBodyRequest {
            document: self.document?,
            generation: state.fillet_selection.as_ref()?.0,
            body_key: key_bytes(&body.key)?,
            input: MoveBody {
                body_index: body.index,
                operations: Some(vec![MoveBodyOp {
                    type_: "freeDrag".into(),
                    x: self.offset[0].clone(),
                    y: self.offset[1].clone(),
                    z: self.offset[2].clone(),
                    ..Default::default()
                }]),
                ..Default::default()
            },
        })
    }
    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::PreviewMoveBody(None));
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
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        let display = ui.io().display_size;
        Self::fit_dockable(
            ui,
            "Move Bodies",
            [(display[0] - 380.).max(0.), 170.],
            [360., 240.],
        );
        let mut open = true;
        Self::dockable_window(ui, "Move Bodies", &mut open).build(|| {
            self.body_choice(ui, state);
            for (label, value) in ["X", "Y", "Z"].into_iter().zip(&mut self.offset) {
                ui.input_text(label, value).build();
            }
            self.refresh_preview(state, actions);
            self.controls(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }
    fn body_choice(&mut self, ui: &Ui, state: &GuiCommandState) {
        let Some(browser) = &state.browser else {
            ui.text("No body available");
            return;
        };
        let name = browser
            .bodies
            .iter()
            .find(|body| Some(&body.key) == self.body.as_ref())
            .map_or("Select body", |body| body.name.as_str());
        let Some(_combo) = ui.begin_combo("Body", name) else {
            return;
        };
        for body in &browser.bodies {
            if ui.selectable(&body.name) {
                self.body = Some(body.key.clone());
            }
        }
    }
    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|request| {
            GuiShellAction::PreviewMoveBody(Some(request.clone())).dispatchable(state)
        });
        if request == self.attempted {
            return;
        }
        if self.attempted.is_some() {
            actions.push(GuiShellAction::PreviewMoveBody(None));
        }
        self.attempted = request.clone();
        if let Some(request) = request {
            actions.push(GuiShellAction::PreviewMoveBody(Some(request)));
        }
    }
    fn controls(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self
            .request(state)
            .filter(|request| GuiShellAction::CreateMoveBody(request.clone()).dispatchable(state));
        let disabled = ui.begin_disabled(request.is_none());
        if ui.button("OK")
            && let Some(request) = request
        {
            self.submitted = Some(request.clone());
            actions.push(GuiShellAction::CreateMoveBody(request));
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
pub(crate) fn key_bytes(hex: &str) -> Option<Vec<u8>> {
    if !hex.len().is_multiple_of(2) {
        return None;
    }
    hex.as_bytes()
        .as_chunks::<2>()
        .0
        .iter()
        .map(|pair| {
            std::str::from_utf8(pair)
                .ok()
                .and_then(|text| u8::from_str_radix(text, 16).ok())
        })
        .collect()
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn body_keys_require_complete_hexadecimal_pairs() {
        assert_eq!(key_bytes("00aF"), Some(vec![0, 175]));
        assert_eq!(key_bytes("0"), None);
        assert_eq!(key_bytes("zz"), None);
    }
    #[test]
    fn closing_an_open_dialog_cancels_its_preview() {
        let state = GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            fillet_available: true,
            ..Default::default()
        };
        let mut dialog = MoveBodyDialog::default();
        dialog.open(&state);
        let mut actions = Vec::new();
        dialog.close(&mut actions);
        assert_eq!(actions, vec![GuiShellAction::PreviewMoveBody(None)]);
    }
    #[test]
    fn selected_body_drives_the_request_and_stale_generation_is_rejected() {
        let browser = crate::GuiModelBrowser {
            bodies: vec![crate::GuiBrowserBody {
                name: "Solid1".into(),
                feature: oblikovati_model::feature::FeatureId(1),
                key: "00af".into(),
                index: 0,
                visible: true,
                faces: 6,
                appearance: String::new(),
                physical: None,
            }],
            ..Default::default()
        };
        let mut state = GuiCommandState {
            document: Some(7),
            authoring: crate::GuiAuthoringContext::Part,
            fillet_available: true,
            fillet_selection: Some((3, Vec::new())),
            browser: Some(browser),
            ..Default::default()
        };
        let mut dialog = MoveBodyDialog::default();
        dialog.open(&state);
        dialog.offset[0] = "width".into();
        let request = dialog.request(&state).unwrap();
        assert_eq!(request.body_key, vec![0, 175]);
        assert_eq!(request.input.operations.as_ref().unwrap()[0].x, "width");
        assert!(GuiShellAction::CreateMoveBody(request.clone()).enabled(&state));
        state.fillet_selection = Some((4, Vec::new()));
        assert!(!GuiShellAction::CreateMoveBody(request).enabled(&state));
    }
}
