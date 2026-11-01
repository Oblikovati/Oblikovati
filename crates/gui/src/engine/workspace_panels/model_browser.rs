// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiShellAction;
use imgui::{Condition, StyleColor, Ui, WindowFlags};

use super::{GO_DOCK_HEADER, draw_dock_tab, workspace_layout};

/// Draw the document model tree; example: `draw(ui, tree, actions, open)`.
pub(crate) fn draw(
    ui: &Ui,
    tree: Option<&crate::GuiModelTree>,
    selected: &mut Vec<GuiShellAction>,
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
) {
    let layout = workspace_layout(ui.io().display_size);
    let mut open = workspace.model_browser_open;
    let _title = ui.push_style_color(StyleColor::TitleBg, GO_DOCK_HEADER);
    let _active = ui.push_style_color(StyleColor::TitleBgActive, GO_DOCK_HEADER);
    ui.window("##model-window")
        .flags(WindowFlags::NO_DOCKING | WindowFlags::NO_SAVED_SETTINGS)
        .opened(&mut open)
        .position(layout.model_position, Condition::Always)
        .size(layout.model_size, Condition::Always)
        .movable(false)
        .resizable(false)
        .build(|| draw_tree_or_placeholder(ui, tree, selected, workspace));
    workspace.model_browser_open = open;
    draw_dock_tab(ui, layout.model_position, "Model", true);
}

fn draw_tree_or_placeholder(
    ui: &Ui,
    tree: Option<&crate::GuiModelTree>,
    selected: &mut Vec<GuiShellAction>,
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
) {
    let Some(tree) = tree else {
        ui.bullet_text("(no document)");
        return;
    };
    draw_tree(ui, tree, selected, workspace);
}

fn draw_tree(
    ui: &Ui,
    tree: &crate::GuiModelTree,
    selected: &mut Vec<GuiShellAction>,
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
) {
    let Some(_document) = ui
        .tree_node_config(&tree.document)
        .default_open(true)
        .push()
    else {
        return;
    };
    draw_origin(ui);
    draw_parameters(ui, &tree.parameters);
    draw_sketches(ui, tree.sketches);
    draw_features(ui, &tree.features);
    draw_bodies(ui, &tree.bodies, selected, workspace);
}

fn draw_origin(ui: &Ui) {
    if let Some(_origin) = ui.tree_node("Origin") {
        ui.bullet_text("XY Plane");
        ui.bullet_text("XZ Plane");
        ui.bullet_text("YZ Plane");
    }
}

fn draw_parameters(ui: &Ui, parameters: &[String]) {
    if parameters.is_empty() {
        ui.bullet_text("Parameters");
        return;
    }
    let label = format!("Parameters ({})", parameters.len());
    let Some(_parameters) = ui.tree_node(&label) else {
        return;
    };
    for parameter in parameters {
        ui.bullet_text(parameter);
    }
}

fn draw_sketches(ui: &Ui, count: usize) {
    if count == 0 {
        ui.bullet_text("Sketches");
        return;
    }
    let label = format!("Sketches ({count})");
    let Some(_sketches) = ui.tree_node(&label) else {
        return;
    };
    for index in 1..=count {
        ui.bullet_text(format!("Sketch {index}"));
    }
}

fn draw_features(ui: &Ui, features: &[crate::GuiModelFeature]) {
    if features.is_empty() {
        ui.bullet_text("Features");
        return;
    }
    let label = format!("Features ({})", features.len());
    let Some(_features) = ui.tree_node(&label) else {
        return;
    };
    for feature in features {
        ui.bullet_text(feature_label(feature));
    }
}

pub(super) fn feature_label(feature: &crate::GuiModelFeature) -> String {
    format!("{} ({}) — {}", feature.name, feature.kind, feature.health)
}

fn draw_bodies(
    ui: &Ui,
    bodies: &[crate::GuiModelBody],
    selected: &mut Vec<GuiShellAction>,
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
) {
    if bodies.is_empty() {
        ui.bullet_text("Bodies");
        return;
    }
    let label = format!("Bodies ({})", bodies.len());
    let Some(_bodies) = ui.tree_node_config(&label).default_open(true).push() else {
        return;
    };
    for body in bodies {
        draw_body_row(ui, body, selected, workspace);
    }
}

fn draw_body_row(
    ui: &Ui,
    body: &crate::GuiModelBody,
    selected: &mut Vec<GuiShellAction>,
    workspace: &mut super::super::workspace_shell::WorkspaceShellState,
) {
    let mut visible = body.visible;
    if ui.checkbox(format!("##body-visible-{}", body.index), &mut visible) {
        selected.push(GuiShellAction::SetBodyVisible {
            index: body.index,
            visible,
        });
    }
    workspace.record_target(
        &format!("Model.BodyVisibility.{}", body.index),
        true,
        ui.item_rect_min(),
        ui.item_rect_max(),
    );
    ui.same_line();
    ui.text(format!("{} — {} faces", body.name, body.faces));
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn feature_labels_keep_name_kind_and_health() {
        assert_eq!(
            feature_label(&crate::GuiModelFeature {
                name: "Extrude1".into(),
                kind: "Extrude".into(),
                health: "Healthy".into(),
            }),
            "Extrude1 (Extrude) — Healthy"
        );
    }
}
