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
            let Some(_tabs) = ui.tab_bar("##document-tabs") else {
                return;
            };
            for document in documents {
                draw_document_tab(ui, document, selected);
            }
        });
}

fn draw_document_tab(
    ui: &Ui,
    document: &crate::GuiDocumentTab,
    selected: &mut Vec<GuiShellAction>,
) {
    let mut open = true;
    let label = document_tab_label(document);
    let active = ui.tab_item_with_opened(label, &mut open).is_some();
    if active && !document.active {
        selected.push(GuiShellAction::ActivateDocument(document.id));
    }
    if !open {
        selected.push(GuiShellAction::CloseDocument(document.id));
    }
}

fn document_tab_label(document: &crate::GuiDocumentTab) -> String {
    let dirty = if document.dirty { " *" } else { "" };
    format!("{}{dirty}###document-{}", document.name, document.id)
}

pub(super) fn draw_model_browser(
    ui: &Ui,
    tree: Option<&crate::GuiModelTree>,
    selected: &mut Vec<GuiShellAction>,
) {
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
                draw_body_row(ui, body, selected);
            }
        });
}

fn draw_body_row(ui: &Ui, body: &crate::GuiModelBody, selected: &mut Vec<GuiShellAction>) {
    let mut visible = body.visible;
    if ui.checkbox(format!("##body-visible-{}", body.index), &mut visible) {
        selected.push(GuiShellAction::SetBodyVisible {
            index: body.index,
            visible,
        });
    }
    ui.same_line();
    ui.text(format!("{} — {} faces", body.name, body.faces));
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
