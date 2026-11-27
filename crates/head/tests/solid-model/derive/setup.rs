// SPDX-License-Identifier: GPL-2.0-only
//! Shared desktop BDD helpers for Derived Part scenarios (derive-01..derive-06).
use super::super::super::*;
use oblikovati_gui::GuiDerivePreview;
use oblikovati_viewport::ViewportPhysicalKey;

pub(super) const PICKER_WINDOW: &str = "Select Component";
pub(super) const DERIVE_WINDOW: &str = "Derived Part";

pub(super) fn open_disposable_destination(world: &mut ApplicationWorld, fixture_rel: &str) {
    let source = super::super::super::scenario_root().join(fixture_rel);
    let dest = world.scratch_folder().join("dest.opd");
    std::fs::copy(&source, &dest)
        .unwrap_or_else(|error| panic!("derive destination fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&dest, true)
        .unwrap();
    world.other_document = Some(document);
    world.pump(Vec::new());
    world.activate_through_tab(document);
}

pub(super) fn stage_source_component(world: &mut ApplicationWorld, fixture_rel: &str) {
    let source = super::super::super::scenario_root().join(fixture_rel);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("derive source fixture {source:?}: {error}"));
    world.saved_path = Some(copy);
}

pub(super) fn click_derive_ribbon(world: &mut ApplicationWorld) {
    if !world.shows("Derive") {
        world.click_text("Manage");
        world.pump(Vec::new());
    }
    world.click_text("Derive");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn select_and_open_source_in_picker(world: &mut ApplicationWorld) {
    assert!(
        super::super::super::diagnostics::window_geometry(PICKER_WINDOW).is_some(),
        "expected {PICKER_WINDOW} picker to be open, saw {}",
        world.text()
    );
    let source_path = world
        .saved_path
        .clone()
        .expect("expected staged derive source path");
    let file_name = source_path
        .file_name()
        .and_then(|name| name.to_str())
        .expect("expected UTF-8 source filename")
        .to_string();
    let row = world
        .run_in_window(PICKER_WINDOW, &file_name.replace(' ', ""))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    let open = world.run_in_window(PICKER_WINDOW, "Open").center();
    world.click(super::super::super::diagnostics::pointer_pixels(open));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn open_derive_dialog_for_source(world: &mut ApplicationWorld) {
    click_derive_ribbon(world);
    select_and_open_source_in_picker(world);
    assert!(
        super::super::super::diagnostics::window_geometry(DERIVE_WINDOW).is_some(),
        "expected {DERIVE_WINDOW} dialog to open, saw {}",
        world.text()
    );
    assert_eq!(
        world
            .state()
            .derive_preview
            .as_ref()
            .map(|preview| &preview.result),
        Some(&GuiDerivePreview::Ready),
        "expected certified derive preview"
    );
}

pub(super) fn click_derive_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(DERIVE_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn select_derive_tab(world: &mut ApplicationWorld, tab: &str) {
    let center = world.run_in_window(DERIVE_WINDOW, tab).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

pub(super) fn choose_mirror_plane(world: &mut ApplicationWorld, plane_text: &str) {
    let label = world.run_in_window(DERIVE_WINDOW, "Mirrorplane").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(DERIVE_WINDOW).unwrap();
    let combo = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        label.center()[1],
    ]);
    world.click(combo);
    world.pump(Vec::new());
    let squeezed = super::super::super::world::squeezed(plane_text);
    let item = world
        .frame
        .runs
        .iter()
        .find(|row| world.frame.is_popup_row(row) && row.text == squeezed)
        .unwrap_or_else(|| {
            panic!(
                "expected {plane_text} in Mirror plane dropdown, saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn replace_scale_factor(world: &mut ApplicationWorld, value: &str) {
    let row = world.run_in_window(DERIVE_WINDOW, "ScaleFactor").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(DERIVE_WINDOW).unwrap();
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
    world.pump(Vec::new());
}

pub(super) fn scene_max_span(world: &mut ApplicationWorld) -> f32 {
    let mut lo = [f32::INFINITY; 3];
    let mut hi = [f32::NEG_INFINITY; 3];
    for tri in &world.app().window.scene_triangles {
        for p in tri.positions {
            for axis in 0..3 {
                lo[axis] = lo[axis].min(p[axis]);
                hi[axis] = hi[axis].max(p[axis]);
            }
        }
    }
    (0..3)
        .map(|axis| hi[axis] - lo[axis])
        .fold(0.0_f32, f32::max)
}

pub(super) fn configure_separate_solid_yz_mirror(world: &mut ApplicationWorld) {
    open_derive_dialog_for_source(world);
    select_derive_tab(world, "Objects");
    click_derive_control(world, "Maintaineachsolidasasolidbody");
    select_derive_tab(world, "Options");
    click_derive_control(world, "Mirrorpart");
    choose_mirror_plane(world, "YZ Plane");
}

pub(super) fn derived_browser_name(world: &ApplicationWorld) -> String {
    world
        .saved_path
        .as_ref()
        .and_then(|path| path.file_name())
        .and_then(|name| name.to_str())
        .expect("expected staged derive source filename")
        .to_string()
}
