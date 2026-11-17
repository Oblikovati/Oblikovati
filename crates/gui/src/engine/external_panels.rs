// SPDX-License-Identifier: GPL-2.0-only
//! Draw host-owned tool content through persistent presentation lifecycles.
use super::*;

pub(super) fn draw(
    docks: &crate::external_dock_window::ExternalDockWindows,
    ui: &imgui::Ui,
    state: &GuiCommandState,
    precise: &mut crate::sketch::precise_input::PreciseInputBar,
    actions: &mut Vec<GuiShellAction>,
) {
    crate::sketch::move_tool::panel(ui, state, precise, actions, &docks.sketch_move);
    crate::sketch::inspect::measure_panel(ui, state, actions, &docks.measure);
    crate::face_properties::draw(ui, state, actions, &docks.face);
    crate::sketch::region_properties::panel(ui, state, actions, &docks.region);
}
