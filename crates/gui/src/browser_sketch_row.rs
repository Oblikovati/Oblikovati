// SPDX-License-Identifier: GPL-2.0-only
//! One sketch's browser row: choosing, opening and its menu.
use super::*;
use crate::application::model_browser::relationships::RelatedRow;
pub(super) fn sketch_row(
    ui: &Ui,
    state: &GuiCommandState,
    row: &GuiBrowserSketch,
    view: &mut BrowserView,
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
    #[cfg(test)]
    view.test_sketch_bounds
        .insert(row.id.0, [ui.item_rect_min(), ui.item_rect_max()]);
    sketch_context_menu(ui, state, row, view, action, actions);
    block_instances(ui, state, row.id, actions);
    rectangular_patterns(ui, state, row.id, actions);
}
fn rectangular_patterns(
    ui: &Ui,
    state: &GuiCommandState,
    sketch: DocumentSketchId,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(browser) = &state.browser else {
        return;
    };
    ui.indent();
    for row in browser
        .rectangular_patterns
        .iter()
        .filter(|row| row.sketch == sketch)
    {
        let _id = ui.push_id(format!("rectangular-pattern-{}", row.pattern.id.0));
        let position = ui.cursor_screen_pos();
        let action = GuiShellAction::OpenSketchPattern {
            sketch,
            pattern: row.pattern.id,
        };
        if crate::widget_abi::selectable(ui, "##pattern", false, true)
            && ui.is_mouse_double_clicked(MouseButton::Left)
            && action.dispatchable(state)
        {
            actions.push(action.clone());
        }
        crate::widget_abi::literal_text(
            ui,
            position,
            ui.style_color(StyleColor::Text),
            &format!("Rectangular Pattern{}", row.ordinal),
        );
        if let Some(_popup) = ui.begin_popup_context_item()
            && ui
                .menu_item_config("Edit Pattern")
                .enabled(action.dispatchable(state))
                .build()
        {
            actions.push(action);
        }
    }
    ui.unindent();
}
fn block_instances(
    ui: &Ui,
    state: &GuiCommandState,
    sketch: DocumentSketchId,
    actions: &mut Vec<GuiShellAction>,
) {
    let Some(browser) = &state.browser else {
        return;
    };
    ui.indent();
    for instance in browser
        .block_instances
        .iter()
        .filter(|instance| instance.sketch == sketch)
    {
        block_instance_row(ui, state, instance, actions);
    }
    ui.unindent();
}
fn block_instance_row(
    ui: &Ui,
    state: &GuiCommandState,
    instance: &GuiBrowserBlockInstance,
    actions: &mut Vec<GuiShellAction>,
) {
    let action = GuiShellAction::SelectBrowserBlock {
        document: state.document.unwrap_or(0),
        sketch: instance.sketch,
        instance: instance.id,
        toggle: ui.io().key_shift || ui.io().key_ctrl,
    };
    let selected = state
        .sketch_selection
        .entities
        .contains(&(instance.sketch, instance.id));
    let _id = ui.push_id(format!("block-{}", instance.id.0));
    let position = ui.cursor_screen_pos();
    let disabled = ui.begin_disabled(!action.dispatchable(state));
    if crate::widget_abi::selectable(ui, "##block-instance", selected, true) {
        actions.push(action);
    }
    crate::widget_abi::literal_text(
        ui,
        position,
        ui.style_color(StyleColor::Text),
        &format!("{}:{}", instance.definition, instance.ordinal),
    );
    drop(disabled);
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
    view: &mut BrowserView,
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
    let edit_selected = ui
        .menu_item_config("Edit Sketch")
        .enabled(action.dispatchable(state))
        .build();
    #[cfg(test)]
    {
        view.test_edit_bounds = Some([ui.item_rect_min(), ui.item_rect_max()]);
    }
    if edit_selected {
        actions.push(action);
    }
    use crate::application::workspace_layout::browser_view::muted_menu_item;
    for listed in ["Redefine", "Share Sketch"] {
        muted_menu_item(ui, listed);
    }
    if ui.menu_item("Properties") {
        view.geometry.request(row.id, row.line_style.clone());
    }
    muted_menu_item(ui, "Measure");
    sketch_visibility_menu(ui, state, row, actions);
    #[cfg(test)]
    {
        view.test_visibility_bounds = Some([ui.item_rect_min(), ui.item_rect_max()]);
    }
    if ui
        .menu_item_config("Dimension Visibility")
        .selected(row.dimensions_shown)
        .enabled(row.visible)
        .build()
    {
        actions.push(GuiShellAction::BrowserChoice {
            document: state.document.unwrap_or(0),
            choice: crate::GuiBrowserChoice::SketchDimensionsShown {
                sketch: row.id,
                shown: !row.dimensions_shown,
            },
        });
    }
    if ui.menu_item("Show Input") {
        show_input(state, row.id, view, actions);
    }
    if ui.menu_item("Relationships") {
        view.relationships.request(RelatedRow::Sketch(row.id));
    }
    muted_menu_item(ui, "Export Sketch As");
}
/// The plane the sketch stands on takes the row's place as what is chosen.
fn show_input(
    state: &GuiCommandState,
    sketch: DocumentSketchId,
    view: &mut BrowserView,
    actions: &mut Vec<GuiShellAction>,
) {
    let browser = state.browser.as_ref();
    let plane = browser.and_then(|browser| {
        crate::application::model_browser::relationships::plane_of_sketch(browser, sketch)
    });
    let Some(RelatedRow::Origin(place)) = plane else {
        return;
    };
    view.selected_origin_row = Some(place);
    actions.push(GuiShellAction::BrowserChoice {
        document: state.document.unwrap_or(0),
        choice: crate::GuiBrowserChoice::Nothing,
    });
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
