// SPDX-License-Identifier: GPL-2.0-only
//! Exercise scope selection with the native ribbon and pointer (#4107).
use super::{ApplicationWorld, diagnostics, sketch_constrain};
use oblikovati_api::types::{
    INFERENCE_SCOPE_ALL_GEOMETRY, INFERENCE_SCOPE_CURRENT_COMMAND,
    INFERENCE_SCOPE_SELECTED_GEOMETRY,
};

const DIALOG: &str = "Constraint Inference Scope";
const CONSTRAIN: &str = "##ribbon-additional-Constrain";

fn click_window(world: &mut ApplicationWorld, window: &str, label: &str) {
    let point = world.run_in_window(window, label).center();
    super::super::x11_input::click(diagnostics::pointer_pixels(point));
    world.pump(Vec::new());
}
fn open_scope(world: &mut ApplicationWorld) {
    sketch_constrain::open_constrain(world);
    click_window(world, CONSTRAIN, "ConstraintInferenceScope");
    assert!(diagnostics::window_geometry(DIALOG).is_some());
}
#[test]
fn scope_selection_accepts_reopens_and_keeps_a_normal_second_click() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
    let edge = world.edited_geometry().lines()[0].id;
    open_scope(&mut world);
    assert_eq!(
        world.state().sketch_inference_scope.unwrap().scope,
        INFERENCE_SCOPE_CURRENT_COMMAND
    );
    click_window(&mut world, DIALOG, "Select");
    assert!(world.run_in_window(DIALOG, "OK").alpha < u8::MAX);
    let (position, _) = diagnostics::window_geometry(DIALOG).unwrap();
    world.drag(
        [(position[0] + 80.) as i32, (position[1] + 10.) as i32],
        [360, 210],
    );
    let point = world.line_point(0, 0.5);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert!(
        world
            .state()
            .sketch_selection
            .entities
            .iter()
            .any(|(_, id)| *id == edge)
    );
    assert_eq!(world.run_in_window(DIALOG, "OK").alpha, u8::MAX);
    click_window(&mut world, DIALOG, "OK");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(DIALOG).is_none());
    assert_eq!(
        world.state().sketch_inference_scope.unwrap().scope,
        INFERENCE_SCOPE_SELECTED_GEOMETRY
    );
    open_scope(&mut world);
    assert!(
        world
            .state()
            .sketch_selection
            .entities
            .iter()
            .any(|(_, id)| *id == edge)
    );
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    assert!(
        world
            .state()
            .sketch_selection
            .entities
            .iter()
            .any(|(_, id)| *id == edge)
    );
    click_window(&mut world, DIALOG, "Cancel");
    open_scope(&mut world);
    assert_eq!(
        world.state().sketch_inference_scope.unwrap().entities,
        Some(vec![edge.0])
    );
    click_window(&mut world, DIALOG, "AllGeometry");
    click_window(&mut world, DIALOG, "OK");
    world.pump(Vec::new());
    open_scope(&mut world);
    assert_eq!(
        world.state().sketch_inference_scope.unwrap().scope,
        INFERENCE_SCOPE_ALL_GEOMETRY
    );
    click_window(&mut world, DIALOG, "Geometryincurrentcommand");
    click_window(&mut world, DIALOG, "OK");
    world.pump(Vec::new());
    assert_eq!(
        world.state().sketch_inference_scope.unwrap().scope,
        INFERENCE_SCOPE_CURRENT_COMMAND
    );
    world.close();
}

#[test]
fn pinned_constrain_tracks_sketch_edit_entry_and_exit() {
    use super::super::workspace_layout::browser_feature::{
        SKETCH, open_visible_extruded_part, right_click_row,
    };
    let mut world = ApplicationWorld::new();
    open_visible_extruded_part(&mut world);
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
    let tab = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Sketch")
        .min_by(|left, right| left.center()[1].total_cmp(&right.center()[1]))
        .expect("expected the Sketch ribbon tab")
        .center();
    world.click(diagnostics::pointer_pixels(tab));
    sketch_constrain::open_constrain(&mut world);
    for command in ["Edit Coordinate System", "Constraint Inference Scope"] {
        assert!(!world.run_is_enabled(command));
    }
    click_window(&mut world, CONSTRAIN, "EditCoordinateSystem");
    assert!(diagnostics::window_geometry(CONSTRAIN).is_some());
    super::toggle_pin(&mut world);
    right_click_row(&mut world, SKETCH);
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(CONSTRAIN).is_some());
    super::pin_highlight(&mut world);
    for command in ["Edit Coordinate System", "Constraint Inference Scope"] {
        assert!(world.run_is_enabled(command));
    }
    world.finish_sketch();
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(CONSTRAIN).is_some());
    super::pin_highlight(&mut world);
    for command in ["Edit Coordinate System", "Constraint Inference Scope"] {
        assert!(!world.run_is_enabled(command));
    }
    world.close();
}
