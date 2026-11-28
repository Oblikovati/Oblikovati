// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Modify Thicken/Offset (#3918).
mod controls;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::{SessionPresets, draw_properties_header};
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    thicken_offset::{
        ThickenApproximationTolerance, ThickenApproximationType, ThickenBoolean, ThickenDirection,
        ThickenOutputMode,
    },
};
pub use types::{
    GuiThickenOffsetAction, GuiThickenOffsetEditSeed, GuiThickenOffsetPick,
    GuiThickenOffsetPreview, GuiThickenOffsetPreviewState, GuiThickenOffsetRequest,
};

pub(super) const PANEL_WINDOW_TITLE: &str = "Thicken/Offset";
pub(super) const EDIT_ERROR_TITLE: &str = "Edit Thicken";

#[derive(Clone, Debug, PartialEq)]
pub(super) struct ThickenPresetFields {
    pub(super) output: ThickenOutputMode,
    pub(super) distance_text: String,
    pub(super) direction: ThickenDirection,
    pub(super) boolean: ThickenBoolean,
    pub(super) automatic_face_chain: bool,
    pub(super) automatic_blending: bool,
}

pub(crate) struct ThickenOffsetDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) target_body: Option<Vec<u8>>,
    pub(super) faces: Vec<Vec<u8>>,
    pub(super) output: ThickenOutputMode,
    pub(super) distance_text: String,
    pub(super) direction: ThickenDirection,
    pub(super) boolean: ThickenBoolean,
    pub(super) automatic_face_chain: bool,
    pub(super) automatic_blending: bool,
    pub(super) allow_approximation: bool,
    pub(super) approximation_type: ThickenApproximationType,
    pub(super) approximation_tolerance: ThickenApproximationTolerance,
    pub(super) approximation_percent_text: String,
    preview_hidden: bool,
    pub(super) input_geometry_open: bool,
    pub(super) advanced_open: bool,
    presets: SessionPresets<ThickenPresetFields>,
    attempted: Option<GuiThickenOffsetRequest>,
    pub(super) submitted: Option<GuiThickenOffsetRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_pick: Option<GuiThickenOffsetPick>,
    pub(super) edit_error: Option<String>,
    pub(super) edit_error_open: bool,
}

impl Default for ThickenOffsetDialog {
    fn default() -> Self {
        let mut presets = SessionPresets::default();
        presets.shown = true;
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            target_body: None,
            faces: Vec::new(),
            output: ThickenOutputMode::Solid,
            distance_text: "1 mm".to_string(),
            direction: ThickenDirection::Positive,
            boolean: ThickenBoolean::Join,
            automatic_face_chain: false,
            automatic_blending: false,
            allow_approximation: false,
            approximation_type: ThickenApproximationType::Mean,
            approximation_tolerance: ThickenApproximationTolerance::Optimized,
            approximation_percent_text: "0,0 %".to_string(),
            preview_hidden: false,
            input_geometry_open: true,
            advanced_open: true,
            presets,
            attempted: None,
            submitted: None,
            reopen_after_commit: false,
            consumed_pick: None,
            edit_error: None,
            edit_error_open: false,
        }
    }
}

impl DockableWindow for ThickenOffsetDialog {}

impl ThickenOffsetDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(PANEL_WINDOW_TITLE, &self.dock)
    }

    pub(crate) fn open(
        &mut self,
        state: &GuiCommandState,
        edit: Option<FeatureId>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
        self.preview_hidden = false;
        self.consumed_pick = state.thicken_offset_pick.clone();
        self.edit_error = None;
        self.edit_error_open = false;
        self.input_geometry_open = true;
        self.advanced_open = true;
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.thicken_offset_edits.get(&id))
        {
            self.presets.clear_selection();
            self.target_body = Some(seed.target_body.clone());
            self.faces = seed.faces.clone();
            self.output = seed.output;
            self.distance_text = format_mm_distance(seed.distance);
            self.direction = seed.direction;
            self.boolean = seed.boolean;
            self.automatic_face_chain = seed.automatic_face_chain;
            self.automatic_blending = seed.automatic_blending;
            self.allow_approximation = seed.allow_approximation;
            self.approximation_type = seed.approximation_type;
            self.approximation_tolerance = seed.approximation_tolerance;
            self.approximation_percent_text = match seed.approximation_percent {
                Some(pct) => format_locale_percent(pct),
                None => "0,0 %".to_string(),
            };
            actions.push(GuiShellAction::ThickenOffset(
                GuiThickenOffsetAction::SetAutomaticFaceChain(self.automatic_face_chain),
            ));
            return;
        }
        self.target_body = None;
        self.faces.clear();
        self.allow_approximation = false;
        self.approximation_type = ThickenApproximationType::Mean;
        self.approximation_tolerance = ThickenApproximationTolerance::Optimized;
        self.approximation_percent_text = "0,0 %".to_string();
        if let Some(saved) = self.presets.initial_fields() {
            self.apply_preset(saved);
        } else {
            self.output = ThickenOutputMode::Solid;
            self.distance_text = "1 mm".to_string();
            self.direction = ThickenDirection::Positive;
            self.boolean = ThickenBoolean::Join;
            self.automatic_face_chain = false;
            self.automatic_blending = false;
        }
        actions.push(GuiShellAction::ThickenOffset(
            GuiThickenOffsetAction::SetAutomaticFaceChain(self.automatic_face_chain),
        ));
    }

    fn current_preset(&self) -> ThickenPresetFields {
        ThickenPresetFields {
            output: self.output,
            distance_text: self.distance_text.clone(),
            direction: self.direction,
            boolean: self.boolean,
            automatic_face_chain: self.automatic_face_chain,
            automatic_blending: self.automatic_blending,
        }
    }

    fn apply_preset(&mut self, preset: ThickenPresetFields) {
        self.output = preset.output;
        self.distance_text = preset.distance_text;
        self.direction = preset.direction;
        self.boolean = preset.boolean;
        self.automatic_face_chain = preset.automatic_face_chain;
        self.automatic_blending = preset.automatic_blending;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::ThickenOffset(GuiThickenOffsetAction::Clear));
        }
        self.editing = None;
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
        self.edit_error = None;
        self.edit_error_open = false;
    }

    pub(crate) fn sync_before_docking(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_some()
            && (self.document != state.document || state.authoring != GuiAuthoringContext::Part)
        {
            self.close(actions);
            return;
        }
        if let Some(submitted) = &self.submitted
            && state.document_generation != Some(submitted.generation)
        {
            self.presets.record_accepted(self.current_preset());
            if self.reopen_after_commit {
                self.reopen_after_commit = false;
                self.open(state, None, actions);
            } else {
                self.close(actions);
            }
        }
    }

    pub(super) fn parsed_distance_cm(&self) -> Option<f64> {
        let cm = parse_distance_cm(&self.distance_text)?;
        match self.output {
            ThickenOutputMode::Solid if cm > 0.0 => Some(cm),
            ThickenOutputMode::Surface if cm >= 0.0 => Some(cm),
            _ => None,
        }
    }

    pub(super) fn parsed_approximation_percent(&self) -> Option<f64> {
        let trimmed = self.approximation_percent_text.trim();
        let value = trimmed.parse::<f64>().ok()?;
        (value.is_finite() && value > 0.0 && value <= 100.0).then_some(value)
    }

    pub(super) fn approximation_valid(&self) -> bool {
        if !self.allow_approximation
            || self.approximation_tolerance != ThickenApproximationTolerance::Specified
        {
            return true;
        }
        self.parsed_approximation_percent().is_some()
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiThickenOffsetRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        let approx_pct = if self.allow_approximation
            && self.approximation_tolerance == ThickenApproximationTolerance::Specified
        {
            self.parsed_approximation_percent()
        } else {
            None
        };
        Some(GuiThickenOffsetRequest {
            document,
            generation,
            feature: self.editing,
            target_body: self.target_body.clone(),
            faces: self.faces.clone(),
            output: self.output,
            distance: self.parsed_distance_cm(),
            direction: self.direction,
            boolean: self.boolean,
            automatic_face_chain: self.automatic_face_chain,
            automatic_blending: self.automatic_blending,
            allow_approximation: self.allow_approximation,
            approximation_type: self.approximation_type,
            approximation_tolerance: self.approximation_tolerance,
            approximation_percent: approx_pct,
            approximation_valid: self.approximation_valid(),
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState) {
        let Some(pick) = &state.thicken_offset_pick else {
            return;
        };
        if self.consumed_pick.as_ref() == Some(pick) {
            return;
        }
        self.consumed_pick = Some(pick.clone());
        if self.target_body.as_ref() != Some(&pick.target_body) {
            self.target_body = Some(pick.target_body.clone());
            self.faces = pick.faces.clone();
            return;
        }
        for face in &pick.faces {
            if !self.faces.iter().any(|existing| existing == face) {
                self.faces.push(face.clone());
            }
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        icons: &GuiRibbonIcons,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state);
        let height = ((area.bottom - area.top) - 16.0).clamp(480.0, 640.0);
        widget_abi::fit_viewport_properties(ui, PANEL_WINDOW_TITLE, area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, PANEL_WINDOW_TITLE, &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header_title = match self.output {
                ThickenOutputMode::Solid => "Thicken/Offset",
                ThickenOutputMode::Surface => "OffsetSrf",
            };
            let header = draw_properties_header(
                ui,
                icons,
                header_title,
                "##thicken-adv",
                &mut self.presets.shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            let had_popup = self.presets.popup_open;
            if let Some(chosen) = self.presets.draw(
                ui,
                icons,
                "thicken-offset",
                "New Thicken Default",
                Some(self.current_preset()),
                "Thicken Preset",
            ) {
                self.apply_preset(chosen);
                actions.push(GuiShellAction::ThickenOffset(
                    GuiThickenOffsetAction::SetAutomaticFaceChain(self.automatic_face_chain),
                ));
            }
            ui.separator();
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(160.0);
            ui.child_window("##thicken-body")
                .size([0.0, body_h])
                .build(|| {
                    self.draw_body(ui, state, actions);
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer_controls(
                ui,
                state,
                had_popup || self.presets.popup_open || self.edit_error.is_some(),
                actions,
            );
        });
        if self.draw_edit_error_modal(ui) {
            self.close(actions);
            return;
        }
        if !open {
            self.close(actions);
        }
    }

    fn draw_body(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        self.draw_input_geometry(ui, actions);
        ui.separator();
        self.draw_behavior_controls(ui, actions);
        ui.separator();
        self.draw_output_controls(ui, state);
        ui.separator();
        self.draw_advanced_properties(ui);
        if let Some(preview) = &state.thicken_offset_preview
            && let GuiThickenOffsetPreview::Failed(msg) = &preview.result
            && self.editing.is_none()
        {
            ui.spacing();
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::ThickenOffset(GuiThickenOffsetAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::ThickenOffset(
                    GuiThickenOffsetAction::Preview(Box::new(req)),
                )),
                None => actions.push(GuiShellAction::ThickenOffset(GuiThickenOffsetAction::Clear)),
            }
        }
    }
}

pub(super) fn parse_distance_cm(raw: &str) -> Option<f64> {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return None;
    }
    let lower = trimmed.to_ascii_lowercase();
    let (num_str, scale_to_cm) = if let Some(prefix) = lower.strip_suffix("mm") {
        (prefix.trim(), 0.1)
    } else if let Some(prefix) = lower.strip_suffix("cm") {
        (prefix.trim(), 1.0)
    } else if let Some(prefix) = lower.strip_suffix("in") {
        (prefix.trim(), 2.54)
    } else {
        (trimmed, 0.1)
    };
    let val = num_str.parse::<f64>().ok()?;
    val.is_finite().then_some(val * scale_to_cm)
}

pub(super) fn format_mm_distance(cm: f64) -> String {
    let mm = cm * 10.0;
    if (mm - mm.round()).abs() < 1.0e-6 {
        format!("{} mm", mm.round() as i64)
    } else {
        let formatted = format!("{mm:.3}");
        let trimmed = formatted.trim_end_matches('0').trim_end_matches('.');
        format!("{trimmed} mm")
    }
}

pub(super) fn format_locale_percent(pct: f64) -> String {
    format!("{pct:.1} %").replace('.', ",")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn distance_and_approximation_validation_match_thicken_offset_rules() {
        let mut dialog = ThickenOffsetDialog {
            distance_text: "0".into(),
            ..Default::default()
        };
        assert!(dialog.parsed_distance_cm().is_none());
        dialog.output = ThickenOutputMode::Surface;
        assert_eq!(dialog.parsed_distance_cm(), Some(0.0));
        dialog.distance_text = "-1".into();
        assert!(dialog.parsed_distance_cm().is_none());
        dialog.distance_text = "5 mm".into();
        assert_eq!(dialog.parsed_distance_cm(), Some(0.5));

        dialog.allow_approximation = true;
        dialog.approximation_tolerance = ThickenApproximationTolerance::Specified;
        dialog.approximation_percent_text = format_locale_percent(1.0);
        assert_eq!(dialog.approximation_percent_text, "1,0 %");
        assert!(!dialog.approximation_valid());
        dialog.approximation_percent_text = "1".into();
        assert!(dialog.approximation_valid());
        assert_eq!(dialog.parsed_approximation_percent(), Some(1.0));
        dialog.allow_approximation = false;
        assert!(dialog.approximation_valid());
    }
}
