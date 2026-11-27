// SPDX-License-Identifier: GPL-2.0-only
//! Radial slot layout for the sketch canvas menu.
use crate::{
    GuiCommandState, GuiDimensionAction, GuiShellAction, GuiSketchConstraint, GuiSketchTool,
};
use imgui::Ui;
use oblikovati_api::wire::MarkingMenuView;

pub(super) fn draw(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    icons: &crate::icons::GuiRibbonIcons,
) {
    crate::marking_menu::draw(ui, menu, state, actions, entry, icons);
}
pub(super) fn draw_classic(
    ui: &Ui,
    menu: &MarkingMenuView,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    crate::marking_menu::draw_classic(ui, menu, state, actions, entry);
}
fn entry(id: &str) -> Option<crate::marking_menu::MarkingEntry> {
    action(id).map(crate::marking_menu::MarkingEntry::command)
}
pub(crate) const COMMANDS: [(&str, &str); 22] = [
    ("Sketch.Line", "Line"),
    ("Sketch.Circle", "Circle"),
    ("Sketch.Rectangle", "Rectangle"),
    ("Sketch.Arc", "Arc"),
    ("Sketch.Arc.CenterPoint", "Center Point Arc"),
    ("Sketch.Rectangle.ThreePoint", "Three Point Rectangle"),
    ("Sketch.Rectangle.Center", "Two Point Center Rectangle"),
    ("Sketch.Circle.ThreePoint", "Three Point Circle"),
    ("Sketch.Point", "Point"),
    ("Sketch.Ellipse", "Ellipse"),
    ("Sketch.Horizontal", "Horizontal"),
    ("Sketch.Vertical", "Vertical"),
    ("Sketch.Coincident", "Coincident"),
    ("Sketch.Fix", "Fix"),
    ("Sketch.Parallel", "Parallel"),
    ("Sketch.Perpendicular", "Perpendicular"),
    ("Sketch.Collinear", "Collinear"),
    ("Sketch.Equal", "Equal"),
    ("Sketch.Concentric", "Concentric"),
    ("Sketch.Tangent", "Tangent"),
    ("Sketch.Dimension", "Dimension"),
    ("Sketch.Finish", "Finish Sketch"),
];
pub(crate) fn action(id: &str) -> Option<(&'static str, GuiShellAction)> {
    Some(match id {
        "Sketch.Line" => ("Line", GuiShellAction::PlaceSketchTool(GuiSketchTool::Line)),
        "Sketch.Circle" => (
            "Circle",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::Circle),
        ),
        "Sketch.Rectangle" => (
            "Rectangle",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::Rectangle),
        ),
        "Sketch.Arc" => (
            "Arc",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::ThreePointArc),
        ),
        "Sketch.Arc.CenterPoint" => (
            "Center Point Arc",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::CenterPointArc),
        ),
        "Sketch.Rectangle.ThreePoint" => (
            "Three Point Rectangle",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::ThreePointRectangle),
        ),
        "Sketch.Rectangle.Center" => (
            "Two Point Center Rectangle",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::CenterRectangle),
        ),
        "Sketch.Circle.ThreePoint" => (
            "Three Point Circle",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::ThreePointCircle),
        ),
        "Sketch.Point" => (
            "Point",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::Point),
        ),
        "Sketch.Ellipse" => (
            "Ellipse",
            GuiShellAction::PlaceSketchTool(GuiSketchTool::Ellipse),
        ),
        "Sketch.Horizontal" => (
            "Horizontal",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Horizontal),
        ),
        "Sketch.Vertical" => (
            "Vertical",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Vertical),
        ),
        "Sketch.Coincident" => (
            "Coincident",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Coincident),
        ),
        "Sketch.Fix" => (
            "Fix",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Fix),
        ),
        "Sketch.Parallel" => (
            "Parallel",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Parallel),
        ),
        "Sketch.Perpendicular" => (
            "Perpendicular",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Perpendicular),
        ),
        "Sketch.Collinear" => (
            "Collinear",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Collinear),
        ),
        "Sketch.Equal" => (
            "Equal",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Equal),
        ),
        "Sketch.Concentric" => (
            "Concentric",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Concentric),
        ),
        "Sketch.Tangent" => (
            "Tangent",
            GuiShellAction::ConstrainSketch(GuiSketchConstraint::Tangent),
        ),
        "Sketch.Dimension" => (
            "Dimension",
            GuiShellAction::Dimension(GuiDimensionAction::Start),
        ),
        "Sketch.Finish" => ("Finish Sketch", GuiShellAction::FinishSketch),
        _ => return None,
    })
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn menu_commands_use_the_existing_sketch_actions() {
        assert_eq!(
            action("Sketch.Rectangle"),
            Some((
                "Rectangle",
                GuiShellAction::PlaceSketchTool(GuiSketchTool::Rectangle)
            ))
        );
        for (id, label) in COMMANDS {
            assert_eq!(action(id).map(|(name, _)| name), Some(label));
        }
        assert_eq!(
            action("Sketch.Arc"),
            Some((
                "Arc",
                GuiShellAction::PlaceSketchTool(GuiSketchTool::ThreePointArc)
            ))
        );
        assert_eq!(
            action("Sketch.Horizontal"),
            Some((
                "Horizontal",
                GuiShellAction::ConstrainSketch(GuiSketchConstraint::Horizontal)
            ))
        );
        assert_eq!(action("unknown"), None);
    }
}
