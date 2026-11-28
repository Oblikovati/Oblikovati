// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Modify Combine (Join, Cut, Intersect, Keep Toolbodies, #3918).
mod controls;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::{SessionPresets, draw_properties_header};
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{FeatureId, combine::CombineOperation};
pub use types::{
    GuiCombineAction, GuiCombineEditSeed, GuiCombinePick, GuiCombinePickMode, GuiCombinePreview,
    GuiCombinePreviewState, GuiCombineRequest,
};

pub(super) const EDIT_ERROR_TITLE: &str = "Edit Combine Feature";

#[derive(Clone, Debug, PartialEq, Eq)]
pub(super) struct CombinePresetFields {
    pub(super) operation: CombineOperation,
    pub(super) keep_toolbodies: bool,
}

pub(crate) struct CombineDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) base_body: Option<Vec<u8>>,
    pub(super) tool_bodies: Vec<Vec<u8>>,
    pub(super) operation: CombineOperation,
    pub(super) keep_toolbodies: bool,
    pub(super) pick_mode: GuiCombinePickMode,
    preview_hidden: bool,
    pub(super) input_geometry_open: bool,
    presets: SessionPresets<CombinePresetFields>,
    attempted: Option<GuiCombineRequest>,
    pub(super) submitted: Option<GuiCombineRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_pick: Option<GuiCombinePick>,
    pub(super) edit_error: Option<String>,
    pub(super) edit_error_open: bool,
}

impl Default for CombineDialog {
    fn default() -> Self {
        let mut presets = SessionPresets::default();
        presets.shown = true;
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            base_body: None,
            tool_bodies: Vec::new(),
            operation: CombineOperation::Join,
            keep_toolbodies: false,
            pick_mode: GuiCombinePickMode::Base,
            preview_hidden: false,
            input_geometry_open: true,
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

impl DockableWindow for CombineDialog {}

impl CombineDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Combine", &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, edit: Option<FeatureId>) {
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
        self.preview_hidden = false;
        self.consumed_pick = state.combine_pick.clone();
        self.edit_error = None;
        self.edit_error_open = false;
        self.input_geometry_open = true;
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.combine_edits.get(&id))
        {
            self.presets.clear_selection();
            self.base_body = Some(seed.base_body.clone());
            self.tool_bodies = seed.tool_bodies.clone();
            self.operation = seed.operation;
            self.keep_toolbodies = seed.keep_toolbodies;
            self.pick_mode = GuiCombinePickMode::Toolbodies;
            return;
        }
        self.base_body = None;
        self.tool_bodies.clear();
        self.pick_mode = GuiCombinePickMode::Base;
        if let Some(saved) = self.presets.initial_fields() {
            self.apply_preset(saved);
        } else {
            self.operation = CombineOperation::Join;
            self.keep_toolbodies = false;
        }
    }

    fn current_preset(&self) -> CombinePresetFields {
        CombinePresetFields {
            operation: self.operation,
            keep_toolbodies: self.keep_toolbodies,
        }
    }

    fn apply_preset(&mut self, preset: CombinePresetFields) {
        self.operation = preset.operation;
        self.keep_toolbodies = preset.keep_toolbodies;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Combine(GuiCombineAction::Clear));
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
                self.open(state, None);
            } else {
                self.close(actions);
            }
        }
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiCombineRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiCombineRequest {
            document,
            generation,
            feature: self.editing,
            base_body: self.base_body.clone(),
            tool_bodies: self.tool_bodies.clone(),
            operation: self.operation,
            keep_toolbodies: self.keep_toolbodies,
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let Some(pick) = &state.combine_pick else {
            return;
        };
        if self.consumed_pick.as_ref() == Some(pick) {
            return;
        }
        self.consumed_pick = Some(pick.clone());
        match self.pick_mode {
            GuiCombinePickMode::Base => {
                self.tool_bodies.retain(|body| body != &pick.body);
                let was_empty = self.base_body.is_none();
                self.base_body = Some(pick.body.clone());
                if was_empty {
                    self.pick_mode = GuiCombinePickMode::Toolbodies;
                    actions.push(GuiShellAction::Combine(GuiCombineAction::SetPickMode(
                        self.pick_mode,
                    )));
                }
            }
            GuiCombinePickMode::Toolbodies => {
                if self.base_body.as_ref() == Some(&pick.body) {
                    return;
                }
                if !self.tool_bodies.iter().any(|body| body == &pick.body) {
                    self.tool_bodies.push(pick.body.clone());
                }
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
        self.consume_picks(state, actions);
        let height = ((area.bottom - area.top) - 24.0).clamp(320.0, 520.0);
        widget_abi::fit_viewport_properties(ui, "Combine", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Combine", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Combine",
                "##combine-adv",
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
                "combine",
                "New Combine Default",
                Some(self.current_preset()),
                "Combine Preset",
            ) {
                self.apply_preset(chosen);
            }
            ui.separator();
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(120.0);
            ui.child_window("##combine-body")
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
        self.draw_output_controls(ui);
        if let Some(preview) = &state.combine_preview
            && let GuiCombinePreview::Failed(msg) = &preview.result
            && self.editing.is_none()
        {
            ui.spacing();
            ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Combine(GuiCombineAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::Combine(GuiCombineAction::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::Combine(GuiCombineAction::Clear)),
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn combine_picks_advance_from_base_to_multiple_tools_and_clear_restores_mode() {
        let mut dialog = CombineDialog::default();
        let state = GuiCommandState {
            document: Some(1),
            document_generation: Some(2),
            authoring: GuiAuthoringContext::Part,
            ..Default::default()
        };
        dialog.open(&state, None);
        assert_eq!(dialog.pick_mode, GuiCombinePickMode::Base);
        let mut actions = Vec::new();
        let mut with_pick = state.clone();
        with_pick.combine_pick = Some(GuiCombinePick {
            document: 1,
            generation: 2,
            sequence: 1,
            body: vec![1, 2, 3],
        });
        dialog.consume_picks(&with_pick, &mut actions);
        assert_eq!(dialog.base_body, Some(vec![1, 2, 3]));
        assert_eq!(dialog.pick_mode, GuiCombinePickMode::Toolbodies);

        with_pick.combine_pick = Some(GuiCombinePick {
            document: 1,
            generation: 2,
            sequence: 2,
            body: vec![4, 5, 6],
        });
        dialog.consume_picks(&with_pick, &mut actions);
        with_pick.combine_pick = Some(GuiCombinePick {
            document: 1,
            generation: 2,
            sequence: 3,
            body: vec![7, 8, 9],
        });
        dialog.consume_picks(&with_pick, &mut actions);
        assert_eq!(dialog.tool_bodies.len(), 2);
    }
}
