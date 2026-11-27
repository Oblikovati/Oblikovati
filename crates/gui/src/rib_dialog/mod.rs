// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Rib Properties panel built on the v1.1 Feature Properties shell.
mod modals;
mod tabs;
mod types;
pub use types::{
    GuiRibAction, GuiRibEditSeed, GuiRibPreview, GuiRibPreviewState, GuiRibRequest,
    GuiRibSketchCandidate,
};

use crate::{
    GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    feature_properties::{draw_properties_footer, draw_properties_header},
    icons::GuiRibbonIcons,
    widget_abi,
};
use imgui::Ui;
use modals::{RibModalOutcome, RibModals};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        sketch_rib::{RibDirection, RibThickenSide, RibThicknessPlane},
    },
    sketch::SketchEntityId,
};

pub(super) const ERROR_TEXT_RED: [f32; 4] = [0.92, 0.25, 0.25, 1.0];

#[derive(Default)]
pub(crate) struct RibDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    attempted: Option<GuiRibRequest>,
    submitted: Option<GuiRibRequest>,
    feature: Option<FeatureId>,
    pub(super) sketch: Option<DocumentSketchId>,
    pub(super) profile_curves: Vec<SketchEntityId>,
    pub(super) thickness: String,
    pub(super) depth: String,
    pub(super) to_next: bool,
    pub(super) direction: RibDirection,
    pub(super) flip_direction: bool,
    pub(super) thicken_side: RibThickenSide,
    pub(super) extend_profile: bool,
    pub(super) draft: String,
    pub(super) thickness_plane: RibThicknessPlane,
    pub(super) target_body: Option<String>,
    pub(super) boss_select_all: bool,
    pub(super) boss_diameter: String,
    pub(super) boss_offset: String,
    pub(super) boss_draft: String,
    preview_hidden: bool,
    presets_shown: bool,
    popup_was_open: bool,
    warned_invalid_thickness: Option<String>,
    modals: RibModals,
}

impl DockableWindow for RibDialog {}

impl RibDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Rib", &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, feature: Option<FeatureId>) {
        let seed = match feature {
            Some(id) => {
                let Some(seed) = state
                    .browser
                    .as_ref()
                    .and_then(|browser| browser.rib_edits.get(&id))
                else {
                    return;
                };
                seed.clone()
            }
            None => {
                let has_visible_sketch = state.browser.as_ref().is_some_and(|browser| {
                    browser
                        .sketches
                        .iter()
                        .any(|sketch| sketch.visible && sketch.consumed_by.is_none())
                });
                if !has_visible_sketch {
                    self.modals.reset();
                    self.modals.show_sketch_tip();
                    self.document = None;
                    self.dock.set_open(false);
                    return;
                }
                let mut seed = GuiRibEditSeed::default();
                if let Some(browser) = &state.browser {
                    if browser.bodies.len() == 1 {
                        seed.target_body = Some(browser.bodies[0].key.clone());
                    }
                    if let Some(selected) = browser.selected_sketches.first().copied()
                        && let Some(candidate) = state
                            .rib_sketches
                            .iter()
                            .find(|item| item.sketch == selected && !item.curves.is_empty())
                    {
                        seed.sketch = Some(candidate.sketch);
                        seed.profile_curves = candidate.curves.clone();
                    }
                }
                seed
            }
        };
        self.feature = feature;
        self.sketch = seed.sketch;
        self.profile_curves = seed.profile_curves;
        self.thickness = seed.thickness;
        self.depth = seed.depth;
        self.to_next = seed.to_next;
        self.direction = seed.direction;
        self.flip_direction = seed.flip_direction;
        self.thicken_side = seed.thicken_side;
        self.extend_profile = seed.extend_profile;
        self.draft = seed.draft;
        self.thickness_plane = seed.thickness_plane;
        self.target_body = seed.target_body;
        self.boss_select_all = false;
        self.boss_diameter = "5 mm".into();
        self.boss_offset = "0 mm".into();
        self.boss_draft = "0 deg".into();
        self.preview_hidden = false;
        self.popup_was_open = false;
        self.warned_invalid_thickness = None;
        self.modals.reset();
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.submitted = None;
            self.modals.reset();
            self.dock.set_open(false);
            actions.push(GuiShellAction::Rib(GuiRibAction::Clear));
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
        if self.document != state.document || !GuiRibAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if state.rib_preview.is_none()
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
        if self.document.is_none() {
            self.modals.draw(ui);
            return;
        }
        let Some(document) = self.document else {
            return;
        };
        let Some(generation) = state.document_generation else {
            return;
        };
        let current = self.build_request(document, generation);
        let current_result = state
            .rib_preview
            .as_ref()
            .filter(|entry| entry.request == current)
            .map(|entry| &entry.result);
        let thickness_invalid = is_non_positive_thickness(&self.thickness)
            || matches!(current_result, Some(GuiRibPreview::RangeError(_)));
        if !thickness_invalid {
            self.warned_invalid_thickness = None;
        }
        let has_profile = self.sketch.is_some() && !self.profile_curves.is_empty();
        let has_solid = self.target_body.is_some()
            || state
                .browser
                .as_ref()
                .is_some_and(|browser| browser.bodies.len() == 1);
        let ok_enabled = has_profile
            && has_solid
            && matches!(
                current_result,
                Some(
                    GuiRibPreview::Ready | GuiRibPreview::Failed(_) | GuiRibPreview::RangeError(_)
                )
            );

        if !self.modals.any_active()
            && !self.popup_was_open
            && ui.is_key_pressed_no_repeat(imgui::Key::Escape)
        {
            self.close(actions);
            return;
        }

        let mut keep_open = true;
        let mut close_now = false;
        let mut commit_now = false;

        widget_abi::fit_viewport_properties(ui, "Rib", area, [304., 468.]);
        let _border = crate::theme::property_window_border(ui);
        Self::dockable_window(ui, "Rib", &mut keep_open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Rib",
                "##rib-advanced-settings",
                &mut self.presets_shown,
                Some((self.preview_hidden, ok_enabled || self.preview_hidden)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            let footer_height = 44.;
            let body_height = (ui.content_region_avail()[1] - footer_height).max(160.);
            {
                let body = widget_abi::scroll_region(ui, "##rib-body", [0., body_height]);
                if body.visible {
                    self.draw_tabs(ui, state, current_result, thickness_invalid);
                }
            }
            let footer = draw_properties_footer(ui, ok_enabled);
            commit_now = footer.ok;
            close_now = footer.cancel;
        });

        self.popup_was_open = widget_abi::any_popup_open(ui);

        if matches!(self.modals.draw(ui), RibModalOutcome::CloseRib) {
            self.close(actions);
            return;
        }
        if !keep_open {
            self.close(actions);
            return;
        }
        if close_now {
            if thickness_invalid
                && self.warned_invalid_thickness.as_deref() != Some(self.thickness.as_str())
            {
                self.warned_invalid_thickness = Some(self.thickness.clone());
                self.modals
                    .show_range_warning("The expression is out of the expected range.".into());
                return;
            }
            self.close(actions);
            return;
        }
        let next = self.build_request(document, generation);
        if commit_now && ok_enabled && next == current {
            if thickness_invalid {
                self.warned_invalid_thickness = Some(self.thickness.clone());
                self.modals
                    .show_range_warning("The expression is out of the expected range.".into());
                return;
            }
            match current_result {
                Some(GuiRibPreview::Ready) => {
                    self.submitted = Some(next.clone());
                    actions.push(GuiShellAction::Rib(GuiRibAction::Commit(Box::new(next))));
                }
                Some(GuiRibPreview::RangeError(message)) => {
                    self.warned_invalid_thickness = Some(self.thickness.clone());
                    self.modals.show_range_warning(message.clone());
                }
                Some(GuiRibPreview::Failed(message)) => {
                    self.modals.show_create_failed(message.clone());
                }
                _ => {}
            }
            return;
        }
        if self.attempted.as_ref() != Some(&next) {
            self.attempted = Some(next.clone());
            actions.push(GuiShellAction::Rib(GuiRibAction::Preview(Box::new(next))));
        }
    }

    fn draw_tabs(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        result: Option<&GuiRibPreview>,
        thickness_invalid: bool,
    ) {
        let Some(_tabs) = ui.tab_bar("##rib-tabs") else {
            return;
        };
        if let Some(_tab) = ui.tab_item("Shape") {
            self.draw_shape_tab(ui, state, result, thickness_invalid);
        }
        if let Some(_tab) = ui.tab_item("Draft") {
            self.draw_draft_tab(ui);
        }
        if let Some(_tab) = ui.tab_item("Boss") {
            self.draw_boss_tab(ui);
        }
    }

    fn build_request(&self, document: u64, generation: u64) -> GuiRibRequest {
        GuiRibRequest {
            document,
            generation,
            feature: self.feature,
            sketch: self.sketch,
            profile_curves: self.profile_curves.clone(),
            thickness: self.thickness.clone(),
            depth: self.depth.clone(),
            to_next: self.to_next,
            direction: self.direction,
            flip_direction: self.flip_direction,
            thicken_side: self.thicken_side,
            extend_profile: self.extend_profile,
            draft: self.draft.clone(),
            thickness_plane: self.thickness_plane,
            target_body: self.target_body.clone(),
            preview_hidden: self.preview_hidden,
        }
    }
}

fn is_non_positive_thickness(raw: &str) -> bool {
    let trimmed = raw.trim().replace(',', ".");
    if trimmed.is_empty() {
        return true;
    }
    let numeric_prefix: String = trimmed
        .chars()
        .take_while(|c| c.is_ascii_digit() || matches!(c, '.' | '-' | '+' | 'e' | 'E'))
        .collect();
    if let Ok(value) = numeric_prefix.parse::<f64>() {
        return !value.is_finite() || value <= 0.0;
    }
    false
}
