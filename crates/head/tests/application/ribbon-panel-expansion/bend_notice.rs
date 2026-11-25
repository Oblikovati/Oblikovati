// SPDX-License-Identifier: GPL-2.0-only
//! Bend prerequisite notice reached from an expanded command context (#4109).
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

const EXPANSION: &str = "##ribbon-additional-PartModify";
const NOTICE: &str = "Bend Part##prerequisite";

#[given("the part has no visible unadaptive sketch suitable for Bend Part")]
fn no_visible_sketch(world: &mut ApplicationWorld) {
    if world
        .state()
        .browser
        .is_some_and(|browser| browser.sketches.iter().any(|sketch| sketch.visible))
    {
        super::super::workspace_layout::native_part::prepare_sketch_menu_fixture(world);
    }
    let browser = world.state().browser.expect("expected active part browser");
    assert!(browser.sketches.iter().all(|sketch| !sketch.visible));
    assert!(
        browser
            .features
            .iter()
            .all(|feature| !feature.kind.to_lowercase().contains("bend"))
    );
}

fn bend_point(world: &mut ApplicationWorld) -> [i32; 2] {
    let bend = world.run_in_window(EXPANSION, "BendPart").clone();
    let menu = world.run("Add to Quick Access Toolbar").clone();
    let point = [bend.minimum[0] + 1., bend.center()[1]];
    assert!(
        point[0] < menu.minimum[0] - 8.,
        "Bend Part point {point:?} must be left of the menu {:?}",
        menu.minimum
    );
    diagnostics::pointer_pixels(point)
}

#[when("the user clicks Bend Part outside that menu")]
fn click_bend(world: &mut ApplicationWorld) {
    let point = bend_point(world);
    world.click(point);
}

#[then("the context menu closes and a notice says a visible unadaptive sketch is needed")]
fn prerequisite_notice(world: &mut ApplicationWorld) {
    world.expect_no_text("Add to Quick Access Toolbar");
    assert!(diagnostics::window_geometry(NOTICE).is_some());
    world.expect_text("A visible unadaptive sketch is needed to bend the part.");
}

#[then("the Modify additional command area remains visible behind the notice")]
fn expansion_behind_notice(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EXPANSION).is_some());
}

#[when("the user acknowledges the notice")]
fn acknowledge(world: &mut ApplicationWorld) {
    let point = world.run_in_window(NOTICE, "OK").center();
    world.click(diagnostics::pointer_pixels(point));
}

#[then("the notice closes and the Modify additional command area closes")]
fn notice_and_expansion_close(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(NOTICE).is_none());
    assert!(diagnostics::window_geometry(EXPANSION).is_none());
}

#[then("no bend feature has been created")]
fn no_bend_feature(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected active part browser");
    assert!(
        browser
            .features
            .iter()
            .all(|feature| !feature.kind.to_lowercase().contains("bend"))
    );
}

#[cfg(target_os = "linux")]
#[test]
fn x11_click_outside_menu_reaches_bend_notice() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    no_visible_sketch(&mut world);
    super::opened_unpinned(&mut world);
    let move_bodies = world.run_in_window(EXPANSION, "MoveBodies").center();
    super::super::x11_input::right_click(diagnostics::pointer_pixels(move_bodies));
    world.pump(Vec::new());
    let point = bend_point(&mut world);
    super::super::x11_input::click(point);
    world.pump(Vec::new());
    prerequisite_notice(&mut world);
    expansion_behind_notice(&mut world);
    let ok = world.run_in_window(NOTICE, "OK").center();
    super::super::x11_input::click(diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    notice_and_expansion_close(&mut world);
    no_bend_feature(&mut world);
    world.close();
}

#[cfg(target_os = "linux")]
#[test]
fn x11_visible_sketch_gets_unavailable_notice_not_a_false_prerequisite() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    world.add_finished_sketch();
    let browser = world.state().browser.expect("expected active part browser");
    assert!(browser.sketches.iter().any(|sketch| sketch.visible));
    super::opened_unpinned(&mut world);
    let bend = world.run_in_window(EXPANSION, "BendPart").center();
    super::super::x11_input::click(diagnostics::pointer_pixels(bend));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(NOTICE).is_some());
    world.expect_text("Bend Part is not available in this build.");
    world.expect_no_text("A visible unadaptive sketch is needed to bend the part.");
    let ok = world.run_in_window(NOTICE, "OK").center();
    super::super::x11_input::click(diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    notice_and_expansion_close(&mut world);
    no_bend_feature(&mut world);
    world.close();
}
