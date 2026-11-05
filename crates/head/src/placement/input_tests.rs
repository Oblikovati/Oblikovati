// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::placement_test_inputs::{button, key, motion};

fn rectangle_state() -> SketchPlacementState {
    SketchPlacementState::new(
        7,
        DocumentSketchId(3),
        11,
        SketchPlane::xz(),
        crate::SketchPlacementTool::Rectangle,
    )
    .unwrap()
}

fn enter_rectangle_width(state: &mut SketchPlacementState) {
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    state.cursor(Some([0., 0.]), true, &construction).unwrap();
    state.cursor(Some([1., 2.]), false, &construction).unwrap();
    for character in "size".chars() {
        let action = state.event(ViewportEvent::Input(ViewportInput::Text(character)));
        assert!(matches!(action, PlacementAction::FieldsChanged));
    }
}

#[test]
fn restart_preserves_document_identity_and_clears_dimension_entry() {
    let mut state = rectangle_state();
    enter_rectangle_width(&mut state);
    assert!(state.fields_engaged());
    assert!(state.field_prompt().contains("size"));
    let locks = state.preview_locks().unwrap();
    assert_eq!(locks[0].as_ref().unwrap().expression, "size");
    state.restart(12);
    let identity = (state.document, state.sketch, state.generation);
    assert_eq!(identity, (7, DocumentSketchId(3), 12));
    assert_eq!(state.plane, SketchPlane::xz());
    assert!(!state.fields_engaged());
    assert!(state.preview.is_none());
    assert!(state.accepted.is_empty());
}

#[test]
fn navigation_suppresses_placement_until_the_key_is_released() {
    let mut state = rectangle_state();
    let pressed = state.event(key(ViewportKey::Pan, true));
    assert!(matches!(pressed, PlacementAction::Idle));
    assert!(matches!(
        state.event(motion([20, 30])),
        PlacementAction::Idle
    ));
    let click = state.event(button([20, 30], true));
    assert!(matches!(click, PlacementAction::Idle));
    state.event(key(ViewportKey::Pan, false));
    let cursor = state.event(motion([20, 30]));
    assert!(matches!(
        cursor,
        PlacementAction::Cursor {
            pixel: [20, 30],
            place: false
        }
    ));
    let cancelled = state.event(key(ViewportKey::Cancel, true));
    assert!(matches!(cancelled, PlacementAction::Cancel));
}

#[test]
fn release_places_only_after_a_drag_above_four_pixels() {
    let mut state = rectangle_state();
    for (distance, expected) in [(4, false), (5, true)] {
        let pressed = state.event(button([10, 10], true));
        assert!(matches!(
            pressed,
            PlacementAction::Cursor { place: true, .. }
        ));
        state.event(motion([10 + distance, 10]));
        let released = state.event(button([10 + distance, 10], false));
        let placed = matches!(released, PlacementAction::Cursor { place: true, .. });
        assert_eq!(placed, expected, "drag distance {distance}");
    }
}

#[test]
fn abandoned_press_cannot_place_on_release() {
    let mut state = rectangle_state();
    state.event(button([10, 10], true));
    state.abandon_press();
    let released = state.event(button([30, 10], false));
    assert!(matches!(released, PlacementAction::Idle));
}
