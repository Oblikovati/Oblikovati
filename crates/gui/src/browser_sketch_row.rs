// SPDX-License-Identifier: GPL-2.0-only
//! One sketch's browser row: choosing, opening and its menu.
use super::*;
pub(super) fn sketch_row(
    ui: &Ui,
    state: &GuiCommandState,
    row: &GuiBrowserSketch,
    actions: &mut Vec<GuiShellAction>,
) {
    let _sketch = ui.push_id(format!("sketch-{}", row.id.0));
    let action = GuiShellAction::EditSketch {
        document: state.document.unwrap_or(0),
        sketch: row.id,
    };
    if sketch_selectable(
        ui,
        row,
        state
            .browser
            .as_ref()
            .is_some_and(|browser| browser.selected_sketches.contains(&row.id)),
    ) {
        sketch_activation(ui, state, row.id, &action, actions);
    }
    sketch_context_menu(ui, state, row, action, actions);
}
fn sketch_activation(
    ui: &Ui,
    state: &GuiCommandState,
    sketch: DocumentSketchId,
    edit: &GuiShellAction,
    actions: &mut Vec<GuiShellAction>,
) {
    let select = GuiShellAction::SelectBrowserSketch {
        document: state.document.unwrap_or(0),
        sketch,
        // #3811: edit activation must retain its sketch even after a modified first click.
        toggle: !ui.is_mouse_double_clicked(MouseButton::Left)
            && (ui.io().key_shift || ui.io().key_ctrl),
    };
    if select.dispatchable(state) {
        actions.push(select);
    }
    if ui.is_mouse_double_clicked(MouseButton::Left) && edit.dispatchable(state) {
        actions.push(edit.clone());
    }
}
fn sketch_selectable(ui: &Ui, row: &GuiBrowserSketch, selected: bool) -> bool {
    let position = ui.cursor_screen_pos();
    let activated = crate::widget_abi::selectable(ui, "##sketch-row", selected, true);
    // Literal labels are drawn separately so imported names containing ## remain visible.
    crate::widget_abi::literal_text(
        ui,
        position,
        ui.style_color(StyleColor::Text),
        &sketch_caption(row),
    );
    if ui.is_item_hovered() {
        ui.tooltip_text(sketch_tooltip(row));
    }
    activated
}
fn sketch_context_menu(
    ui: &Ui,
    state: &GuiCommandState,
    row: &GuiBrowserSketch,
    action: GuiShellAction,
    actions: &mut Vec<GuiShellAction>,
) {
    let select = GuiShellAction::SelectBrowserSketch {
        document: state.document.unwrap_or(0),
        sketch: row.id,
        toggle: false,
    };
    // An open sketch keeps what its tools chose; a right-click there only asks for the menu.
    let may_choose =
        select.dispatchable(state) && state.authoring != crate::GuiAuthoringContext::Sketch;
    if may_choose && ui.is_item_hovered() && ui.is_mouse_clicked(imgui::MouseButton::Right) {
        actions.push(select);
    }
    let Some(_popup) = ui.begin_popup_context_item() else {
        return;
    };
    if ui
        .menu_item_config("Edit Sketch")
        .enabled(action.dispatchable(state))
        .build()
    {
        actions.push(action);
    }
    for listed in ["Redefine", "Share Sketch", "Properties", "Measure"] {
        crate::application::workspace_layout::browser_view::muted_menu_item(ui, listed);
    }
    sketch_visibility_menu(ui, state, row, actions);
    crate::application::workspace_layout::browser_view::muted_menu_item(ui, "Export Sketch As");
}
fn sketch_visibility_menu(
    ui: &Ui,
    state: &GuiCommandState,
    row: &GuiBrowserSketch,
    actions: &mut Vec<GuiShellAction>,
) {
    let visibility = GuiShellAction::SetSketchVisibility {
        document: state.document.unwrap_or(0),
        sketch: row.id,
        visible: !row.visible,
    };
    if ui
        .menu_item_config("Visibility")
        .selected(row.visible)
        .enabled(visibility.dispatchable(state))
        .build()
    {
        actions.push(visibility);
    }
}
fn sketch_caption(row: &GuiBrowserSketch) -> String {
    match &row.inspection {
        Ok(summary) => format!(
            "{}  [DOF {}{}]",
            row.name,
            summary.dof,
            if summary.healthy { "" } else { ", sick" }
        ),
        Err(_) => format!("{}  [unavailable]", row.name),
    }
}
fn sketch_tooltip(row: &GuiBrowserSketch) -> String {
    match &row.inspection {
        Ok(summary) => format!(
            "{} plane, {} entities. Double-click to edit.",
            summary.plane, summary.entity_count
        ),
        Err(cause) => format!("{}: {cause}", row.name),
    }
}

#[cfg(test)]
#[path = "../tests/browser/unit.rs"]
mod tests;
