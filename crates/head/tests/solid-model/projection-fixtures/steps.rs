// SPDX-License-Identifier: GPL-2.0-only
//! Build the projection source through the native part and sketch tools.
use super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;

#[given("a standard millimetre part template is selected")]
fn select_metric_template(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::N, true);
    world.click_text("Standard metric part");
    world.expect_text("millimeters (mm)");
}

#[when("the recorded rectangular profile is created")]
fn create_projection_profile(world: &mut ApplicationWorld) {
    world.click_text("Create");
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    world.click_text("XZ Plane");
    world.pump(Vec::new());
    world.click_text("Rectangle");
    let plane = world.app().owner.active_sketch_edit().unwrap().plane;
    let origin = world
        .app()
        .window
        .native
        .view
        .sketch_pixel(plane, [0., 0.])
        .unwrap();
    world.click(origin);
    world.pump(vec![oblikovati_viewport::ViewportEvent::Input(
        oblikovati_viewport::ViewportInput::Motion {
            position: [1000, 600],
            modifiers: Default::default(),
        },
    )]);
    // The recording did not retain dimensions; this fixture chooses 10 mm (#3976).
    for key in [ViewportPhysicalKey::Tab, ViewportPhysicalKey::Enter] {
        world.type_text("10 mm");
        world.key(key, false);
    }
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.key(ViewportPhysicalKey::Escape, false);
    world.finish_sketch();
}

#[when("Extrude is accepted with Distance A 10 mm and zero taper")]
fn accept_projection_extrusion(world: &mut ApplicationWorld) {
    world.click_text("Extrude");
    let label = world.run("Distance A").clone();
    world.click([label.minimum[0] as i32 - 100, label.center()[1] as i32]);
    world.type_text("10 mm");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.expect_text("1 Profile");
    world.expect_text("10 mm");
    assert!(world.state().extrude_preview_active);
    world.click_text("OK");
    world.pump(Vec::new());
}

#[then("Solid1 is available as the source for sketch projection observations")]
fn projection_source_exists(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected a part browser");
    assert_eq!(browser.bodies.len(), 1, "expected exactly one source body");
    assert_eq!(browser.bodies[0].name, "Solid1");
    assert_eq!(browser.bodies[0].faces, 6, "expected a rectangular prism");
    assert!(
        browser.bodies[0].visible,
        "expected the source body visible"
    );
    assert_eq!(browser.features.len(), 1, "expected one extrude feature");
    assert_projection_depth(world);
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}

fn assert_projection_depth(world: &mut ApplicationWorld) {
    use oblikovati_api::{client::Client, wire::MeasureArgs};
    let client = Client::new(world.app().owner.document_host());
    let body = client
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap()
        .remove(0);
    let edges = body.edges.expect("expected named solid edges");
    let matching = edges
        .iter()
        .filter(|edge| {
            let args = MeasureArgs {
                body_index: 0,
                type_: "length".into(),
                key_a: edge.key.clone(),
                ..Default::default()
            };
            let length = client.analysis().measure(&args).unwrap();
            length.unit == "mm" && (length.value - 10.0).abs() < 1e-6
        })
        .count();
    assert!(
        matching >= 4,
        "expected four 10 mm extrusion edges, got {matching}"
    );
}

#[test]
fn projection_fixture_creates_a_visible_source_solid() {
    let mut world = ApplicationWorld::new();
    select_metric_template(&mut world);
    create_projection_profile(&mut world);
    accept_projection_extrusion(&mut world);
    projection_source_exists(&mut world);
    world.close();
}
