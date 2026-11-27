// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Decal Properties panel built on the v1.1 Feature Properties shell.
mod modals;
mod types;
pub use types::{
    GuiDecalAction, GuiDecalEditSeed, GuiDecalImageCandidate, GuiDecalPreview,
    GuiDecalPreviewState, GuiDecalRequest,
};

use crate::{
    GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    feature_properties::{draw_properties_footer, draw_properties_header},
    icons::GuiRibbonIcons,
    widget_abi,
};
use imgui::Ui;
use modals::DecalModals;
use oblikovati_model::{
    document_sketch::DocumentSketchId, feature::FeatureId, sketch::SketchEntityId,
};

#[derive(Default)]
pub(crate) struct DecalDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    submitted: Option<GuiDecalRequest>,
    feature: Option<FeatureId>,
    sketch: Option<DocumentSketchId>,
    image: Option<SketchEntityId>,
    target_face: Option<Vec<u8>>,
    target_body: Option<String>,
    observed_face: Option<Vec<u8>>,
    chain_faces: bool,
    wrap_to_face: bool,
    wrap_to_face_available: bool,
    preview_hidden: bool,
    presets_shown: bool,
    popup_was_open: bool,
    modals: DecalModals,
}

impl DockableWindow for DecalDialog {}

impl DecalDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Decal", &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, feature: Option<FeatureId>) {
        let seed = match feature {
            Some(id) => {
                let Some(seed) = state
                    .browser
                    .as_ref()
                    .and_then(|browser| browser.decal_edits.get(&id))
                else {
                    return;
                };
                seed.clone()
            }
            None => {
                if state.decal_images.is_empty() {
                    self.modals.reset();
                    self.modals.show_image_tip();
                    self.document = None;
                    self.dock.set_open(false);
                    return;
                }
                let mut seed = GuiDecalEditSeed::default();
                if let Some(browser) = &state.browser
                    && browser.bodies.len() == 1
                {
                    seed.target_body = Some(browser.bodies[0].key.clone());
                }
                seed
            }
        };
        self.feature = feature;
        self.sketch = seed.sketch;
        self.image = seed.image;
        self.target_face = seed.target_face;
        self.target_body = seed.target_body;
        self.observed_face = state.selected_model_face.clone();
        self.chain_faces = seed.chain_faces;
        self.wrap_to_face = seed.wrap_to_face;
        self.wrap_to_face_available = seed.wrap_to_face_available;
        self.preview_hidden = false;
        self.popup_was_open = false;
        self.modals.reset();
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.submitted = None;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.submitted = None;
            self.modals.reset();
            self.dock.set_open(false);
            actions.push(GuiShellAction::Decal(GuiDecalAction::Clear));
        }
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_none() {
            return;
        }
        if self.document != state.document || !GuiDecalAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if self.target_body.is_none()
            && let Some(browser) = &state.browser
            && browser.bodies.len() == 1
        {
            self.target_body = Some(browser.bodies[0].key.clone());
        }
        if state.selected_model_face != self.observed_face {
            self.observed_face = state.selected_model_face.clone();
            if let Some(face) = state.selected_model_face.clone() {
                self.target_face = Some(face);
            }
        }
        let Some(request) = self.request(state) else {
            return;
        };
        if let Some(preview) = state
            .decal_preview
            .as_ref()
            .filter(|preview| preview.request == request)
        {
            self.wrap_to_face_available = preview.wrap_to_face_available;
            if !self.wrap_to_face_available {
                self.wrap_to_face = false;
            }
        }
        if self.submitted.as_ref() != Some(&request) {
            self.submitted = Some(request.clone());
            actions.push(GuiShellAction::Decal(GuiDecalAction::Preview(Box::new(
                request,
            ))));
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        (area, icons): (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &GuiRibbonIcons,
        ),
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.modals.draw(ui);
        if self.document.is_none() {
            return;
        }
        if self.document != state.document || !GuiDecalAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if !self.popup_was_open && ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.close(actions);
            return;
        }
        let mut keep_open = true;
        let mut cancel = false;
        let mut commit = false;
        let request = self.request(state);
        let ready = request
            .as_ref()
            .is_some_and(|req| GuiDecalAction::Commit(Box::new(req.clone())).enabled(state));
        let preview_error = request.as_ref().and_then(|req| {
            state
                .decal_preview
                .as_ref()
                .filter(|preview| preview.request == *req)
                .and_then(|preview| match &preview.result {
                    GuiDecalPreview::Failed(msg) => Some(msg.as_str()),
                    _ => None,
                })
        });
        let title = match self.feature {
            Some(id) => state
                .browser
                .as_ref()
                .and_then(|browser| browser.features.iter().find(|row| row.id == id))
                .map(|row| format!("Decal: {}", row.name))
                .unwrap_or_else(|| "Decal".into()),
            None => "Decal".into(),
        };
        widget_abi::fit_viewport_properties(ui, "Decal", area, [296., 320.]);
        let _border = crate::theme::property_window_border(ui);
        Self::dockable_window(ui, "Decal", &mut keep_open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                &title,
                "##decal-advanced-settings",
                &mut self.presets_shown,
                Some((self.preview_hidden, ready || self.preview_hidden)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            let footer_height = 44.;
            let body_height = (ui.content_region_avail()[1] - footer_height).max(120.);
            {
                let body = widget_abi::scroll_region(ui, "##decal-body", [0., body_height]);
                if body.visible {
                    self.draw_input_geometry(ui, state);
                    ui.spacing();
                    self.draw_behavior(ui);
                    if let Some(error) = preview_error {
                        ui.spacing();
                        ui.text_wrapped(error);
                    }
                }
            }
            let footer = draw_properties_footer(ui, ready);
            commit = footer.ok;
            cancel = footer.cancel;
        });
        self.popup_was_open = widget_abi::any_popup_open(ui);
        if commit && let Some(req) = request {
            actions.push(GuiShellAction::Decal(GuiDecalAction::Commit(Box::new(req))));
            self.document = None;
            self.submitted = None;
            self.dock.set_open(false);
        } else if cancel || !keep_open {
            self.close(actions);
        }
    }

    fn draw_input_geometry(&mut self, ui: &Ui, state: &GuiCommandState) {
        if !ui.collapsing_header("Input Geometry", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        let image_label = if self.image.is_some() {
            "1 Image"
        } else {
            "Select Image"
        };
        crate::theme::property_label(ui, "Image");
        if widget_abi::button(ui, &format!("{image_label}##decal-image"), [128., 24.]) {
            if self.image.is_some() {
                self.sketch = None;
                self.image = None;
            } else if let Some(first) = state.decal_images.first() {
                self.sketch = Some(first.sketch);
                self.image = Some(first.image);
            }
        }
        let face_label = if self.target_face.is_some() {
            "1 Face"
        } else {
            "Select Face"
        };
        crate::theme::property_label(ui, "Face");
        if widget_abi::button(ui, &format!("{face_label}##decal-face"), [128., 24.]) {
            if self.target_face.is_some() {
                self.target_face = None;
            } else if let Some(face) = state.selected_model_face.clone() {
                self.target_face = Some(face);
            }
        }
    }

    fn draw_behavior(&mut self, ui: &Ui) {
        if !ui.collapsing_header("Behavior", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        ui.checkbox("Automatic Face Chain", &mut self.chain_faces);
        let wrap_disabled = self.target_face.is_some() && !self.wrap_to_face_available;
        let _disabled = wrap_disabled.then(|| ui.begin_disabled(true));
        ui.checkbox("Wrap to Face", &mut self.wrap_to_face);
    }

    fn request(&self, state: &GuiCommandState) -> Option<GuiDecalRequest> {
        Some(GuiDecalRequest {
            document: self.document?,
            generation: state.document_generation?,
            feature: self.feature,
            sketch: self.sketch,
            image: self.image,
            target_face: self.target_face.clone(),
            target_body: self.target_body.clone(),
            chain_faces: self.chain_faces,
            wrap_to_face: self.wrap_to_face,
            preview_hidden: self.preview_hidden,
        })
    }
}
