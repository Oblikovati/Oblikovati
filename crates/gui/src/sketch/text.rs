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
use std::cell::{Cell, RefCell};
#[path = "text_format.rs"]
mod format_controls;
#[path = "text_height.rs"]
mod text_height;
#[path = "text_presets.rs"]
mod text_presets;

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
    height_input: RefCell<text_height::TextHeightInput>,
    presets: RefCell<text_presets::TextPresets>,
    escape_consumed: Cell<bool>,
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
        self.escape_consumed.set(false);
        self.dock_state.set_open(state.sketch_text.is_some());
        self.presets
            .borrow_mut()
            .synchronize(state.sketch_text.is_some());
        if state.sketch_text.is_none() {
            *self.height_input.borrow_mut() = Default::default();
        }
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
                self.text_editor(
                    ui,
                    draft,
                    state.sketch_text_font_families.as_deref().unwrap_or(&[]),
                    state.sketch_text_existing,
                )
            })
            .flatten();
        if !open {
            self.dock_state.set_open(false);
            self.presets.borrow_mut().cancel();
            actions.push(GuiShellAction::SketchText(GuiSketchTextAction::Cancel));
        } else if let Some(action) = action {
            actions.push(GuiShellAction::SketchText(action));
        }
    }

    pub(crate) fn escape_consumed(&self) -> bool {
        self.escape_consumed.get()
    }

    fn text_editor(
        &self,
        ui: &Ui,
        mut draft: GuiSketchTextDraft,
        families: &[String],
        existing: bool,
    ) -> Option<GuiSketchTextAction> {
        let mut presets = self.presets.borrow_mut();
        let mut height = self.height_input.borrow_mut();
        let changed_preset = presets.begin(existing, &mut draft) | presets.draw(ui, &mut draft);
        if changed_preset {
            *height = Default::default();
        }
        let escape_used = presets.draw_name(ui, &draft);
        let changed = changed_preset
            | ui.input_text_multiline("Content", &mut draft.content, [295., 100.])
                .build()
            | format_controls::draw(
                ui,
                &mut draft,
                families,
                &mut self.font_filter.borrow_mut(),
                &mut height,
            );
        let accept = ui.button("OK");
        ui.same_line();
        let escape_editor =
            !escape_used && ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape);
        self.escape_consumed.set(escape_used || escape_editor);
        if ui.button("Cancel") || escape_editor {
            presets.cancel();
            return Some(GuiSketchTextAction::Cancel);
        }
        if accept {
            height.apply(&mut draft.height_mm);
            if !height.blocked() {
                presets.submit(&draft);
                return Some(GuiSketchTextAction::Accept(draft));
            }
        }
        height.warning(ui);
        changed.then_some(GuiSketchTextAction::Preview(draft))
    }
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
