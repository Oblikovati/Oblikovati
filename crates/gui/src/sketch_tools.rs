// SPDX-License-Identifier: GPL-2.0-only
/// A ribbon choice for an existing owner placement engine; numeric settings remain in that engine.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiSketchTool {
    Line,
    Rectangle,
    Circle,
    Point,
    CenterPointArc,
    Ellipse,
    CenterRectangle,
    ThreePointRectangle,
    ThreePointCenterRectangle,
    ThreePointCircle,
    ThreePointArc,
}
pub(crate) const SECONDARY_SKETCH_TOOLS: [GuiSketchTool; 6] = [
    GuiSketchTool::Circle,
    GuiSketchTool::Point,
    GuiSketchTool::CenterPointArc,
    GuiSketchTool::Ellipse,
    GuiSketchTool::CenterRectangle,
    GuiSketchTool::ThreePointRectangle,
];
impl GuiSketchTool {
    /// The fuller name a dropdown has room for; example: `GuiSketchTool::Circle.menu_label()`.
    pub fn menu_label(self) -> &'static str {
        match self {
            Self::Circle => "Center Point Circle",
            Self::ThreePointRectangle => "Three Point Rectangle",
            Self::ThreePointCenterRectangle => "Three Point Center Rectangle",
            other => other.label(),
        }
    }
    /// Supply the compact ribbon caption; example: `GuiSketchTool::Circle.label()`.
    pub fn label(self) -> &'static str {
        match self {
            Self::Line => "Line",
            Self::Rectangle => "Rectangle",
            Self::Circle => "Circle",
            Self::Point => "Point",
            Self::CenterPointArc => "Center Arc",
            Self::Ellipse => "Ellipse",
            Self::CenterRectangle => "Center Rect.",
            Self::ThreePointRectangle => "3-Pt Rect.",
            Self::ThreePointCenterRectangle => "3-Pt Center Rect.",
            Self::ThreePointCircle => "3-Point Circle",
            Self::ThreePointArc => "3-Point Arc",
        }
    }
}
