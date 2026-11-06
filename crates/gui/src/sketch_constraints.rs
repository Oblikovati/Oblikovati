// SPDX-License-Identifier: GPL-2.0-only
/// Ribbon choices for geometric constraints; the document owner checks and applies the picks.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSketchConstraint {
    Horizontal,
    Vertical,
    Coincident,
    Fix,
}
impl GuiSketchConstraint {
    /// Name a constraint command; example: `GuiSketchConstraint::Horizontal.label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Horizontal => "Horizontal",
            Self::Vertical => "Vertical",
            Self::Coincident => "Coincident",
            Self::Fix => "Fix",
        }
    }
}
