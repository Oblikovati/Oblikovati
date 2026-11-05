// SPDX-License-Identifier: GPL-2.0-only
use crate::SketchPlacementTool;
use oblikovati_gui::GuiSketchTool;
/// Select the existing placement engine; example: `gui_placement_tool(GuiSketchTool::Circle)`.
pub(super) fn gui_placement_tool(tool: GuiSketchTool) -> SketchPlacementTool {
    match tool {
        GuiSketchTool::Rectangle => SketchPlacementTool::Rectangle,
        GuiSketchTool::Circle => SketchPlacementTool::Circle,
        GuiSketchTool::Point => SketchPlacementTool::Point,
        GuiSketchTool::CenterPointArc => SketchPlacementTool::CenterPointArc,
        GuiSketchTool::Ellipse => SketchPlacementTool::Ellipse,
        GuiSketchTool::CenterRectangle => SketchPlacementTool::CenterRectangle,
        GuiSketchTool::ThreePointRectangle => SketchPlacementTool::ThreePointRectangle,
    }
}
