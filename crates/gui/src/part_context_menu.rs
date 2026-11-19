// SPDX-License-Identifier: GPL-2.0-only
//! Part canvas menu uses the user's configured base marking menu (#4045).
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;

const POPUP: &str = "##part-canvas-menu";
pub(crate) fn canvas_menu(
    ui: &Ui,
    state: &GuiCommandState,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    actions: &mut Vec<GuiShellAction>,
    repeat_wheel: &mut Option<bool>,
    marking_enabled: bool,
) -> bool {
    if state.authoring != GuiAuthoringContext::Part || state.document.is_none() {
        return false;
    }
    let classic = crate::marking_menu::use_classic(state.sketch_classic_menu, marking_enabled);
    if !ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel())
        && ui.is_mouse_released(imgui::MouseButton::Right)
    {
        ui.open_popup(POPUP);
        position_face_popup(ui, state, area, classic);
    }
    let face = selected_face_action(state);
    let separate = face.is_some() && !classic;
    let _background =
        separate.then(|| ui.push_style_color(imgui::StyleColor::PopupBg, [0., 0., 0., 0.]));
    let Some(_popup) = ui.begin_popup(POPUP) else {
        return false;
    };
    if separate {
        draw_face_standard(ui, state, actions, face.as_ref());
    }
    if repeat_wheel.is_some() && ui.menu_item("Repeat Navigation Wheel") {
        *repeat_wheel = Some(true);
    }
    draw_configured_menu(ui, state, actions, classic);
    draw_face_measure(ui, state, actions, face.as_ref());
    if !separate {
        draw_face_properties(ui, state, actions, face.as_ref());
    }
    let look_at = GuiShellAction::LookAtSelection;
    if ui
        .menu_item_config("Look At")
        .enabled(look_at.enabled(state))
        .build()
    {
        actions.push(look_at);
    }
    if !separate && ui.menu_item("Home View") {
        actions.push(GuiShellAction::Home);
    }
    true
}
fn position_face_popup(
    ui: &Ui,
    state: &GuiCommandState,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    classic: bool,
) {
    if classic || selected_face_action(state).is_none() {
        return;
    }
    let Some(ring) = face_ring_size(ui, state) else {
        return;
    };
    let origin = face_popup_origin(ui, state, area, ring);
    crate::widget_abi::next_window_position(ui, origin);
}
fn face_popup_origin(
    ui: &Ui,
    state: &GuiCommandState,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    ring: f32,
) -> [f32; 2] {
    let style = ui.clone_style();
    let above = face_standard_size(ui)[1] + style.item_spacing[1];
    let overflow = state.marking_menus.as_ref().map_or(0, |menus| {
        let menu = menus.get(oblikovati_api::types::BASE_ENVIRONMENT);
        crate::marking_menu::overflow_count(&menu, selected_face_command)
    });
    let below = (overflow + 2) as f32 * ui.frame_height_with_spacing() + style.item_spacing[1];
    crate::marking_menu_layout::popup_origin(
        ui.io().mouse_pos,
        [[area.left, area.top], [area.right, area.bottom]],
        ring,
        above,
        below,
        style.window_padding,
    )
}
fn face_ring_size(ui: &Ui, state: &GuiCommandState) -> Option<f32> {
    let menus = state.marking_menus.as_ref()?;
    let menu = menus.get(oblikovati_api::types::BASE_ENVIRONMENT);
    crate::marking_menu::ring_size(ui, &menu, selected_face_command)
}
fn face_standard_size(ui: &Ui) -> [f32; 2] {
    let style = ui.clone_style();
    let width = ui.calc_text_size("Properties")[0].max(ui.calc_text_size("Home View")[0])
        + 4. * style.frame_padding[0]
        + 2. * style.item_spacing[0];
    [
        width,
        2. * ui.frame_height_with_spacing() + 2. * style.window_padding[1],
    ]
}
fn draw_face_standard(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    face: Option<&GuiShellAction>,
) {
    let size = face_standard_size(ui);
    let origin = ui.cursor_pos();
    let ring = face_ring_size(ui, state).unwrap_or(size[0]);
    ui.set_cursor_pos([origin[0] + ((ring - size[0]) / 2.).max(0.), origin[1]]);
    ui.child_window("##face-standard-menu")
        .size(size)
        .border(true)
        .build(|| draw_face_standard_rows(ui, state, actions, face));
    ui.set_cursor_pos([
        origin[0],
        origin[1] + size[1] + ui.clone_style().item_spacing[1],
    ]);
}
fn draw_face_standard_rows(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    face: Option<&GuiShellAction>,
) {
    draw_face_properties(ui, state, actions, face);
    if ui.menu_item("Home View") {
        actions.push(GuiShellAction::Home);
    }
}
fn draw_face_measure(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    face: Option<&GuiShellAction>,
) {
    let Some(GuiShellAction::StartSketchOnSelectedFace { document, key }) = face else {
        return;
    };
    let action = GuiShellAction::MeasureSelectedFace {
        document: *document,
        key: key.clone(),
    };
    if ui
        .menu_item_config("Measure")
        .enabled(action.enabled(state))
        .build()
    {
        actions.push(action);
    }
}
fn draw_face_properties(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    face: Option<&GuiShellAction>,
) {
    let Some(GuiShellAction::StartSketchOnSelectedFace { document, key }) = face else {
        return;
    };
    let action = GuiShellAction::OpenFaceProperties {
        document: *document,
        key: key.clone(),
    };
    if ui
        .menu_item_config("Properties")
        .enabled(action.enabled(state))
        .build()
    {
        actions.push(action);
    }
}
fn draw_configured_menu(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    classic: bool,
) {
    let Some(menus) = &state.marking_menus else {
        return;
    };
    let menu = menus.get(oblikovati_api::types::BASE_ENVIRONMENT);
    let selected = selected_face_action(state);
    let resolve = if selected.is_some() {
        selected_face_command
    } else {
        command
    };
    let start = actions.len();
    if classic {
        crate::marking_menu::draw_classic(ui, &menu, state, actions, resolve);
    } else {
        crate::marking_menu::draw(ui, &menu, state, actions, resolve);
    }
    for action in &mut actions[start..] {
        if *action == GuiShellAction::ChooseSketchPlane
            && let Some(selected) = &selected
        {
            *action = selected.clone();
        }
    }
}
fn selected_face_action(state: &GuiCommandState) -> Option<GuiShellAction> {
    let document = state.document?;
    let [(key, _)] = state.selected_face_appearances.as_slice() else {
        return None;
    };
    let action = GuiShellAction::StartSketchOnSelectedFace {
        document,
        key: key.clone(),
    };
    action.enabled(state).then_some(action)
}
fn selected_face_command(id: &str) -> Option<(&'static str, GuiShellAction)> {
    if id == "Sketch.Create2D" {
        return Some(("New Sketch", GuiShellAction::ChooseSketchPlane));
    }
    command(id)
}
fn command(id: &str) -> Option<(&'static str, GuiShellAction)> {
    match id {
        "Create.Extrude" => Some(("Extrude", GuiShellAction::OpenExtrude)),
        "Modify.Fillet" => Some(("Fillet", GuiShellAction::OpenFillet)),
        "Modify.Chamfer" => Some(("Chamfer", GuiShellAction::OpenChamfer)),
        "Sketch.Create2D" => Some(("Start 2D Sketch", GuiShellAction::ChooseSketchPlane)),
        "WorkPlane.Offset" => Some(("Offset Plane", GuiShellAction::OpenOffsetPlane)),
        _ => None,
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn one_current_face_produces_a_named_sketch_action() {
        let mut state = GuiCommandState {
            document: Some(12),
            authoring: GuiAuthoringContext::Part,
            model_face_chosen: true,
            selected_face_appearances: vec![("face-1".into(), false)],
            ..Default::default()
        };
        assert_eq!(
            selected_face_action(&state),
            Some(GuiShellAction::StartSketchOnSelectedFace {
                document: 12,
                key: "face-1".into()
            })
        );
        let original = selected_face_action(&state).unwrap();
        let measure = GuiShellAction::MeasureSelectedFace {
            document: 12,
            key: "face-1".into(),
        };
        let properties = GuiShellAction::OpenFaceProperties {
            document: 12,
            key: "face-1".into(),
        };
        assert!(measure.enabled(&state));
        assert!(properties.enabled(&state));
        state.selected_face_appearances[0].0 = "face-2".into();
        assert!(!measure.enabled(&state));
        assert!(!properties.enabled(&state));
        assert!(
            !original.enabled(&state),
            "a changed face must not execute a stale action"
        );
        state.selected_face_appearances[0].0 = "face-1".into();
        state.document = Some(13);
        assert!(
            !original.enabled(&state),
            "a different document must not execute a stale action"
        );
        state.document = Some(12);
        assert_eq!(
            selected_face_command("Sketch.Create2D"),
            Some(("New Sketch", GuiShellAction::ChooseSketchPlane))
        );
        state
            .selected_face_appearances
            .push(("face-2".into(), false));
        assert_eq!(selected_face_action(&state), None);
        state.selected_face_appearances.pop();
        state.selected_body_keys.push("body".into());
        assert_eq!(selected_face_action(&state), None);
        state.selected_body_keys.clear();
        state.selected_edge_count = 1;
        assert_eq!(selected_face_action(&state), None);
        state.selected_edge_count = 0;
        state.transaction_open = true;
        assert_eq!(selected_face_action(&state), None);
    }
    #[test]
    fn popup_height_counts_only_resolvable_overflow_commands() {
        let menu = oblikovati_api::wire::MarkingMenuView {
            overflow: Some(vec![
                "Create.Extrude".into(),
                "Unknown".into(),
                "Modify.Chamfer".into(),
            ]),
            ..Default::default()
        };
        assert_eq!(
            crate::marking_menu::overflow_count(&menu, selected_face_command),
            2
        );
    }
    #[test]
    fn part_commands_resolve_only_known_actions() {
        assert_eq!(
            command("Modify.Fillet"),
            Some(("Fillet", GuiShellAction::OpenFillet))
        );
        assert_eq!(
            command("Create.Extrude"),
            Some(("Extrude", GuiShellAction::OpenExtrude))
        );
        assert_eq!(
            command("Modify.Chamfer"),
            Some(("Chamfer", GuiShellAction::OpenChamfer))
        );
        assert_eq!(command("Modify.Unknown"), None);
    }
}
