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
    if icons.appearance != crate::ribbon_command_layout::RibbonAppearance::Normal {
        icon_formats(ui, origin, state, icons, actions);
        return;
    }
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
    let action = GuiShellAction::ToggleCenterPointFormat;
    let disabled = ui.begin_disabled(!action.dispatchable(state));
    let clicked = widget_abi::toggle_button(
        ui,
        "Center Point",
        icons.cell([108., 18.]),
        state.center_point_format,
    );
    drop(disabled);
    if clicked && action.dispatchable(state) {
        actions.push(action);
    }
}

fn icon_formats(
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
        let (position, size) = icons.appearance.cell(index);
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
