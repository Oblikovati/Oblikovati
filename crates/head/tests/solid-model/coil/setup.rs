// SPDX-License-Identifier: GPL-2.0-only
//! Shared Coil desktop BDD helpers and coil-01 creation steps.
use super::super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::{FeatureOutputMode, GuiCoilPreview};
use oblikovati_model::feature::sketch_coil::CoilMethod;
use oblikovati_viewport::ViewportPhysicalKey;
use std::path::Path;

pub(super) fn click_coil_button(world: &mut ApplicationWorld, label: &str) {
    let pixel = super::super::super::diagnostics::pointer_pixels(
        world.run_in_window("Coil", label).center(),
    );
    world.click(pixel);
    world.pump(Vec::new());
}

pub(super) fn focus_and_replace_coil_field(world: &mut ApplicationWorld, label: &str, value: &str) {
    let row = world.run_in_window("Coil", label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry("Coil").unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        row.center()[1],
    ]);
    world.click(field);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::A, true);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Backspace, false);
    world.pump(Vec::new());
    world.type_text(value);
    world.pump(Vec::new());
}

fn ensure_origin_expanded(world: &mut ApplicationWorld) {
    if world.frame.runs.iter().any(|run| run.text == "YAxis") {
        return;
    }
    let origin = world.run("Origin").clone();
    let arrow = super::super::super::diagnostics::pointer_pixels([
        origin.minimum[0] - 10.,
        origin.center()[1],
    ]);
    world.click(arrow);
    world.pump(Vec::new());
    if !world.frame.runs.iter().any(|run| run.text == "YAxis") {
        world.click_text("Origin");
        world.pump(Vec::new());
    }
}

pub(super) fn pick_origin_y_axis(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    let y_axis = super::super::super::diagnostics::pointer_pixels(world.run("YAxis").center());
    world.click(y_axis);
    world.pump(Vec::new());
}

fn draw_coil_circle_and_finish(world: &mut ApplicationWorld) {
    world.click_text("Circle");
    world.pump(Vec::new());
    let center =
        crate::HeadWindow::model_pixel(&world.app().window.native, [2.0, 2.5, 0.0]).unwrap();
    world.click(center);
    world.pump(Vec::new());
    world.type_text("5");
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    assert_eq!(
        world.edited_geometry().circles().len(),
        1,
        "expected drawn coil profile circle"
    );
    world.finish_sketch();
    world.pump(Vec::new());
}

pub(super) fn reopen_coil1(world: &mut ApplicationWorld) {
    let row = world.run("Coil1").center();
    world.right_click(super::super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
    let edit = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| world.frame.is_popup_row(run) && run.text == "EditFeature")
        .unwrap_or_else(|| panic!("expected EditFeature popup item, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(edit));
    world.pump(Vec::new());
    assert!(super::super::super::diagnostics::window_geometry("Coil").is_some());
}

pub(super) fn scene_bounds(world: &mut ApplicationWorld) -> ([f32; 3], [f32; 3]) {
    let mut lo = [f32::INFINITY; 3];
    let mut hi = [f32::NEG_INFINITY; 3];
    for tri in &world.app().window.scene_triangles {
        for p in tri.positions {
            for i in 0..3 {
                lo[i] = lo[i].min(p[i]);
                hi[i] = hi[i].max(p[i]);
            }
        }
    }
    (lo, hi)
}

fn open_coil_fixture(world: &mut ApplicationWorld) {
    let source = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/docking-visible-solid.opd");
    let copy = world.scratch_folder().join("coil-start.opd");
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("coil fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
    let visible_sketches: Vec<_> = world
        .state()
        .browser
        .unwrap()
        .sketches
        .into_iter()
        .filter(|sketch| sketch.visible)
        .map(|sketch| sketch.id)
        .collect();
    for sketch in visible_sketches {
        world
            .app()
            .owner
            .set_sketch_visibility_from_gui(document, sketch, false)
            .unwrap();
    }
    world.pump(Vec::new());
}

#[given("a part has no available coil profile")]
pub(super) fn no_visible_coil_profile(world: &mut ApplicationWorld) {
    open_coil_fixture(world);
    let browser = world.state().browser.unwrap();
    assert!(
        browser.sketches.iter().all(|s| !s.visible),
        "expected no visible sketch in docking-visible-solid.opd"
    );
    assert_eq!(browser.bodies.len(), 1);
}

#[when("Coil is opened and a planar face is selected")]
pub(super) fn open_coil_and_select_xy_base_face(world: &mut ApplicationWorld) {
    if !world.shows("Coil") {
        let create = world.run("CREATE▼").center();
        world.click(super::super::super::diagnostics::pointer_pixels(create));
        world.pump(Vec::new());
    }
    world.click_text("Coil");
    world.pump(Vec::new());
    world.click_text("XY Plane");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the MCAD Software starts a sketch on that face")]
pub(super) fn starts_sketch_on_face(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
}

#[when("a circle is drawn and the sketch is finished")]
pub(super) fn draw_circle_and_finish(world: &mut ApplicationWorld) {
    draw_coil_circle_and_finish(world);
}

#[then("Coil opens with one profile selected and requests an axis")]
pub(super) fn coil_opens_with_profile_requests_axis(world: &mut ApplicationWorld) {
    assert!(super::super::super::diagnostics::window_geometry("Coil").is_some());
    let panel = super::super::super::world::squeezed(&world.window_text("Coil"));
    assert!(panel.contains("1Profile"), "{panel}");
    assert!(panel.contains("SelectAxis"), "{panel}");
    let preview = world.state().coil_preview.unwrap();
    assert_eq!(preview.result, GuiCoilPreview::MissingAxis);
}

#[when("the origin Y axis is selected")]
pub(super) fn select_origin_y_axis(world: &mut ApplicationWorld) {
    pick_origin_y_axis(world);
}

#[then("a helical preview appears with Pitch and Revolution, 15 mm pitch and 2.5 revolutions")]
pub(super) fn helical_preview_defaults_appear(world: &mut ApplicationWorld) {
    let preview = world.state().coil_preview.unwrap();
    assert_eq!(preview.result, GuiCoilPreview::Ready);
    assert_eq!(preview.request.method, CoilMethod::PitchAndRevolution);
    assert_eq!(preview.request.pitch, "15 mm");
    assert_eq!(preview.request.revolutions, "2.5");
    assert!(!world.app().window.scene_triangles.is_empty());
}

#[when("New Solid is chosen and accepted")]
pub(super) fn choose_new_solid_and_accept(world: &mut ApplicationWorld) {
    click_coil_button(world, "NewSolid");
    assert_eq!(
        world.state().coil_preview.unwrap().request.output,
        FeatureOutputMode::NewSolid
    );
    click_coil_button(world, "OK");
}

#[then("Coil1 appears in the browser and the solid body count increases to two")]
pub(super) fn coil1_and_two_bodies(world: &mut ApplicationWorld) {
    world.expect_text("Coil1");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 2);
    assert!(
        browser
            .features
            .iter()
            .any(|f| f.name == "Coil1" && f.kind == "Coil")
    );
}

pub(super) fn prepare_coil1_open_for_edit(world: &mut ApplicationWorld) {
    no_visible_coil_profile(world);
    open_coil_and_select_xy_base_face(world);
    starts_sketch_on_face(world);
    draw_coil_circle_and_finish(world);
    coil_opens_with_profile_requests_axis(world);
    click_coil_button(world, "NewSolid");
    pick_origin_y_axis(world);
    helical_preview_defaults_appear(world);
    click_coil_button(world, "OK");
    coil1_and_two_bodies(world);
    reopen_coil1(world);
}
