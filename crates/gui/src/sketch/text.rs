// SPDX-License-Identifier: GPL-2.0-only
//! The text editor sends draft edits and completion to the sketch owner.
use crate::{
    GuiCommandState, GuiShellAction, dockable_lifecycle::DockableState,
    dockable_window::DockableWindow,
};
use imgui::{Key, Ui};
use oblikovati_api::types::{
    SketchTextField, SketchTextFraction, SketchTextLineSpacing, SketchTextRun, TEXT_ALIGN_BASELINE,
    TEXT_ALIGN_LEFT, TextHorizontalAlign, TextVerticalAlign,
};
use std::cell::{Cell, RefCell};
#[path = "text_format.rs"]
mod format_controls;
#[path = "text_content.rs"]
mod text_content;
#[path = "text_number.rs"]
mod text_number;
#[path = "text_presets.rs"]
mod text_presets;
#[path = "text_spacing.rs"]
mod text_spacing;
#[path = "text_width.rs"]
mod text_width;

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
    pub formatting: Box<GuiSketchTextFormatting>,
}

/// Text formatting retained by the dialog and its named presets.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiSketchTextFormatting {
    pub line_spacing: Option<Box<SketchTextLineSpacing>>,
    pub runs: Option<Vec<SketchTextRun>>,
    pub stacked_fractions: Option<Vec<SketchTextFraction>>,
    pub linked_fields: Option<Vec<SketchTextField>>,
}

/// Choices available in the current document for inserting a linked Text field.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiSketchTextFieldSources {
    pub document: String,
    pub model_parameters: Vec<String>,
    pub user_parameters: Vec<String>,
}

/// Document-resolved editor content, tied to the source fields that produced it.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiSketchTextEditorFields {
    pub authored: String,
    pub fields: Vec<SketchTextField>,
    pub projection: oblikovati_model::sketch::SketchTextFieldProjection,
}

impl GuiSketchTextEditorFields {
    fn for_draft(
        &self,
        draft: &GuiSketchTextDraft,
    ) -> Option<&oblikovati_model::sketch::SketchTextFieldProjection> {
        (self.authored == draft.content
            && draft.formatting.linked_fields.as_ref() == Some(&self.fields))
        .then_some(&self.projection)
    }
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
            formatting: Box::default(),
        }
    }
}

/// One editor action, for example `GuiSketchTextAction::Cancel`.
#[derive(Clone, Debug, PartialEq)]
pub enum GuiSketchTextAction {
    Preview(GuiSketchTextDraft),
    Accept {
        draft: GuiSketchTextDraft,
        preset: Option<GuiSketchTextPresetId>,
    },
    Cancel,
    Edit {
        token: u64,
        entity: oblikovati_model::sketch::SketchEntityId,
    },
}

/// Stable session identity of a named Format Text preset, independent of its caption.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub struct GuiSketchTextPresetId(u64);

#[derive(Default)]
pub(crate) struct SketchTextDialog {
    dock_state: DockableState,
    font_filter: RefCell<String>,
    height_input: RefCell<text_number::TextNumberInput>,
    width_input: RefCell<text_number::TextNumberInput>,
    content: RefCell<text_content::TextContentEditor>,
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
            *self.width_input.borrow_mut() = Default::default();
            *self.content.borrow_mut() = Default::default();
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
        Self::fit_dockable(ui, TITLE, [560., 135.], [460., 430.]);
        let mut open = true;
        let action = Self::dockable_window(ui, TITLE, &mut open)
            .build(|| self.text_editor(ui, draft, state))
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
        state: &GuiCommandState,
    ) -> Option<GuiSketchTextAction> {
        let mut presets = self.presets.borrow_mut();
        let mut height = self.height_input.borrow_mut();
        let mut width = self.width_input.borrow_mut();
        let changed_preset = presets.begin(
            state.sketch_text_existing,
            state.sketch_text_preset,
            &mut draft,
        ) | presets.draw(ui, &mut draft);
        if changed_preset {
            *height = Default::default();
            *width = Default::default();
            *self.content.borrow_mut() = Default::default();
        }
        let escape_used = presets.draw_name(ui, &draft);
        let row_factor = text_spacing::editor_row_factor(&draft);
        let editor_fields = state
            .sketch_text_editor_fields
            .as_deref()
            .and_then(|fields| fields.for_draft(&draft));
        let changed = changed_preset
            | self.content.borrow_mut().draw(
                ui,
                &mut draft,
                row_factor,
                state.sketch_text_field_sources.as_deref(),
                editor_fields,
            )
            | format_controls::draw(
                ui,
                &mut draft,
                state.sketch_text_font_families.as_deref().unwrap_or(&[]),
                &mut self.font_filter.borrow_mut(),
                &mut height,
            )
            | text_width::draw(ui, &mut draft, &mut width);
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
            height.apply(&mut draft.height_mm, &text_number::HEIGHT);
            text_width::apply(&mut draft, &mut width);
            if !height.blocked() && !width.blocked() {
                presets.submit(&draft);
                return Some(GuiSketchTextAction::Accept {
                    draft,
                    preset: presets.selected_preset(),
                });
            }
        }
        height.warning(ui, &text_number::HEIGHT);
        if !height.blocked() {
            width.warning(ui, &text_number::WIDTH);
        }
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
