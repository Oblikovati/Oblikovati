// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiShellAction;
use imgui::{Condition, TreeNodeFlags, Ui};

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
    open: &mut bool,
) {
    ui.window("Model")
        .opened(open)
        .size([280., 420.], Condition::FirstUseEver)
        .build(|| {
            let Some(tree) = tree else {
                ui.text("No active part");
                return;
            };
            draw_model_tree(ui, tree, selected);
        });
}

fn draw_model_tree(ui: &Ui, tree: &crate::GuiModelTree, selected: &mut Vec<GuiShellAction>) {
    if !ui.collapsing_header(&tree.document, TreeNodeFlags::DEFAULT_OPEN) {
        return;
    }
    draw_model_origin(ui);
    draw_model_parameters(ui, &tree.parameters);
    draw_model_sketches(ui, tree.sketches);
    draw_model_features(ui, &tree.features);
    draw_model_bodies(ui, &tree.bodies, selected);
}

fn draw_model_origin(ui: &Ui) {
    if ui.collapsing_header("Origin", TreeNodeFlags::empty()) {
        ui.bullet_text("XY Plane");
        ui.bullet_text("XZ Plane");
        ui.bullet_text("YZ Plane");
    }
}

fn draw_model_parameters(ui: &Ui, parameters: &[String]) {
    if !ui.collapsing_header(
        format!("Parameters ({})", parameters.len()),
        TreeNodeFlags::empty(),
    ) {
        return;
    }
    for parameter in parameters {
        ui.bullet_text(parameter);
    }
}

fn draw_model_sketches(ui: &Ui, count: usize) {
    if ui.collapsing_header(format!("Sketches ({count})"), TreeNodeFlags::empty()) {
        for index in 1..=count {
            ui.bullet_text(format!("Sketch {index}"));
        }
    }
}

fn draw_model_features(ui: &Ui, features: &[crate::GuiModelFeature]) {
    if !ui.collapsing_header(
        format!("Features ({})", features.len()),
        TreeNodeFlags::empty(),
    ) {
        return;
    }
    for feature in features {
        ui.bullet_text(model_feature_label(feature));
    }
}

fn model_feature_label(feature: &crate::GuiModelFeature) -> String {
    format!("{} ({}) — {}", feature.name, feature.kind, feature.health)
}

fn draw_model_bodies(ui: &Ui, bodies: &[crate::GuiModelBody], selected: &mut Vec<GuiShellAction>) {
    if !ui.collapsing_header(
        format!("Bodies ({})", bodies.len()),
        TreeNodeFlags::DEFAULT_OPEN,
    ) {
        return;
    }
    for body in bodies {
        draw_body_row(ui, body, selected);
    }
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

pub(super) fn draw_command_window(ui: &Ui, status: Option<&str>, open: &mut bool) {
    ui.window("Command")
        .opened(open)
        .size([360., 130.], Condition::FirstUseEver)
        .build(|| {
            ui.text("Command");
            ui.separator();
            ui.text_wrapped(status.unwrap_or("Ready."));
        });
}

#[cfg(test)]
mod tests {
    use super::model_feature_label;
    use crate::GuiModelFeature;

    #[test]
    fn feature_rows_keep_name_kind_and_health() {
        let label = model_feature_label(&GuiModelFeature {
            name: "Extrude1".into(),
            kind: "Extrude".into(),
            health: "Healthy".into(),
        });
        assert_eq!(label, "Extrude1 (Extrude) — Healthy");
    }
}
