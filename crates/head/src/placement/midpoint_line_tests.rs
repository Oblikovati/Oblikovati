// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn midpoint_line_preview_uses_center_and_full_segment_length() {
    let mut state = SketchPlacementState::new(
        7,
        DocumentSketchId(3),
        11,
        SketchPlane::xy(),
        crate::SketchPlacementTool::MidpointLine,
    )
    .unwrap();
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    assert_eq!(state.prompt(), "Select midpoint of line");
    state.cursor(Some([2., -1.]), true, &construction).unwrap();
    assert_eq!(state.prompt(), "Select end of line");
    state.cursor(Some([5., 3.]), false, &construction).unwrap();
    let preview = state.preview.as_ref().unwrap();
    assert_eq!(preview.points, vec![[-1., -5.], [5., 3.], [2., -1.]]);
    assert_eq!(preview.fields[0].value, 10.);
}

#[test]
fn same_position_keeps_midpoint_placement_active() {
    let mut state = SketchPlacementState::new(
        7,
        DocumentSketchId(3),
        11,
        SketchPlane::xy(),
        crate::SketchPlacementTool::MidpointLine,
    )
    .unwrap();
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    state.cursor(Some([0., 0.]), true, &construction).unwrap();
    assert!(
        state
            .cursor(Some([0., 0.]), true, &construction)
            .unwrap()
            .is_none()
    );
    assert_eq!(state.prompt(), "Select end of line");
}

#[test]
fn typed_length_is_full_length_and_angle_selects_direction() {
    let mut state = SketchPlacementState::new(
        7,
        DocumentSketchId(3),
        11,
        SketchPlane::xy(),
        crate::SketchPlacementTool::MidpointLine,
    )
    .unwrap();
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    state.cursor(Some([0., 0.]), true, &construction).unwrap();
    state.cursor(Some([3., 4.]), false, &construction).unwrap();
    for character in "10".chars() {
        state.event(ViewportEvent::Input(ViewportInput::Text(character)));
    }
    state.evaluate_line_values(&|_, source| source.parse().ok());
    state.event(ViewportEvent::Input(ViewportInput::PhysicalKey {
        key: ViewportPhysicalKey::Tab,
        layout_key: ViewportPhysicalKey::Tab,
        command: None,
        pressed: true,
        modifiers: Default::default(),
    }));
    for character in "30".chars() {
        state.event(ViewportEvent::Input(ViewportInput::Text(character)));
    }
    state.evaluate_line_values(&|index, source| {
        source.parse::<f64>().ok().map(|value| {
            if index == 1 {
                value.to_radians()
            } else {
                value
            }
        })
    });
    let recipe = state.typed_seed_recipe(&construction).unwrap().unwrap();
    assert!((recipe.fields[0].value - 10.).abs() < 1e-12);
    assert!((recipe.fields[1].value - 30_f64.to_radians()).abs() < 1e-12);
    assert!((recipe.points[1][0] - 5. * 30_f64.to_radians().cos()).abs() < 1e-12);
    assert!((recipe.points[1][1] - 2.5).abs() < 1e-12);
    let locks = state.locks_for(&recipe).unwrap();
    assert_eq!(locks[0].as_ref().unwrap().expression, "10");
    assert_eq!(locks[1].as_ref().unwrap().expression, "30");
}

#[test]
fn selected_midpoint_identity_is_shared_by_preview_and_typed_commit() {
    let mut state = SketchPlacementState::new(
        7,
        DocumentSketchId(3),
        11,
        SketchPlane::xy(),
        crate::SketchPlacementTool::MidpointLine,
    )
    .unwrap();
    let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
    let id = oblikovati_model::sketch::SketchEntityId(42);
    state.cursor_vertex = Some(([0., 0.], id));
    state.cursor(Some([0., 0.]), true, &construction).unwrap();
    state.cursor_vertex = None;
    state.cursor(Some([3., 4.]), false, &construction).unwrap();
    assert_eq!(
        state.preview.as_ref().unwrap().picked_points.get(&2),
        Some(&id)
    );
    assert_eq!(
        state
            .typed_seed_recipe(&construction)
            .unwrap()
            .unwrap()
            .picked_points
            .get(&2),
        Some(&id)
    );
    state.restart(12);
    state.cursor(Some([1., 1.]), true, &construction).unwrap();
    state.cursor(Some([2., 2.]), false, &construction).unwrap();
    assert!(state.preview.as_ref().unwrap().picked_points.is_empty());
}
