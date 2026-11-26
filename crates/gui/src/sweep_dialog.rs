// SPDX-License-Identifier: GPL-2.0-only
//! Properties and lifecycle for a Sweep of a closed profile along a path.
use crate::{
    GuiAuthoringContext, GuiCommandState, GuiShellAction, dockable_window::DockableWindow,
};
use imgui::Ui;

/// Document and generation whose Sweep inputs may be previewed or accepted.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiSweepRequest {
    pub document: u64,
    pub generation: u64,
}

/// The geometry owner's response to a Sweep preview request.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiSweepPreview {
    Ready,
    MissingInputs,
    Failed(String),
}

/// Actions emitted by the Sweep ribbon and Properties window.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSweepAction {
    Open,
    Preview(GuiSweepRequest),
    Clear,
    Commit(GuiSweepRequest),
}

impl GuiSweepAction {
    pub(crate) fn enabled(self, state: &GuiCommandState) -> bool {
        let can_start = state.document.is_some()
            && state.certified_part_feature_available
            && state.authoring == GuiAuthoringContext::Part
            && !state.transaction_open;
        let current = |request: GuiSweepRequest| {
            can_start
                && state.document == Some(request.document)
                && state.document_generation == Some(request.generation)
        };
        match self {
            Self::Open => can_start,
            Self::Preview(request) => current(request),
            Self::Clear => true,
            Self::Commit(request) => {
                current(request) && state.sweep_preview == Some(GuiSweepPreview::Ready)
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

    pub(super) fn open(&mut self, state: &GuiCommandState) {
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
                    .is_some_and(|request| Some(request.generation) != state.document_generation))
        {
            self.close(actions);
        }
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        area: crate::application::document_creation::workspaces::WorkspaceArea,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let (Some(document), Some(generation)) = (self.document, state.document_generation) else {
            return;
        };
        let request = GuiSweepRequest {
            document,
            generation,
        };
        if self.attempted != Some(request) {
            self.attempted = Some(request);
            actions.push(GuiShellAction::Sweep(GuiSweepAction::Preview(request)));
        }
        let height = if self.show_help { 244. } else { 174. };
        crate::widget_abi::fit_viewport_properties(ui, "Sweep", area, [300., height]);
        let _border = crate::theme::property_window_border(ui);
        let mut open = true;
        Self::dockable_window(ui, "Sweep", &mut open).build(|| {
            crate::theme::floating_property_shadow(ui);
            self.contents(ui, state, request, actions);
        });
        if !open {
            self.close(actions);
        }
    }

    fn contents(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        request: GuiSweepRequest,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let ready = GuiSweepAction::Commit(request).enabled(state);
        let tip = state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.sketches.is_empty());
        if ready {
            crate::theme::property_label(ui, "Profiles");
            ui.text("1 Profile");
            crate::theme::property_label(ui, "Path");
            ui.text("1 Curve");
        } else {
            if let Some(GuiSweepPreview::Failed(cause)) = &state.sweep_preview {
                ui.text_wrapped(cause);
            } else {
                ui.text_wrapped(TIP);
            }
        }
        if self.show_help {
            ui.text_wrapped("Create a closed section in one sketch and a straight path in another. Start the path on the profile plane and keep both sketches visible.");
        }
        ui.separator();
        if crate::theme::property_accept(ui, tip || ready) {
            if ready {
                self.submitted = Some(request);
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
