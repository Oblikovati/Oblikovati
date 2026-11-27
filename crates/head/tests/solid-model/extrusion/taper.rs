// SPDX-License-Identifier: GPL-2.0-only
//! Drive Taper A through the browser, viewport, history and Part package.
use super::super::{capture_framebuffer, diagnostics, solid_visibility};
use super::acceptance::{
    feature_count, feature_distance, named_solid_edges, open_extrusion_editor,
};
use super::*;

const FIXTURE: &str = "fixtures/cad/extrusion/extrusion-rect-8mm.opd";

#[given(
    "a copy of `fixtures/cad/extrusion/extrusion-rect-8mm.opd` is open with its accepted untapered Extrusion1"
)]
#[given("the fixture contains its accepted untapered extrusion")]
#[given(r#"a disposable copy of the native fixture "fixtures/cad/extrusion/extrusion-rect-8mm.opd" is open in the part modeling workspace"#)]
pub(super) fn open_published_part(world: &mut ApplicationWorld) {
    let source = super::super::scenario_root().join(FIXTURE);
    open_copy(world, &std::fs::read(source).unwrap());
}

#[given("the accepted untapered Extrusion1")]
fn open_pinned_part(world: &mut ApplicationWorld) {
    open_copy(
        world,
        include_bytes!("../../../../../test-utilities/fixtures/extrusion/extrusion-rect-8mm.opd"),
    );
}

fn open_copy(world: &mut ApplicationWorld, package: &[u8]) {
    let path = world.scratch_folder().join("extrusion-rect-8mm.opd");
    if !path.is_file() || world.state().document.is_none() {
        std::fs::write(&path, package).unwrap();
        world.open_fixture(&path);
    }
    assert_eq!(feature_count(world), 1);
    assert_eq!(feature_distance(world), 8.);
    assert_eq!(saved_angle(world), 0.);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        oblikovati_viewport::ViewportNavigation::Home { margin: 0.12 },
    )
    .unwrap();
    world.pump(Vec::new());
    check_cap_sizes(world, 0.);
}

#[when("Edit Feature is chosen from its browser context menu")]
#[given("Edit Feature is reopened")]
#[when("the user opens the extrusion for editing")]
#[when("the user reopens the extrusion for editing")]
#[given("the user reopens the extrusion for editing")]
fn edit_feature(world: &mut ApplicationWorld) {
    open_extrusion_editor(world);
    world.pump(Vec::new());
}

#[then("Distance A is retained as 8 mm and Taper A as zero degrees")]
#[then("Distance A is 8 mm and Taper A is zero")]
fn starts_untapered(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "8mm");
    world.run_in_window("Extrude", "TaperA");
    world.run_in_window("Extrude", "0deg");
}

#[when("Taper A is replaced with 5 and OK is clicked")]
#[when("the user enters 5 degrees for Taper A and confirms the edit")]
fn accept_positive_taper(world: &mut ApplicationWorld) {
    let generation = world.state().document_generation;
    let keys = named_solid_edges(world);
    let input = diagnostics::pointer_pixels(world.run_in_window("Extrude", "0deg").center());
    world.click(input);
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("5");
    world.pump(Vec::new());
    assert!(world.state().extrude_preview_active);
    assert_eq!(world.state().document_generation, generation);
    assert_eq!(saved_angle(world), 0.);
    check_cap_sizes(world, 5.);
    check_visible_edit_preview(world);
    click_control(world, "OK");
    assert_eq!(named_solid_edges(world), keys);
    assert_eq!(feature_distance(world), 8.);
    assert!((saved_angle(world) - 5.).abs() < 1e-9);
    assert!(diagnostics::window_geometry("Extrude").is_none());
}

#[given("the user edits Extrusion1, sets Taper A to positive 5 degrees, and accepts the edit")]
fn setup_accepted_taper(world: &mut ApplicationWorld) {
    edit_feature(world);
    accept_positive_taper(world);
}

#[given("the user edits the fixture extrusion, sets Taper A to 5 degrees and accepts the edit")]
fn setup_published_taper(world: &mut ApplicationWorld) {
    open_published_part(world);
    setup_accepted_taper(world);
}

#[given("Extrusion1 has accepted positive taper of 5 degrees")]
fn setup_pinned_taper(world: &mut ApplicationWorld) {
    open_pinned_part(world);
    setup_accepted_taper(world);
}

#[then("the edited solid visibly widens toward the far end")]
#[then("the solid visibly widens toward its far end")]
fn check_accepted_shape(world: &mut ApplicationWorld) {
    check_cap_sizes(world, 5.);
    let frame = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_solid_fill(
        &frame,
        &diagnostics::window_occlusions()
    ));
    world.save_capture("extrusion-taper-accepted");
}

#[when("Undo then Redo are invoked and Edit Feature is opened")]
fn undo_redo_and_reopen(world: &mut ApplicationWorld) {
    let keys = named_solid_edges(world);
    world.key(ViewportPhysicalKey::Z, true);
    world.pump(Vec::new());
    assert_eq!(saved_angle(world), 0.);
    check_cap_sizes(world, 0.);
    world.key(ViewportPhysicalKey::Y, true);
    world.pump(Vec::new());
    check_accepted_shape(world);
    let path = world.scratch_folder().join("extrusion-taper-reopened.opd");
    acceptance::save_and_reopen(world, &path);
    world.pump(Vec::new());
    assert_eq!(named_solid_edges(world), keys);
    assert!((saved_angle(world) - 5.).abs() < 1e-9);
    edit_feature(world);
}

#[then("the widening shape returns and Taper A shows 5 degrees")]
fn check_saved_taper(world: &mut ApplicationWorld) {
    check_cap_sizes(world, 5.);
    retained_positive_taper(world);
}

#[given("Edit Feature with Taper A 5 degrees")]
fn setup_pinned_reversed_edit(world: &mut ApplicationWorld) {
    setup_pinned_taper(world);
    edit_feature(world);
}

#[when("its sign control is clicked")]
#[when("the user reverses the taper sign")]
fn reverse_taper(world: &mut ApplicationWorld) {
    click_control(world, "±");
    world.pump(Vec::new());
}

#[then("Taper A shows -5 degrees and the preview narrows")]
#[then("Taper A is -5 degrees and the preview narrows")]
fn check_reversed_draft(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "-5deg");
    assert!(world.state().extrude_preview_active);
    assert!((saved_angle(world) - 5.).abs() < 1e-9);
    check_cap_sizes(world, -5.);
    check_visible_edit_preview(world);
}

#[when("Cancel is clicked and Edit Feature is reopened")]
#[when("the user cancels the edit and reopens the extrusion for editing")]
fn cancel_reversed_taper(world: &mut ApplicationWorld) {
    let generation = world.state().document_generation;
    let keys = named_solid_edges(world);
    click_control(world, "Cancel");
    assert_eq!(world.state().document_generation, generation);
    assert_eq!(named_solid_edges(world), keys);
    check_cap_sizes(world, 5.);
    edit_feature(world);
}

#[then("the retained value is positive 5 degrees")]
#[then("Taper A is 5 degrees")]
#[then("Taper A remains 5 degrees")]
fn retained_positive_taper(world: &mut ApplicationWorld) {
    world.run_in_window("Extrude", "5deg");
    assert!((saved_angle(world) - 5.).abs() < 1e-9);
}

fn click_control(world: &mut ApplicationWorld, name: &str) {
    let position = diagnostics::pointer_pixels(world.run_in_window("Extrude", name).center());
    world.click(position);
    world.pump(Vec::new());
}

fn saved_angle(world: &mut ApplicationWorld) -> f64 {
    let document = world.active_id();
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let [feature] = snapshot.work_geometry.features() else {
        panic!("expected one extrusion");
    };
    let oblikovati_model::feature::part::PartFeatureDefinition::SketchPrism(prism) =
        feature.definition()
    else {
        panic!("expected an Extrude prism");
    };
    prism
        .taper()
        .map(|angle| angle.resolve(&snapshot.parameters).unwrap().to_degrees())
        .unwrap_or_default()
}

fn check_cap_sizes(world: &mut ApplicationWorld, degrees: f64) {
    let mesh = &world.app().window.scene_triangles;
    for (height, setback) in [(0., 0.), (0.8, 0.8 * degrees.to_radians().tan())] {
        let mut lower = [f32::INFINITY; 2];
        let mut upper = [f32::NEG_INFINITY; 2];
        for [x, y, _] in mesh
            .iter()
            .flat_map(|face| face.positions)
            .filter(|point| (f64::from(point[2]) - height).abs() < 1e-4)
        {
            for (axis, value) in [x, y].into_iter().enumerate() {
                lower[axis] = lower[axis].min(value);
                upper[axis] = upper[axis].max(value);
            }
        }
        let mut spans = std::array::from_fn::<_, 2, _>(|axis| f64::from(upper[axis] - lower[axis]));
        spans.sort_by(f64::total_cmp);
        for (actual, nominal) in spans.into_iter().zip([3., 4.]) {
            let expected = nominal + 2. * setback;
            assert!(
                (actual - expected).abs() < 1e-4,
                "cap at {height}: {actual} instead of {expected}"
            );
        }
    }
}

fn check_visible_edit_preview(world: &mut ApplicationWorld) {
    let frame = capture_framebuffer(&mut world.app().window.native);
    assert!(solid_visibility::visible_replacement_preview_fill(
        &frame,
        &diagnostics::window_occlusions()
    ));
}
