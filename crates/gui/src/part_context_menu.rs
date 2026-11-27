// SPDX-License-Identifier: GPL-2.0-only
//! Part canvas menu uses the user's configured base marking menu (#4045).
use crate::icons::GuiRibbonIcons;
use crate::marking_menu::{MarkingEntry, ResolveCommand};
use crate::{GuiAuthoringContext, GuiCommandState, GuiShellAction};
use imgui::Ui;
use oblikovati_api::{
    types,
    wire::{MarkingMenuItem, MarkingMenuView},
};
use oblikovati_model::marking_menu::{MarkingMenuOverflow, MarkingMenus};

const POPUP: &str = "##part-canvas-menu";
const FACE_EXPORT: &str = "Export Face As...";
const FACE_OTHER: &str = "Face.SelectOther";
const FACE_TANGENCIES: &str = "Face.SelectTangencies";
pub(crate) fn canvas_menu(
    ui: &Ui,
    state: &GuiCommandState,
    area: crate::application::document_creation::workspaces::WorkspaceArea,
    actions: &mut Vec<GuiShellAction>,
    repeat_wheel: &mut Option<bool>,
    marking_enabled: bool,
    icons: &GuiRibbonIcons,
) -> bool {
    if state.authoring != GuiAuthoringContext::Part || state.document.is_none() {
        return false;
    }
    let classic = crate::marking_menu::use_classic(state.sketch_classic_menu, marking_enabled);
    let full = classic || state.marking_menu_display.overflow == MarkingMenuOverflow::Full;
    if !ui.is_window_hovered_with_flags(crate::widget_abi::over_any_panel())
        && ui.is_mouse_released(imgui::MouseButton::Right)
    {
        ui.open_popup(POPUP);
        position_face_popup(ui, state, area, classic, icons);
    }
    let face = selected_face_action(state);
    let separate = face.is_some() && !classic && full;
    let _background =
        separate.then(|| ui.push_style_color(imgui::StyleColor::PopupBg, [0., 0., 0., 0.]));
    let Some(_popup) = ui.begin_popup(POPUP) else {
        return false;
    };
    if separate {
        draw_face_standard(ui, state, actions, face.as_ref(), icons);
    }
    if full && repeat_wheel.is_some() && ui.menu_item("Repeat Navigation Wheel") {
        *repeat_wheel = Some(true);
    }
    draw_configured_menu(ui, state, actions, classic, icons);
    if !full {
        return true;
    }
    draw_face_measure(ui, state, actions, face.as_ref());
    if !separate {
        draw_face_details(ui, state, actions, face.as_ref());
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
    icons: &GuiRibbonIcons,
) {
    if classic || selected_face_action(state).is_none() {
        return;
    }
    let Some(ring) = face_ring_size(ui, state, icons) else {
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
    let mode = state.marking_menu_display.overflow;
    let full = mode == MarkingMenuOverflow::Full;
    let above = if full {
        face_standard_size(ui)[1] + style.item_spacing[1]
    } else {
        0.
    };
    let below = if mode == MarkingMenuOverflow::RadialOnly {
        0.
    } else {
        let overflow = part_menu(state, true).map_or(0, |menu| {
            crate::marking_menu::overflow_count(&menu, selected_face_command)
        });
        (overflow + if full { 2 } else { 0 }) as f32 * ui.frame_height_with_spacing()
            + style.item_spacing[1]
    };
    crate::marking_menu_layout::popup_origin(
        ui.io().mouse_pos,
        [[area.left, area.top], [area.right, area.bottom]],
        ring,
        above,
        below,
        style.window_padding,
    )
}
fn face_ring_size(ui: &Ui, state: &GuiCommandState, icons: &GuiRibbonIcons) -> Option<f32> {
    let menu = part_menu(state, true)?;
    crate::marking_menu::ring_size(
        ui,
        &menu,
        selected_face_command,
        state.marking_menu_display.style,
        icons,
    )
}
fn face_standard_size(ui: &Ui) -> [f32; 2] {
    let style = ui.clone_style();
    let width = ["Properties", "Home View", FACE_EXPORT]
        .into_iter()
        .map(|label| ui.calc_text_size(label)[0])
        .fold(0., f32::max)
        + 4. * style.frame_padding[0]
        + 2. * style.item_spacing[0];
    [
        width,
        3. * ui.frame_height_with_spacing() + 2. * style.window_padding[1],
    ]
}
fn draw_face_standard(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    face: Option<&GuiShellAction>,
    icons: &GuiRibbonIcons,
) {
    let size = face_standard_size(ui);
    let origin = ui.cursor_pos();
    let ring = face_ring_size(ui, state, icons).unwrap_or(size[0]);
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
    draw_face_details(ui, state, actions, face);
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
fn draw_face_details(
    ui: &Ui,
    state: &GuiCommandState,
    actions: &mut Vec<GuiShellAction>,
    face: Option<&GuiShellAction>,
) {
    let Some(GuiShellAction::StartSketchOnSelectedFace { document, key }) = face else {
        return;
    };
    ui.menu_item_config(FACE_EXPORT).enabled(false).build();
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
    icons: &GuiRibbonIcons,
) {
    let selected = selected_face_action(state);
    let Some(menu) = part_menu(state, selected.is_some()) else {
        return;
    };
    let resolve: ResolveCommand = if selected.is_some() {
        selected_face_command
    } else {
        part_entry
    };
    let start = actions.len();
    if classic {
        crate::marking_menu::draw_classic(ui, &menu, state, actions, resolve);
    } else {
        crate::marking_menu::draw(ui, &menu, state, actions, resolve, icons);
    }
    for action in &mut actions[start..] {
        if *action == GuiShellAction::ChooseSketchPlane
            && let Some(selected) = &selected
        {
            *action = selected.clone();
        }
    }
}
fn part_menu(state: &GuiCommandState, face: bool) -> Option<MarkingMenuView> {
    let mut menu = state.marking_menus.as_ref()?.get(types::BASE_ENVIRONMENT);
    if !face || menu != MarkingMenus::default().get(types::BASE_ENVIRONMENT) {
        return Some(menu);
    }
    // The selected-face preset has its own selection commands and positions (#4055).
    menu.quadrants = Some(default_face_slots());
    menu.overflow = None;
    Some(menu)
}
fn default_face_slots() -> Vec<MarkingMenuItem> {
    [
        (types::QUADRANT_NORTH, "Sketch.Create2D"),
        (types::QUADRANT_NORTH_EAST, "Modify.Fillet"),
        (types::QUADRANT_EAST, "Create.Extrude"),
        (types::QUADRANT_SOUTH_EAST, "Modify.Chamfer"),
        (types::QUADRANT_SOUTH, FACE_TANGENCIES),
        (types::QUADRANT_SOUTH_WEST, "WorkPlane.Offset"),
        (types::QUADRANT_WEST, FACE_OTHER),
    ]
    .into_iter()
    .map(|(quadrant, command_id)| MarkingMenuItem {
        quadrant,
        command_id: command_id.into(),
    })
    .collect()
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
fn selected_face_command(id: &str) -> Option<MarkingEntry> {
    match id {
        "Sketch.Create2D" => Some(MarkingEntry::command((
            "New Sketch",
            GuiShellAction::ChooseSketchPlane,
        ))),
        FACE_OTHER => Some(MarkingEntry::disabled("Select Other...")),
        FACE_TANGENCIES => Some(MarkingEntry::disabled("Select Tangencies")),
        _ => part_entry(id),
    }
}
fn part_entry(id: &str) -> Option<MarkingEntry> {
    command(id).map(MarkingEntry::command)
}
const MARKING_COMMANDS: [(&str, &str, GuiShellAction); 9] = [
    ("Create.Extrude", "Extrude", GuiShellAction::OpenExtrude),
    (
        "Create.Revolve",
        "Revolve",
        GuiShellAction::Revolve(crate::GuiRevolveAction::Open),
    ),
    ("Modify.Fillet", "Fillet", GuiShellAction::OpenFillet),
    ("Modify.Chamfer", "Chamfer", GuiShellAction::OpenChamfer),
    ("Modify.Hole", "Hole", GuiShellAction::OpenHole),
    ("edit.undo", "Undo", GuiShellAction::Undo),
    (
        "Inspect.Measure",
        "Measure",
        GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure),
    ),
    (
        "Sketch.Create2D",
        "Start 2D Sketch",
        GuiShellAction::ChooseSketchPlane,
    ),
    (
        "WorkPlane.Offset",
        "Offset Plane",
        GuiShellAction::OpenOffsetPlane,
    ),
];
pub(crate) fn marking_commands() -> impl Iterator<Item = (&'static str, &'static str)> {
    MARKING_COMMANDS.iter().map(|&(id, label, _)| match id {
        "Sketch.Create2D" => (id, "New Sketch"),
        "WorkPlane.Offset" => (id, "Work Plane"),
        _ => (id, label),
    })
}
fn command(id: &str) -> Option<(&'static str, GuiShellAction)> {
    MARKING_COMMANDS
        .iter()
        .find_map(|(key, label, action)| (*key == id).then(|| (*label, action.clone())))
}
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn face_only_entries_leave_the_base_profile_and_custom_slots_intact() {
        let mut menus = MarkingMenus::default();
        let mut state = GuiCommandState {
            marking_menus: Some(menus.clone()),
            ..Default::default()
        };
        let ordinary = part_menu(&state, false).unwrap();
        let face = part_menu(&state, true).unwrap();
        let ids = face
            .quadrants
            .unwrap()
            .into_iter()
            .map(|item| item.command_id)
            .collect::<Vec<_>>();
        assert!(ids.iter().any(|id| id == FACE_OTHER));
        assert!(ids.iter().any(|id| id == FACE_TANGENCIES));
        assert_eq!(part_menu(&state, false), Some(ordinary.clone()));
        let mut custom = ordinary;
        custom
            .quadrants
            .as_mut()
            .unwrap()
            .iter_mut()
            .find(|item| item.quadrant == types::QUADRANT_WEST)
            .unwrap()
            .command_id = "Modify.Fillet".into();
        menus.set(custom.clone()).unwrap();
        state.marking_menus = Some(menus);
        assert_eq!(part_menu(&state, true), Some(custom));
    }
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
            Some(MarkingEntry::command((
                "New Sketch",
                GuiShellAction::ChooseSketchPlane
            )))
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
        assert_eq!(
            command("Modify.Hole"),
            Some(("Hole", GuiShellAction::OpenHole))
        );
        assert_eq!(command("edit.undo"), Some(("Undo", GuiShellAction::Undo)));
        assert_eq!(
            command("Inspect.Measure"),
            Some((
                "Measure",
                GuiShellAction::Inspect(crate::GuiInspectAction::StartMeasure)
            ))
        );
        assert_eq!(
            part_entry("Create.Revolve"),
            Some(MarkingEntry::command((
                "Revolve",
                GuiShellAction::Revolve(crate::GuiRevolveAction::Open),
            )))
        );
        assert_eq!(command("Modify.Unknown"), None);
    }
}
