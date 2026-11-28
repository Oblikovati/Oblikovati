// SPDX-License-Identifier: GPL-2.0-only
//! Dockable dialog for Part Surface Stitch with Stitch and Analyze tabs (#3918).
mod controls;
pub mod types;

use crate::dockable_window::DockableWindow;
use crate::icons::GuiRibbonIcons;
use crate::widget_abi;
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::feature::FeatureId;
pub use types::{
    GuiStitchAction, GuiStitchEditSeed, GuiStitchFreeEdge, GuiStitchPick, GuiStitchPreview,
    GuiStitchPreviewState, GuiStitchRequest,
};

pub(super) const PANEL_WINDOW_TITLE: &str = "Stitch";
pub(super) const DEFAULT_TOLERANCE_MM: f64 = 0.01;

pub(crate) struct StitchDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    pub(super) editing: Option<FeatureId>,
    pub(super) surfaces: Vec<Vec<u8>>,
    pub(super) surface_names: Vec<String>,
    pub(super) maintain_as_surface: bool,
    pub(super) tolerance_text: String,
    pub(super) show_edge_conditions: bool,
    pub(super) show_near_tangent: bool,
    pub(super) applied_free_edges: Vec<GuiStitchFreeEdge>,
    pub(super) applied_stitched_count: Option<usize>,
    attempted: Option<GuiStitchRequest>,
    pub(super) submitted: Option<GuiStitchRequest>,
    consumed_pick_seq: u64,
    last_browser_body: Option<String>,
}

impl Default for StitchDialog {
    fn default() -> Self {
        Self {
            dock: Default::default(),
            document: None,
            editing: None,
            surfaces: Vec::new(),
            surface_names: Vec::new(),
            maintain_as_surface: true,
            tolerance_text: format_mm_tolerance(DEFAULT_TOLERANCE_MM),
            show_edge_conditions: false,
            show_near_tangent: false,
            applied_free_edges: Vec::new(),
            applied_stitched_count: None,
            attempted: None,
            submitted: None,
            consumed_pick_seq: 0,
            last_browser_body: None,
        }
    }
}

impl DockableWindow for StitchDialog {}

impl StitchDialog {
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
        self.applied_free_edges.clear();
        self.applied_stitched_count = None;
        self.consumed_pick_seq = state.stitch_pick.as_ref().map_or(0, |p| p.sequence);
        self.last_browser_body = state.selected_body_key.clone();
        if let Some(id) = edit
            && let Some(seed) = state
                .browser
                .as_ref()
                .and_then(|browser| browser.stitch_edits.get(&id))
        {
            self.surfaces = seed.surfaces.clone();
            self.surface_names = resolve_surface_names(state, &self.surfaces);
            self.maintain_as_surface = seed.maintain_as_surface;
            self.tolerance_text = format_mm_tolerance(seed.maximum_tolerance_mm);
            self.show_edge_conditions = seed.show_edge_conditions;
            self.show_near_tangent = seed.show_near_tangent;
            actions.push(GuiShellAction::Stitch(GuiStitchAction::OpenEdit(id)));
            return;
        }
        self.surfaces.clear();
        self.surface_names.clear();
        self.maintain_as_surface = false;
        self.tolerance_text = format_mm_tolerance(DEFAULT_TOLERANCE_MM);
        self.show_edge_conditions = false;
        self.show_near_tangent = false;
        actions.push(GuiShellAction::Stitch(GuiStitchAction::Open));
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Stitch(GuiStitchAction::Done));
        }
        self.editing = None;
        self.attempted = None;
        self.submitted = None;
        self.applied_free_edges.clear();
        self.applied_stitched_count = None;
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
            // Apply keeps the Stitch dialog open and retains the free-edge report.
            self.submitted = None;
            self.attempted = None;
            if self.editing.is_none() {
                self.surfaces.clear();
                self.surface_names.clear();
            }
        }
    }

    pub(super) fn parsed_tolerance_mm(&self) -> Option<f64> {
        parse_tolerance_mm(&self.tolerance_text)
    }

    pub(super) fn request(&self, state: &GuiCommandState) -> Option<GuiStitchRequest> {
        let document = self.document?;
        let generation = state.document_generation?;
        Some(GuiStitchRequest {
            document,
            generation,
            feature: self.editing,
            surfaces: self.surfaces.clone(),
            maintain_as_surface: self.maintain_as_surface,
            maximum_tolerance_mm: self.parsed_tolerance_mm(),
            show_edge_conditions: self.show_edge_conditions,
            show_near_tangent: self.show_near_tangent,
        })
    }

    fn consume_picks(&mut self, state: &GuiCommandState) {
        if let Some(pick) = &state.stitch_pick
            && pick.sequence > self.consumed_pick_seq
        {
            self.consumed_pick_seq = pick.sequence;
            self.add_surface(pick.surface_key.clone(), pick.surface_name.clone());
        }
        if state.selected_body_key != self.last_browser_body {
            self.last_browser_body = state.selected_body_key.clone();
            if let Some(hex) = &state.selected_body_key
                && let Some(browser) = &state.browser
                && let Some(body) = browser.surface_bodies.iter().find(|b| &b.key == hex)
                && let Some(bytes) = decode_hex_key(hex)
            {
                self.add_surface(bytes, body.name.clone());
            }
        }
    }

    fn add_surface(&mut self, key: Vec<u8>, name: String) {
        if self.surfaces.iter().any(|existing| existing == &key) {
            return;
        }
        self.surfaces.push(key);
        self.surface_names.push(name);
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        (area, _icons): (
            crate::application::document_creation::workspaces::WorkspaceArea,
            &GuiRibbonIcons,
        ),
        actions: &mut Vec<GuiShellAction>,
    ) {
        self.synchronize(state, actions);
        if self.document.is_none() {
            return;
        }
        self.consume_picks(state);
        self.capture_preview_edges(state);
        let height = ((area.bottom - area.top) - 16.0).clamp(420.0, 560.0);
        widget_abi::fit_viewport_properties(ui, PANEL_WINDOW_TITLE, area, [300.0, height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, PANEL_WINDOW_TITLE, &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let footer_h = 44.0;
            let body_h = (ui.content_region_avail()[1] - footer_h).max(180.0);
            ui.child_window("##stitch-body")
                .size([0.0, body_h])
                .build(|| {
                    self.draw_tabs(ui, state);
                });
            self.refresh_preview(state, actions);
            ui.separator();
            self.draw_footer(ui, state, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn capture_preview_edges(&mut self, state: &GuiCommandState) {
        if let Some(preview) = &state.stitch_preview
            && let GuiStitchPreview::Ready {
                stitched_edge_count,
                free_edges,
                ..
            } = &preview.result
            && !self.surfaces.is_empty()
        {
            self.applied_free_edges = free_edges.clone();
            self.applied_stitched_count = Some(*stitched_edge_count);
        }
    }

    fn refresh_preview(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let request = self.request(state).filter(|req| {
            GuiShellAction::Stitch(GuiStitchAction::Preview(Box::new(req.clone())))
                .dispatchable(state)
        });
        if self.attempted != request {
            self.attempted = request.clone();
            match request {
                Some(req) => actions.push(GuiShellAction::Stitch(GuiStitchAction::Preview(
                    Box::new(req),
                ))),
                None => actions.push(GuiShellAction::Stitch(GuiStitchAction::Clear)),
            }
        }
    }
}

pub(super) fn parse_tolerance_mm(raw: &str) -> Option<f64> {
    let trimmed = raw.trim();
    if trimmed.is_empty() {
        return None;
    }
    let lower = trimmed.to_ascii_lowercase();
    let (num_str, scale_to_mm) = if let Some(prefix) = lower.strip_suffix("mm") {
        (prefix.trim(), 1.0)
    } else if let Some(prefix) = lower.strip_suffix("cm") {
        (prefix.trim(), 10.0)
    } else if let Some(prefix) = lower.strip_suffix("in") {
        (prefix.trim(), 25.4)
    } else {
        (trimmed, 1.0)
    };
    let val = num_str.parse::<f64>().ok()?;
    (val.is_finite() && val > 0.0).then_some(val * scale_to_mm)
}

pub(super) fn format_mm_tolerance(mm: f64) -> String {
    format!("{mm:.3} mm")
}

fn decode_hex_key(hex: &str) -> Option<Vec<u8>> {
    if !hex.len().is_multiple_of(2) {
        return None;
    }
    let mut out = Vec::with_capacity(hex.len() / 2);
    for pair in hex.as_bytes().as_chunks::<2>().0 {
        let text = std::str::from_utf8(pair).ok()?;
        let byte = u8::from_str_radix(text, 16).ok()?;
        out.push(byte);
    }
    Some(out)
}

fn encode_hex_key(bytes: &[u8]) -> String {
    bytes.iter().map(|b| format!("{b:02x}")).collect()
}

fn resolve_surface_names(state: &GuiCommandState, surfaces: &[Vec<u8>]) -> Vec<String> {
    surfaces
        .iter()
        .enumerate()
        .map(|(idx, key)| {
            let hex = encode_hex_key(key);
            state
                .browser
                .as_ref()
                .and_then(|b| b.surface_bodies.iter().find(|s| s.key == hex))
                .map_or_else(|| format!("Srf{}", idx + 1), |s| s.name.clone())
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tolerance_parsing_and_three_decimal_formatting_match_stitch_spec() {
        assert_eq!(parse_tolerance_mm("0.5 mm"), Some(0.5));
        assert_eq!(parse_tolerance_mm("0.5"), Some(0.5));
        assert_eq!(parse_tolerance_mm("0.500 mm"), Some(0.5));
        assert_eq!(parse_tolerance_mm("0.01 mm"), Some(0.01));
        assert_eq!(parse_tolerance_mm("0"), None);
        assert_eq!(parse_tolerance_mm("-0.1 mm"), None);
        assert_eq!(format_mm_tolerance(0.5), "0.500 mm");
        assert_eq!(format_mm_tolerance(0.01), "0.010 mm");
    }
}
