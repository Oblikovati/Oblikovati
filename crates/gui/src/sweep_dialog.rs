// SPDX-License-Identifier: GPL-2.0-only
//! Properties and lifecycle for a Sweep of a closed profile along a path.
use crate::{
    GuiAuthoringContext, GuiCommandState, GuiShellAction, dockable_window::DockableWindow,
};
use imgui::Ui;
#[path = "sweep_orientation.rs"]
mod orientation;
use crate::icons::GuiRibbonIcons;
use oblikovati_model::feature::FeatureId;

/// Sweep orientation choices available in Properties.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiSweepOrientation {
    #[default]
    FollowPath,
    Fixed,
    Guide,
}

/// Authored Sweep options shown when editing an existing feature.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiSweepEditSeed {
    pub taper: String,
    pub orientation: GuiSweepOrientation,
}

/// Document and generation whose Sweep inputs may be previewed or accepted.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiSweepRequest {
    pub document: u64,
    pub generation: u64,
    pub feature: Option<FeatureId>,
    pub orientation: GuiSweepOrientation,
    pub taper: String,
}

/// The geometry owner's response to a Sweep preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSweepPreview {
    Ready,
    MissingInputs,
    MissingGuide,
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

#[derive(Default)]
pub(crate) struct SweepDialog {
    dock: crate::dockable_lifecycle::DockableState,
    document: Option<u64>,
    attempted: Option<GuiSweepRequest>,
    submitted: Option<GuiSweepRequest>,
    feature: Option<FeatureId>,
    orientation: GuiSweepOrientation,
    selector: orientation::OrientationSelector,
    taper: String,
    show_help: bool,
}
impl DockableWindow for SweepDialog {}
impl SweepDialog {
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
                ..Default::default()
            },
        };
        self.feature = feature;
        self.taper = seed.taper;
        self.orientation = seed.orientation;
        self.document = state.document;
        self.dock.set_open(self.document.is_some());
        self.attempted = None;
        self.submitted = None;
        self.show_help = false;
    }

    pub(super) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Sweep(GuiSweepAction::Clear));
        }
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_some()
            && (self.document != state.document
                || !GuiSweepAction::Open.enabled(state)
                || self
                    .submitted
                    .as_ref()
                    .is_some_and(|request| Some(request.generation) != state.document_generation))
        {
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
        let height = if self.show_help { 363. } else { 289. };
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
            crate::theme::property_label(ui, "Profiles");
            ui.text("1 Profile");
            crate::theme::property_label(ui, "Path");
            ui.text("1 Curve");
        }
        if !tip {
            self.selector.draw(ui, &mut self.orientation, icons);
        }
        let follows_path = self.orientation == GuiSweepOrientation::FollowPath;
        if !tip && follows_path {
            crate::theme::property_label(ui, "Taper");
            let _width = ui.push_item_width(-1.);
            ui.input_text("##sweep-taper", &mut self.taper).build();
        }
        let request = GuiSweepRequest {
            document,
            generation,
            feature: self.feature,
            orientation: self.orientation,
            taper: if follows_path {
                self.taper.clone()
            } else {
                String::new()
            },
        };
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
            Some(GuiSweepPreview::Failed(cause)) => ui.text_wrapped(cause),
            None if tip => ui.text_wrapped(TIP),
            _ => {}
        }
        let ready = GuiSweepAction::Commit(request.clone()).enabled(state);
        if !tip && ui.collapsing_header("Advanced Properties", imgui::TreeNodeFlags::empty()) {
            let _disabled = ui.begin_disabled(true);
            ui.checkbox("Optimize for Single Selection", &mut false);
        }
        if self.show_help {
            ui.text_wrapped("Create a closed section in one sketch and a straight path in another. Start the path on the profile plane and keep both sketches visible.");
        }
        ui.separator();
        if crate::theme::property_accept(ui, tip || ready) {
            if ready {
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
    }
}
