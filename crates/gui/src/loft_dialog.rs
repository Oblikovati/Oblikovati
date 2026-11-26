// SPDX-License-Identifier: GPL-2.0-only
//! The Loft section draft and empty-Part guidance.
use crate::{
    GuiAuthoringContext, GuiCommandState, GuiShellAction,
    application::document_creation::workspaces::WorkspaceArea,
    dockable_lifecycle::{DockRegistry, DockableState},
    dockable_window::DockableWindow,
};
use imgui::{TabItemFlags, Ui};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};

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

/// Lifecycle requests for the active Loft draft.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiLoftAction {
    Open,
    Begin(u64),
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
            Self::Begin(document) => available && state.document == Some(document),
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
    select_curves: bool,
    manual_mapping: bool,
    point_sets: usize,
}

impl DockableWindow for LoftDialog {}

impl LoftDialog {
    pub(crate) fn register_dockable(&self, registry: &mut DockRegistry) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.dock)
    }

    pub(crate) fn open(&mut self, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
        let Some((document, generation)) = state
            .document
            .zip(state.document_generation)
            .filter(|_| GuiLoftAction::Open.enabled(state))
        else {
            return;
        };
        self.document = Some((document, generation));
        self.show_help = false;
        self.select_curves = true;
        self.manual_mapping = false;
        self.point_sets = 0;
        self.dock.set_open(true);
        actions.push(GuiShellAction::Loft(GuiLoftAction::Begin(document)));
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
        let height = if tip { 175. } else { 290. };
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
            self.tabs(ui, &state.loft_sections);
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
            .filter(|action| !self.manual_mapping && action.enabled(state));
        let accepted = crate::theme::property_accept(ui, tip || commit.is_some());
        if accepted && let Some(commit) = commit {
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
        (accepted && tip) || cancelled || ui.is_key_pressed_no_repeat(imgui::Key::Escape)
    }
    fn tabs(&mut self, ui: &Ui, sections: &[GuiLoftSection]) {
        let Some(_tabs) = ui.tab_bar("##loft-pages") else {
            return;
        };
        let flags = if std::mem::take(&mut self.select_curves) {
            TabItemFlags::SET_SELECTED
        } else {
            TabItemFlags::empty()
        };
        if let Some(_curves) = ui.tab_item_with_flags("Curves", None, flags) {
            draw_sections(ui, sections);
        }
        if let Some(_transition) = ui.tab_item("Transition") {
            self.transition(ui);
        }
    }

    fn transition(&mut self, ui: &Ui) {
        let mut automatic = !self.manual_mapping;
        if ui.checkbox("Automatic Mapping", &mut automatic) {
            self.manual_mapping = !automatic;
        }
        if automatic {
            return;
        }
        ui.text("Point Set");
        for index in 1..=self.point_sets {
            ui.text(format!("Set {index} *"));
        }
        if crate::widget_abi::button(ui, "Click to add", [0., 0.]) {
            self.point_sets += 1;
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
    ui.text("Select sections in the viewport");
}
