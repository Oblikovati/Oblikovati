// SPDX-License-Identifier: GPL-2.0-only
//! Exercise accepted startup planes through the preferences and template chooser.
use super::super::{ApplicationWorld, viewport_orientation};
use cucumber::{given, then, when};
use oblikovati_gui::GuiAuthoringContext;
use oblikovati_model::feature::work_plane::WorkPlaneOrigin;
use oblikovati_viewport::ViewportPhysicalKey;

#[given("No new sketch is the starting preference")]
fn startup_baseline(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().part_options.unwrap().startup_sketch_plane,
        None
    );
    for _ in 0..3 {
        world.add_part();
    }
}

#[when("the user accept each automatic startup plane in turn")]
fn accept_startup_planes(world: &mut ApplicationWorld) {
    let mut observed = Vec::new();
    for (before, choice, plane, name) in [
        ("No new sketch", "x-y", WorkPlaneOrigin::XY, "Part4"),
        ("x-y", "y-z", WorkPlaneOrigin::YZ, "Part5"),
        ("y-z", "x-z", WorkPlaneOrigin::XZ, "Part6"),
    ] {
        set_startup_plane(world, before, choice, Some(plane));
        create_metric_part(world);
        observe_startup(world, plane, name);
        observed.push(name);
        world.finish_sketch();
    }
    world.observed_rows = Some(observed.join(","));
}

fn set_startup_plane(
    world: &mut ApplicationWorld,
    before: &str,
    choice: &str,
    plane: Option<WorkPlaneOrigin>,
) {
    world.click_text("Tools");
    world.click_text("Application Options");
    world.click_text(before);
    world.click_text(choice);
    world.click_text("Apply");
    assert_eq!(
        world.state().part_options.unwrap().startup_sketch_plane,
        plane
    );
    world.escape();
}

fn create_metric_part(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Metric");
    world.click_text("Standard metric part");
    world.expect_text("millimeters (mm)");
    world.click_text("Create");
    world.pump(Vec::new());
}

fn observe_startup(world: &mut ApplicationWorld, plane: WorkPlaneOrigin, name: &str) {
    let id = world.active_id();
    assert_eq!(world.document_name(id), name);
    let state = world.state();
    assert_eq!(state.authoring, GuiAuthoringContext::Sketch);
    let sketches = state.browser.unwrap().sketches;
    assert_eq!(sketches.len(), 1);
    assert_eq!(sketches[0].name, "Sketch1");
    assert_eq!(state.active_sketch, Some(sketches[0].id));
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(id)
        .unwrap()
        .snapshot();
    assert_eq!(
        snapshot.sketches.get(sketches[0].id).unwrap().placement(),
        oblikovati_model::document_sketch::SketchPlacement::OriginPlane(plane)
    );
    let camera = world.app().window.native.view.camera_state().unwrap();
    let (eye, up) = match plane {
        WorkPlaneOrigin::XY => ([0., 0., 20.], [0., 1., 0.]),
        WorkPlaneOrigin::YZ => ([20., 0., 0.], [0., 1., 0.]),
        WorkPlaneOrigin::XZ => ([0., 20., 0.], [0., 0., 1.]),
    };
    assert_eq!((camera.eye, camera.target, camera.up), (eye, [0.; 3], up));
    let label = match plane {
        WorkPlaneOrigin::XY => "Front",
        WorkPlaneOrigin::YZ => "Right",
        WorkPlaneOrigin::XZ => "Top",
    };
    viewport_orientation::expect_control_shows(world, label);
}

#[when("the user create a fresh metric part document after each change")]
fn metric_parts_created(world: &mut ApplicationWorld) {
    assert_eq!(world.observed_rows.as_deref(), Some("Part4,Part5,Part6"));
}

#[then("x-y creates Part4 Sketch1 in FRONT view")]
#[then("y-z creates Part5 Sketch1 in RIGHT view")]
#[then("x-z creates Part6 Sketch1 in TOP orientation")]
#[then("no separate Start 2D Sketch action is needed in those routes")]
fn observed_each_route(world: &mut ApplicationWorld) {
    assert_eq!(world.observed_rows.as_deref(), Some("Part4,Part5,Part6"));
}

#[then("No new sketch is restored accepted and verified")]
fn restore_startup_baseline(world: &mut ApplicationWorld) {
    set_startup_plane(world, "x-z", "No new sketch", None);
    assert_eq!(
        world.state().part_options.unwrap().startup_sketch_plane,
        None
    );
}
