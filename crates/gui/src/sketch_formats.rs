// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction, widget_abi};
use imgui::Ui;
use oblikovati_model::sketch::SketchCurveFormat::{Centerline, Construction};
pub(crate) fn sketch_format_controls(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    let modes = &state.sketch_formats;
    for (format, y, armed) in [
        (Construction, 59., modes.construction),
        (Centerline, 79., modes.centerline),
    ] {
        widget_abi::cursor_position(ui, [1084., y]);
        let action = GuiShellAction::ToggleSketchFormat(format);
        let disabled = ui.begin_disabled(!action.enabled(state));
        let clicked = widget_abi::toggle_button(ui, format.label(), [108., 18.], armed);
        drop(disabled);
        if clicked && action.dispatchable(state) {
            actions.push(action);
        }
    }
    widget_abi::cursor_position(ui, [1084., 125.]);
    ui.text("Format");
}
