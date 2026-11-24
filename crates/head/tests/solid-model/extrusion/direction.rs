// SPDX-License-Identifier: GPL-2.0-only
//! Compare native direction drafts and verify the accepted geometry.
use super::acceptance::feature_count;
use super::viewport_interaction::{click_text_native, preview_changed_outside_windows};
use super::*;

#[given("the rectangle extrusion preview is open at Default direction")]
fn default_direction(world: &mut ApplicationWorld) {
    super::selected_rectangle_preview(world);
    world.expect_text("Direction");
    let view = &mut world.app().window.native;
    crate::HeadWindow::navigate_view(
        view,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.2 },
    )
    .unwrap();
    world.pump(Vec::new());
    world.last_capture = Some(super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}

#[when("Flipped is selected in Direction")]
fn choose_flipped(world: &mut ApplicationWorld) {
    click_text_native(world, "Flipped");
    world.pump(Vec::new());
}

#[then("the preview changes its side or extent relative to Default direction")]
fn preview_is_flipped(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert_eq!(feature_count(world), 0);
    world.expect_text("10 mm");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(preview_changed_outside_windows(
        world.last_capture.as_ref().expect("Default preview"),
        &capture,
        &windows,
    ));
}

#[test]
fn flipped_literal_preview_and_commit_use_the_same_direction() {
    use oblikovati_api::{client::Client, wire::BodyPhysicalPropertiesArgs};
    let mut world = ApplicationWorld::new();
    default_direction(&mut world);
    choose_flipped(&mut world);
    preview_is_flipped(&mut world);
    click_text_native(&mut world, "OK");
    assert_eq!(feature_count(&mut world), 1);
    let body = Client::new(world.app().owner.document_host())
        .body()
        .physical_properties(&BodyPhysicalPropertiesArgs {
            body_index: 0,
            density_g_cm3: 1.,
            accuracy: String::new(),
        })
        .unwrap();
    assert!((body.centroid_z_mm + 5.).abs() < 1e-8);
    world.close();
}
