// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiShellAction;
use imgui::{Condition, Ui};

pub(super) fn draw_status_bar(ui: &Ui, status: Option<&str>) {
    let display = ui.io().display_size;
    ui.window("Status")
        .position([0., display[1] - 28.], Condition::Always)
        .size([display[0], 28.], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| ui.text(status.unwrap_or("Press F1 for more help")));
}

pub(super) fn draw_document_tabs(
    ui: &Ui,
    documents: &[crate::GuiDocumentTab],
    selected: &mut Vec<GuiShellAction>,
) {
    if documents.is_empty() {
        return;
    }
    let display = ui.io().display_size;
    ui.window("Documents")
        .position([0., 112.], Condition::Always)
        .size([display[0], 30.], Condition::Always)
        .title_bar(false)
        .movable(false)
        .resizable(false)
        .build(|| {
            for document in documents {
                let label = if document.dirty {
                    format!("{} *##document-{}", document.name, document.id)
                } else {
                    format!("{}##document-{}", document.name, document.id)
                };
                if ui
                    .selectable_config(&label)
                    .selected(document.active)
                    .build()
                    && !document.active
                {
                    selected.push(GuiShellAction::ActivateDocument(document.id));
                }
                ui.same_line();
            }
        });
}

pub(super) fn draw_model_browser(ui: &Ui, tree: Option<&crate::GuiModelTree>) {
    ui.window("Model")
        .size([280., 420.], Condition::FirstUseEver)
        .build(|| {
            let Some(tree) = tree else {
                ui.text("No active part");
                return;
            };
            ui.text(&tree.document);
            ui.bullet_text("Origin");
            ui.bullet_text(format!("Parameters ({})", tree.parameters.len()));
            for parameter in &tree.parameters {
                ui.text(format!("  {parameter}"));
            }
            ui.bullet_text(format!("Sketches ({})", tree.sketches));
            ui.bullet_text(format!("Features ({})", tree.features.len()));
            for feature in &tree.features {
                ui.text(format!(
                    "  {} ({}) {}",
                    feature.name, feature.kind, feature.health
                ));
            }
            ui.bullet_text(format!("Bodies ({})", tree.bodies.len()));
            for body in &tree.bodies {
                let visibility = if body.visible { "visible" } else { "hidden" };
                ui.text(format!(
                    "  {} — {} faces, {}",
                    body.name, body.faces, visibility
                ));
            }
        });
}

pub(super) fn draw_command_window(ui: &Ui, status: Option<&str>) {
    ui.window("Command")
        .size([360., 130.], Condition::FirstUseEver)
        .build(|| {
            ui.text("Command");
            ui.separator();
            ui.text_wrapped(status.unwrap_or("Ready."));
        });
}
