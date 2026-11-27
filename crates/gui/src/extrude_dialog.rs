// SPDX-License-Identifier: GPL-2.0-only
//! Desktop extrusion draft. The document changes only when the owner accepts Apply.
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiExtrudeDirection, GuiShellAction};
use imgui::Ui;
use oblikovati_model::document_sketch::DocumentSketchId;
#[path = "extrude_direction.rs"]
mod direction;
#[path = "extrude_distance_editor.rs"]
mod distance_editor;
pub(crate) use distance_editor::ExtrudePanelControls;
#[path = "extrude_output.rs"]
mod output;
pub(crate) use output::single_body;
#[path = "extrude_panel.rs"]
mod panel;
#[path = "extrude_pending.rs"]
mod pending;
#[path = "extrude_presets.rs"]
mod presets;
#[path = "extrude_profile_pick.rs"]
mod profile_pick;
#[path = "extrude_reference.rs"]
mod reference;
#[path = "extrude_selector.rs"]
mod selector;
#[path = "extrude_taper.rs"]
mod taper;
const DEFAULT_DISTANCE: &str = "10.00 mm";

#[derive(Default)]
pub(crate) struct ExtrudeDialog {
    pub requested: bool,
    start_without_sketch: bool,
    document: Option<u64>,
    dock_state: crate::dockable_lifecycle::DockableState,
    sketch: Option<DocumentSketchId>,
    profile: i32,
    profile_pick_sketch: Option<DocumentSketchId>,
    release_sketch_on_apply: bool,
    distance: String,
    distance_editor: distance_editor::DistanceEditor,
    second_distance: String,
    taper: Option<taper::TaperDraft>,
    direction: GuiExtrudeDirection,
    direction_style: selector::SelectorStyle,
    operation_style: selector::SelectorStyle,
    manual_boolean: bool,
    presets: presets::ExtrudePresets,
    reference: reference::ReferenceDraft,
    operation: Option<oblikovati_model::feature::sketch_prism::PrismBoolean>,
    body_name: String,
    previewed: bool,
    handle: crate::extrude_handle::DistanceHandle,
    preview_hidden: bool,
    size: panel::PropertiesSize,
    attempted: Option<crate::GuiExtrudePreview>,
    pending: pending::PendingExtrude,
    submitted: Option<crate::GuiExtrudePreview>,
    commit_error: Option<String>,
    edit_feature: Option<oblikovati_model::feature::FeatureId>,
    submitted_edit: Option<crate::GuiExtrudeEditRequest>,
}
pub(crate) fn start_action(state: &GuiCommandState) -> GuiShellAction {
    if state
        .browser
        .as_ref()
        .is_some_and(|browser| browser.sketches.is_empty())
    {
        GuiShellAction::ChooseSketchPlaneForExtrude
    } else {
        GuiShellAction::OpenExtrude
    }
}
impl DockableWindow for ExtrudeDialog {}
impl ExtrudeDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Extrude", &self.dock_state)
    }
    fn set_document(&mut self, document: Option<u64>) {
        if document.is_none() || self.document != document {
            self.distance_editor = Default::default();
            self.handle = Default::default();
            self.taper = None;
            self.operation = None;
            self.preview_hidden = false;
            self.profile_pick_sketch = None;
            self.size = Default::default();
        }
        self.document = document;
        self.dock_state.set_open(document.is_some());
    }
    pub(crate) fn open_edit(
        &mut self,
        state: &GuiCommandState,
        feature: oblikovati_model::feature::FeatureId,
    ) {
        let Some(seed) = state
            .browser
            .as_ref()
            .and_then(|browser| browser.extrusion_edits.get(&feature))
        else {
            return;
        };
        self.set_document(state.document);
        self.presets.clear_selection();
        self.sketch = Some(seed.sketch);
        self.profile_pick_sketch = None;
        self.profile = 0;
        self.distance = seed.distance.clone();
        self.distance_editor = Default::default();
        self.second_distance = seed.second_distance.clone().unwrap_or_default();
        self.taper = seed.taper.clone().map(taper::TaperDraft::new);
        self.direction = seed.direction;
        self.reference = Default::default();
        self.operation = None;
        self.edit_feature = Some(feature);
        self.attempted = None;
        self.commit_error = None;
    }
    pub(crate) fn needs_frame(&self) -> bool {
        self.requested || self.size.needs_frame()
    }
    pub(crate) fn wait_for_sketch(&mut self, document: Option<u64>) {
        self.pending.start(document);
    }
    fn synchronize(&mut self, state: &GuiCommandState) {
        if self.requested {
            self.open_requested_draft(state);
        }
        if self.document != state.document
            || !state.extrude_available
            || state.authoring != crate::GuiAuthoringContext::Part
        {
            self.set_document(None);
            self.edit_feature = None;
        }
    }
    fn initialize_fresh_draft(&mut self, state: &GuiCommandState) {
        self.sketch = state.browser.as_ref().and_then(|browser| {
            browser
                .selected_sketches
                .first()
                .copied()
                .or_else(|| browser.sketches.first().map(|row| row.id))
        });
        self.distance = DEFAULT_DISTANCE.into();
        self.second_distance = "5 mm".into();
        self.direction = GuiExtrudeDirection::Positive;
        self.reference = Default::default();
        self.seed_preset();
    }

    pub(crate) fn sync_before_docking(&mut self, state: &GuiCommandState) {
        if !self.requested {
            self.synchronize(state);
        }
    }
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        plane_active: bool,
        actions: &mut Vec<GuiShellAction>,
        mut controls: ExtrudePanelControls<'_>,
    ) {
        self.presets.observe_command_state(state);
        if let Some(sketch) = self.pending.observe(state, plane_active, actions) {
            self.set_document(state.document);
            self.sketch = Some(sketch);
            self.body_name.clear();
            self.profile = 0;
            self.distance = DEFAULT_DISTANCE.into();
            self.second_distance = "5 mm".into();
            self.direction = GuiExtrudeDirection::Positive;
            self.reference = Default::default();
            self.attempted = None;
            self.edit_feature = None;
            self.seed_preset();
        }
        if self.requested && self.previewed {
            actions.push(GuiShellAction::PreviewExtrude(None));
            self.previewed = false;
        }
        self.synchronize(state);
        self.observe_profile_pick(state);
        if self.document.is_none() && self.previewed {
            actions.push(GuiShellAction::PreviewExtrude(None));
            self.previewed = false;
        }
        let Some(document) = self.document else {
            controls.parameters.close_tolerance_inspection();
            return;
        };
        if let Some(reference) = controls.browser_plane.take() {
            self.reference.pick(reference);
        }
        if self.update_distance_handle(ui, state, controls.area) {
            controls.parameters.close_tolerance_inspection();
        }
        self.size.fit(ui, controls.area);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Extrude", &mut open)
            .scroll_bar(false)
            .scrollable(false)
            .build(|| {
                crate::theme::floating_property_shadow(ui);
                let style = ui.clone_style();
                let chrome = ui.cursor_pos()[1] + style.window_padding[1] + style.item_spacing[1];
                let mut properties = 0.;
                let mut edited = false;
                {
                    let region = crate::widget_abi::child_region(
                        ui,
                        "##extrude-properties",
                        [0., -panel::FOOTER_HEIGHT],
                        0.,
                        imgui::WindowFlags::empty(),
                    );
                    if region.visible {
                        let start = ui.cursor_pos()[1];
                        edited = self.properties(ui, state, document, &mut controls, actions);
                        properties = ui.cursor_pos()[1] - start;
                    }
                }
                self.size.measure(chrome, properties);
                // Child input still owns the draft. Popups keep their own Escape key.
                let shortcuts = !controls.popup_was_open
                    && !crate::widget_abi::any_popup_open(ui)
                    && ui.is_window_focused_with_flags(imgui::WindowFocusedFlags::CHILD_WINDOWS);
                self.controls(ui, state, document, (edited, shortcuts), actions);
            });
        if !open {
            self.close(actions);
        }
        if self.document.is_none() || !self.reference.numeric() {
            controls.parameters.close_tolerance_inspection();
        }
    }
    fn properties(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        document: u64,
        controls: &mut ExtrudePanelControls<'_>,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        let expanded = imgui::TreeNodeFlags::DEFAULT_OPEN;
        self.draw_advanced_settings(ui, controls.icons);
        self.breadcrumb(ui, state);
        self.preview_control(ui, controls.icons, state.extrude_preview_active, actions);
        let mut edited = self.draw_presets(ui, state, controls.icons);
        edited |=
            ui.collapsing_header("Input Geometry", expanded) && self.geometry_controls(ui, state);
        if ui.collapsing_header("Behavior", expanded) {
            edited |= self.draw_extent(ui, state, controls.icons, controls.parameters);
            if self.reference.numeric()
                && let Some(taper) = &mut self.taper
            {
                edited |= taper.draw(ui);
            }
        }
        edited |= self.draw_output(ui, state, controls.icons);
        if edited {
            self.commit_error = None;
            if self.previewed {
                actions.push(GuiShellAction::PreviewExtrude(None));
                self.previewed = false;
            }
        }
        self.refresh_preview(state, document, edited, actions);
        self.draw_validation(ui, state, document, actions);
        edited
    }
    fn draw_validation(
        &self,
        ui: &Ui,
        state: &GuiCommandState,
        document: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let validation = self.current_distance_validation(state, document);
        if self.reference.numeric() && !self.distance.trim().is_empty() && validation.is_none() {
            actions.push(GuiShellAction::ValidateExtrudeDistance {
                document,
                distance: self.distance.clone(),
                second_distance: self.active_second_distance().map(str::to_owned),
            });
        }
        if let Some(error) = validation.and_then(|result| result.error.as_ref()) {
            ui.text_colored([1.0, 0.25, 0.25, 1.0], format!("Invalid distance: {error}"));
        }
        if let Some(error) = &self.commit_error {
            ui.text_colored([1.0, 0.25, 0.25, 1.0], format!("Extrusion failed: {error}"));
        }
    }
    fn refresh_preview(
        &mut self,
        state: &GuiCommandState,
        document: u64,
        edited: bool,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let request = self.valid_preview_request(state, document, edited);
        if self.preview_hidden || request == self.attempted {
            return;
        }
        if self.previewed {
            actions.push(GuiShellAction::PreviewExtrude(None));
            self.previewed = false;
        }
        self.attempted = request.clone();
        if let Some(request) = request {
            if let Some(edit) = self.edit_request(&request) {
                actions.push(GuiShellAction::PreviewExtrudeEdit(edit));
            } else {
                actions.push(GuiShellAction::PreviewExtrude(Some(request)));
            }
            self.previewed = true;
        }
    }
    fn valid_preview_request(
        &self,
        state: &GuiCommandState,
        document: u64,
        edited: bool,
    ) -> Option<crate::GuiExtrudePreview> {
        if !self.reference.numeric() {
            // A completed reference pick is ready in this frame. Numeric edits below
            // wait for fresh validation, which schedules another frame.
            return Some(crate::GuiExtrudePreview {
                generation: Some(state.document_generation?),
                document,
                sketch: self.sketch?,
                profile: i64::from(self.profile),
                distance: String::new(),
                second_distance: None,
                direction: self.reference_direction(),
                operation: self.reference_operation(),
                reference_extent: Some(self.reference.request()?),
            });
        }
        if edited
            || self.taper.as_ref().is_some_and(|taper| {
                taper.is_empty() || (taper.change().is_some() && self.commit_error.is_some())
            })
        {
            return None;
        }
        let validation = self.current_distance_validation(state, document)?;
        let reference_extent = self.reference.request();
        if validation.error.is_some()
            || (self.reference.needs_start_plane() && reference_extent.is_none())
        {
            return None;
        }
        Some(crate::GuiExtrudePreview {
            generation: Some(validation.generation),
            document,
            sketch: self.sketch?,
            profile: i64::from(self.profile),
            distance: self.distance.clone(),
            second_distance: self.active_second_distance().map(str::to_owned),
            direction: self.direction,
            operation: self.operation,
            reference_extent,
        })
    }
    fn sketch_choice(&mut self, ui: &Ui, state: &GuiCommandState) -> bool {
        let Some(browser) = state.browser.as_ref() else {
            return false;
        };
        let shown = browser
            .sketches
            .iter()
            .find(|row| Some(row.id) == self.sketch)
            .map_or("Choose a sketch", |row| row.name.as_str());
        crate::theme::property_label(ui, "Sketch");
        let Some(_list) = ui.begin_combo("##extrude-sketch", shown) else {
            return false;
        };
        let previous = self.sketch;
        for row in &browser.sketches {
            if ui
                .selectable_config(&row.name)
                .selected(Some(row.id) == self.sketch)
                .build()
            {
                self.sketch = Some(row.id);
            }
        }
        self.sketch != previous
    }
    fn edit_request(
        &self,
        request: &crate::GuiExtrudePreview,
    ) -> Option<crate::GuiExtrudeEditRequest> {
        Some(crate::GuiExtrudeEditRequest {
            generation: request.generation,
            document: request.document,
            feature: self.edit_feature?,
            distance: request.distance.clone(),
            second_distance: request.second_distance.clone(),
            taper: self.taper.as_ref().and_then(taper::TaperDraft::change),
        })
    }
    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.previewed {
            actions.push(GuiShellAction::PreviewExtrude(None));
            self.previewed = false;
        }
        self.requested = false;
        self.start_without_sketch = false;
        self.pending = Default::default();
        self.set_document(None);
        self.edit_feature = None;
        self.attempted = None;
        self.submitted = None;
        self.submitted_edit = None;
    }
    fn current_distance_validation<'a>(
        &self,
        state: &'a GuiCommandState,
        document: u64,
    ) -> Option<&'a crate::GuiExtrudeDistanceValidation> {
        if !self.reference.numeric() {
            return None;
        }
        state.extrude_distance_validation.as_ref().filter(|result| {
            result.document == document
                && state.document_generation == Some(result.generation)
                && result.distance == self.distance
                && result.second_distance.as_deref() == self.active_second_distance()
        })
    }
    fn active_second_distance(&self) -> Option<&str> {
        (self.direction == GuiExtrudeDirection::Asymmetric).then_some(self.second_distance.as_str())
    }
    pub(crate) fn reject(&mut self, request: &crate::GuiExtrudePreview, cause: &str) {
        if self.submitted.as_ref() != Some(request) {
            return;
        }
        self.presets.reject_submission();
        self.requested = false;
        self.start_without_sketch = false;
        self.set_document(Some(request.document));
        self.operation = request.operation;
        self.attempted = None;
        self.commit_error = Some(cause.into());
    }
    pub(crate) fn reject_edit(&mut self, request: &crate::GuiExtrudeEditRequest, cause: &str) {
        if self.submitted_edit.as_ref() != Some(request) {
            if self.document == Some(request.document)
                && self.edit_feature == Some(request.feature)
                && self.distance == request.distance
                && request.taper.is_some()
                && self.taper.as_ref().and_then(taper::TaperDraft::change) == request.taper
            {
                self.commit_error = Some(cause.into());
            }
            return;
        }
        let taper = self.taper.take();
        self.set_document(Some(request.document));
        self.taper = taper;
        self.edit_feature = Some(request.feature);
        self.commit_error = Some(cause.into());
    }
}

#[cfg(test)]
#[path = "extrude_dialog_tests.rs"]
mod tests;
