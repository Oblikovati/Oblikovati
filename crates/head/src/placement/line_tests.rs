// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
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
#[test]
fn line_draft_keeps_vertices_private_and_emits_explicit_finish_actions() {
    let mut state = draft();
    assert!(state.line_scope.is_none());
    for point in [[0., 0.], [50., 20.], [80., 90.]] {
        state
            .cursor(Some(point), true, &MartensiteSketchConstruction)
            .unwrap();
        state.confirm_line_vertex();
    }
    state.undo_line_vertex().unwrap();
    assert_eq!(
        state.line_input(false, false).unwrap().points,
        [[0., 0.], [50., 20.]]
    );
    assert_eq!(state.preview.as_ref().unwrap().points.len(), 2);
    assert!(matches!(
        state.event(ViewportEvent::Input(ViewportInput::Text('c'))),
        PlacementAction::FinishLine { closed: true }
    ));
    assert!(matches!(
        state.event(ViewportEvent::Input(ViewportInput::Key {
            key: ViewportKey::Cancel,
            pressed: true
        })),
        PlacementAction::FinishLine { closed: false }
    ));
    assert!(matches!(
        state.event(ViewportEvent::Focus(false)),
        PlacementAction::Cancel
    ));
}
#[test]
fn a_rejected_line_preview_keeps_confirmed_vertices_and_ignores_numeric_field_input() {
    let mut state = draft();
    state
        .cursor(Some([0., 0.]), true, &MartensiteSketchConstruction)
        .unwrap();
    state.confirm_line_vertex();
    state
        .cursor(Some([50., 20.]), true, &MartensiteSketchConstruction)
        .unwrap();
    state.abandon_press();
    assert!(state.line_input(false, false).is_none());
    assert!(
        state
            .cursor(Some([0., 0.]), true, &MartensiteSketchConstruction)
            .is_err()
    );
    state.abandon_press();
    state
        .cursor(Some([80., 30.]), true, &MartensiteSketchConstruction)
        .unwrap();
    state.confirm_line_vertex();
    assert_eq!(
        state.line_input(false, false).unwrap().points,
        [[0., 0.], [80., 30.]]
    );
    assert!(matches!(
        state.event(ViewportEvent::Input(ViewportInput::Text('7'))),
        PlacementAction::Idle
    ));
    assert!(!state.fields_engaged());
}
