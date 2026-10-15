// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
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
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiDocumentTab {
    pub id: u64,
    pub name: String,
    pub dirty: bool,
    pub active: bool,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiModelFeature {
    pub name: String,
    pub kind: String,
    pub health: String,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiModelTree {
    pub document: String,
    pub parameters: Vec<String>,
    pub sketches: usize,
    pub features: Vec<GuiModelFeature>,
    pub bodies: usize,
}
/// A bounded owner snapshot; the GUI holds no document history or transaction token.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiCommandState {
    pub undo_label: Option<String>,
    pub redo_label: Option<String>,
    pub transaction_open: bool,
    pub documents: Vec<GuiDocumentTab>,
    pub model_tree: Option<GuiModelTree>,
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
            | Self::ActivateDocument(_) => true,
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
