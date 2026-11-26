// SPDX-License-Identifier: GPL-2.0-only
//! Properties and lifecycle for a Sweep of a closed profile along a path.
use crate::{
    GuiAuthoringContext, GuiCommandState, GuiShellAction, dockable_window::DockableWindow,
};
use imgui::Ui;
#[path = "sweep_orientation.rs"]
mod orientation;
#[path = "sweep_output.rs"]
mod output;
#[path = "sweep_presets.rs"]
mod presets;
use crate::icons::GuiRibbonIcons;
use oblikovati_model::feature::{FeatureId, sketch_prism::SketchPrismSelection};
pub use output::GuiSweepOutput;
pub use presets::GuiSweepPresetStore;

/// Sweep orientation choices available in Properties.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum GuiSweepOrientation {
    #[default]
    FollowPath,
    Fixed,
    Guide,
}

/// The section retained or selected for the current feature edit.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub enum GuiSweepProfile {
    #[default]
    Saved,
    Empty,
    Picked(SketchPrismSelection),
}

/// Authored Sweep options shown when editing an existing feature.
#[derive(Clone, Debug, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct GuiSweepEditSeed {
    pub taper: String,
    pub twist: String,
    pub orientation: GuiSweepOrientation,
    #[serde(skip)]
    pub output: GuiSweepOutput,
}

/// Document and generation whose Sweep inputs may be previewed or accepted.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSweepRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub profile: GuiSweepProfile,
    pub path_cleared: bool,
    pub output: GuiSweepOutput,
    pub orientation: GuiSweepOrientation,
    pub taper: String,
    pub twist: String,
}

/// The geometry owner's response to a Sweep preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSweepPreview {
    Ready,
    MissingInputs,
    MissingProfile,
    MissingPath,
    MissingGuide,
    UnchangedCut(String),
    Failed(String),
}

/// The exact draft and its validation result.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSweepPreviewState {
    pub request: GuiSweepRequest,
    pub result: GuiSweepPreview,
}

/// Actions emitted by the Sweep ribbon and Properties window.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSweepAction {
    Open,
    OpenEdit(FeatureId),
    Preview(GuiSweepRequest),
    Clear,
    Commit(GuiSweepRequest),
}

impl GuiSweepAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: &GuiSweepRequest| {
            can_start
                && state.document == Some(request.document)
                && state.document_generation == Some(request.generation)
        };
        match self {
            Self::Open => can_start,
            Self::OpenEdit(feature) => {
                can_start
                    && state
                        .browser
                        .as_ref()
                        .is_some_and(|browser| browser.sweep_edits.contains_key(feature))
            }
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request)
                    && state.sweep_preview.as_ref().is_some_and(|preview| {
                        preview.request == *request && preview.result == GuiSweepPreview::Ready
                    })
            }
        }
    }
}

const TIP: &str = "Sweep needs a closed profile sketch and a path sketch. Make both sketches visible to select an unambiguous pair.";
const RECENT_TWIST_LIMIT: usize = 8;

#[derive(Default)]
pub(crate) struct SweepDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    attempted: Option<GuiSweepRequest>,
    submitted: Option<GuiSweepRequest>,
    feature: Option<FeatureId>,
    profile: GuiSweepProfile,
    path_cleared: bool,
    output: GuiSweepOutput,
    orientation: GuiSweepOrientation,
    selector: orientation::OrientationSelector,
    presets: presets::SweepPresets,
    taper: String,
    twist: String,
    initial_twist: String,
    recent_twist: Vec<String>,
    optimize_single_selection: bool,
    show_help: bool,
}
impl DockableWindow for SweepDialog {}
impl SweepDialog {
    pub(crate) fn needs_frame(&self) -> bool {
        self.document.is_some() && self.presets.needs_frame
    }

    pub(crate) fn set_preset_store(
        &mut self,
        store: Box<dyn GuiSweepPresetStore>,
    ) -> Result<(), String> {
        self.presets.set_store(store)
    }

    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>("Sweep", &self.dock)
    }

    pub(super) fn open(&mut self, state: &GuiCommandState, feature: Option<FeatureId>) {
        let seed = match feature {
            Some(id) => {
                let Some(taper) = state
                    .browser
                    .as_ref()
                    .and_then(|browser| browser.sweep_edits.get(&id))
                else {
                    return;
                };
                taper.clone()
            }
            None => GuiSweepEditSeed {
                taper: "0 deg".into(),
                twist: "0 deg".into(),
                ..Default::default()
            },
        };
        self.feature = feature;
        self.profile = GuiSweepProfile::Saved;
        self.path_cleared = false;
        self.output = seed.output;
        self.taper = seed.taper;
        self.twist = seed.twist;
        self.initial_twist.clone_from(&self.twist);
        self.optimize_single_selection = false;
        self.presets.reset();
        self.orientation = seed.orientation;
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.show_help = false;
    }

    pub(super) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.submitted = None;
            self.dock.set_open(false);
            actions.push(GuiShellAction::Sweep(GuiSweepAction::Clear));
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
        if self.document != state.document || !GuiSweepAction::Open.enabled(state) {
            self.close(actions);
            return;
        }
        // The owner clears the exact preview after accepting the request, including a no-op edit.
        if state.sweep_preview.is_none()
            && let Some(request) = self.submitted.take()
        {
            self.remember_twist(&request);
            self.close(actions);
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
        let (Some(document), Some(generation)) = (self.document, state.document_generation) else {
            return;
        };
        let height = if self.show_help { 471. } else { 397. };
        crate::widget_abi::fit_viewport_properties(ui, "Sweep", area, [300., height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Sweep", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            self.contents(ui, state, (document, generation), icons, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        (document, generation): (u64, u64),
        icons: &GuiRibbonIcons,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let tip = state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.sketches.is_empty());
        let preview = state.sweep_preview.as_ref();
        self.adopt_profile_pick(preview);
        if self.feature.is_some()
            || preview.is_some_and(|state| {
                matches!(
                    state.result,
                    GuiSweepPreview::Ready
                        | GuiSweepPreview::MissingGuide
                        | GuiSweepPreview::Failed(_)
                )
            })
        {
            self.profile_selector(ui);
            crate::theme::property_label(ui, "Path");
            if self.path_cleared {
                ui.text("Select Curve or Edge");
                if crate::widget_abi::button(ui, "Restore Path", [0., 0.]) {
                    self.path_cleared = false;
                }
            } else {
                ui.text("1 Curve");
                if self.feature.is_some() && crate::widget_abi::button(ui, "Clear Path", [0., 0.]) {
                    self.path_cleared = true;
                }
            }
        }
        if !tip {
            self.selector.draw(ui, &mut self.orientation, icons);
        }
        let follows_path = self.orientation == GuiSweepOrientation::FollowPath;
        if !tip && follows_path {
            crate::theme::property_label(ui, "Taper");
            let _width = ui.push_item_width(-1.);
            ui.input_text("##sweep-taper", &mut self.taper).build();
            self.twist_field(ui);
        }
        if !tip {
            self.draw_operation(ui, state);
        }
        let mut request = self.request(document, generation);
        if !tip {
            let certified = Self::certified_preset(state, &request);
            if let Some(fields) = self.presets.draw(ui, certified) {
                self.orientation = fields.orientation;
                self.taper = fields.taper;
                self.twist = fields.twist;
                request = self.request(document, generation);
            }
        }
        if self.attempted.as_ref() != Some(&request) {
            self.attempted = Some(request.clone());
            actions.push(GuiShellAction::Sweep(GuiSweepAction::Preview(
                request.clone(),
            )));
        }
        match preview
            .filter(|preview| preview.request == request)
            .map(|preview| &preview.result)
        {
            Some(GuiSweepPreview::MissingInputs) => ui.text_wrapped(TIP),
            Some(GuiSweepPreview::MissingGuide) => ui.text_wrapped("Select Rail or Surfaces"),
            Some(GuiSweepPreview::Failed(cause)) => {
                let _error = ui.push_style_color(imgui::StyleColor::Text, [1., 0.25, 0.25, 1.]);
                ui.text_wrapped(cause);
            }
            None if tip => ui.text_wrapped(TIP),
            _ => {}
        }
        let ready = GuiSweepAction::Commit(request.clone()).enabled(state);
        let cut_warning = output::cut_warning(preview, &request);
        if !tip && ui.collapsing_header("Advanced Properties", imgui::TreeNodeFlags::empty()) {
            ui.checkbox(
                "Optimize for Single Selection",
                &mut self.optimize_single_selection,
            );
        }
        if self.show_help {
            ui.text_wrapped("Create a closed section in one sketch and a straight path in another. Start the path on the profile plane and keep both sketches visible.");
        }
        ui.separator();
        if crate::theme::property_accept(ui, tip || ready || cut_warning.is_some()) {
            if cut_warning.is_some() {
                ui.open_popup(output::EDIT_WARNING);
            } else if ready {
                self.submitted = Some(request.clone());
                actions.push(GuiShellAction::Sweep(GuiSweepAction::Commit(request)));
            } else {
                self.close(actions);
            }
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel", [64., 28.])
            || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
        {
            self.close(actions);
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Help", [58., 28.]) {
            self.show_help = !self.show_help;
        }
        if Self::draw_cut_warning(ui, cut_warning) {
            self.close(actions);
        }
    }

    fn profile_selector(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Profiles");
        if self.profile == GuiSweepProfile::Empty {
            ui.text("Select Profiles");
            return;
        }
        ui.text("1 Profile");
        if self.feature.is_some() && crate::widget_abi::button(ui, "Clear Profile", [0., 0.]) {
            self.profile = GuiSweepProfile::Empty;
        }
    }

    fn twist_field(&mut self, ui: &Ui) {
        crate::theme::property_label(ui, "Twist");
        if !self.recent_twist.is_empty() {
            ui.set_next_item_width(-68.);
        }
        ui.input_text("##sweep-twist", &mut self.twist).build();
        if self.recent_twist.is_empty() {
            return;
        }
        ui.same_line();
        if crate::widget_abi::button(ui, "Recent##sweep-twist-recent", [60., 0.]) {
            ui.open_popup("##sweep-recent-expressions");
        }
        ui.popup("##sweep-recent-expressions", || {
            for expression in &self.recent_twist {
                if ui.selectable(expression) {
                    self.twist.clone_from(expression);
                }
            }
        });
    }

    fn remember_twist(&mut self, request: &GuiSweepRequest) {
        let expression = request.twist.trim();
        if request.orientation != GuiSweepOrientation::FollowPath
            || expression.is_empty()
            || expression == self.initial_twist.trim()
        {
            return;
        }
        self.recent_twist.retain(|value| value != expression);
        self.recent_twist.insert(0, expression.into());
        self.recent_twist.truncate(RECENT_TWIST_LIMIT);
    }

    fn request(&self, document: u64, generation: u64) -> GuiSweepRequest {
        let follows_path = self.orientation == GuiSweepOrientation::FollowPath;
        GuiSweepRequest {
            document,
            generation,
            feature: self.feature,
            profile: self.profile.clone(),
            path_cleared: self.path_cleared,
            output: self.output,
            orientation: self.orientation,
            taper: if follows_path {
                self.taper.clone()
            } else {
                String::new()
            },
            twist: if follows_path {
                self.twist.clone()
            } else {
                String::new()
            },
        }
    }

    fn certified_preset(
        state: &GuiCommandState,
        request: &GuiSweepRequest,
    ) -> Option<GuiSweepEditSeed> {
        state
            .sweep_preview
            .as_ref()
            .filter(|preview| {
                &preview.request == request && preview.result == GuiSweepPreview::Ready
            })
            .map(|_| GuiSweepEditSeed {
                taper: preset_angle(state, &request.taper),
                twist: preset_angle(state, &request.twist),
                orientation: request.orientation,
                output: GuiSweepOutput::NewSolid,
            })
    }

    fn adopt_profile_pick(&mut self, preview: Option<&GuiSweepPreviewState>) {
        let Some(preview) = preview.filter(|_| self.profile == GuiSweepProfile::Empty) else {
            return;
        };
        let mut prior = preview.request.clone();
        let profile = std::mem::replace(&mut prior.profile, GuiSweepProfile::Empty);
        if matches!(profile, GuiSweepProfile::Picked(_)) && self.attempted.as_ref() == Some(&prior)
        {
            self.profile = profile;
            self.attempted = Some(preview.request.clone());
        }
    }
}

fn preset_angle(state: &GuiCommandState, expression: &str) -> String {
    let expression = expression.trim();
    if let Some(units) = &state.document_units
        && expression.parse::<f64>().is_ok()
    {
        return format!("{expression} {}", units.angle_unit);
    }
    expression.into()
}
