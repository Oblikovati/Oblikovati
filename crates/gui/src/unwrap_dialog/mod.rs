// SPDX-License-Identifier: GPL-2.0-only
//! Dockable Unwrap Properties panel built on the v1.1 Feature Properties shell.
mod types;
pub use types::{
    GuiUnwrapAction, GuiUnwrapEdgeMode, GuiUnwrapEditSeed, GuiUnwrapFacePick, GuiUnwrapPreview,
    GuiUnwrapPreviewState, GuiUnwrapRequest,
};

use crate::{
    GuiCommandState, GuiShellAction,
    dockable_window::DockableWindow,
    feature_properties::{draw_properties_footer, draw_properties_header},
    icons::GuiRibbonIcons,
    widget_abi,
};
use imgui::Ui;
use oblikovati_model::feature::{FeatureId, unwrap::UnwrapAlignment};

#[derive(Default)]
pub(crate) struct UnwrapDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    submitted: Option<GuiUnwrapRequest>,
    feature: Option<FeatureId>,
    target_body: Option<String>,
    faces: Vec<Vec<u8>>,
    alignment_vertex: Option<Vec<u8>>,
    observed_face: Option<Vec<u8>>,
    applied_pick: Option<(Vec<u8>, bool)>,
    auto_face_chain: bool,
    merge_result_body: bool,
    alignment: UnwrapAlignment,
    linear_result_edges: Vec<Vec<u8>>,
    rigid_result_edges: Vec<Vec<u8>>,
    seam_edges: Vec<Vec<u8>>,
    edge_mode: GuiUnwrapEdgeMode,
    preview_hidden: bool,
    presets_shown: bool,
    popup_was_open: bool,
}

impl DockableWindow for UnwrapDialog {}

impl UnwrapDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Unwrap", &self.dock)
    }

    pub(crate) fn status_prompt(&self) -> Option<&'static str> {
        self.document?;
        self.edge_mode.status_prompt()
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, feature: Option<FeatureId>) {
        let seed = match feature {
            Some(id) => {
                let Some(seed) = state
                    .browser
                    .as_ref()
                    .and_then(|browser| browser.unwrap_edits.get(&id))
                else {
                    return;
                };
                seed.clone()
            }
            None => {
                let mut seed = GuiUnwrapEditSeed::default();
                if let Some(browser) = &state.browser
                    && browser.bodies.len() == 1
                {
                    seed.target_body = Some(browser.bodies[0].key.clone());
                }
                seed
            }
        };
        self.feature = feature;
        self.target_body = seed.target_body;
        self.faces = seed.faces;
        self.alignment_vertex = seed.alignment_vertex;
        self.observed_face = state.selected_model_face.clone();
        self.applied_pick = None;
        self.auto_face_chain = seed.auto_face_chain;
        self.merge_result_body = seed.merge_result_body;
        self.alignment = seed.alignment;
        self.linear_result_edges = seed.linear_result_edges;
        self.rigid_result_edges = seed.rigid_result_edges;
        self.seam_edges = seed.seam_edges;
        self.edge_mode = GuiUnwrapEdgeMode::None;
        self.preview_hidden = false;
        self.popup_was_open = false;
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.submitted = None;
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.submitted = None;
            self.edge_mode = GuiUnwrapEdgeMode::None;
            self.dock.set_open(false);
            actions.push(GuiShellAction::Unwrap(GuiUnwrapAction::Clear));
        }
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(document) = self.document else {
            return;
        };
        if self.document != state.document || !GuiUnwrapAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        let Some(generation) = state.document_generation else {
            return;
        };
        if let Some(pick) = state.unwrap_pick.as_ref().filter(|p| {
            p.document == document
                && p.generation == generation
                && self.applied_pick.as_ref() != Some(&(p.seed_face.clone(), p.auto_face_chain))
        }) {
            self.applied_pick = Some((pick.seed_face.clone(), pick.auto_face_chain));
            self.target_body = Some(pick.target_body.clone());
            self.faces = pick.faces.clone();
            self.alignment_vertex = pick.alignment_vertex.clone();
        }
        if state.selected_model_face != self.observed_face {
            self.observed_face = state.selected_model_face.clone();
            if self.edge_mode == GuiUnwrapEdgeMode::None
                && let Some(face) = state.selected_model_face.clone()
            {
                actions.push(GuiShellAction::Unwrap(GuiUnwrapAction::PickFace {
                    document,
                    generation,
                    face,
                    auto_face_chain: self.auto_face_chain,
                }));
            }
        }
        let Some(request) = self.request(state) else {
            return;
        };
        if self.submitted.as_ref() != Some(&request) {
            self.submitted = Some(request.clone());
            actions.push(GuiShellAction::Unwrap(GuiUnwrapAction::Preview(Box::new(
                request,
            ))));
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
            return;
        }
        if self.document != state.document || !GuiUnwrapAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        if !self.popup_was_open && ui.is_key_pressed_no_repeat(imgui::Key::Escape) {
            self.close(actions);
            return;
        }
        let mut keep_open = true;
        let mut cancel = false;
        let mut commit = false;
        let request = self.request(state);
        let ready = request
            .as_ref()
            .is_some_and(|req| GuiUnwrapAction::Commit(Box::new(req.clone())).enabled(state));
        let preview_error = request.as_ref().and_then(|req| {
            state
                .unwrap_preview
                .as_ref()
                .filter(|preview| preview.request == *req)
                .and_then(|preview| match &preview.result {
                    GuiUnwrapPreview::Failed(msg) => Some(msg.as_str()),
                    _ => None,
                })
        });
        let title = match self.feature {
            Some(id) => state
                .browser
                .as_ref()
                .and_then(|browser| browser.features.iter().find(|row| row.id == id))
                .map(|row| format!("Unwrap: {}", row.name))
                .unwrap_or_else(|| "Unwrap".into()),
            None => "Unwrap".into(),
        };
        widget_abi::fit_viewport_properties(ui, "Unwrap", area, [316., 456.]);
        let _border = crate::theme::property_window_border(ui);
        Self::dockable_window(ui, "Unwrap", &mut keep_open).build(|| {
            crate::theme::floating_property_shadow(ui);
            let header = draw_properties_header(
                ui,
                icons,
                &title,
                "##unwrap-advanced-settings",
                &mut self.presets_shown,
                Some((self.preview_hidden, ready || self.preview_hidden)),
                |_| {},
            );
            if header.toggle_preview {
                self.preview_hidden = !self.preview_hidden;
            }
            let footer_height = 44.;
            let body_height = (ui.content_region_avail()[1] - footer_height).max(160.);
            {
                let body = widget_abi::scroll_region(ui, "##unwrap-body", [0., body_height]);
                if body.visible {
                    self.draw_input_geometry(ui, state, actions);
                    ui.spacing();
                    self.draw_behavior(ui);
                    if let Some(error) = preview_error {
                        ui.spacing();
                        ui.text_wrapped(error);
                    }
                }
            }
            let footer = draw_properties_footer(ui, ready);
            commit = footer.ok;
            cancel = footer.cancel;
        });
        self.popup_was_open = widget_abi::any_popup_open(ui);
        if commit && let Some(req) = request {
            actions.push(GuiShellAction::Unwrap(GuiUnwrapAction::Commit(Box::new(
                req,
            ))));
            self.document = None;
            self.submitted = None;
            self.edge_mode = GuiUnwrapEdgeMode::None;
            self.dock.set_open(false);
        } else if cancel || !keep_open {
            self.close(actions);
        }
    }

    fn draw_input_geometry(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if !ui.collapsing_header("Input Geometry", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        let faces_label = match self.faces.len() {
            0 => "Select Faces".to_string(),
            1 => "1 Face".to_string(),
            n => format!("{n} Faces"),
        };
        crate::theme::property_label(ui, "Faces");
        if widget_abi::button(ui, &format!("{faces_label}##unwrap-faces"), [112., 24.]) {
            self.edge_mode = GuiUnwrapEdgeMode::None;
            if self.faces.is_empty()
                && let (Some(document), Some(generation), Some(face)) = (
                    self.document,
                    state.document_generation,
                    state.selected_model_face.clone(),
                )
            {
                actions.push(GuiShellAction::Unwrap(GuiUnwrapAction::PickFace {
                    document,
                    generation,
                    face,
                    auto_face_chain: self.auto_face_chain,
                }));
            }
        }
        if !self.faces.is_empty() {
            ui.same_line_with_spacing(0., 6.);
            if widget_abi::button(ui, "Clear##unwrap-clear-faces", [50., 24.]) {
                self.faces.clear();
                self.alignment_vertex = None;
                self.applied_pick = None;
                self.observed_face = None;
                self.submitted = None;
                self.edge_mode = GuiUnwrapEdgeMode::None;
                actions.push(GuiShellAction::Unwrap(GuiUnwrapAction::Clear));
            }
        }
        if !self.faces.is_empty() {
            let vertex_label = if self.alignment_vertex.is_some() {
                "1 Vertex"
            } else {
                "Select Vertex"
            };
            crate::theme::property_label(ui, "Alignment Vertex");
            let _ = widget_abi::button(ui, &format!("{vertex_label}##unwrap-vertex"), [128., 24.]);
            self.draw_edge_selector(
                ui,
                "Linear Result",
                GuiUnwrapEdgeMode::LinearResult,
                self.linear_result_edges.len(),
            );
            self.draw_edge_selector(
                ui,
                "Rigid Result",
                GuiUnwrapEdgeMode::RigidResult,
                self.rigid_result_edges.len(),
            );
            self.draw_edge_selector(
                ui,
                "Seam Edges",
                GuiUnwrapEdgeMode::SeamEdges,
                self.seam_edges.len(),
            );
        }
    }

    fn draw_edge_selector(&mut self, ui: &Ui, label: &str, mode: GuiUnwrapEdgeMode, count: usize) {
        let caption = match count {
            0 => "Select Edges".to_string(),
            1 => "1 Edge".to_string(),
            n => format!("{n} Edges"),
        };
        crate::theme::property_label(ui, label);
        let active = self.edge_mode == mode;
        let _hl = active.then(|| {
            ui.push_style_color(
                imgui::StyleColor::Button,
                ui.style_color(imgui::StyleColor::ButtonActive),
            )
        });
        if widget_abi::button(ui, &format!("{caption}##unwrap-{label}"), [128., 24.]) {
            self.edge_mode = if active {
                GuiUnwrapEdgeMode::None
            } else {
                mode
            };
        }
    }

    fn draw_behavior(&mut self, ui: &Ui) {
        if !ui.collapsing_header("Behavior", imgui::TreeNodeFlags::DEFAULT_OPEN) {
            return;
        }
        ui.checkbox("Auto Face Chain", &mut self.auto_face_chain);
        ui.checkbox("Merge Result Body", &mut self.merge_result_body);
        crate::theme::property_label(ui, "Alignment");
        for (index, option) in UnwrapAlignment::ALL.into_iter().enumerate() {
            if index > 0 {
                ui.same_line_with_spacing(0., 4.);
            }
            let active = self.alignment == option;
            let _hl = active.then(|| {
                ui.push_style_color(
                    imgui::StyleColor::Button,
                    ui.style_color(imgui::StyleColor::ButtonActive),
                )
            });
            if widget_abi::button(
                ui,
                &format!("{}##unwrap-align-{index}", option.label()),
                [36., 24.],
            ) {
                self.alignment = option;
            }
        }
    }

    fn request(&self, state: &GuiCommandState) -> Option<GuiUnwrapRequest> {
        Some(GuiUnwrapRequest {
            document: self.document?,
            generation: state.document_generation?,
            feature: self.feature,
            target_body: self.target_body.clone(),
            faces: self.faces.clone(),
            alignment_vertex: self.alignment_vertex.clone(),
            auto_face_chain: self.auto_face_chain,
            merge_result_body: self.merge_result_body,
            alignment: self.alignment,
            linear_result_edges: self.linear_result_edges.clone(),
            rigid_result_edges: self.rigid_result_edges.clone(),
            seam_edges: self.seam_edges.clone(),
            preview_hidden: self.preview_hidden,
        })
    }
}
