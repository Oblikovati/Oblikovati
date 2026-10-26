// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Debug, PartialEq)]
pub enum GuiShellAction {
    Save,
    Undo,
    Redo,
    Home,
    PreviousView,
    NewPart,
    NewAssembly,
    NewDrawing,
    SketchXY,
    Rectangle,
    Circle,
    ExitSketch,
    Extrude,
    ActivateDocument(u64),
    CloseDocument(u64),
    JumpHistoryTo { document: u64, position: i64 },
    SetLengthUnit(String),
    SetAngleUnit(String),
    SetMassUnit(String),
    SetTimeUnit(String),
    SetLengthPrecision(i64),
    SetAnglePrecision(i64),
    SetLengthFormat(String),
    SetBodyVisible { index: i64, visible: bool },
    SetSelectionEnabled { kind: String, enabled: bool },
    MoveSelectionKind { from: usize, to: usize },
    SetGridSpacingCm(f64),
    SetGridVisible(bool),
    SetGridMajorEvery(i64),
    SetHeadsUpDisplay(bool),
    SetPointerInput(bool),
    SetDimensionInput(bool),
    SetSnapToPoints(bool),
    SetSnapToGrid(bool),
    SetRelaxMode(bool),
    SetAutoProjectOrigin(bool),
    SetSuppressFormatOverrides(bool),
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
    pub active: bool,
    pub button_style: u8,
}
/// A visible ribbon-control hit target; example: test code finds `Sketch.Create` before clicking it.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiControlTarget {
    pub command_id: String,
    pub center: [f32; 2],
    pub enabled: bool,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiRibbonPanel {
    pub name: String,
    pub controls: Vec<GuiRibbonControl>,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiRibbonTab {
    pub name: String,
    pub panels: Vec<GuiRibbonPanel>,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiModelFeature {
    pub name: String,
    pub kind: String,
    pub health: String,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiModelBody {
    pub index: i64,
    pub name: String,
    pub visible: bool,
    pub faces: usize,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiHistoryEntry {
    pub label: String,
    pub saved: bool,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiDocumentUnits {
    pub length_unit: String,
    pub angle_unit: String,
    pub mass_unit: String,
    pub time_unit: String,
    pub length_precision: i64,
    pub angle_precision: i64,
    pub length_format: String,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiDocumentHistory {
    pub document: u64,
    pub name: String,
    pub position: i64,
    pub entries: Vec<GuiHistoryEntry>,
}

#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiSketchPreferences {
    pub grid_spacing_cm: f64,
    pub grid_visible: bool,
    pub grid_major_every: i64,
    pub heads_up_display: bool,
    pub pointer_input: bool,
    pub dimension_input: bool,
    pub snap_to_points: bool,
    pub snap_to_grid: bool,
    pub relax_mode: bool,
    pub auto_project_origin: bool,
    pub suppress_format_overrides: bool,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiSelectionFilterEntry {
    pub kind: String,
    pub label: String,
    pub enabled: bool,
}
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiSelectionFilter {
    pub entries: Vec<GuiSelectionFilterEntry>,
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
    pub histories: Vec<GuiDocumentHistory>,
    pub document_units: Option<GuiDocumentUnits>,
    pub ribbon_key: String,
    pub ribbon_tabs: Vec<GuiRibbonTab>,
    pub model_tree: Option<GuiModelTree>,
    pub sketch_preferences: Option<GuiSketchPreferences>,
    pub selection_filter: Option<GuiSelectionFilter>,
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
    pub fn enabled(&self, state: &GuiCommandState) -> bool {
        match self {
            Self::Undo => state.undo_label.is_some(),
            Self::Redo => state.redo_label.is_some(),
            Self::Save
            | Self::Home
            | Self::PreviousView
            | Self::NewPart
            | Self::NewAssembly
            | Self::NewDrawing
            | Self::SketchXY
            | Self::Rectangle
            | Self::Circle
            | Self::ExitSketch
            | Self::Extrude
            | Self::ActivateDocument(_)
            | Self::CloseDocument(_)
            | Self::JumpHistoryTo { .. }
            | Self::SetLengthUnit(_)
            | Self::SetAngleUnit(_)
            | Self::SetMassUnit(_)
            | Self::SetTimeUnit(_)
            | Self::SetLengthPrecision(_)
            | Self::SetAnglePrecision(_)
            | Self::SetLengthFormat(_)
            | Self::SetBodyVisible { .. }
            | Self::SetSelectionEnabled { .. }
            | Self::MoveSelectionKind { .. }
            | Self::SetGridSpacingCm(_)
            | Self::SetGridVisible(_)
            | Self::SetGridMajorEvery(_)
            | Self::SetHeadsUpDisplay(_)
            | Self::SetPointerInput(_)
            | Self::SetDimensionInput(_)
            | Self::SetSnapToPoints(_)
            | Self::SetSnapToGrid(_)
            | Self::SetRelaxMode(_)
            | Self::SetAutoProjectOrigin(_)
            | Self::SetSuppressFormatOverrides(_) => true,
        }
    }
    /// Keep available history controls inert during an edit; example: `action.dispatchable(&state)` after a click.
    pub fn dispatchable(&self, state: &GuiCommandState) -> bool {
        let history = matches!(self, Self::Undo | Self::Redo);
        self.enabled(state) && !(history && state.transaction_open)
    }
}
#[cfg(test)]
#[path = "../tests/commands/unit.rs"]
mod tests;
