// SPDX-License-Identifier: GPL-2.0-only
//! `features/sketch/line`: a segment commits when accepted; the command keeps the chain going.
use super::super::ApplicationWorld;
use cucumber::{given, then, when};

/// Two viewport points that lie clear of the origin axes and every other fixture.
const SEGMENT: [[i32; 2]; 2] = [[620, 300], [840, 420]];

#[given("the MCAD Software is running")]
fn software_running(world: &mut ApplicationWorld) {
    assert!(world.app.is_some(), "expected a running application window");
    world.pump(Vec::new());
}
#[test]
fn running_precondition_is_independent_of_document_and_sketch_state() {
    let mut world = ApplicationWorld::new();
    running_at_home(&mut world);
    running_with_part(&mut world);
    running_with_sketch(&mut world);
    world.close();
    assert!(
        std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| software_running(
            &mut world
        )))
        .is_err()
    );
}
fn running_at_home(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::NoDocument
    );
    software_running(world);
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::NoDocument
    );
}
fn running_with_part(world: &mut ApplicationWorld) {
    let document = world.add_part();
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    software_running(world);
    assert_eq!(world.state().document, Some(document));
    assert!(!world.state().sketch_at_ready);
}
fn running_with_sketch(world: &mut ApplicationWorld) {
    world.start_sketch();
    let sketch = world.state().active_sketch;
    software_running(world);
    assert_eq!(world.state().active_sketch, sketch);
    assert!(world.state().sketch_at_ready);
}
#[given("an empty editable planar sketch is active")]
fn empty_sketch(world: &mut ApplicationWorld) {
    let geometry = world.edited_geometry();
    assert_eq!(geometry.entity_ids().len(), geometry.points().len());
    assert!(geometry.points().len() <= 1);
    assert!(
        geometry
            .points()
            .iter()
            .all(|point| point.position == [0., 0.])
    );
    assert!(world.state().sketch_at_ready);
}
#[when("I select Line and click two distinct points")]
#[when("the user select Line and click two distinct points")]
fn line_two_points(world: &mut ApplicationWorld) {
    world.select_line_tool();
    for point in SEGMENT {
        world.sketch_click(point);
    }
}
#[then("one segment joins the points and the command offers a connected next segment")]
fn one_segment_and_next(world: &mut ApplicationWorld) {
    assert_eq!(
        world.edited_geometry().lines().len(),
        1,
        "expected one segment"
    );
    world.expect_segment_drawn(SEGMENT[0], SEGMENT[1]);
    world.expect_status("Select end of line");
    assert!(
        !world.window_text("##sketch-hud").is_empty(),
        "expected the fields of the next segment"
    );
}
#[then("the command ends and the visible segment remains")]
fn command_ended(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_at_ready);
    world.expect_segment_drawn(SEGMENT[0], SEGMENT[1]);
}

#[path = "chain.rs"]
mod chain;
#[path = "inspect.rs"]
mod inspect;
#[path = "numeric.rs"]
mod numeric;
#[path = "recorded.rs"]
mod recorded;
#[path = "repeated_click.rs"]
mod repeated_click;
#[path = "retained_arc.rs"]
mod retained_arc;
