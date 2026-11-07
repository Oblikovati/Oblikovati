// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction, widget_abi};
use imgui::Ui;
use oblikovati_model::sketch::SketchCurveFormat::{Centerline, Construction};
pub(crate) fn sketch_format_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    icons: &crate::icons::GuiRibbonIcons,
    actions: &mut Vec<GuiShellAction>,
) {
    let modes = &state.sketch_formats;
    for (format, y, armed) in [
        (Construction, 0., modes.construction),
        (Centerline, 20., modes.centerline),
    ] {
        icons.cursor(ui, origin, [0., y]);
        let action = GuiShellAction::ToggleSketchFormat(format);
        let disabled = ui.begin_disabled(!action.enabled(state));
        let lit = ui.style_color(imgui::StyleColor::ButtonActive);
        let highlight = armed.then(|| ui.push_style_color(imgui::StyleColor::Button, lit));
        let glyph = if format == Construction {
            crate::icons::IconGlyph::Construction
        } else {
            crate::icons::IconGlyph::Centerline
        };
        let clicked = icons.button(ui, format.label(), glyph, [108., 18.]);
        drop(highlight);
        drop(disabled);
        if clicked && action.dispatchable(state) {
            actions.push(action);
        }
    }
    icons.cursor(ui, origin, [0., 40.]);
    if widget_abi::toggle_button(
        ui,
        "Center Point",
        icons.cell([108., 18.]),
        state.center_point_format,
    ) {
        actions.push(GuiShellAction::ToggleCenterPointFormat);
    }
}
