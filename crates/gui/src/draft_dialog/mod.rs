// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Modify Draft (Fixed Edge, Fixed Plane, Parting Tool, #3918).
mod controls;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::feature_properties::draw_properties_header;
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::{
    FeatureId,
    draft::{DraftAngleMode, DraftMode, DraftPartingTool},
    hole::HoleOriginPlane,
};
pub use types::{
    GuiDraftAction, GuiDraftEditSeed, GuiDraftPick, GuiDraftPickTarget, GuiDraftPreview,
    GuiDraftPreviewState, GuiDraftRequest,
};

pub(crate) struct DraftDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    body: Option<Vec<u8>>,
    pub(super) mode: DraftMode,
    pub(super) pull_direction: Option<Vec<u8>>,
    pub(super) parting_tool: Option<DraftPartingTool>,
    pub(super) faces: Vec<Vec<u8>>,
    pub(super) angle: String,
    pub(super) angle_mode: DraftAngleMode,
    pub(super) direction_flipped: bool,
    pub(super) automatic_blending: bool,
    pub(super) pick_target: GuiDraftPickTarget,
    pub(super) options_expanded: bool,
    preview_hidden: bool,
    presets_shown: bool,
    attempted: Option<GuiDraftRequest>,
    pub(super) submitted: Option<GuiDraftRequest>,
    consumed_pick_seq: u64,
    pub(super) failure_modal: Option<String>,
    pub(super) failure_detail_expanded: bool,
}

impl Default for DraftDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            body: None,
            mode: DraftMode::FixedEdge,
            pull_direction: None,
            parting_tool: None,
            faces: Vec::new(),
            angle: "2.5 deg".into(),
            angle_mode: DraftAngleMode::Absolute,
            direction_flipped: false,
            automatic_blending: true,
            pick_target: GuiDraftPickTarget::PullDirection,
            options_expanded: false,
            preview_hidden: false,
            presets_shown: false,
            attempted: None,
            submitted: None,
            consumed_pick_seq: 0,
            failure_modal: None,
            failure_detail_expanded: false,
        }
    }
}

impl DockableWindow for DraftDialog {}

impl DraftDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Draft", &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, edit: Option<FeatureId>) {
        self.document = state.document;
        self.editing = edit;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.preview_hidden = false;
        self.failure_modal = None;
        self.failure_detail_expanded = false;
        self.options_expanded = false;
        self.consumed_pick_seq = state.draft_pick.as_ref().map_or(0, |p| p.sequence);
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.draft_edits.get(&id))
        {
            self.body = Some(seed.body.clone());
            self.mode = seed.mode;
            self.pull_direction = Some(seed.pull_direction.clone());
            self.parting_tool = seed.parting_tool.clone();
            self.faces = seed.faces.clone();
            self.angle = seed.angle.clone();
            self.angle_mode = seed.angle_mode;
            self.direction_flipped = seed.direction_flipped;
            self.automatic_blending = seed.automatic_blending;
            self.pick_target = GuiDraftPickTarget::Faces;
            return;
        }
        self.body = None;
        self.mode = DraftMode::FixedEdge;
        self.pull_direction = None;
        self.parting_tool = None;
        self.faces.clear();
        self.angle = "2.5 deg".into();
        self.angle_mode = DraftAngleMode::Absolute;
        self.direction_flipped = false;
        self.automatic_blending = true;
        self.pick_target = GuiDraftPickTarget::PullDirection;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Draft(GuiDraftAction::Clear));
        }
        self.editing = None;
        self.attempted = None;
        self.submitted = None;
        self.failure_modal = None;
        self.failure_detail_expanded = false;
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
            self.close(actions);
        }
    }

    pub(super) fn faces_selector_enabled(&self) -> bool {
        match self.mode {
            DraftMode::FixedEdge | DraftMode::FixedPlane => self.pull_direction.is_some(),
            DraftMode::PartingTool => self.pull_direction.is_some() && self.parting_tool.is_some(),
        }
    }

    pub(super) fn geometry_complete(&self) -> bool {
        self.faces_selector_enabled() && !self.faces.is_empty()
    }

    pub(super) fn sync_pick_target_after_mode_change(&mut self) {
        if self.pull_direction.is_none() {
            self.pick_target = GuiDraftPickTarget::PullDirection;
        } else if self.mode == DraftMode::PartingTool && self.parting_tool.is_none() {
            self.pick_target = GuiDraftPickTarget::PartingTool;
        } else {
            self.pick_target = GuiDraftPickTarget::Faces;
        }
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiDraftRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiDraftRequest {
            document,
            generation,
            feature: self.editing,
            body: self.body.clone(),
            mode: self.mode,
            pull_direction: self.pull_direction.clone(),
            parting_tool: self.parting_tool.clone(),
            faces: self.faces.clone(),
            angle: self.angle.clone(),
            angle_mode: self.angle_mode,
            direction_flipped: self.direction_flipped,
            automatic_blending: self.automatic_blending,
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState, browser_plane: Option<&str>) {
        if let Some(plane_ref) = browser_plane
            && self.mode == DraftMode::PartingTool
            && let Some(origin_plane) = parse_browser_origin_plane(plane_ref)
        {
            self.parting_tool = Some(DraftPartingTool::OriginPlane(origin_plane));
            if self.pull_direction.is_some() {
                self.pick_target = GuiDraftPickTarget::Faces;
            }
        }
        let Some(pick) = &state.draft_pick else {
            return;
        };
        if pick.sequence <= self.consumed_pick_seq {
            return;
        }
        self.consumed_pick_seq = pick.sequence;
        if self.body.is_none() {
            self.body = Some(pick.body.clone());
        }
        match self.pick_target {
            GuiDraftPickTarget::PullDirection => {
                self.pull_direction = Some(pick.face.clone());
                self.faces.retain(|f| f != &pick.face);
                self.sync_pick_target_after_mode_change();
            }
            GuiDraftPickTarget::PartingTool => {
                self.parting_tool = Some(DraftPartingTool::Face(pick.face.clone()));
                if self.pull_direction.is_some() {
                    self.pick_target = GuiDraftPickTarget::Faces;
                }
            }
            GuiDraftPickTarget::Faces => {
                if self.pull_direction.as_ref() != Some(&pick.face)
                    && !self.faces.iter().any(|f| f == &pick.face)
                {
                    self.faces.push(pick.face.clone());
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
        browser_plane: Option<&str>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.sync_before_docking(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state, browser_plane);
        let base_h: f32 = if self.mode == DraftMode::PartingTool {
            340.0
        } else {
            305.0
        };
        let extra_h: f32 = if self.options_expanded { 76.0 } else { 0.0 };
        let height = (base_h + extra_h)
            .min((area.bottom - area.top) - 24.0)
            .max(240.0);
        widget_abi::fit_viewport_properties(ui, "Draft", area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        let mut enter_commit = false;
        Self::dockable_window(ui, "Draft", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Face Draft",
                "##draft-adv",
                &mut self.presets_shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            ui.separator();
            self.draw_mode_buttons(ui);
            ui.separator();
            self.draw_selectors(ui);
            ui.separator();
            enter_commit = self.draw_angle_and_options(ui);
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer(ui, state, enter_commit, actions);
        });
        self.draw_failure_modal(ui, actions);
        if !open {
            self.close(actions);
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Draft(GuiDraftAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::Draft(GuiDraftAction::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::Draft(GuiDraftAction::Clear)),
            }
        }
    }
}

pub(super) fn parse_angle_deg(raw: &str) -> Option<f64> {
    let cleaned = raw
        .trim()
        .replace(',', ".")
        .replace('\u{00b0}', "")
        .replace("deg", "");
    let value: f64 = cleaned.trim().parse().ok()?;
    value.is_finite().then_some(value)
}

pub(super) fn parting_tool_label(tool: &DraftPartingTool) -> &'static str {
    match tool {
        DraftPartingTool::OriginPlane(HoleOriginPlane::Xy) => "XY Plane",
        DraftPartingTool::OriginPlane(HoleOriginPlane::Xz) => "XZ Plane",
        DraftPartingTool::OriginPlane(HoleOriginPlane::Yz) => "YZ Plane",
        DraftPartingTool::WorkPlane(_) => "Work Plane",
        DraftPartingTool::Face(_) => "1 Face",
    }
}

fn parse_browser_origin_plane(reference: &str) -> Option<HoleOriginPlane> {
    let lower = reference.trim().to_ascii_lowercase();
    if lower.contains("xy") {
        Some(HoleOriginPlane::Xy)
    } else if lower.contains("xz") {
        Some(HoleOriginPlane::Xz)
    } else if lower.contains("yz") {
        Some(HoleOriginPlane::Yz)
    } else {
        None
    }
}
