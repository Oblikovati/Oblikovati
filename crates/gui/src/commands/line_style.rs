// SPDX-License-Identifier: GPL-2.0-only
/// How a sketch's lines are drawn; a part left unset draws as the application's default.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct GuiSketchLineStyle {
    pub color: Option<[u8; 3]>,
    pub line_type: Option<String>,
    /// In millimeters.
    pub weight: Option<f32>,
    pub scale: Option<f32>,
}
