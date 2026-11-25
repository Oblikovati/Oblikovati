// SPDX-License-Identifier: GPL-2.0-only
//! Available profiles in the canvas menu editor.
use crate::GuiAuthoringContext;
use imgui::{ComboBoxFlags, Ui};
use oblikovati_api::types::{self, Environment};

const CHOICES: [(&str, Option<Environment>); 7] = [
    ("2D Sketch", Some(types::SKETCH_ENVIRONMENT)),
    ("3D Sketch", Some(types::SKETCH3_D_ENVIRONMENT)),
    ("Assembly", None),
    ("Drawing", None),
    ("Part", Some(types::BASE_ENVIRONMENT)),
    ("Presentation", None),
    ("Sheet Metal", None),
];

const SKETCH_SUB_ENVIRONMENTS: &[&str] = &[
    "Arc",
    "Circle",
    "Constraint Glyphs",
    "Ctrl + Right Click Menu",
    "Dimension",
    "Ellipse",
    "Equation Curve",
    "Line",
    "None Selected",
    "Point",
    "Projected Geometry",
    "Sketch Block",
    "Sketch Pattern",
    "Spline",
    "Text",
];

const SKETCH_3D_COMMANDS: &[(&str, &str)] = &[
    ("Sketch3D.Line", "Line"),
    ("Sketch3D.Bend", "Bend"),
    ("Sketch3D.Arc", "Three Point Arc"),
    ("edit.undo", "Undo"),
    ("Sketch3D.Helix", "Helical Curve"),
    ("Sketch3D.Dimension", "General Dimension"),
    ("Sketch3D.Include", "Include Geometry"),
    ("Sketch3D.Finish", "Finish 3D Sketch"),
];

const SKETCH_3D_SUB_ENVIRONMENTS: &[&str] = &[
    "3D Line, 3D Arc",
    "3D Point",
    "Ctrl + Right Click Menu",
    "Equation Curve",
    "Helical Curve",
    "None Selected",
    "Projected Curves",
    "Spline",
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
    match environment {
        types::BASE_ENVIRONMENT => crate::part_context_menu::marking_commands().collect(),
        types::SKETCH3_D_ENVIRONMENT => SKETCH_3D_COMMANDS.to_vec(),
        _ => crate::sketch::marking_menu::COMMANDS.to_vec(),
    }
}

pub(super) fn sub_environment_selector(ui: &Ui, environment: Environment) -> bool {
    let choices = match environment {
        types::SKETCH_ENVIRONMENT => SKETCH_SUB_ENVIRONMENTS,
        types::SKETCH3_D_ENVIRONMENT => SKETCH_3D_SUB_ENVIRONMENTS,
        _ => &[],
    };
    let _disabled = ui.begin_disabled(choices.is_empty());
    ui.set_next_item_width(190.);
    let Some(_menu) = ui.begin_combo_with_flags(
        "Sub-Environment",
        "None Selected",
        ComboBoxFlags::HEIGHT_LARGEST,
    ) else {
        return false;
    };
    for &label in choices {
        let selected = label == "None Selected";
        ui.selectable_config(label)
            .selected(selected)
            .disabled(!selected)
            .build();
        if !selected && ui.is_item_hovered_with_flags(imgui::ItemHoveredFlags::ALLOW_WHEN_DISABLED)
        {
            ui.tooltip_text("This sub-environment has no editable marking menu.");
        }
    }
    true
}
