// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Emboss Properties panel built on the shared Feature Properties shell.
mod modals;
mod types;
pub use types::{
    GuiEmbossAction, GuiEmbossEditSeed, GuiEmbossPreview, GuiEmbossPreviewState, GuiEmbossRequest,
};

use crate::{
    GuiAuthoringContext, GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    feature_properties::{draw_error_badge, draw_properties_footer, draw_properties_header},
    icons::GuiRibbonIcons,
    widget_abi,
};
use imgui::Ui;
use modals::{EmbossModals, ModalOutcome};
use oblikovati_model::feature::{FeatureId, sketch_emboss::EmbossType};

const OPERATIONS: [(EmbossType, &str); 3] = [
    (EmbossType::EmbossFromFace, "Emboss from Face"),
    (EmbossType::EngraveFromFace, "Engrave from Face"),
    (
        EmbossType::EmbossEngraveFromPlane,
        "Emboss/Engrave from Plane",
    ),
];

#[derive(Default)]
pub(crate) struct EmbossDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    attempted: Option<GuiEmbossRequest>,
    submitted: Option<GuiEmbossRequest>,
    feature: Option<FeatureId>,
    depth: String,
    emboss_type: EmbossType,
    direction_reversed: bool,
    taper: String,
    wrap_to_face: bool,
    wrap_face: Option<Vec<u8>>,
    preview_hidden: bool,
    presets_shown: bool,
    modals: EmbossModals,
    waiting_for_sketch_doc: Option<u64>,
    request_sketch_plane: bool,
}

impl DockableWindow for EmbossDialog {}

impl EmbossDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Emboss", &self.dock)
    }

    pub(crate) fn take_sketch_plane_request(&mut self) -> bool {
        std::mem::take(&mut self.request_sketch_plane)
    }

    pub(super) fn open(&mut self, state: &GuiCommandState, feature: Option<FeatureId>) {
        let seed =
            match feature {
                Some(id) => {
                    let Some(seed) = state
                        .browser
                        .as_ref()
                        .and_then(|browser| browser.emboss_edits.get(&id))
                    else {
                        return;
                    };
                    seed.clone()
                }
                None => {
                    if state.browser.as_ref().is_some_and(|browser| {
                        browser.sketches.iter().all(|sketch| !sketch.visible)
                    }) {
                        self.waiting_for_sketch_doc = state.document;
                        self.request_sketch_plane = true;
                        return;
                    }
                    GuiEmbossEditSeed::default()
                }
            };
        self.feature = feature;
        self.depth = seed.depth;
        self.emboss_type = seed.emboss_type;
        self.direction_reversed = seed.direction_reversed;
        self.taper = seed.taper;
        self.wrap_to_face = seed.wrap_to_face;
        self.wrap_face = seed.wrap_face;
        self.preview_hidden = false;
        self.modals.reset();
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.waiting_for_sketch_doc = None;
    }

    pub(super) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.submitted = None;
            self.modals.reset();
            self.dock.set_open(false);
            actions.push(GuiShellAction::Emboss(GuiEmbossAction::Clear));
        }
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let Some(doc) = self.waiting_for_sketch_doc {
            if state.document != Some(doc) {
                self.waiting_for_sketch_doc = None;
            } else if state.authoring == GuiAuthoringContext::Part
                && state
                    .browser
                    .as_ref()
                    .is_some_and(|browser| browser.sketches.iter().any(|s| s.visible))
            {
                self.waiting_for_sketch_doc = None;
                self.open(state, None);
            }
            return;
        }
        if self.document.is_none() {
            return;
        }
        if self.document != state.document || !GuiEmbossAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if state.emboss_preview.is_none()
            && let Some(request) = self.submitted.take()
            && state.document_generation != Some(request.generation)
        {
            self.document = None;
            self.modals.reset();
            self.dock.set_open(false);
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
        let Some(document) = self.document else {
            return;
        };
        let Some(generation) = state.document_generation else {
            return;
        };
        if self.wrap_to_face
            && self.wrap_face.is_none()
            && let Some(face) = state.selected_model_face.as_ref()
        {
            self.wrap_face = Some(face.clone());
        }
        let current = self.build_request(document, generation);
        let current_result = state
            .emboss_preview
            .as_ref()
            .filter(|entry| entry.request == current)
            .map(|entry| &entry.result);
        if let Some(GuiEmbossPreview::OpenLoop(message)) = current_result {
            self.modals.show_sketch_doctor(message.clone());
        }
        let wrap_needs_face = self.wrap_to_face && self.wrap_face.is_none();
        let ok_enabled = !wrap_needs_face
            && matches!(
                current_result,
                Some(
                    GuiEmbossPreview::Ready
                        | GuiEmbossPreview::Failed(_)
                        | GuiEmbossPreview::RangeError(_)
                )
            );

        if !self.modals.any_active() && ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.close(actions);
            return;
        }

        let mut keep_open = true;
        let mut close_now = false;
        let mut commit_now = false;
        let mut clear_selected_face = false;

        widget_abi::fit_viewport_properties(ui, "Emboss", area, [320., 460.]);
        let _border = crate::theme::property_window_border(ui);
        Self::dockable_window(ui, "Emboss", &mut keep_open).build(|| {
            crate::theme::floating_property_shadow(ui);
            draw_properties_header(
                ui,
                icons,
                "##emboss-advanced-settings",
                &mut self.presets_shown,
                |_| {},
            );
            let footer_height = ui.frame_height_with_spacing() + 12.;
            let body_height = (ui.content_region_avail()[1] - footer_height).max(120.);
            {
                let body = widget_abi::scroll_region(ui, "##emboss-body", [0., body_height]);
                if body.visible {
                    self.draw_body(ui, current_result, &mut clear_selected_face);
                }
            }
            let footer = draw_properties_footer(ui, ok_enabled, self.preview_hidden);
            commit_now = footer.ok;
            close_now = footer.cancel;
            if footer.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
        });

        if matches!(self.modals.draw(ui), ModalOutcome::CloseEmboss) {
            self.close(actions);
            return;
        }
        if clear_selected_face {
            actions.push(GuiShellAction::Emboss(GuiEmbossAction::ClearSelectedFace));
        }
        if !keep_open || close_now {
            self.close(actions);
            return;
        }
        let next = self.build_request(document, generation);
        if commit_now && ok_enabled && next == current {
            match current_result {
                Some(GuiEmbossPreview::Ready) => {
                    self.submitted = Some(next.clone());
                    actions.push(GuiShellAction::Emboss(GuiEmbossAction::Commit(Box::new(
                        next,
                    ))));
                }
                Some(GuiEmbossPreview::RangeError(message)) => {
                    self.modals.show_range_warning(message.clone());
                }
                Some(GuiEmbossPreview::Failed(message)) => {
                    self.modals.show_edit_failed(message.clone());
                }
                _ => {}
            }
            return;
        }
        if self.attempted.as_ref() != Some(&next) {
            self.attempted = Some(next.clone());
            actions.push(GuiShellAction::Emboss(GuiEmbossAction::Preview(Box::new(
                next,
            ))));
        }
    }

    fn draw_body(
        &mut self,
        ui: &Ui,
        result: Option<&GuiEmbossPreview>,
        clear_selected_face: &mut bool,
    ) {
        if ui.collapsing_header("Input Geometry", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            crate::theme::property_label(ui, "Profile");
            ui.text(match result {
                Some(GuiEmbossPreview::MissingProfile) => "Select Profile",
                Some(GuiEmbossPreview::OpenLoop(_)) => "Open Loop",
                _ => "1 Profile",
            });
            crate::theme::property_label(ui, "Operation");
            ui.set_next_item_width(-1.);
            if let Some(_combo) = ui.begin_combo("##emboss-operation", op_label(self.emboss_type)) {
                for (op, label) in OPERATIONS {
                    if ui
                        .selectable_config(label)
                        .selected(self.emboss_type == op)
                        .build()
                    {
                        self.emboss_type = op;
                        if op == EmbossType::EmbossEngraveFromPlane {
                            self.wrap_to_face = false;
                            self.wrap_face = None;
                            if self.taper.trim().is_empty() {
                                self.taper = "0,0".into();
                            }
                        }
                    }
                }
            }
        }
        if ui.collapsing_header("Behavior", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            crate::theme::property_label(ui, "Depth");
            ui.set_next_item_width(-1.);
            ui.input_text("##emboss-depth", &mut self.depth).build();

            if self.emboss_type == EmbossType::EmbossEngraveFromPlane {
                crate::theme::property_label(ui, "Taper");
                ui.set_next_item_width(-1.);
                ui.input_text("##emboss-taper", &mut self.taper).build();
            } else {
                crate::theme::property_label(ui, "Direction");
                if widget_abi::button(ui, "Flip Direction", [0.; 2]) {
                    self.direction_reversed = !self.direction_reversed;
                }
                ui.same_line_with_spacing(0., 8.);
                ui.text(if self.direction_reversed {
                    "-Normal"
                } else {
                    "+Normal"
                });

                let was_wrapped = self.wrap_to_face;
                ui.checkbox("Wrap to Face", &mut self.wrap_to_face);
                if self.wrap_to_face && !was_wrapped {
                    self.wrap_face = None;
                    *clear_selected_face = true;
                } else if !self.wrap_to_face && was_wrapped {
                    self.wrap_face = None;
                }
                if self.wrap_to_face {
                    crate::theme::property_label(ui, "Face");
                    ui.text(if self.wrap_face.is_some() {
                        "1 Face"
                    } else {
                        "Select Face"
                    });
                    if self.wrap_face.is_some() {
                        ui.same_line_with_spacing(0., 8.);
                        if widget_abi::button(ui, "ClearFace", [0.; 2]) {
                            self.wrap_face = None;
                            *clear_selected_face = true;
                        }
                    }
                }
            }
            if let Some(GuiEmbossPreview::Failed(msg)) = result {
                draw_error_badge(ui, msg);
            }
        }
    }

    fn build_request(&self, document: u64, generation: u64) -> GuiEmbossRequest {
        GuiEmbossRequest {
            document,
            generation,
            feature: self.feature,
            depth: self.depth.clone(),
            emboss_type: self.emboss_type,
            direction_reversed: self.direction_reversed,
            taper: self.taper.clone(),
            wrap_to_face: self.wrap_to_face,
            wrap_face: self.wrap_face.clone(),
            preview_hidden: self.preview_hidden,
        }
    }
}

fn op_label(op: EmbossType) -> &'static str {
    match op {
        EmbossType::EmbossFromFace => "Emboss from Face",
        EmbossType::EngraveFromFace => "Engrave from Face",
        EmbossType::EmbossEngraveFromPlane => "Emboss/Engrave from Plane",
    }
}
