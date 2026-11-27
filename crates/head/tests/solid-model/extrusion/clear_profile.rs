// SPDX-License-Identifier: GPL-2.0-only
use super::super::distance_editor::dimension_pick::{
    open_dimension_package, open_rectangle_preview,
};
use super::super::viewport_interaction::click_at;
use super::{ApplicationWorld, apply_next::click_settled, feature_count};

#[test]
fn native_clear_and_repick_an_extrude_profile_without_reusing_a_stale_hit() {
    let mut world = ApplicationWorld::new();
    open_dimension_package(
        &mut world,
        include_bytes!(
            "../../../../../test-utilities/fixtures/extrusion/extrusion-dimension-pick.opd"
        ),
    );
    open_rectangle_preview(&mut world);
    let generation = world.state().document_generation;
    for (expected, point) in [(1, [3., 0., 0.]), (0, [1., 0.4, 0.])] {
        click_settled(&mut world, "Clear");
        world.run_in_window("Extrude", "SelectProfile");
        assert!(!world.state().extrude_preview_active);
        assert!(world.state().extrude_draft_preview.is_none());
        assert!(!world.run_is_enabled("OK"));
        assert!(world.state().browser.unwrap().sketches[0].visible);

        let pixel = world.app().window.native.view.model_pixel(point).unwrap();
        world.hover(pixel);
        click_at(&mut world, pixel);
        world.pump(Vec::new());
        let state = world.state();
        let preview = state.extrude_draft_preview.expect("repicked profile");
        assert_eq!(preview.profile, expected);
        assert_eq!(preview.distance, "10.00 mm");
        assert_eq!(state.document_generation, generation);
        assert!(state.undo_labels.is_empty());
        assert_eq!(feature_count(&mut world), 0);
        assert!(world.run_is_enabled("OK"));
    }
}
