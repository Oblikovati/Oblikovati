// SPDX-License-Identifier: GPL-2.0-only
//! Shared desktop BDD helpers for Decal scenarios (decal-01..decal-04).
use super::super::super::*;
use oblikovati_api::{
    client::Client,
    wire::{AddFeatureArgs, AddSketchImageArgs, featureargs::Extrude},
};
use oblikovati_viewport::ViewportNavigation;

pub(super) const DECAL_WINDOW: &str = "Decal";
pub(super) const IMAGE_TIP_WINDOW: &str = "Decal - Image Sketch Required";

pub(super) fn open_disposable_decal_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("decal fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
}

pub(super) fn extrude_trapezoid_20mm(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    let args = serde_json::to_value(Extrude {
        sketch_index: 0,
        profile_index: 0,
        distance: "20 mm".into(),
        ..Default::default()
    })
    .unwrap();
    client
        .features()
        .add(&AddFeatureArgs {
            kind: "extrude".into(),
            args: oblikovati_api::json::JsonValue(Some(args)),
        })
        .expect("expected trapezoid extrusion 20 mm to succeed");
    world.pump(Vec::new());
    world.pump(Vec::new());
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    world.pump(Vec::new());
}

pub(super) fn insert_decal_image(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let client = Client::new(world.app().owner.document_host());
    let reply = client
        .sketch()
        .add_image(&AddSketchImageArgs {
            sketch_index: 1,
            ref_: source.to_string_lossy().into_owned(),
            anchor: Some(vec![5.0, 5.0]),
            width: "10 mm".into(),
            height: "10 mm".into(),
            rotation: String::new(),
            opacity: 1.0,
        })
        .expect("expected sketch image insertion to succeed");
    assert_ne!(reply.entity_id, 0);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_decal_ribbon(world: &mut ApplicationWorld) {
    world.click_text("Decal");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_decal_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(DECAL_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn select_sloped_face(world: &mut ApplicationWorld) {
    // Orient to Back so the sloped +Y face connecting (-2, 2, z) to (2, 1, z) faces the camera.
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
    world.pump(Vec::new());
    let pixel =
        crate::HeadWindow::model_pixel(&world.app().window.native, [0.0, 1.5, 1.0]).unwrap();
    world.click(pixel);
    world.pump(Vec::new());
    world.pump(Vec::new());
    if world.state().selected_model_face.is_none() {
        crate::HeadWindow::navigate_view(
            &mut world.app().window.native,
            ViewportNavigation::Orbit {
                yaw: std::f64::consts::PI,
                pitch: 0.0,
            },
        )
        .unwrap();
        world.pump(Vec::new());
        let pixel =
            crate::HeadWindow::model_pixel(&world.app().window.native, [0.0, 1.5, 1.0]).unwrap();
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
    assert!(
        world.state().selected_model_face.is_some(),
        "expected sloped face click to select a model face"
    );
}
