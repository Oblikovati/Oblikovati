// SPDX-License-Identifier: GPL-2.0-only
//! The text editor sends draft edits and completion to the sketch owner.
use crate::{
    GuiCommandState, GuiShellAction, dockable_lifecycle::DockableState,
    dockable_window::DockableWindow,
};
use imgui::{Key, Ui};

const TITLE: &str = "Format Text";

/// Values typed into the sketch text editor. Height is entered in millimetres.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiSketchTextDraft {
    pub content: String,
    pub family: String,
    pub height_mm: String,
    pub rotation_degrees: String,
}

impl Default for GuiSketchTextDraft {
    fn default() -> Self {
        Self {
            content: String::new(),
            family: String::new(),
            height_mm: "3.05".into(),
            rotation_degrees: "0".into(),
        }
    }
}

/// One editor action, for example `GuiSketchTextAction::Cancel`.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiSketchTextAction {
    Preview(GuiSketchTextDraft),
    Accept(GuiSketchTextDraft),
    Cancel,
}

#[derive(Default)]
pub(crate) struct SketchTextDialog {
    dock_state: DockableState,
}
impl DockableWindow for SketchTextDialog {}

impl SketchTextDialog {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.dock_state)
    }

    pub(crate) fn synchronize(&self, state: &GuiCommandState) {
        self.dock_state.set_open(state.sketch_text.is_some());
    }

    pub(crate) fn panel(
        &self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(draft) = state.sketch_text.clone() else {
            return;
        };
        Self::fit_dockable(ui, TITLE, [560., 175.], [440., 285.]);
        let mut open = true;
        let action = Self::dockable_window(ui, TITLE, &mut open)
            .build(|| text_editor(ui, draft))
            .flatten();
        if !open {
            self.dock_state.set_open(false);
            actions.push(GuiShellAction::SketchText(GuiSketchTextAction::Cancel));
        } else if let Some(action) = action {
            actions.push(GuiShellAction::SketchText(action));
        }
    }
}

fn text_editor(ui: &Ui, mut draft: GuiSketchTextDraft) -> Option<GuiSketchTextAction> {
    let changed = text_fields(ui, &mut draft);
    if ui.button("OK") {
        return Some(GuiSketchTextAction::Accept(draft));
    }
    ui.same_line();
    if ui.button("Cancel") || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape) {
        return Some(GuiSketchTextAction::Cancel);
    }
    changed.then_some(GuiSketchTextAction::Preview(draft))
}

fn text_fields(ui: &Ui, draft: &mut GuiSketchTextDraft) -> bool {
    ui.input_text_multiline("Content", &mut draft.content, [295., 100.])
        .build()
        | ui.input_text("Font family", &mut draft.family)
            .hint("System default")
            .build()
        | ui.input_text("Height (mm)", &mut draft.height_mm).build()
        | ui.input_text("Rotation (deg)", &mut draft.rotation_degrees)
            .build()
}
