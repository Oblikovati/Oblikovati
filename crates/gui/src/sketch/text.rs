// SPDX-License-Identifier: GPL-2.0-only
//! The text editor sends draft edits and completion to the sketch owner.
use crate::{
    GuiCommandState, GuiShellAction, dockable_lifecycle::DockableState,
    dockable_window::DockableWindow,
};
use imgui::{Key, Ui};
use oblikovati_api::types::{
    TEXT_ALIGN_BASELINE, TEXT_ALIGN_LEFT, TextHorizontalAlign, TextVerticalAlign,
};
use std::cell::RefCell;
#[path = "text_format.rs"]
mod format_controls;

const TITLE: &str = "Format Text";

/// Values typed into the sketch text editor. Height is entered in millimetres.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiSketchTextDraft {
    pub content: String,
    pub family: String,
    pub height_mm: String,
    pub rotation_degrees: String,
    pub h_align: TextHorizontalAlign,
    pub v_align: TextVerticalAlign,
}

impl Default for GuiSketchTextDraft {
    fn default() -> Self {
        Self {
            content: String::new(),
            family: String::new(),
            height_mm: "3.05".into(),
            rotation_degrees: "0".into(),
            h_align: TEXT_ALIGN_LEFT,
            v_align: TEXT_ALIGN_BASELINE,
        }
    }
}

/// One editor action, for example `GuiSketchTextAction::Cancel`.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiSketchTextAction {
    Preview(GuiSketchTextDraft),
    Accept(GuiSketchTextDraft),
    Cancel,
    Edit {
        token: u64,
        entity: oblikovati_model::sketch::SketchEntityId,
    },
}

#[derive(Default)]
pub(crate) struct SketchTextDialog {
    dock_state: DockableState,
    font_filter: RefCell<String>,
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
        Self::fit_dockable(ui, TITLE, [560., 175.], [460., 345.]);
        let mut open = true;
        let action = Self::dockable_window(ui, TITLE, &mut open)
            .build(|| {
                let mut filter = self.font_filter.borrow_mut();
                text_editor(
                    ui,
                    draft,
                    state.sketch_text_font_families.as_deref().unwrap_or(&[]),
                    &mut filter,
                )
            })
            .flatten();
        if !open {
            self.dock_state.set_open(false);
            actions.push(GuiShellAction::SketchText(GuiSketchTextAction::Cancel));
        } else if let Some(action) = action {
            actions.push(GuiShellAction::SketchText(action));
        }
    }
}

fn text_editor(
    ui: &Ui,
    mut draft: GuiSketchTextDraft,
    families: &[String],
    filter: &mut String,
) -> Option<GuiSketchTextAction> {
    let changed = ui
        .input_text_multiline("Content", &mut draft.content, [295., 100.])
        .build()
        | format_controls::draw(ui, &mut draft, families, filter);
    if ui.button("OK") {
        return Some(GuiSketchTextAction::Accept(draft));
    }
    ui.same_line();
    if ui.button("Cancel") || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape) {
        return Some(GuiSketchTextAction::Cancel);
    }
    changed.then_some(GuiSketchTextAction::Preview(draft))
}

pub(super) fn edit_menu_row(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(menu) = &state.sketch_context else {
        return;
    };
    let Some(entity) = menu.text else {
        return;
    };
    let action = GuiShellAction::SketchText(GuiSketchTextAction::Edit {
        token: menu.token,
        entity,
    });
    if ui
        .menu_item_config("Edit Text")
        .enabled(action.enabled(state))
        .build()
    {
        actions.push(action);
    }
}

impl GuiSketchTextAction {
    pub(crate) fn enabled(&self, state: &GuiCommandState) -> bool {
        if state.transaction_open {
            return false;
        }
        match self {
            Self::Edit { token, entity } => {
                state.sketch_at_ready
                    && state
                        .sketch_context
                        .as_ref()
                        .is_some_and(|menu| menu.token == *token && menu.text == Some(*entity))
            }
            _ => state.sketch_text.is_some(),
        }
    }
}
