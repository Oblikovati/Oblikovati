// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_model::sketch::SketchCurveFormat::{Centerline, Construction};
pub(crate) fn sketch_format_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    use crate::icons::IconGlyph;
    let entries = [
        (
            "Construction",
            GuiShellAction::ToggleSketchFormat(Construction),
            IconGlyph::Construction,
            state.sketch_formats.construction,
        ),
        (
            "Centerline",
            GuiShellAction::ToggleSketchFormat(Centerline),
            IconGlyph::Centerline,
            state.sketch_formats.centerline,
        ),
        (
            "Center Point",
            GuiShellAction::ToggleCenterPointFormat,
            IconGlyph::CenterPoint,
            state.center_point_format,
        ),
    ];
    for (index, (label, action, glyph, active)) in entries.into_iter().enumerate() {
        let (position, size) =
            if icons.appearance == crate::ribbon_command_layout::RibbonAppearance::Normal {
                ([0., index as f32 * 22.], [108., 20.])
            } else {
                icons.appearance.cell(index)
            };
        icons.cursor(ui, origin, position);
        let disabled = ui.begin_disabled(!action.dispatchable(state));
        let lit = ui.style_color(imgui::StyleColor::ButtonActive);
        let _highlight = active.then(|| ui.push_style_color(imgui::StyleColor::Button, lit));
        if icons.button(ui, label, glyph, size) && action.dispatchable(state) {
            actions.push(action);
        }
        drop(disabled);
    }
}
