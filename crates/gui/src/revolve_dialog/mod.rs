// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Revolution Properties panel built on the shared Feature Properties shell.
mod behavior;
mod types;
pub use types::{
    GuiRevolveAction, GuiRevolveEditSeed, GuiRevolveExtent, GuiRevolvePreview,
    GuiRevolvePreviewState, GuiRevolveRequest,
};

use crate::{
    GuiAuthoringContext, GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    feature_properties::{
        FeatureOutputMode, SelectorStyle, SessionPresets, draw_output_section,
        draw_properties_footer, draw_properties_header,
    },
    icons::{GuiRibbonIcons, IconGlyph},
    parameters::GuiParameterEditor,
    widget_abi,
};
use imgui::Ui;
use oblikovati_model::{
    feature::{FeatureId, sketch_revolve::SketchRevolveDirection},
    sketch::SketchEntityId,
};

const DIRECTIONS: [(SketchRevolveDirection, &str, IconGlyph); 4] = [
    (
        SketchRevolveDirection::Positive,
        "Default",
        IconGlyph::DirectionDefault,
    ),
    (
        SketchRevolveDirection::Negative,
        "Flipped",
        IconGlyph::DirectionFlipped,
    ),
    (
        SketchRevolveDirection::Symmetric,
        "Symmetric",
        IconGlyph::DirectionSymmetric,
    ),
    (
        SketchRevolveDirection::Asymmetric,
        "Asymmetric",
        IconGlyph::DirectionAsymmetric,
    ),
];

#[derive(Clone)]
struct RevolvePresetFields {
    extent: GuiRevolveExtent,
    direction: SketchRevolveDirection,
    angle: String,
    second_angle: String,
}

#[derive(Default)]
pub(crate) struct RevolveDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    attempted: Option<GuiRevolveRequest>,
    submitted: Option<GuiRevolveRequest>,
    feature: Option<FeatureId>,
    profile_cleared: bool,
    axis: Option<SketchEntityId>,
    extent: GuiRevolveExtent,
    direction: SketchRevolveDirection,
    angle: String,
    second_angle: String,
    output: FeatureOutputMode,
    preview_hidden: bool,
    direction_style: SelectorStyle,
    boolean_style: SelectorStyle,
    presets: SessionPresets<RevolvePresetFields>,
    waiting_for_sketch_doc: Option<u64>,
    request_sketch_plane: bool,
    requested: bool,
}

impl DockableWindow for RevolveDialog {}

impl RevolveDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Revolution", &self.dock)
    }

    pub(crate) fn request_open(&mut self) {
        self.requested = true;
    }

    pub(crate) fn take_sketch_plane_request(&mut self) -> bool {
        std::mem::take(&mut self.request_sketch_plane)
    }

    pub(super) fn open(&mut self, state: &GuiCommandState, feature: Option<FeatureId>) {
        let has_existing_body = state
            .browser
            .as_ref()
            .is_some_and(|browser| !browser.bodies.is_empty());
        let seed = match feature {
            Some(id) => {
                let Some(seed) = state
                    .browser
                    .as_ref()
                    .and_then(|browser| browser.revolve_edits.get(&id))
                else {
                    return;
                };
                seed.clone()
            }
            None => {
                let mut seed = GuiRevolveEditSeed {
                    output: if has_existing_body {
                        FeatureOutputMode::Join
                    } else {
                        FeatureOutputMode::NewSolid
                    },
                    ..Default::default()
                };
                if let Some(fields) = self.presets.initial_fields() {
                    seed.extent = fields.extent;
                    seed.direction = fields.direction;
                    seed.angle = fields.angle;
                    seed.second_angle = fields.second_angle;
                }
                seed
            }
        };
        if feature.is_none()
            && state.browser.as_ref().is_some_and(|browser| {
                browser.sketches.iter().all(|sketch| !sketch.visible) && browser.bodies.is_empty()
            })
        {
            self.waiting_for_sketch_doc = state.document;
            self.request_sketch_plane = true;
            return;
        }
        self.feature = feature;
        self.profile_cleared = false;
        self.axis = seed.axis;
        self.extent = seed.extent;
        self.direction = seed.direction;
        self.angle = seed.angle;
        self.second_angle = seed.second_angle;
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
            actions.push(GuiShellAction::Revolve(GuiRevolveAction::Clear));
        }
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if std::mem::take(&mut self.requested) && GuiRevolveAction::Open.enabled(state) {
            self.open(state, None);
        }
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
        if self.document != state.document || !GuiRevolveAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if state.revolve_preview.is_none()
            && let Some(request) = self.submitted.take()
            && state.document_generation != Some(request.generation)
        {
            self.presets.record_accepted(RevolvePresetFields {
                extent: request.extent,
                direction: request.direction,
                angle: request.angle,
                second_angle: request.second_angle,
            });
            self.document = None;
            self.dock.set_open(false);
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        (area, icons, parameters): (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &GuiRibbonIcons,
            &mut GuiParameterEditor,
        ),
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = self.document else {
            return;
        };
        self.adopt_viewport_picks(state.revolve_preview.as_ref());
        let Some(generation) = state.document_generation else {
            return;
        };
        let mut keep_open = true;
        let mut close_now = false;
        let mut commit_now = false;
        let preview = state.revolve_preview.as_ref();
        let current = self.build_request(document, generation);
        let current_result = preview
            .filter(|entry| entry.request == current)
            .map(|entry| &entry.result);
        let ok_enabled = matches!(current_result, Some(GuiRevolvePreview::Ready));

        if ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.close(actions);
            return;
        }

        crate::widget_abi::fit_viewport_properties(ui, "Revolution", area, [280., 440.]);
        let _border = crate::theme::property_window_border(ui);
        Self::dockable_window(ui, "Revolution", &mut keep_open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let title = self
                .feature
                .and_then(|id| {
                    state
                        .browser
                        .as_ref()?
                        .features
                        .iter()
                        .find(|row| row.id == id)
                        .map(|row| row.name.as_str())
                })
                .unwrap_or("New Revolution");
            let header = draw_properties_header(
                ui,
                icons,
                title,
                "##revolve-advanced-settings",
                &mut self.presets.shown,
                Some((self.preview_hidden, ok_enabled || self.preview_hidden)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            let preset_fields = ok_enabled.then(|| RevolvePresetFields {
                extent: self.extent,
                direction: self.direction,
                angle: self.angle.clone(),
                second_angle: self.second_angle.clone(),
            });
            let default_name = format!("{}_{}", direction_name(self.direction), self.angle);
            if let Some(chosen) = self.presets.draw(
                ui,
                icons,
                "revolve",
                "New Revolve Default",
                preset_fields,
                &default_name,
            ) {
                self.extent = chosen.extent;
                self.direction = chosen.direction;
                self.angle = chosen.angle;
                self.second_angle = chosen.second_angle;
            }
            self.draw_input_geometry(ui, preview);
            self.draw_behavior(ui, icons, current_result, document, parameters);
            self.draw_output(ui, icons, state);
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
            actions.push(GuiShellAction::Revolve(GuiRevolveAction::Commit(next)));
            return;
        }
        if self.attempted.as_ref() != Some(&next) {
            self.attempted = Some(next.clone());
            actions.push(GuiShellAction::Revolve(GuiRevolveAction::Preview(next)));
        }
    }

    fn adopt_viewport_picks(&mut self, preview: Option<&GuiRevolvePreviewState>) {
        let Some(preview) = preview else {
            return;
        };
        if self.axis.is_none() && preview.request.axis.is_some() {
            let mut prior = preview.request.clone();
            prior.axis = None;
            if self.attempted.as_ref() == Some(&prior) {
                self.axis = preview.request.axis;
                self.attempted = Some(preview.request.clone());
            }
        }
        if self.profile_cleared && !preview.request.profile_cleared {
            let mut prior = preview.request.clone();
            prior.profile_cleared = true;
            if self.attempted.as_ref() == Some(&prior) {
                self.profile_cleared = false;
                self.attempted = Some(preview.request.clone());
            }
        }
    }

    fn draw_input_geometry(&mut self, ui: &Ui, preview: Option<&GuiRevolvePreviewState>) {
        if !ui.collapsing_header("Input Geometry", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        let curves = preview.map_or(4, |entry| entry.profile_curves.max(1));
        crate::theme::property_label(ui, "Profiles");
        let profile_text = if self.profile_cleared {
            "Select Profiles".to_owned()
        } else if self.output.is_surface() {
            format!("{curves} Curves")
        } else {
            "1 Profile".to_owned()
        };
        crate::feature_properties::draw_pick_pill(ui, &profile_text, !self.profile_cleared);
        if !self.profile_cleared {
            ui.same_line_with_spacing(0., 6.);
            if widget_abi::button(ui, "ClearProfile", [0.; 2]) {
                self.profile_cleared = true;
            }
        }
        crate::theme::property_label(ui, "Axis");
        let axis_text = if self.axis.is_some() {
            "1 Axis"
        } else {
            "Select Axis"
        };
        crate::feature_properties::draw_pick_pill(ui, axis_text, self.axis.is_some());
        if self.axis.is_some() {
            ui.same_line_with_spacing(0., 6.);
            if widget_abi::button(ui, "ClearAxis", [0.; 2]) {
                self.axis = None;
            }
        }
    }

    fn draw_output(&mut self, ui: &Ui, icons: &GuiRibbonIcons, state: &GuiCommandState) {
        let Some(browser) = &state.browser else {
            return;
        };
        let saved_boolean = self
            .feature
            .and_then(|id| browser.revolve_edits.get(&id))
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
            "revolve",
            &mut self.output,
            boolean_available,
            (&solid_name, &surface_name),
        );
    }

    fn build_request(&self, document: u64, generation: u64) -> GuiRevolveRequest {
        GuiRevolveRequest {
            document,
            generation,
            feature: self.feature,
            profile_cleared: self.profile_cleared,
            axis: self.axis,
            extent: self.extent,
            direction: self.direction,
            angle: self.angle.clone(),
            second_angle: self.second_angle.clone(),
            output: self.output,
            preview_hidden: self.preview_hidden,
        }
    }
}

fn direction_name(direction: SketchRevolveDirection) -> &'static str {
    match direction {
        SketchRevolveDirection::Positive => "Default",
        SketchRevolveDirection::Negative => "Flipped",
        SketchRevolveDirection::Symmetric => "Symmetric",
        SketchRevolveDirection::Asymmetric => "Asymmetric",
    }
}
