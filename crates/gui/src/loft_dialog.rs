// SPDX-License-Identifier: GPL-2.0-only
//! The Loft section draft and empty-Part guidance.
use crate::{
    GuiAuthoringContext, GuiCommandState, GuiShellAction,
    application::document_creation::workspaces::WorkspaceArea,
    dockable_lifecycle::{DockRegistry, DockableState},
    dockable_window::DockableWindow,
};
use imgui::{TabItemFlags, Ui};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{FeatureId, sketch_loft::SketchLoftOutput},
    sketch::SketchEntityId,
};

const TITLE: &str = "Loft";
const EMPTY_PART: &str = "Loft requires a body or at least two visible sketches.";
const HELP: &str = "Select each profile or point in the viewport in loft order.";

/// An authored point or circular profile in a Loft draft.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiLoftElement {
    Point(SketchEntityId),
    Profile(SketchEntityId),
}

/// The sketch and entity the viewport selected for one Loft section.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiLoftSection {
    pub sketch: DocumentSketchId,
    pub name: String,
    pub element: GuiLoftElement,
}

/// The geometry that controls a Loft between its sections.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub enum GuiLoftPathMode {
    #[default]
    Rails,
    CenterLine,
    PlacedSections,
}

/// The authored output and path of a saved Loft.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct GuiLoftEditSeed {
    pub output: SketchLoftOutput,
    pub path_mode: GuiLoftPathMode,
}

/// Lifecycle requests for the active Loft draft.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiLoftAction {
    Open,
    OpenEdit(FeatureId),
    Begin {
        document: u64,
        feature: Option<FeatureId>,
    },
    SetPath {
        document: u64,
        mode: GuiLoftPathMode,
    },
    SetOutput {
        document: u64,
        output: SketchLoftOutput,
    },
    ChooseCenterLine(u64),
    Commit(u64),
    Clear,
}

impl GuiLoftAction {
    pub(crate) fn enabled(self, state: &GuiCommandState) -> bool {
        let available = state.authoring == GuiAuthoringContext::Part
            && state.document.is_some()
            && state.certified_part_feature_available
            && !state.transaction_open;
        match self {
            Self::Open => available,
            Self::OpenEdit(feature) => {
                available
                    && state.browser.as_ref().is_some_and(|browser| {
                        browser
                            .features
                            .iter()
                            .any(|row| row.id == feature && row.kind == "Loft" && !row.suppressed)
                    })
            }
            Self::Begin { document, feature } => {
                available
                    && state.document == Some(document)
                    && feature.is_none_or(|id| Self::OpenEdit(id).enabled(state))
            }
            Self::SetPath { document, .. } | Self::SetOutput { document, .. } => {
                available && state.document == Some(document)
            }
            Self::ChooseCenterLine(document) => available && state.document == Some(document),
            Self::Commit(document) => {
                available
                    && state.document == Some(document)
                    && state.loft_preview.as_ref().is_some_and(Result::is_ok)
            }
            Self::Clear => true,
        }
    }
}

#[derive(Default)]
pub(crate) struct LoftDialog {
    dock: DockableState,
    document: Option<(u64, u64)>,
    show_help: bool,
    editing: bool,
    path_mode: GuiLoftPathMode,
    output: SketchLoftOutput,
    closed_loop: bool,
    merge_tangent_faces: bool,
    select_curves: bool,
    manual_mapping: bool,
    supplemental_point_sets: usize,
}

impl DockableWindow for LoftDialog {}

impl LoftDialog {
    pub(crate) fn register_dockable(&self, registry: &mut DockRegistry) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.dock)
    }

    pub(crate) fn open(
        &mut self,
        state: &GuiCommandState,
        feature: Option<FeatureId>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let open = feature.map_or(GuiLoftAction::Open, GuiLoftAction::OpenEdit);
        let Some((document, generation)) = state
            .document
            .zip(state.document_generation)
            .filter(|_| open.enabled(state))
        else {
            return;
        };
        self.document = Some((document, generation));
        self.show_help = false;
        self.editing = feature.is_some();
        let seed = state
            .browser
            .as_ref()
            .and_then(|browser| browser.loft_edits.get(&feature?))
            .copied()
            .unwrap_or_default();
        self.path_mode = seed.path_mode;
        self.output = seed.output;
        self.closed_loop = false;
        self.merge_tangent_faces = false;
        self.select_curves = true;
        self.manual_mapping = false;
        self.supplemental_point_sets = 0;
        self.dock.set_open(true);
        actions.push(GuiShellAction::Loft(GuiLoftAction::Begin {
            document,
            feature,
        }));
    }

    pub(crate) fn close(&mut self, actions: &mut Vec<GuiShellAction>) {
        if self.document.take().is_some() {
            self.dock.set_open(false);
            actions.push(GuiShellAction::Loft(GuiLoftAction::Clear));
        }
    }

    pub(crate) fn synchronize(
        &mut self,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if state.document.zip(state.document_generation) != self.document
            || !GuiLoftAction::Open.enabled(state)
        {
            self.close(actions);
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if self.document.is_none() {
            return;
        }
        let tip = state.browser.as_ref().is_none_or(empty_part);
        let height = if tip { 175. } else { 325. };
        crate::widget_abi::fit_viewport_properties(ui, TITLE, area, [300., height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, TITLE, &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            if self.contents(ui, state, tip, actions) {
                self.close(actions);
            }
        });
        if !open {
            self.close(actions);
        }
    }

    fn contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        tip: bool,
        actions: &mut Vec<GuiShellAction>,
    ) -> bool {
        if tip {
            ui.text_wrapped(EMPTY_PART);
        } else {
            self.tabs(ui, state, actions);
            if let Some(Err(cause)) = &state.loft_preview {
                ui.text_wrapped(cause);
            }
        }
        if self.show_help {
            ui.text_wrapped(HELP);
        }
        ui.separator();
        let commit = self
            .document
            .map(|(document, _)| GuiLoftAction::Commit(document))
            .filter(|action| {
                (self.path_mode == GuiLoftPathMode::Rails || state.loft_center_line.is_some())
                    && !self.manual_mapping
                    && !self.closed_loop
                    && !self.merge_tangent_faces
                    && action.enabled(state)
            });
        let accepted = crate::theme::property_accept(ui, tip || commit.is_some());
        if accepted
            && !self.editing
            && let Some(commit) = commit
        {
            actions.push(GuiShellAction::Loft(commit));
        }
        ui.same_line();
        let cancelled = !tip && crate::widget_abi::button(ui, "Cancel", [58., 28.]);
        if !tip {
            ui.same_line();
        }
        if crate::widget_abi::button(ui, "Help", [58., 28.]) {
            self.show_help = !self.show_help;
        }
        (accepted && (tip || self.editing))
            || cancelled
            || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
    }
    fn tabs(&mut self, ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let Some(_tabs) = ui.tab_bar("##loft-pages") else {
            return;
        };
        let flags = if std::mem::take(&mut self.select_curves) {
            TabItemFlags::SET_SELECTED
        } else {
            TabItemFlags::empty()
        };
        if let Some(_curves) = ui.tab_item_with_flags("Curves", None, flags) {
            draw_sections(ui, &state.loft_sections);
            if !self.editing && self.path_mode == GuiLoftPathMode::Rails {
                ui.text("Select sections in the viewport");
            }
            self.output_controls(ui, actions);
            self.path_controls(ui, state, actions);
            ui.checkbox("Closed Loop", &mut self.closed_loop);
            ui.checkbox("Merge Tangent Faces", &mut self.merge_tangent_faces);
        }
        if let Some(_transition) = ui.tab_item("Transition") {
            self.transition(ui, &state.loft_sections);
        }
    }

    fn output_controls(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let before = self.output;
        let _disabled = ui.begin_disabled(self.editing);
        ui.radio_button("Solid", &mut self.output, SketchLoftOutput::Solid);
        ui.same_line();
        ui.radio_button("Surface", &mut self.output, SketchLoftOutput::Surface);
        if self.output != before
            && let Some((document, _)) = self.document
        {
            actions.push(GuiShellAction::Loft(GuiLoftAction::SetOutput {
                document,
                output: self.output,
            }));
        }
    }

    fn path_controls(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let before = self.path_mode;
        ui.radio_button("Rails", &mut self.path_mode, GuiLoftPathMode::Rails);
        ui.same_line();
        ui.radio_button(
            "Center Line",
            &mut self.path_mode,
            GuiLoftPathMode::CenterLine,
        );
        ui.radio_button(
            "Center Line with Placed Sections",
            &mut self.path_mode,
            GuiLoftPathMode::PlacedSections,
        );
        if self.path_mode != before
            && let Some((document, _)) = self.document
        {
            actions.push(GuiShellAction::Loft(GuiLoftAction::SetPath {
                document,
                mode: self.path_mode,
            }));
        }
        match self.path_mode {
            GuiLoftPathMode::Rails => {
                ui.child_window("##loft-rails")
                    .size([0., 27.])
                    .border(true)
                    .build(|| {
                        ui.text_disabled("No rails selected");
                    });
            }
            GuiLoftPathMode::CenterLine => self.center_line_selector(ui, state, actions),
            GuiLoftPathMode::PlacedSections => {
                self.center_line_selector(ui, state, actions);
                let _disabled = ui.begin_disabled(true);
                crate::widget_abi::button(ui, "Placed Sections", [0., 0.]);
            }
        }
    }

    fn center_line_selector(
        &self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if let Some(sketch) = &state.loft_center_line {
            ui.text(sketch);
        } else {
            ui.text("Select a Center Line sketch");
        }
        let _disabled = ui.begin_disabled(self.editing);
        if crate::widget_abi::button(ui, "Select Center Line", [0., 0.])
            && let Some((document, _)) = self.document
        {
            actions.push(GuiShellAction::Loft(GuiLoftAction::ChooseCenterLine(
                document,
            )));
        }
    }

    fn transition(&mut self, ui: &Ui, sections: &[GuiLoftSection]) {
        let mut automatic = !self.manual_mapping;
        if ui.checkbox("Automatic Mapping", &mut automatic) {
            self.manual_mapping = !automatic;
        }
        if automatic {
            return;
        }
        ui.text("Point Set");
        let _disabled = ui.begin_disabled(sections.len() < 2);
        if sections.len() >= 2 {
            let height = if self.supplemental_point_sets == 0 {
                76.
            } else {
                156.
            };
            ui.child_window("##loft-mapping-point-sets")
                .size([0., height])
                .border(true)
                .build(|| {
                    draw_point_set(ui, sections, 1, "");
                    for index in 0..self.supplemental_point_sets {
                        draw_point_set(ui, sections, index + 2, " *");
                    }
                });
        }
        if crate::widget_abi::button(ui, "Click to add", [0., 0.]) {
            self.supplemental_point_sets += 1;
        }
    }
}

fn empty_part(browser: &crate::GuiModelBrowser) -> bool {
    browser.bodies.is_empty()
        && browser.surface_bodies.is_empty()
        && !browser.sketches.iter().any(|sketch| sketch.visible)
}

fn draw_sections(ui: &Ui, sections: &[GuiLoftSection]) {
    ui.text("Sections");
    for section in sections {
        match section.element {
            GuiLoftElement::Point(_) => ui.text(format!("Point {}", section.name)),
            GuiLoftElement::Profile(_) => ui.text(&section.name),
        }
    }
}

fn draw_point_set(ui: &Ui, sections: &[GuiLoftSection], number: usize, incomplete: &str) {
    ui.text(format!("Set {number}"));
    ui.indent_by(12.);
    for section in sections {
        ui.text(format!("{} Map Point{incomplete}", section.name));
    }
    ui.unindent_by(12.);
}
