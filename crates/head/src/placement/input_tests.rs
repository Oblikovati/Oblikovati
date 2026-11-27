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
    let recipe = state.preview.clone().unwrap();
    let locks = state.locks_for(&recipe).unwrap();
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

fn arc_awaiting_its_point() -> SketchPlacementState {
    let mut state = SketchPlacementState::new(
        7,
        DocumentSketchId(3),
        11,
        SketchPlane::xz(),
        crate::SketchPlacementTool::ThreePointArc {
            options: oblikovati_model::sketch::CircumcircleOptions {
                maximum_point_residual: 1e-9,
            },
        },
    )
    .unwrap();
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    state.cursor(Some([0., 0.]), true, &construction).unwrap();
    state.cursor(Some([8., 0.]), true, &construction).unwrap();
    state
}
fn typed_radius(state: &mut SketchPlacementState, radius: f64) {
    for character in radius.to_string().chars() {
        state.event(ViewportEvent::Input(ViewportInput::Text(character)));
    }
    state.evaluate_line_values(&|_, text| text.parse().ok());
}
#[test]
fn a_typed_radius_bows_the_arc_to_the_pointers_side_on_the_nearer_circle() {
    let mut state = arc_awaiting_its_point();
    typed_radius(&mut state, 5.);
    // The ends' chord is 8 long, so the circles of radius 5 have centers at (4, 3) and (4, -3).
    assert_eq!(state.typed_finishing_point([4., 2.]), [4., 2.]);
    assert_eq!(state.typed_finishing_point([4., -2.]), [4., -2.]);
    // Past the near circle's top, the pointer is nearer the far circle's: the longer arc.
    assert_eq!(state.typed_finishing_point([4., 9.]), [4., 8.]);
    assert_eq!(state.least_typed_radius(), 4.);
}
#[test]
fn enter_at_the_arcs_point_goes_back_for_its_end_because_a_radius_names_no_side() {
    let mut state = arc_awaiting_its_point();
    typed_radius(&mut state, 5.);
    assert!(!state.enter_finishes_shape());
    assert!(state.accept_typed_stage().unwrap());
    assert_eq!(state.prompt(), "Select end of arc");
    assert!(!state.fields_engaged());
}

fn centered_arc_state() -> SketchPlacementState {
    let mut state = SketchPlacementState::new(
        7,
        DocumentSketchId(3),
        11,
        SketchPlane::xy(),
        crate::SketchPlacementTool::CenterPointArc,
    )
    .unwrap();
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    state.cursor(Some([0., 0.]), true, &construction).unwrap();
    state
}
fn type_field(state: &mut SketchPlacementState, text: &str) {
    for character in text.chars() {
        state.event(ViewportEvent::Input(ViewportInput::Text(character)));
    }
    state.evaluate_line_values(&|_, source| source.parse().ok());
}
fn tab_field(state: &mut SketchPlacementState) {
    state.event(ViewportEvent::Input(ViewportInput::PhysicalKey {
        key: ViewportPhysicalKey::Tab,
        layout_key: ViewportPhysicalKey::Tab,
        command: None,
        pressed: true,
        modifiers: Default::default(),
    }));
}
#[test]
fn centered_arc_keeps_radius_and_sweep_at_their_recipe_fields() {
    let mut state = centered_arc_state();
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    type_field(&mut state, "4");
    tab_field(&mut state);
    assert!(state.field_view(0).locked && state.field_view(1).active);
    state.cursor(Some([3., 0.]), true, &construction).unwrap();
    assert!(state.field_view(2).active);
    type_field(&mut state, "1");
    state.cursor(Some([0., 4.]), false, &construction).unwrap();
    let recipe = state.preview.clone().unwrap();
    let locks = state.locks_for(&recipe).unwrap();
    assert_eq!(locks[0].as_ref().unwrap().expression, "4");
    assert!(locks[1].is_none());
    assert_eq!(locks[2].as_ref().unwrap().expression, "1");
}
#[test]
fn centered_arc_start_angle_zero_is_direction_not_zero_radius() {
    let mut state = centered_arc_state();
    type_field(&mut state, "4");
    tab_field(&mut state);
    type_field(&mut state, "0");
    assert!(!state.typed_size_is_zero());
    assert_eq!(state.arc_start_at_radius([0., 3.]), [4., 0.]);
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    state.cursor(Some([0., 3.]), true, &construction).unwrap();
    assert_eq!(state.prompt(), "Select end point");
    state.cursor(Some([0., 4.]), false, &construction).unwrap();
    let recipe = state.preview.clone().unwrap();
    let locks = state.locks_for(&recipe).unwrap();
    assert_eq!(locks[1].as_ref().unwrap().expression, "0");
}
