// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Coil Properties panel built on the shared Feature Properties shell.
mod behavior;
mod types;
pub use types::{
    GuiCoilAction, GuiCoilEditSeed, GuiCoilPreview, GuiCoilPreviewState, GuiCoilRequest,
};

use crate::{
    GuiAuthoringContext, GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    feature_properties::{
        FeatureOutputMode, SelectorStyle, SessionPresets, draw_output_section,
        draw_properties_footer, draw_properties_header,
    },
    icons::GuiRibbonIcons,
};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    sketch_coil::{CoilAxisSource, CoilHandedness, CoilMethod},
    work_axis::WorkAxisReference,
};

#[derive(Clone)]
struct CoilPresetFields {
    method: CoilMethod,
    pitch: String,
    revolutions: String,
    height: String,
    taper: String,
    handedness: CoilHandedness,
}

#[derive(Default)]
pub(crate) struct CoilDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    attempted: Option<GuiCoilRequest>,
    submitted: Option<GuiCoilRequest>,
    feature: Option<FeatureId>,
    profile_cleared: bool,
    axis: Option<CoilAxisSource>,
    axis_reversed: bool,
    method: CoilMethod,
    pitch: String,
    revolutions: String,
    height: String,
    taper: String,
    handedness: CoilHandedness,
    close_start: bool,
    start_flat: String,
    start_transition: String,
    close_end: bool,
    end_flat: String,
    end_transition: String,
    output: FeatureOutputMode,
    preview_hidden: bool,
    boolean_style: SelectorStyle,
    presets: SessionPresets<CoilPresetFields>,
    waiting_for_sketch_doc: Option<u64>,
    request_sketch_plane: bool,
}

impl DockableWindow for CoilDialog {}

impl CoilDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Coil", &self.dock)
    }

    pub(crate) fn take_sketch_plane_request(&mut self) -> bool {
        std::mem::take(&mut self.request_sketch_plane)
    }

    pub(super) fn open(&mut self, state: &GuiCommandState, feature: Option<FeatureId>) {
        let has_existing_body = state
            .browser
            .as_ref()
            .is_some_and(|browser| !browser.bodies.is_empty());
        let seed =
            match feature {
                Some(id) => {
                    let Some(seed) = state
                        .browser
                        .as_ref()
                        .and_then(|browser| browser.coil_edits.get(&id))
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
                    let mut seed = GuiCoilEditSeed {
                        output: if has_existing_body {
                            FeatureOutputMode::Join
                        } else {
                            FeatureOutputMode::NewSolid
                        },
                        ..Default::default()
                    };
                    if let Some(fields) = self.presets.initial_fields() {
                        seed.method = fields.method;
                        seed.pitch = fields.pitch;
                        seed.revolutions = fields.revolutions;
                        seed.height = fields.height;
                        seed.taper = fields.taper;
                        seed.handedness = fields.handedness;
                    }
                    seed
                }
            };
        self.feature = feature;
        self.profile_cleared = false;
        self.axis = seed.axis;
        self.axis_reversed = seed.axis_reversed;
        self.method = seed.method;
        self.pitch = seed.pitch;
        self.revolutions = seed.revolutions;
        self.height = seed.height;
        self.taper = seed.taper;
        self.handedness = seed.handedness;
        self.close_start = seed.close_start;
        self.start_flat = seed.start_flat;
        self.start_transition = seed.start_transition;
        self.close_end = seed.close_end;
        self.end_flat = seed.end_flat;
        self.end_transition = seed.end_transition;
        self.output = seed.output;
        self.preview_hidden = false;
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.waiting_for_sketch_doc = None;
    }

    pub(super) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.submitted = None;
            self.dock.set_open(false);
            actions.push(GuiShellAction::Coil(GuiCoilAction::Clear));
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
        if self.document != state.document || !GuiCoilAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if state.coil_preview.is_none()
            && let Some(request) = self.submitted.take()
            && state.document_generation != Some(request.generation)
        {
            self.presets.record_accepted(CoilPresetFields {
                method: request.method,
                pitch: request.pitch,
                revolutions: request.revolutions,
                height: request.height,
                taper: request.taper,
                handedness: request.handedness,
            });
            self.document = None;
            self.dock.set_open(false);
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        (area, icons, picked_axis): (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &GuiRibbonIcons,
            Option<WorkAxisReference>,
        ),
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = self.document else {
            return;
        };
        if let Some(axis_ref) = picked_axis {
            self.axis = Some(CoilAxisSource::WorkAxis(axis_ref));
        }
        let Some(generation) = state.document_generation else {
            return;
        };
        let mut keep_open = true;
        let mut close_now = false;
        let mut commit_now = false;
        let preview = state.coil_preview.as_ref();
        let current = self.build_request(document, generation);
        let current_result = preview
            .filter(|entry| entry.request == current)
            .map(|entry| &entry.result);
        let ok_enabled = matches!(current_result, Some(GuiCoilPreview::Ready));

        if ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.close(actions);
            return;
        }

        crate::widget_abi::fit_viewport_properties(ui, "Coil", area, [280., 620.]);
        let _border = crate::theme::property_window_border(ui);
        Self::dockable_window(ui, "Coil", &mut keep_open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Coil",
                "##coil-advanced-settings",
                &mut self.presets.shown,
                Some((self.preview_hidden, ok_enabled || self.preview_hidden)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            let preset_fields = ok_enabled.then(|| CoilPresetFields {
                method: self.method,
                pitch: self.pitch.clone(),
                revolutions: self.revolutions.clone(),
                height: self.height.clone(),
                taper: self.taper.clone(),
                handedness: self.handedness,
            });
            let default_name = format!("{}_{}", behavior::method_label(self.method), self.pitch);
            if let Some(chosen) = self.presets.draw(
                ui,
                icons,
                "coil",
                "New Coil Default",
                preset_fields,
                &default_name,
            ) {
                self.method = chosen.method;
                self.pitch = chosen.pitch;
                self.revolutions = chosen.revolutions;
                self.height = chosen.height;
                self.taper = chosen.taper;
                self.handedness = chosen.handedness;
            }
            let footer_height = 44.;
            let body_height = (ui.content_region_avail()[1] - footer_height).max(120.);
            {
                let body = crate::widget_abi::scroll_region(ui, "##coil-body", [0., body_height]);
                if body.visible {
                    self.draw_input_geometry(ui);
                    self.draw_coil_size(ui, current_result);
                    self.draw_coil_ends(ui);
                    self.draw_output(ui, icons, state);
                }
            }
            let footer = draw_properties_footer(ui, ok_enabled);
            commit_now = footer.ok;
            close_now = footer.cancel;
        });

        if !keep_open || close_now {
            self.close(actions);
            return;
        }
        let next = self.build_request(document, generation);
        if commit_now && ok_enabled && next == current {
            self.submitted = Some(next.clone());
            actions.push(GuiShellAction::Coil(GuiCoilAction::Commit(Box::new(next))));
            return;
        }
        if self.attempted.as_ref() != Some(&next) {
            self.attempted = Some(next.clone());
            actions.push(GuiShellAction::Coil(GuiCoilAction::Preview(Box::new(next))));
        }
    }

    fn draw_output(&mut self, ui: &Ui, icons: &GuiRibbonIcons, state: &GuiCommandState) {
        let Some(browser) = &state.browser else {
            return;
        };
        let saved_boolean = self
            .feature
            .and_then(|id| browser.coil_edits.get(&id))
            .is_some_and(|seed| seed.output.boolean().is_some());
        let boolean_available = saved_boolean
            || browser
                .bodies
                .iter()
                .filter(|body| Some(body.feature) != self.feature)
                .count()
                == 1;
        let solid_name = format!("Solid{}", browser.bodies.len() + 1);
        let surface_name = format!("Srf{}", browser.surface_bodies.len() + 1);
        draw_output_section(
            ui,
            icons,
            &mut self.boolean_style,
            "coil",
            &mut self.output,
            boolean_available,
            (&solid_name, &surface_name),
        );
    }

    fn build_request(&self, document: u64, generation: u64) -> GuiCoilRequest {
        GuiCoilRequest {
            document,
            generation,
            feature: self.feature,
            profile_cleared: self.profile_cleared,
            axis: self.axis,
            axis_reversed: self.axis_reversed,
            method: self.method,
            pitch: self.pitch.clone(),
            revolutions: self.revolutions.clone(),
            height: self.height.clone(),
            taper: self.taper.clone(),
            handedness: self.handedness,
            close_start: self.close_start,
            start_flat: self.start_flat.clone(),
            start_transition: self.start_transition.clone(),
            close_end: self.close_end,
            end_flat: self.end_flat.clone(),
            end_transition: self.end_transition.clone(),
            output: self.output,
            preview_hidden: self.preview_hidden,
        }
    }
}
