// SPDX-License-Identifier: GPL-2.0-only
use crate::SketchPlacementTool as Tool;
use oblikovati_gui::GuiSketchTool as Gui;
use oblikovati_model::sketch::{CircumcircleOptions, SketchConstructionPolicy};
/// Select the existing placement engine; example: `gui_placement_tool(Gui::Circle, 1.)`.
pub(super) fn gui_placement_tool(tool: Gui, scale: f64) -> Result<Tool, String> {
    let options = gui_circular_accuracy(scale)?;
    Ok(match tool {
        Gui::ThreePointCircle => Tool::ThreePointCircle { options },
        Gui::ThreePointArc => Tool::ThreePointArc { options },
        Gui::Line => Tool::Line,
        Gui::Rectangle => Tool::Rectangle,
        Gui::Circle => Tool::Circle,
        Gui::Point => Tool::Point,
        Gui::CenterPointArc => Tool::CenterPointArc,
        Gui::Ellipse => Tool::Ellipse,
        Gui::CenterRectangle => Tool::CenterRectangle,
        Gui::ThreePointRectangle => Tool::ThreePointRectangle,
    })
}
fn gui_circular_accuracy(scale: f64) -> Result<CircumcircleOptions, String> {
    SketchConstructionPolicy::default()
        .in_working_units(scale)
        .map_err(|cause| cause.to_string())
}
