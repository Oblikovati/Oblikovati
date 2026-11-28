// SPDX-License-Identifier: GPL-2.0-only
//! Dockable dialog for 3D Model > Surface > Trim (Trim Surface, #3918).
mod controls;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    surface_trim::{SurfaceTrimSide, SurfaceTrimTool},
};
pub use types::{
    GuiSurfaceTrimAction, GuiSurfaceTrimEditSeed, GuiSurfaceTrimPick, GuiSurfaceTrimPickKind,
    GuiSurfaceTrimPickMode, GuiSurfaceTrimPreview, GuiSurfaceTrimPreviewState,
    GuiSurfaceTrimRequest,
};

pub(crate) struct SurfaceTrimDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) tool: Option<SurfaceTrimTool>,
    pub(super) target_body: Option<Vec<u8>>,
    pub(super) remove_side: Option<SurfaceTrimSide>,
    pub(super) pick_mode: GuiSurfaceTrimPickMode,
    attempted: Option<GuiSurfaceTrimRequest>,
    pub(super) submitted: Option<GuiSurfaceTrimRequest>,
    pub(super) reopen_after_commit: bool,
    consumed_pick_seq: u64,
    last_browser_feature: Option<FeatureId>,
}

impl Default for SurfaceTrimDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            tool: None,
            target_body: None,
            remove_side: None,
            pick_mode: GuiSurfaceTrimPickMode::CuttingTool,
            attempted: None,
            submitted: None,
            reopen_after_commit: false,
            consumed_pick_seq: 0,
            last_browser_feature: None,
        }
    }
}

impl DockableWindow for SurfaceTrimDialog {}

impl SurfaceTrimDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Trim Surface", &self.dock)
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
        self.consumed_pick_seq = state
            .surface_trim_pick
            .as_ref()
            .map_or(0, |pick| pick.sequence);
        self.last_browser_feature = state.browser.as_ref().and_then(|b| b.selected_feature);
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.surface_trim_edits.get(&id))
        {
            self.tool = Some(seed.tool.clone());
            self.target_body = Some(seed.target_body.clone());
            self.remove_side = Some(seed.remove_side);
            self.pick_mode = GuiSurfaceTrimPickMode::Remove;
            actions.push(GuiShellAction::SurfaceTrim(GuiSurfaceTrimAction::OpenEdit(
                id,
            )));
            actions.push(GuiShellAction::SurfaceTrim(
                GuiSurfaceTrimAction::SetPickMode(self.pick_mode),
            ));
            return;
        }
        self.tool = None;
        self.target_body = None;
        self.remove_side = None;
        self.pick_mode = GuiSurfaceTrimPickMode::CuttingTool;
        actions.push(GuiShellAction::SurfaceTrim(GuiSurfaceTrimAction::Open));
        actions.push(GuiShellAction::SurfaceTrim(
            GuiSurfaceTrimAction::SetPickMode(self.pick_mode),
        ));
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::SurfaceTrim(GuiSurfaceTrimAction::Clear));
        }
        self.editing = None;
        self.tool = None;
        self.target_body = None;
        self.remove_side = None;
        self.pick_mode = GuiSurfaceTrimPickMode::CuttingTool;
        self.attempted = None;
        self.submitted = None;
        self.reopen_after_commit = false;
    }

    pub(crate) fn synchronize(
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
            if self.reopen_after_commit {
                self.reopen_after_commit = false;
                self.open(state, None, actions);
            } else {
                self.close(actions);
            }
        }
    }

    pub(super) fn set_pick_mode(
        &mut self,
        mode: GuiSurfaceTrimPickMode,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.pick_mode = mode;
        actions.push(GuiShellAction::SurfaceTrim(
            GuiSurfaceTrimAction::SetPickMode(mode),
        ));
    }

    pub(super) fn is_complete(&self) -> bool {
        self.tool.is_some() && self.target_body.is_some() && self.remove_side.is_some()
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiSurfaceTrimRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiSurfaceTrimRequest {
            document,
            generation,
            feature: self.editing,
            tool: self.tool.clone(),
            target_body: self.target_body.clone(),
            remove_side: self.remove_side,
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
            self.accept_tool(tool, actions);
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
                self.accept_tool(SurfaceTrimTool::WorkPlane(id), actions);
            }
        }
        let Some(pick) = &state.surface_trim_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        match &pick.kind {
            GuiSurfaceTrimPickKind::CuttingTool(tool) => {
                self.accept_tool(tool.clone(), actions);
            }
            GuiSurfaceTrimPickKind::RemoveRegion {
                target_body,
                remove_side,
            } => {
                self.target_body = Some(target_body.clone());
                self.remove_side = Some(*remove_side);
            }
        }
    }

    fn accept_tool(&mut self, tool: SurfaceTrimTool, actions: &mut Vec<GuiShellAction>) {
        self.tool = Some(tool);
        self.set_pick_mode(GuiSurfaceTrimPickMode::Remove, actions);
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        browser_plane: Option<&str>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.synchronize(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state, browser_plane, actions);
        let height = ((area.bottom - area.top) - 24.0).clamp(180.0, 240.0);
        widget_abi::fit_viewport_properties(ui, "Trim Surface", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Trim Surface", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            ui.text("Trim Surface");
            ui.separator();
            let footer_h = 38.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(76.0);
            ui.child_window("##surface-trim-body")
                .size([0.0, body_h])
                .build(|| {
                    self.draw_cutting_tool_row(ui, actions);
                    ui.spacing();
                    self.draw_remove_row(ui, actions);
                    if let Some(preview) = &state.surface_trim_preview
                        && let GuiSurfaceTrimPreview::Failed(msg) = &preview.result
                    {
                        ui.spacing();
                        ui.text_colored([0.92, 0.34, 0.34, 1.0], format!("! {msg}"));
                    }
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer_controls(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::SurfaceTrim(GuiSurfaceTrimAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::SurfaceTrim(
                    GuiSurfaceTrimAction::Preview(Box::new(req)),
                )),
                None => actions.push(GuiShellAction::SurfaceTrim(GuiSurfaceTrimAction::Clear)),
            }
        }
    }
}
