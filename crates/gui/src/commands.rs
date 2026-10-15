// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Copy, Debug, PartialEq)]
pub enum GuiShellAction {
    Save,
    Undo,
    Redo,
    Home,
    NewPart,
    SketchXY,
    Rectangle,
    Extrude,
    ActivateDocument(u64),
    CloseDocument(u64),
    SetGridSpacingCm(f64),
    SetGridVisible(bool),
    SetGridMajorEvery(i64),
    SetHeadsUpDisplay(bool),
    SetPointerInput(bool),
    SetDimensionInput(bool),
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiDocumentTab {
    pub id: u64,
    pub name: String,
    pub dirty: bool,
    pub active: bool,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiRibbonControl {
    pub command_id: String,
    pub display_name: String,
    pub icon: String,
    pub enabled: bool,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiRibbonTab {
    pub name: String,
    pub controls: Vec<GuiRibbonControl>,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiModelFeature {
    pub name: String,
    pub kind: String,
    pub health: String,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiModelBody {
    pub name: String,
    pub visible: bool,
    pub faces: usize,
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiSketchPreferences {
    pub grid_spacing_cm: f64,
    pub grid_visible: bool,
    pub grid_major_every: i64,
    pub heads_up_display: bool,
    pub pointer_input: bool,
    pub dimension_input: bool,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiModelTree {
    pub document: String,
    pub parameters: Vec<String>,
    pub sketches: usize,
    pub features: Vec<GuiModelFeature>,
    pub bodies: Vec<GuiModelBody>,
}
/// A bounded owner snapshot; the GUI holds no document history or transaction token.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiCommandState {
    pub undo_label: Option<String>,
    pub redo_label: Option<String>,
    pub transaction_open: bool,
    pub documents: Vec<GuiDocumentTab>,
    pub ribbon_tabs: Vec<GuiRibbonTab>,
    pub model_tree: Option<GuiModelTree>,
    pub sketch_preferences: Option<GuiSketchPreferences>,
}
impl GuiCommandState {
    pub(crate) fn validate(&self) -> Result<(), String> {
        for label in [&self.undo_label, &self.redo_label].into_iter().flatten() {
            if label.contains('\0') {
                return Err(format!(
                    "GUI history label {label:?}: expected text without NUL"
                ));
            }
        }
        Ok(())
    }
}
impl GuiShellAction {
    /// Preserve Go empty-stack enablement; example: `GuiShellAction::Undo.enabled(&state)`.
    pub fn enabled(self, state: &GuiCommandState) -> bool {
        match self {
            Self::Undo => state.undo_label.is_some(),
            Self::Redo => state.redo_label.is_some(),
            Self::Save
            | Self::Home
            | Self::NewPart
            | Self::SketchXY
            | Self::Rectangle
            | Self::Extrude
            | Self::ActivateDocument(_)
            | Self::CloseDocument(_)
            | Self::SetGridSpacingCm(_)
            | Self::SetGridVisible(_)
            | Self::SetGridMajorEvery(_)
            | Self::SetHeadsUpDisplay(_)
            | Self::SetPointerInput(_)
            | Self::SetDimensionInput(_) => true,
        }
    }
    /// Keep available history controls inert during an edit; example: `action.dispatchable(&state)` after a click.
    pub fn dispatchable(self, state: &GuiCommandState) -> bool {
        let history = matches!(self, Self::Undo | Self::Redo);
        self.enabled(state) && !(history && state.transaction_open)
    }
    pub(crate) fn next_label(self, state: &GuiCommandState) -> Option<&str> {
        match self {
            Self::Undo => state.undo_label.as_deref(),
            Self::Redo => state.redo_label.as_deref(),
            _ => None,
        }
    }
}
#[cfg(test)]
#[path = "../tests/commands/unit.rs"]
mod tests;
