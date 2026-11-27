// SPDX-License-Identifier: GPL-2.0-only
//! The Tools resource panel follows the Go desktop's dockable browser handoff.
use crate::GuiShellAction;
use crate::dockable_lifecycle::{DockRegistry, DockableState};
use crate::dockable_window::DockableWindow;
use crate::icons::{GuiRibbonIcons, IconGlyph};
use imgui::{Condition, Ui};

const TITLE: &str = "Team Web";
const RESOURCE: &str = "https://docs.oblikovati.org/";

#[derive(Default)]
pub(crate) struct TeamWebWindow {
    open: DockableState,
}

impl DockableWindow for TeamWebWindow {}

impl TeamWebWindow {
    pub(crate) fn register_dockable(&self, registry: &mut DockRegistry) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.open)
    }

    pub(crate) fn ribbon_button(&mut self, ui: &Ui, icons: &GuiRibbonIcons) -> bool {
        let clicked = icons.button(
            ui,
            "Open Team Web",
            IconGlyph::TeamWeb,
            icons.appearance.single_cell(),
        );
        if clicked {
            self.open.set_open(true);
        }
        clicked
    }

    pub(crate) fn draw(&mut self, ui: &Ui, actions: &mut Vec<GuiShellAction>) {
        let mut open = self.open.is_open();
        if !open {
            return;
        }
        crate::widget_abi::window_size(ui, [360., 145.], Condition::FirstUseEver);
        Self::dockable_window(ui, TITLE, &mut open).build(|| {
            ui.text(RESOURCE);
            if crate::widget_abi::button(ui, "Open in Browser", [140., 26.]) {
                actions.push(GuiShellAction::OpenTeamWebInBrowser(RESOURCE));
            }
        });
        self.open.set_open(open);
    }
}
