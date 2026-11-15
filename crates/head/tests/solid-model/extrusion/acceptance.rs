// SPDX-License-Identifier: GPL-2.0-only
//! Native extrusion acceptance and history before solid-edge measurement.
use super::*;

#[given("Distance A 8 mm and zero taper")]
fn eight_millimeter_preview(world: &mut ApplicationWorld) {
    super::selected_rectangle_preview(world);
    super::replace_extrusion_distance(world, "8");
    world.pump(Vec::new());
    super::positive_distance_is_accepted(world);
    assert!(
        world.state().extrude_preview_active,
        "expected an active straight-prism preview"
    );
}

#[when("OK is clicked then Undo then Redo")]
fn accept_and_replay(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 1, "accepted feature count");
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 0, "feature count after Undo");
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
    assert_eq!(feature_count(world), 1, "feature count after Redo");
}

#[then("Extrusion1 and Solid Bodies(1) return with the shaded solid")]
fn restored_solid_is_visible(world: &mut ApplicationWorld) {
    let browser = world
        .state()
        .browser
        .expect("expected a part model browser");
    assert_eq!(browser.features.len(), 1, "feature rows");
    assert_eq!(browser.features[0].name, "Extrusion1");
    assert_eq!(browser.bodies.len(), 1, "solid body rows");
    world.expect_text("Extrusion1");
    world.expect_text("Solid Bodies");
    assert_eight_millimeter_edges(world);
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(
        super::super::solid_visibility::visible_solid_fill(&capture, &windows),
        "expected a visible shaded solid after Redo"
    );
}

fn assert_eight_millimeter_edges(world: &mut ApplicationWorld) {
    let edges = named_solid_edges(world);
    let matching = edges
        .iter()
        .filter(|key| edge_is_eight_mm(world, key))
        .count();
    assert!(
        matching >= 4,
        "expected four 8 mm extrusion edges; found {matching}"
    );
}

fn named_solid_edges(world: &mut ApplicationWorld) -> Vec<String> {
    let client = oblikovati_api::client::Client::new(world.app().owner.document_host());
    let body = client
        .model()
        .reference_keys()
        .unwrap()
        .bodies
        .unwrap()
        .remove(0);
    body.edges
        .expect("expected named solid edges")
        .into_iter()
        .map(|edge| edge.key)
        .collect()
}

fn edge_is_eight_mm(world: &mut ApplicationWorld, key: &str) -> bool {
    use oblikovati_api::{client::Client, wire::MeasureArgs};
    let client = Client::new(world.app().owner.document_host());
    let value = client
        .analysis()
        .measure(&MeasureArgs {
            body_index: 0,
            type_: "length".into(),
            key_a: key.into(),
            ..Default::default()
        })
        .unwrap();
    value.unit == "mm" && (value.value - 8.0).abs() < 1e-6
}

fn feature_count(world: &mut ApplicationWorld) -> usize {
    let document = world.active_id();
    world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .work_geometry
        .features()
        .len()
}

#[test]
fn extrusion_acceptance_and_history_restore_a_visible_solid() {
    let mut world = ApplicationWorld::new();
    eight_millimeter_preview(&mut world);
    accept_and_replay(&mut world);
    restored_solid_is_visible(&mut world);
    world.close();
}

#[test]
fn free_rectangle_extrusion_survives_undo_and_redo() {
    let mut world = ApplicationWorld::new();
    super::rectangle_for_waiting_extrusion(&mut world);
    super::finish_requested_profile(&mut world);
    world.pump(Vec::new());
    super::waiting_extrusion_shows_profile(&mut world);
    super::replace_extrusion_distance(&mut world, "8");
    world.pump(Vec::new());
    accept_and_replay(&mut world);
    restored_solid_is_visible(&mut world);
    let directory = tempfile::tempdir().unwrap();
    save_and_reopen(&mut world, &directory.path().join("free-rectangle.opd"));
    world.pump(Vec::new());
    restored_solid_is_visible(&mut world);
    world.close();
}

fn save_and_reopen(world: &mut ApplicationWorld, path: &std::path::Path) {
    use oblikovati_api::{
        client::Client,
        wire::{CloseDocumentArgs, OpenDocumentArgs, SaveDocumentAsArgs},
    };
    let document = world.active_id();
    let path = path.to_str().unwrap().to_owned();
    let client = Client::new(world.app().owner.document_host());
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.clone(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path,
            ..Default::default()
        })
        .unwrap();
}
