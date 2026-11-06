// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::SketchLineChainCreationReport;
fn draft() -> SketchPlacementState {
    SketchPlacementState::new(
        1,
        DocumentSketchId(1),
        0,
        SketchPlane::xy(),
        crate::SketchPlacementTool::Line,
    )
    .unwrap()
}
fn report(vertices: [u64; 2]) -> SketchLineChainCreationReport {
    SketchLineChainCreationReport {
        sketch: DocumentSketchId(1),
        points: vertices.map(SketchEntityId).to_vec(),
        lines: Vec::new(),
        dimensions: Vec::new(),
    }
}
fn press(state: &mut SketchPlacementState, pixel: [i32; 2]) -> PlacementAction {
    state.button(ViewportButton::Left, true, pixel)
}
fn release(state: &mut SketchPlacementState, pixel: [i32; 2]) -> PlacementAction {
    state.button(ViewportButton::Left, false, pixel)
}
#[test]
fn the_pending_end_follows_the_pointer_from_the_start_point() {
    let mut state = draft();
    assert!(state.line_input().is_none());
    state.line_start_at([1., 2.], [10, 10]).unwrap();
    state.line_cursor(Some([4., 6.]));
    let input = state.line_input().unwrap();
    assert_eq!(input.points, [[1., 2.], [4., 6.]]);
    assert_eq!(input.joins, [None, None]);
    state.line_cursor(Some([1., 2.]));
    assert!(state.line_input().is_none(), "a zero-length end is no end");
}
#[test]
fn typed_length_and_angle_replace_the_pointer_and_become_locks() {
    let mut state = draft();
    state.committing_value_dimensions = true;
    state.line_start_at([0., 0.], [10, 10]).unwrap();
    state.line_cursor(Some([5., 5.]));
    state.fields.input(ViewportInput::Text('2'), 2);
    state.evaluate_line_values(&|_, _| Some(2.));
    let [x, y] = state.line_end().unwrap();
    let reach = 2f64.sqrt();
    assert!(
        (x - reach).abs() < 1e-12 && (y - reach).abs() < 1e-12,
        "{x} {y}"
    );
    let locks = state.line_input().unwrap().locks;
    assert_eq!(locks[0].as_ref().unwrap().expression, "2");
    assert!(locks[1].is_none());
    state.committing_value_dimensions = false;
    assert!(state.line_input().unwrap().locks[0].is_none());
}
#[test]
fn a_value_that_does_not_evaluate_keeps_its_field_and_blocks_tab() {
    let mut state = draft();
    state.line_start_at([0., 0.], [10, 10]).unwrap();
    state.fields.input(ViewportInput::Text('0'), 2);
    state.evaluate_line_values(&|_, _| None);
    assert!(state.fields_invalid());
    assert!(state.field_view(0).invalid);
    let tab = ViewportInput::PhysicalKey {
        key: ViewportPhysicalKey::Tab,
        layout_key: ViewportPhysicalKey::Tab,
        command: None,
        pressed: true,
        modifiers: Default::default(),
    };
    state.fields.input(tab, 2);
    assert!(
        state.field_view(0).active,
        "Tab must not leave an invalid value"
    );
}
#[test]
fn select_all_makes_the_next_character_replace_the_field() {
    let mut state = draft();
    state.line_start_at([0., 0.], [10, 10]).unwrap();
    state.fields.input(ViewportInput::Text('0'), 2);
    let select = ViewportInput::PhysicalKey {
        key: ViewportPhysicalKey::A,
        layout_key: ViewportPhysicalKey::A,
        command: None,
        pressed: true,
        modifiers: oblikovati_viewport::ViewportModifiers {
            control: true,
            ..Default::default()
        },
    };
    state.fields.input(select, 2);
    assert!(state.field_view(0).selected);
    state.fields.input(ViewportInput::Text('4'), 2);
    assert_eq!(state.field_view(0).typed.as_deref(), Some("4"));
}
#[test]
fn a_second_click_at_the_accepted_end_ends_the_chain_but_a_drag_does_not() {
    let mut state = draft();
    state.line_start_at([0., 0.], [100, 100]).unwrap();
    state.accept_line_segment(&report([1, 2]), [5., 0.], Some([200, 100]), 3);
    assert!(matches!(
        press(&mut state, [201, 101]),
        PlacementAction::Idle
    ));
    assert!(matches!(
        release(&mut state, [201, 101]),
        PlacementAction::RestartLine
    ));
    assert!(matches!(
        press(&mut state, [200, 100]),
        PlacementAction::Idle
    ));
    state.motion([300, 100]);
    assert!(matches!(
        release(&mut state, [300, 100]),
        PlacementAction::Idle
    ));
    assert!(state.line_restart_available());
}
#[test]
fn a_repeated_click_on_a_lone_start_point_keeps_waiting_for_the_end() {
    let mut state = draft();
    state.line_start_at([0., 0.], [100, 100]).unwrap();
    assert!(matches!(
        press(&mut state, [100, 100]),
        PlacementAction::Idle
    ));
    assert!(matches!(
        release(&mut state, [100, 100]),
        PlacementAction::Idle
    ));
    assert!(state.line_restart_available());
}
#[test]
fn close_needs_two_segments_and_joins_the_first_vertex() {
    let mut state = draft();
    state.line_start_at([0., 0.], [100, 100]).unwrap();
    assert!(state.line_close_input().is_none());
    state.accept_line_segment(&report([1, 2]), [5., 0.], Some([200, 100]), 3);
    assert!(!state.line_close_available());
    state.accept_line_segment(&report([2, 3]), [5., 5.], Some([200, 200]), 4);
    assert!(state.line_close_available());
    let close = state.line_close_input().unwrap();
    assert_eq!(close.points, [[5., 5.], [0., 0.]]);
    assert_eq!(
        close.joins,
        [Some(SketchEntityId(3)), Some(SketchEntityId(1))]
    );
}
#[test]
fn restart_drops_the_chain_and_escape_ends_the_command() {
    let mut state = draft();
    state.line_start_at([0., 0.], [100, 100]).unwrap();
    assert!(matches!(
        state.event(ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Cancel,
            pressed: true
        })),
        PlacementAction::FinishLine
    ));
    state.restart_line(9);
    assert!(state.line_awaits_start());
    assert_eq!(state.generation, 9);
}
