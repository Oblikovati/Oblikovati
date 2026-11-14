// SPDX-License-Identifier: GPL-2.0-only
//! The Region Properties dialog: loops the owner has accepted, and what they calculate to.
use crate::dockable_window::DockableWindow;
use crate::{GuiCommandState, GuiInspectAction, GuiShellAction};
use imgui::Ui;

/// Accepted loops and, once calculated, their result rows.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct GuiRegionPanel {
    pub loops: Vec<String>,
    pub results: Vec<String>,
}
impl DockableWindow for GuiRegionPanel {}
const SIZE: [f32; 2] = [360., 320.];

/// Draw the dialog; Calculate needs an accepted loop, as nothing else has a result.
pub(crate) fn panel(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some(panel) = &state.region_properties else {
        return;
    };
    let [width, _] = ui.io().display_size;
    GuiRegionPanel::fit_dockable(ui, "Region Properties", [width - SIZE[0] - 24., 220.], SIZE);
    let mut open = true;
    GuiRegionPanel::dockable_window(ui, "Region Properties", &mut open).build(|| {
        ui.text("Selected loops");
        for name in &panel.loops {
            ui.text(name);
        }
        ui.separator();
        let disabled = ui.begin_disabled(panel.loops.is_empty());
        if ui.button("Calculate") {
            actions.push(GuiShellAction::Inspect(GuiInspectAction::CalculateRegions));
        }
        drop(disabled);
        for row in &panel.results {
            ui.text(row);
        }
        ui.separator();
        if ui.button("Done") {
            actions.push(GuiShellAction::Inspect(
                GuiInspectAction::FinishRegionProperties,
            ));
        }
    });
    if !open {
        actions.push(GuiShellAction::Inspect(
            GuiInspectAction::FinishRegionProperties,
        ));
    }
}
