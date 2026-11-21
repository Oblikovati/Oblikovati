// SPDX-License-Identifier: GPL-2.0-only
//! Curvature context rows and the owner's preview controls.
use crate::{
    GuiCommandState, GuiShellAction, dockable_lifecycle::DockableState,
    dockable_window::DockableWindow,
};
use imgui::Ui;

/// Two positions on the curvature dialog's Minimum-to-Maximum sliders.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiCurvatureSliders {
    pub density: u8,
    pub scale: u8,
}

/// Display state for one current owner dialog.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct GuiCurvatureSettings {
    pub token: u64,
    pub applied: GuiCurvatureSliders,
    pub preview: GuiCurvatureSliders,
}

/// An operation on one curve context or an open settings dialog.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiCurvatureAction {
    Toggle {
        context: u64,
    },
    Setup {
        context: u64,
    },
    Preview {
        dialog: u64,
        sliders: GuiCurvatureSliders,
    },
    Apply {
        dialog: u64,
        close: bool,
    },
    Cancel {
        dialog: u64,
    },
}

impl GuiCurvatureAction {
    pub(crate) fn enabled(self, state: &GuiCommandState) -> bool {
        match self {
            Self::Toggle { context } | Self::Setup { context } => {
                state.sketch_context.as_ref().is_some_and(|menu| {
                    menu.token == context
                        && menu
                            .curvature
                            .is_some_and(|shown| shown || matches!(self, Self::Toggle { .. }))
                })
            }
            Self::Preview { dialog, .. } | Self::Cancel { dialog } => state
                .curvature_settings
                .is_some_and(|view| view.token == dialog),
            Self::Apply { dialog, .. } => state
                .curvature_settings
                .is_some_and(|view| view.token == dialog && view.applied != view.preview),
        }
    }
}

pub(super) fn context_menu(ui: &Ui, state: &GuiCommandState, actions: &mut Vec<GuiShellAction>) {
    let Some((context, shown)) = state
        .sketch_context
        .as_ref()
        .and_then(|menu| menu.curvature.map(|value| (menu.token, value)))
    else {
        return;
    };
    if ui
        .menu_item_config("Display Curvature")
        .selected(shown)
        .build()
    {
        actions.push(GuiShellAction::SketchCurvature(
            GuiCurvatureAction::Toggle { context },
        ));
    }
    if shown && ui.menu_item("Setup Curvature Display") {
        actions.push(GuiShellAction::SketchCurvature(GuiCurvatureAction::Setup {
            context,
        }));
    }
}

const TITLE: &str = "Curvature Settings";

#[derive(Default)]
pub(crate) struct CurvatureSettingsWindow {
    open: DockableState,
    visible_token: Option<u64>,
}

impl DockableWindow for CurvatureSettingsWindow {}

impl CurvatureSettingsWindow {
    pub(crate) fn register_dockable(
        &self,
        registry: &mut crate::dockable_lifecycle::DockRegistry,
    ) -> Result<(), String> {
        registry.register::<Self>(TITLE, &self.open)
    }

    pub(crate) fn panel(
        &mut self,
        ui: &Ui,
        settings: Option<GuiCurvatureSettings>,
        actions: &mut Vec<GuiShellAction>,
    ) {
        let Some(settings) = settings else {
            self.visible_token = None;
            self.open.set_open(false);
            return;
        };
        let focus = self.visible_token.replace(settings.token) != Some(settings.token);
        if focus {
            self.open.set_open(true);
        }
        Self::center_dockable(ui, TITLE, [410., 165.]);
        let mut open = self.open.is_open();
        if open {
            Self::dockable_window(ui, TITLE, &mut open)
                .focused(focus)
                .build(|| controls(ui, settings, actions));
        }
        if !open {
            actions.push(GuiShellAction::SketchCurvature(
                GuiCurvatureAction::Cancel {
                    dialog: settings.token,
                },
            ));
        }
    }
}

fn controls(ui: &Ui, settings: GuiCurvatureSettings, actions: &mut Vec<GuiShellAction>) {
    let mut sliders = settings.preview;
    slider(ui, "Density", &mut sliders.density);
    slider(ui, "Scale", &mut sliders.scale);
    let dialog = settings.token;
    if sliders != settings.preview {
        actions.push(GuiShellAction::SketchCurvature(
            GuiCurvatureAction::Preview { dialog, sliders },
        ));
    }
    let buttons = ui.begin_disabled(settings.applied == settings.preview);
    for (label, close) in [("OK", true), ("Apply", false)] {
        if ui.button(label) {
            actions.push(GuiShellAction::SketchCurvature(GuiCurvatureAction::Apply {
                dialog,
                close,
            }));
        }
        ui.same_line();
    }
    buttons.end();
    if ui.button("Cancel")
        || (ui.is_window_focused() && ui.is_key_pressed_no_repeat(imgui::Key::Escape))
    {
        actions.push(GuiShellAction::SketchCurvature(
            GuiCurvatureAction::Cancel { dialog },
        ));
    }
}

fn slider(ui: &Ui, name: &str, position: &mut u8) {
    ui.text("Minimum");
    ui.same_line();
    ui.set_next_item_width(200.);
    ui.slider_config(format!("##curvature-{name}"), 0_u8, 100)
        .display_format("")
        .flags(imgui::SliderFlags::NO_INPUT)
        .build(position);
    ui.same_line();
    ui.text("Maximum");
}
