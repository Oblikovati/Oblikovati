// SPDX-License-Identifier: GPL-2.0-only
use crate::{GuiCommandState, GuiShellAction, widget_abi};
use imgui::Ui;
use oblikovati_model::sketch::SketchCurveFormat::{Centerline, Construction};
pub(crate) fn sketch_format_controls(
    ui: &Ui,
    origin: [f32; 2],
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
) {
    let modes = &state.sketch_formats;
    for (format, y, armed) in [
        (Construction, 0., modes.construction),
        (Centerline, 20., modes.centerline),
    ] {
        widget_abi::cursor_position(ui, [origin[0], origin[1] + y]);
        let action = GuiShellAction::ToggleSketchFormat(format);
        let disabled = ui.begin_disabled(!action.enabled(state));
        let clicked = widget_abi::toggle_button(ui, format.label(), [108., 18.], armed);
        drop(disabled);
        if clicked && action.dispatchable(state) {
            actions.push(action);
        }
    }
}
