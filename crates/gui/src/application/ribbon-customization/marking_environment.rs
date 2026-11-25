// SPDX-License-Identifier: GPL-2.0-only
//! Available profiles in the canvas menu editor.
use crate::GuiAuthoringContext;
use imgui::Ui;
use oblikovati_api::types::{self, Environment};

const CHOICES: [(&str, Option<Environment>); 7] = [
    ("2D Sketch", Some(types::SKETCH_ENVIRONMENT)),
    ("3D Sketch", None),
    ("Assembly", None),
    ("Drawing", None),
    ("Part", Some(types::BASE_ENVIRONMENT)),
    ("Presentation", None),
    ("Sheet Metal", None),
];

pub(super) fn for_context(context: GuiAuthoringContext) -> Environment {
    if context == GuiAuthoringContext::Part {
        types::BASE_ENVIRONMENT
    } else {
        types::SKETCH_ENVIRONMENT
    }
}

pub(super) fn selector(ui: &Ui, selected: &mut Environment) -> bool {
    let preview = CHOICES
        .iter()
        .find_map(|(label, choice)| (*choice == Some(*selected)).then_some(*label));
    ui.set_next_item_width(190.);
    let Some(_menu) = ui.begin_combo("Environment", preview.unwrap_or("2D Sketch")) else {
        return false;
    };
    for (label, choice) in CHOICES {
        if ui
            .selectable_config(label)
            .selected(choice == Some(*selected))
            .disabled(choice.is_none())
            .build()
            && let Some(choice) = choice
        {
            *selected = choice;
        }
        if choice.is_none()
            && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
        {
            ui.tooltip_text("This marking menu is unavailable in this build.");
        }
    }
    true
}

pub(super) fn commands(environment: Environment) -> Vec<(&'static str, &'static str)> {
    if environment == types::BASE_ENVIRONMENT {
        crate::part_context_menu::marking_commands().collect()
    } else {
        crate::sketch::marking_menu::COMMANDS.to_vec()
    }
}
