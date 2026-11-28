// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Modify Split (Faces and Solid modes, #3918).
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
    split::{SplitKeepSide, SplitMode, SplitTool},
};
pub use types::{
    GuiSplitAction, GuiSplitEditSeed, GuiSplitPick, GuiSplitPickKind, GuiSplitPickMode,
    GuiSplitPreview, GuiSplitPreviewState, GuiSplitRequest,
};

#[derive(Clone, Debug, PartialEq, Eq)]
struct SplitPresetFields {
    mode: SplitMode,
    all_faces: bool,
    keep_side: SplitKeepSide,
}

pub(crate) struct SplitDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) mode: SplitMode,
    pub(super) tool: Option<SplitTool>,
    pub(super) body: Option<Vec<u8>>,
    pub(super) all_faces: bool,
    pub(super) all_faces_count: usize,
    pub(super) faces: Vec<Vec<u8>>,
    pub(super) keep_side: SplitKeepSide,
    pub(super) pick_mode: GuiSplitPickMode,
    preview_hidden: bool,
    pub(super) input_geometry_open: bool,
    presets: SessionPresets<SplitPresetFields>,
    attempted: Option<GuiSplitRequest>,
    pub(super) submitted: Option<GuiSplitRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_pick_seq: u64,
    last_browser_feature: Option<FeatureId>,
    last_browser_body: Option<String>,
}

impl Default for SplitDialog {
    fn default() -> Self {
        let mut presets = SessionPresets::default();
        presets.shown = true;
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            mode: SplitMode::Faces,
            tool: None,
            body: None,
            all_faces: false,
            all_faces_count: 0,
            faces: Vec::new(),
            keep_side: SplitKeepSide::Both,
            pick_mode: GuiSplitPickMode::Tool,
            preview_hidden: false,
            input_geometry_open: true,
            presets,
            attempted: None,
            submitted: None,
            reopen_after_commit: false,
            consumed_pick_seq: 0,
            last_browser_feature: None,
            last_browser_body: None,
        }
    }
}

impl DockableWindow for SplitDialog {}

impl SplitDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Split", &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, edit: Option<FeatureId>) {
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
        self.preview_hidden = false;
        self.input_geometry_open = true;
        self.consumed_pick_seq = state.split_pick.as_ref().map_or(0, |p| p.sequence);
        self.last_browser_feature = state.browser.as_ref().and_then(|b| b.selected_feature);
        self.last_browser_body = state.selected_body_key.clone();
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.split_edits.get(&id))
        {
            self.presets.clear_selection();
            self.mode = seed.mode;
            self.tool = Some(seed.tool.clone());
            self.body = Some(seed.body.clone());
            self.all_faces = seed.all_faces;
            self.all_faces_count = seed.all_faces_count;
            self.faces = seed.faces.clone();
            self.keep_side = seed.keep_side;
            self.pick_mode = GuiSplitPickMode::Target;
            return;
        }
        self.mode = SplitMode::Faces;
        self.tool = None;
        self.body = None;
        self.all_faces = false;
        self.all_faces_count = 0;
        self.faces.clear();
        self.keep_side = SplitKeepSide::Both;
        self.pick_mode = GuiSplitPickMode::Tool;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Split(GuiSplitAction::Clear));
        }
        self.editing = None;
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
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

    fn current_preset(&self) -> SplitPresetFields {
        SplitPresetFields {
            mode: self.mode,
            all_faces: self.all_faces,
            keep_side: self.keep_side,
        }
    }

    fn apply_preset(&mut self, preset: SplitPresetFields) {
        self.mode = preset.mode;
        self.all_faces = preset.all_faces;
        self.keep_side = preset.keep_side;
    }

    pub(super) fn switch_mode(&mut self, next: SplitMode, actions: &mut Vec<GuiShellAction>) {
        if self.mode == next {
            return;
        }
        self.mode = next;
        self.all_faces = false;
        self.all_faces_count = 0;
        self.faces.clear();
        if next == SplitMode::Solid {
            self.body = None;
        }
        self.pick_mode = if self.tool.is_none() {
            GuiSplitPickMode::Tool
        } else {
            GuiSplitPickMode::Target
        };
        actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
            self.pick_mode,
            self.mode,
        )));
    }

    pub(super) fn set_all_faces(
        &mut self,
        enabled: bool,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.all_faces = enabled;
        if !enabled {
            self.faces.clear();
            self.all_faces_count = 0;
            self.pick_mode = GuiSplitPickMode::Target;
            actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
                self.pick_mode,
                self.mode,
            )));
            return;
        }
        self.faces.clear();
        if self.tool.is_some() {
            self.populate_default_all_faces(state);
        }
    }

    fn populate_default_all_faces(&mut self, state: &GuiCommandState) {
        if let Some(browser) = &state.browser
            && browser.bodies.len() == 1
        {
            if self.body.is_none()
                && let Ok(bytes) = controls::decode_hex_key(&browser.bodies[0].key)
            {
                self.body = Some(bytes);
            }
            if self.all_faces_count == 0 {
                self.all_faces_count = browser.bodies[0].faces;
            }
        }
    }

    pub(super) fn displayed_face_count(&self) -> usize {
        if self.all_faces {
            self.all_faces_count
        } else {
            self.faces.len()
        }
    }

    pub(super) fn is_complete(&self) -> bool {
        if self.tool.is_none() || self.body.is_none() {
            return false;
        }
        match self.mode {
            SplitMode::Faces => self.all_faces || !self.faces.is_empty(),
            SplitMode::Solid => true,
        }
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiSplitRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiSplitRequest {
            document,
            generation,
            feature: self.editing,
            mode: self.mode,
            tool: self.tool.clone(),
            body: self.body.clone(),
            all_faces: self.all_faces,
            faces: self.faces.clone(),
            keep_side: self.keep_side,
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(
        &mut self,
        state: &GuiCommandState,
        browser_plane: Option<&str>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let Some(plane_ref) = browser_plane
            && let Some(tool) = controls::parse_browser_plane_tool(plane_ref)
        {
            self.accept_tool(tool, None, 0, state, actions);
        }
        if let Some(browser) = &state.browser
            && browser.selected_feature != self.last_browser_feature
        {
            self.last_browser_feature = browser.selected_feature;
            if let Some(id) = browser.selected_feature
                && browser
                    .features
                    .iter()
                    .any(|f| f.id == id && f.kind == "Work Plane")
            {
                self.accept_tool(SplitTool::WorkPlane(id), None, 0, state, actions);
            }
        }
        if state.selected_body_key != self.last_browser_body {
            self.last_browser_body = state.selected_body_key.clone();
            if self.mode == SplitMode::Solid
                && let Some(hex) = &state.selected_body_key
                && let Ok(bytes) = controls::decode_hex_key(hex)
            {
                self.body = Some(bytes);
            }
        }
        let Some(pick) = &state.split_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        match &pick.kind {
            GuiSplitPickKind::SketchCurve {
                tool,
                default_body,
                default_face_count,
            } => {
                self.accept_tool(
                    tool.clone(),
                    default_body.clone(),
                    *default_face_count,
                    state,
                    actions,
                );
            }
            GuiSplitPickKind::WorkPlane(id) => {
                self.accept_tool(SplitTool::WorkPlane(*id), None, 0, state, actions);
            }
            GuiSplitPickKind::Face { body, face } => {
                if self.mode == SplitMode::Faces {
                    self.all_faces = false;
                    self.all_faces_count = 0;
                    if self.body.as_ref() != Some(body) {
                        self.body = Some(body.clone());
                        self.faces.clear();
                    }
                    if !self.faces.iter().any(|existing| existing == face) {
                        self.faces.push(face.clone());
                    }
                }
            }
            GuiSplitPickKind::Solid { body } => {
                if self.mode == SplitMode::Solid {
                    self.body = Some(body.clone());
                }
            }
        }
    }

    fn accept_tool(
        &mut self,
        tool: SplitTool,
        default_body: Option<Vec<u8>>,
        default_face_count: usize,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.tool = Some(tool);
        if self.mode == SplitMode::Faces && self.all_faces {
            if self.body.is_none() {
                self.body = default_body;
            }
            if default_face_count > 0 {
                self.all_faces_count = default_face_count;
            }
            self.populate_default_all_faces(state);
        }
        self.pick_mode = GuiSplitPickMode::Target;
        actions.push(GuiShellAction::Split(GuiSplitAction::SetPickMode(
            self.pick_mode,
            self.mode,
        )));
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        icons: &GuiRibbonIcons,
        browser_plane: Option<&str>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state, browser_plane, actions);
        let height = ((area.bottom - area.top) - 24.0).clamp(280.0, 440.0);
        widget_abi::fit_viewport_properties(ui, "Split", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Split", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Split",
                "##split-adv",
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
                "split",
                "New Split Default",
                Some(self.current_preset()),
                "Split Preset",
            ) {
                self.apply_preset(chosen);
            }
            ui.separator();
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(110.0);
            ui.child_window("##split-body")
                .size([0.0, body_h])
                .build(|| {
                    self.draw_input_geometry(ui, state, actions);
                    if let Some(preview) = &state.split_preview
                        && let GuiSplitPreview::Failed(msg) = &preview.result
                    {
                        ui.spacing();
                        ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
                    }
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer_controls(ui, state, had_popup || self.presets.popup_open, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Split(GuiSplitAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::Split(GuiSplitAction::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::Split(GuiSplitAction::Clear)),
            }
        }
    }
}

#[cfg(test)]
mod tests;
