// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the Loft command through native ribbon and viewport input.
#[path = "../chamfer/steps.rs"]
mod solid_chamfer;
#[path = "../coil/steps.rs"]
mod solid_coil;
#[path = "../combine/steps.rs"]
mod solid_combine;
#[path = "../decal/steps.rs"]
mod solid_decal;
#[path = "../delete_face/steps.rs"]
mod solid_delete_face;
#[path = "../derive/steps.rs"]
mod solid_derive;
#[path = "../direct_edit/steps.rs"]
mod solid_direct_edit;
#[path = "../draft/steps.rs"]
mod solid_draft;
#[path = "../emboss/steps.rs"]
mod solid_emboss;
#[path = "../fillet/steps.rs"]
mod solid_fillet;
#[path = "../hole/steps.rs"]
mod solid_hole;
#[path = "../import/steps.rs"]
mod solid_import;
#[path = "../mark/steps.rs"]
mod solid_mark;
#[path = "../rib/steps.rs"]
mod solid_rib;
#[path = "../shell/steps.rs"]
mod solid_shell;
#[path = "../slot/steps.rs"]
mod solid_slot;
#[path = "../split/steps.rs"]
mod solid_split;
#[path = "../thicken-offset/steps.rs"]
mod solid_thicken_offset;
#[path = "../thread/steps.rs"]
mod solid_thread;
#[path = "../ucs/steps.rs"]
mod solid_ucs;
#[path = "../unwrap/steps.rs"]
mod solid_unwrap;

use super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiLoftElement;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
use oblikovati_viewport::ViewportNavigation;

mod center_line;
mod conditions;
mod edit;
mod fixtures;
mod placed;
use fixtures::parallel_sections_if_needed;
mod surface;

#[given("the part contains no visible sketches or solid bodies")]
fn empty_part_has_no_loft_inputs(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert!(!browser.sketches.iter().any(|sketch| sketch.visible));
    assert!(browser.bodies.is_empty());
}

#[when("Loft is clicked")]
fn open_loft(world: &mut ApplicationWorld) {
    world.click_text("Loft");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Loft").is_some());
}

#[then("a tip says Loft requires a body or at least two visible sketches")]
fn loft_geometry_tip(world: &mut ApplicationWorld) {
    let text = world::squeezed(&world.window_text("Loft"));
    assert!(
        text.contains("Loftrequiresabodyoratleasttwovisiblesketches"),
        "{text}"
    );
}

#[then("the tip closes")]
fn loft_tip_is_dismissed(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Loft").is_none());
    assert!(world.state().loft_sections.is_empty());
}

#[given("Loft is open and Sketch1 contains a visible circle and center point")]
fn loft_is_open_on_the_parallel_circles(world: &mut ApplicationWorld) {
    parallel_sections_if_needed(world);
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1" && sketch.visible)
    );
    open_loft_with_fitted_view(world);
}

fn open_loft_with_fitted_view(world: &mut ApplicationWorld) {
    open_loft(world);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
}

#[when("the center point and then the circle edge are clicked")]
fn pick_center_then_circumference(world: &mut ApplicationWorld) {
    pick_model_points(world, [[0., 0., 0.], [1., 0., 0.]]);
}

fn pick_model_points(world: &mut ApplicationWorld, points: [[f64; 3]; 2]) {
    for point in points {
        let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point).unwrap();
        world.click(pixel);
        world.pump(Vec::new());
    }
}

#[then("the Sections list contains Point Sketch1 and Sketch1 as separate entries")]
fn loft_lists_both_sketch_entities(world: &mut ApplicationWorld) {
    let sections = world.state().loft_sections;
    assert_eq!(sections.len(), 2, "{sections:?}");
    assert!(
        sections
            .iter()
            .all(|section| section.sketch == DocumentSketchId(1))
    );
    assert_eq!(
        sections[0].element,
        GuiLoftElement::Point(SketchEntityId(2))
    );
    assert_eq!(
        sections[1].element,
        GuiLoftElement::Profile(SketchEntityId(3))
    );
    let text = world::squeezed(&world.window_text("Loft"));
    assert!(text.contains("SectionsPointSketch1Sketch1"), "{text}");
}

#[given("both circular sketches on separate parallel planes are visible")]
#[given("the two circular sketches on separate parallel planes are visible")]
fn independent_circles_are_visible(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    let names: Vec<_> = browser
        .sketches
        .iter()
        .filter(|sketch| sketch.visible)
        .map(|sketch| sketch.name.as_str())
        .collect();
    assert_eq!(names, ["Sketch1", "Sketch2"]);
    assert!(browser.bodies.is_empty());
}

#[when("their circular edges are selected in a fresh Loft dialog")]
fn pick_both_circles(world: &mut ApplicationWorld) {
    open_loft_with_fitted_view(world);
    pick_model_points(world, [[1., 0., 0.], [0.5, 0., 3.]]);
}

#[then("the Sections list contains both sketches and a tapered solid preview appears")]
fn tapped_circles_display_a_frustum(world: &mut ApplicationWorld) {
    let state = world.state();
    let sections = &state.loft_sections;
    assert_eq!(
        sections
            .iter()
            .map(|section| section.name.as_str())
            .collect::<Vec<_>>(),
        ["Sketch1", "Sketch2"]
    );
    assert!(
        sections
            .iter()
            .all(|section| matches!(section.element, GuiLoftElement::Profile(_)))
    );
    assert_eq!(state.loft_preview, Some(Ok(())));
    assert!(world.run_is_enabled("OK"));
    let text = world::squeezed(&world.window_text("Loft"));
    assert!(text.contains("SectionsSketch1Sketch2"), "{text}");
    let triangles = &world.app().window.scene_triangles;
    for (height, expected_radius) in [(0., 1.), (3., 0.5)] {
        let radius = triangles
            .iter()
            .filter(|triangle| triangle.color == crate::part_scene::NEW_SOLID_PREVIEW)
            .flat_map(|triangle| triangle.positions)
            .filter(|point| (point[2] - height).abs() < 0.001)
            .map(|point| point[0].hypot(point[1]))
            .fold(0.0_f32, f32::max);
        assert!(
            (radius - expected_radius).abs() < 0.005,
            "preview radius at {height}: {radius}"
        );
    }
    assert!(state.browser.unwrap().bodies.is_empty());
    let capture = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_preview_fill(
        &capture,
        &diagnostics::window_occlusions()
    ));
    world.save_capture("loft-tapered-solid-preview");
}

#[when("OK is clicked with the default settings")]
fn accept_loft(world: &mut ApplicationWorld) {
    click_loft_control(world, "OK");
}

#[then("Loft1 appears in the browser and the part contains one solid body")]
fn committed_loft_owns_one_solid(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Loft").is_none());
    let before = loft_body_key(world);
    let path = world.scratch_folder().join("accepted-loft.opd");
    solid_extrusion::acceptance::save_and_reopen(world, &path);
    world.pump(Vec::new());
    assert_eq!(loft_body_key(world), before);
}

fn loft_body_key(world: &mut ApplicationWorld) -> String {
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 3);
    assert!(browser.surface_bodies.is_empty());
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Loft1" && feature.id == browser.bodies[0].feature)
    );
    assert!(browser.sketches.iter().all(|sketch| !sketch.visible));
    world.expect_text("Loft1");
    browser.bodies[0].key.clone()
}

#[given(
    "Loft is open with the two circular sections selected and the Transition tab has Automatic Mapping selected"
)]
#[given("the Loft Transition tab has Automatic Mapping selected")]
fn transition_uses_automatic_mapping(world: &mut ApplicationWorld) {
    parallel_sections_if_needed(world);
    pick_both_circles(world);
    click_loft_control(world, "Transition");
    assert!(loft_panel_has_text(world, "AutomaticMapping"));
    assert!(!loft_panel_has_text(world, "Clicktoadd"));
    assert!(world.run_is_enabled("OK"));
}

#[when("Automatic Mapping is cleared")]
#[when("Automatic Mapping is selected again")]
fn toggle_automatic_mapping(world: &mut ApplicationWorld) {
    click_loft_control(world, "AutomaticMapping");
}

#[then("the Point Set list offers Click to add")]
fn manual_mapping_offers_a_point_set(world: &mut ApplicationWorld) {
    assert!(loft_panel_has_text(world, "PointSet"));
    assert!(world.run_is_enabled("Click to add"));
    assert_eq!(world.state().loft_sections.len(), 2);
}

#[then("that entry is no longer shown")]
fn automatic_mapping_hides_point_sets(world: &mut ApplicationWorld) {
    assert!(!loft_panel_has_text(world, "Clicktoadd"));
    assert!(world.run_is_enabled("OK"));
    assert_eq!(world.state().loft_preview, Some(Ok(())));
}

fn click_loft_control(world: &mut ApplicationWorld, label: &str) {
    let pixel = diagnostics::pointer_pixels(world.run_in_window("Loft", label).center());
    world.click(pixel);
    world.pump(Vec::new());
}

fn loft_panel_has_text(world: &mut ApplicationWorld, text: &str) -> bool {
    world::squeezed(&world.window_text("Loft")).contains(text)
}
