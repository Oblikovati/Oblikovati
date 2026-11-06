// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction};
use imgui::{Key, Ui};

/// A checked owner draft; example: display `draft.kind` before accepting its expression.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDimensionDraft {
    pub id: u64,
    pub kind: String,
    pub orientation: String,
    pub expression: String,
}
/// UI intent for one owner draft; example: `GuiDimensionAction::Start`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuiDimensionAction {
    Start,
    Apply {
        id: u64,
        expression: String,
        driven: bool,
    },
    Cancel {
        id: u64,
    },
}
#[derive(Default)]
pub(crate) struct GuiDimensionEditor {
    draft: Option<u64>,
    expression: String,
    driven: bool,
}
impl GuiDimensionEditor {
    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let fresh = self.synchronize_draft(state.dimension_draft.as_ref());
        let Some(draft) = state.dimension_draft.as_ref() else {
            return;
        };
        crate::widget_abi::fit_dialog(ui, "Dimension", [640., 240.], [420., 180.]);
        ui.window("Dimension").build(|| {
            self.fields(ui, draft, fresh);
            self.controls(ui, state, draft.id, actions);
        });
    }
    fn synchronize_draft(&mut self, draft: Option<&GuiDimensionDraft>) -> bool {
        let id = draft.map(|draft| draft.id);
        if self.draft == id {
            return false;
        }
        *self = Self {
            draft: id,
            expression: draft.map_or_else(String::new, |draft| draft.expression.clone()),
            ..Default::default()
        };
        true
    }
    fn fields(&mut self, ui: &Ui, draft: &GuiDimensionDraft, fresh: bool) {
        ui.text(format!("{} {}", draft.kind, draft.orientation));
        if fresh {
            ui.set_keyboard_focus_here();
        }
        ui.input_text("Expression", &mut self.expression).build();
        ui.text("Leave blank to keep the measured value.");
        ui.checkbox("Reference (driven)", &mut self.driven);
    }
    fn controls(
        &self,
        ui: &Ui,
        state: &GuiCommandState,
        id: u64,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let apply = GuiShellAction::Dimension(GuiDimensionAction::Apply {
            id,
            expression: self.expression.clone(),
            driven: self.driven,
        });
        let enabled = apply.dispatchable(state);
        let disabled = ui.begin_disabled(!enabled);
        let pressed = crate::widget_abi::button(ui, "Apply", [0.; 2]);
        if enabled && (pressed || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Enter))
        {
            actions.push(apply);
        }
        drop(disabled);
        ui.same_line();
        if crate::widget_abi::button(ui, "Cancel", [0.; 2])
            || ui.is_key_pressed_no_repeat(Key::Escape)
        {
            actions.push(GuiShellAction::Dimension(GuiDimensionAction::Cancel { id }));
        }
    }
}

pub(crate) fn validate_dimension_draft(state: &GuiCommandState) -> Result<(), String> {
    use oblikovati_api::types::{
        DIM_CONSTRAINT_ANGLE, DIM_CONSTRAINT_DISTANCE, DIM_CONSTRAINT_RADIUS,
    };
    let Some(draft) = state.dimension_draft.as_ref() else {
        return Ok(());
    };
    let kind_valid = [
        DIM_CONSTRAINT_ANGLE,
        DIM_CONSTRAINT_DISTANCE,
        DIM_CONSTRAINT_RADIUS,
    ]
    .iter()
    .any(|kind| kind.0.as_ref() == draft.kind);
    let orientation_valid = matches!(
        draft.orientation.as_str(),
        "" | "aligned" | "horizontal" | "vertical"
    );
    if draft.id == 0
        || !kind_valid
        || !orientation_valid
        || draft.expression.contains('\0')
        || state.document.is_none()
        || state.authoring != crate::GuiAuthoringContext::Sketch
    {
        return Err(format!(
            "dimension draft {draft:?}, document {:?}, authoring {:?}: expected a positive owner draft in an active sketch with a canonical kind, distance orientation and source without NUL",
            state.document, state.authoring
        ));
    }
    Ok(())
}
