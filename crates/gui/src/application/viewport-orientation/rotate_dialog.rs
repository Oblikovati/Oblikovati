// SPDX-License-Identifier: GPL-2.0-only
//! Rotate at Angle: each press turns the view at once, and only OK keeps the turn.
use super::*;
use crate::dockable_window::DockableWindow;

const ROTATE_TITLE: &str = "Incremental View Rotate";
const ROTATE_SIZE: [f32; 2] = [300., 170.];
pub(super) const DEFAULT_INCREMENT: f32 = 10.;
/// Each direction button: a turn about the screen's axes, as yaw, pitch and roll multiples.
const ROTATE_BUTTONS: [(&str, [f64; 3]); 6] = [
    ("X+", [0., 1., 0.]),
    ("X-", [0., -1., 0.]),
    ("Y+", [1., 0., 0.]),
    ("Y-", [-1., 0., 0.]),
    ("Z+", [0., 0., 1.]),
    ("Z-", [0., 0., -1.]),
];
impl NavigationBar {
    /// The rotate dialog: each press turns the view at once, and only OK keeps the turn.
    pub(in super::super) fn rotate_dialog(
        &mut self,
        ui: &Ui,
        state: &GuiCommandState,
        actions: &mut Vec<GuiShellAction>,
    ) {
        if std::mem::take(&mut self.rotate_opening) {
            self.rotate_document = state.document;
            self.rotate_dock.set_open(self.rotate_document.is_some());
        }
        if self.rotate_document.is_none() || self.rotate_document != state.document {
            self.rotate_document = None;
            self.rotate_dock.set_open(false);
            return;
        }
        Self::center_dockable(ui, ROTATE_TITLE, ROTATE_SIZE);
        let mut open = true;
        let mut closed = false;
        Self::dockable_window(ui, ROTATE_TITLE, &mut open).build(|| {
            let increment = self.increment.get_or_insert(DEFAULT_INCREMENT);
            ui.input_float("Increment", increment).build();
            let radians = f64::from(*increment).to_radians();
            for (index, (label, [yaw, pitch, roll])) in ROTATE_BUTTONS.into_iter().enumerate() {
                if index % 2 == 1 {
                    ui.same_line();
                }
                if crate::widget_abi::button(ui, label, [44., 0.]) {
                    actions.extend(turn_commands([yaw, pitch, roll].map(|part| part * radians)));
                }
            }
            closed = rotate_ending_buttons(ui, actions);
        });
        if !open {
            actions.push(GuiShellAction::ViewCommand(
                GuiViewCommand::RestoreMarkedView,
            ));
        }
        if !open || closed {
            self.rotate_document = None;
            self.rotate_dock.set_open(false);
        }
    }
}
fn turn_commands([yaw, pitch, roll]: [f64; 3]) -> Vec<GuiShellAction> {
    let command = if roll != 0. {
        GuiViewCommand::Roll(roll)
    } else {
        GuiViewCommand::Orbit { yaw, pitch }
    };
    vec![GuiShellAction::ViewCommand(command)]
}
fn rotate_ending_buttons(ui: &Ui, actions: &mut Vec<GuiShellAction>) -> bool {
    let restore = GuiShellAction::ViewCommand(GuiViewCommand::RestoreMarkedView);
    if crate::widget_abi::button(ui, "Reset", [0.; 2]) {
        actions.push(restore.clone());
    }
    ui.same_line();
    if crate::widget_abi::button(ui, "OK", [0.; 2]) {
        return true;
    }
    ui.same_line();
    if crate::widget_abi::button(ui, "Cancel", [0.; 2])
        || ui.is_window_focused() && ui.is_key_pressed_no_repeat(Key::Escape)
    {
        actions.push(restore);
        return true;
    }
    false
}
