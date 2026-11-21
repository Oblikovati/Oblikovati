// SPDX-License-Identifier: GPL-2.0-only
//! Dockable analytic properties for one named face.
use crate::{GuiCommandState, GuiShellAction, dockable_window::DockableWindow};

/// A measured face's stable identity and analytic area.
/// Example: `GuiFaceProperties { key: "face".into(), area_mm2: 100. }`.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiFaceProperties {
    pub key: String,
    pub area_mm2: f64,
}

pub(crate) fn draw(
    ui: &imgui::Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    dock: &crate::external_dock_window::ExternalDockWindow,
) {
    if !dock.is_open() {
        return;
    }
    let Some(face) = &state.face_properties else {
        return;
    };
    let [width, _] = ui.io().display_size;
    crate::external_dock_window::ExternalDockWindow::fit_dockable(
        ui,
        "Face Properties",
        [width - 390., 220.],
        [360., 190.],
    );
    let mut open = true;
    crate::external_dock_window::ExternalDockWindow::dockable_window(
        ui,
        "Face Properties",
        &mut open,
    )
    .build(|| {
        ui.text("Type: Face");
        ui.text(format!("Area: {:.3} mm²", face.area_mm2));
        ui.text_wrapped(format!("Reference: {}", face.key));
    });
    if !open {
        actions.push(GuiShellAction::CloseFaceProperties);
    }
}
