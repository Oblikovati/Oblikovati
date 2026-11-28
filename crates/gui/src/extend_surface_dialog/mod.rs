// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Properties panel for Part Surface Extend (#3918).
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
    extend_surface::{ExtendSurfaceExtent, ExtendSurfaceMethod, ExtendSurfaceTermination},
};
pub use types::{
    GuiExtendSurfaceAction, GuiExtendSurfaceEditSeed, GuiExtendSurfaceExtentMode,
    GuiExtendSurfacePick, GuiExtendSurfacePreview, GuiExtendSurfacePreviewState,
    GuiExtendSurfaceRequest,
};

pub(super) const PANEL_WINDOW_TITLE: &str = "Extend Surface";
const DEFAULT_DISTANCE_TEXT: &str = "5 mm";

pub(crate) struct ExtendSurfaceDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) target_body: Option<Vec<u8>>,
    pub(super) edges: Vec<Vec<u8>>,
    pub(super) edge_chain: bool,
    pub(super) extent_mode: GuiExtendSurfaceExtentMode,
    pub(super) distance_text: String,
    pub(super) termination: Option<ExtendSurfaceTermination>,
    pub(super) picking_termination: bool,
    pub(super) method: ExtendSurfaceMethod,
    preview_hidden: bool,
    presets_shown: bool,
    pub(super) input_geometry_open: bool,
    pub(super) more_open: bool,
    attempted: Option<GuiExtendSurfaceRequest>,
    pub(super) submitted: Option<GuiExtendSurfaceRequest>,
    consumed_pick: Option<GuiExtendSurfacePick>,
}

impl Default for ExtendSurfaceDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            target_body: None,
            edges: Vec::new(),
            edge_chain: true,
            extent_mode: GuiExtendSurfaceExtentMode::Distance,
            distance_text: DEFAULT_DISTANCE_TEXT.to_string(),
            termination: None,
            picking_termination: false,
            method: ExtendSurfaceMethod::Extend,
            preview_hidden: false,
            presets_shown: false,
            input_geometry_open: true,
            more_open: true,
            attempted: None,
            submitted: None,
            consumed_pick: None,
        }
    }
}

impl DockableWindow for ExtendSurfaceDialog {}

impl ExtendSurfaceDialog {
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
        self.preview_hidden = false;
        self.consumed_pick = state.extend_surface_pick.clone();
        self.input_geometry_open = true;
        self.more_open = true;
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.extend_surface_edits.get(&id))
        {
            self.target_body = Some(seed.target_body.clone());
            self.edges = seed.edges.clone();
            self.edge_chain = seed.edge_chain;
            self.method = seed.method;
            match &seed.extent {
                ExtendSurfaceExtent::Distance(cm) => {
                    self.extent_mode = GuiExtendSurfaceExtentMode::Distance;
                    self.distance_text = format_mm_distance(*cm);
                    self.termination = None;
                    self.picking_termination = false;
                }
                ExtendSurfaceExtent::To(target) => {
                    self.extent_mode = GuiExtendSurfaceExtentMode::To;
                    self.distance_text = DEFAULT_DISTANCE_TEXT.to_string();
                    self.termination = Some(target.clone());
                    self.picking_termination = false;
                }
            }
            actions.push(GuiShellAction::ExtendSurface(
                GuiExtendSurfaceAction::SetEdgeChain(self.edge_chain),
            ));
            actions.push(GuiShellAction::ExtendSurface(
                GuiExtendSurfaceAction::SetPickTermination(self.picking_termination),
            ));
            return;
        }
        self.target_body = None;
        self.edges.clear();
        self.edge_chain = true;
        self.extent_mode = GuiExtendSurfaceExtentMode::Distance;
        self.distance_text = DEFAULT_DISTANCE_TEXT.to_string();
        self.termination = None;
        self.picking_termination = false;
        self.method = ExtendSurfaceMethod::Extend;
        actions.push(GuiShellAction::ExtendSurface(
            GuiExtendSurfaceAction::SetEdgeChain(self.edge_chain),
        ));
        actions.push(GuiShellAction::ExtendSurface(
            GuiExtendSurfaceAction::SetPickTermination(false),
        ));
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::ExtendSurface(GuiExtendSurfaceAction::Clear));
        }
        self.editing = None;
        self.attempted = None;
        self.submitted = None;
        self.picking_termination = false;
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
            self.close(actions);
        }
    }

    pub(super) fn set_picking_termination(
        &mut self,
        enabled: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.picking_termination != enabled {
            self.picking_termination = enabled;
            actions.push(GuiShellAction::ExtendSurface(
                GuiExtendSurfaceAction::SetPickTermination(enabled),
            ));
        }
    }

    pub(super) fn parsed_distance_cm(&self) -> Option<f64> {
        let cm = parse_distance_cm(&self.distance_text)?;
        (cm.is_finite() && cm > 0.0).then_some(cm)
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiExtendSurfaceRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiExtendSurfaceRequest {
            document,
            generation,
            feature: self.editing,
            target_body: self.target_body.clone(),
            edges: self.edges.clone(),
            edge_chain: self.edge_chain,
            extent_mode: self.extent_mode,
            distance: self.parsed_distance_cm(),
            termination: self.termination.clone(),
            method: self.method,
            preview_hidden: self.preview_hidden,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let Some(pick) = &state.extend_surface_pick else {
            return;
        };
        if self.consumed_pick.as_ref() == Some(pick) {
            return;
        }
        self.consumed_pick = Some(pick.clone());
        if let Some(termination) = &pick.termination {
            self.termination = Some(termination.clone());
            self.set_picking_termination(false, actions);
            return;
        }
        let Some(target_body) = &pick.target_body else {
            return;
        };
        if self.target_body.as_ref() != Some(target_body) {
            self.target_body = Some(target_body.clone());
            self.edges = pick.edges.clone();
            return;
        }
        for edge in &pick.edges {
            if !self.edges.iter().any(|existing| existing == edge) {
                self.edges.push(edge.clone());
            }
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
        self.synchronize(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state, actions);
        let height = ((area.bottom - area.top) - 16.0).clamp(360.0, 520.0);
        widget_abi::fit_viewport_properties(ui, PANEL_WINDOW_TITLE, area, [280.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, PANEL_WINDOW_TITLE, &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                "Extend Surface",
                "##extend-surface-adv",
                &mut self.presets_shown,
                Some((self.preview_hidden, true)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            ui.separator();
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(160.0);
            ui.child_window("##extend-surface-body")
                .size([0.0, body_h])
                .build(|| {
                    self.draw_input_geometry(ui, actions);
                    ui.separator();
                    self.draw_extents_controls(ui, actions);
                    ui.separator();
                    self.draw_more_controls(ui);
                    self.draw_status_error(ui, state);
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
            GuiShellAction::ExtendSurface(GuiExtendSurfaceAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::ExtendSurface(
                    GuiExtendSurfaceAction::Preview(Box::new(req)),
                )),
                None => actions.push(GuiShellAction::ExtendSurface(GuiExtendSurfaceAction::Clear)),
            }
        }
    }
}

fn parse_distance_cm(raw: &str) -> Option<f64> {
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

fn format_mm_distance(cm: f64) -> String {
    let mm = cm * 10.0;
    if (mm - mm.round()).abs() < 1.0e-6 {
        format!("{} mm", mm.round() as i64)
    } else {
        let formatted = format!("{mm:.3}");
        let trimmed = formatted.trim_end_matches('0').trim_end_matches('.');
        format!("{trimmed} mm")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn defaults_and_distance_parsing_match_extend_surface_spec() {
        let mut dialog = ExtendSurfaceDialog::default();
        assert!(dialog.edge_chain);
        assert_eq!(dialog.extent_mode, GuiExtendSurfaceExtentMode::Distance);
        assert_eq!(dialog.distance_text, "5 mm");
        assert_eq!(dialog.parsed_distance_cm(), Some(0.5));
        assert_eq!(dialog.method, ExtendSurfaceMethod::Extend);

        dialog.distance_text = "10 mm".into();
        assert_eq!(dialog.parsed_distance_cm(), Some(1.0));
        dialog.distance_text = "0 mm".into();
        assert!(dialog.parsed_distance_cm().is_none());
        dialog.distance_text = "-5 mm".into();
        assert!(dialog.parsed_distance_cm().is_none());
        assert_eq!(format_mm_distance(1.0), "10 mm");
    }
}
