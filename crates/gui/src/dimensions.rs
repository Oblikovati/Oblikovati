// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiCommandState;

/// A placed dimension or a refused placement awaiting explicit reference correction.
/// Example: open `draft.committed` in the dimension value editor.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiDimensionDraft {
    pub id: u64,
    pub kind: String,
    pub orientation: String,
    pub expression: String,
    pub committed: Option<crate::GuiDimensionEdit>,
    pub refusal: Option<String>,
    pub choice_required: bool,
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
pub(crate) fn validate_dimension_draft(state: &GuiCommandState) -> Result<(), String> {
    use oblikovati_api::types::{
        DIM_CONSTRAINT_ANGLE, DIM_CONSTRAINT_DISTANCE, DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS,
        DIM_CONSTRAINT_ELLIPSE_RADIUS, DIM_CONSTRAINT_RADIUS,
    };
    let Some(draft) = state.dimension_draft.as_ref() else {
        return Ok(());
    };
    let kind_valid = [
        DIM_CONSTRAINT_ANGLE,
        DIM_CONSTRAINT_DISTANCE,
        DIM_CONSTRAINT_RADIUS,
        DIM_CONSTRAINT_ELLIPSE_RADIUS,
        DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS,
    ]
    .iter()
    .any(|kind| kind.0.as_ref() == draft.kind);
    let orientation_valid = matches!(
        draft.orientation.as_str(),
        "" | "aligned" | "horizontal" | "vertical"
    );
    let outcome_valid = match (&draft.committed, &draft.refusal) {
        (Some(edit), None) => {
            edit.sketch != 0
                && edit.constraint != 0
                && !edit.name.is_empty()
                && !edit.expression.contains('\0')
        }
        (None, Some(refusal)) => !refusal.is_empty() && !refusal.contains('\0'),
        _ => false,
    };
    if draft.id == 0
        || (draft.choice_required && draft.committed.is_some())
        || !kind_valid
        || !outcome_valid
        || !orientation_valid
        || draft.expression.contains('\0')
        || state.document.is_none()
        || state.authoring != crate::GuiAuthoringContext::Sketch
    {
        return Err(format!(
            "dimension draft {draft:?}, document {:?}, authoring {:?}: expected a positive owner draft in an active sketch with a canonical kind, distance orientation and source without NUL, and either a committed edit or a named refusal",
            state.document, state.authoring
        ));
    }
    Ok(())
}
