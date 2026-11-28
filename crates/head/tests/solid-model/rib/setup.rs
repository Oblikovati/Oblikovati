// SPDX-License-Identifier: GPL-2.0-only
//! Shared desktop BDD helpers for Rib scenarios (rib-01..rib-05).
use super::super::super::*;
use oblikovati_viewport::ViewportPhysicalKey;

pub(super) const RIB_WINDOW: &str = "Rib";
pub(super) const SKETCH_TIP_WINDOW: &str = "Rib - Sketch Required";
pub(super) const CREATE_FAILED_WINDOW: &str = "Create rib feature failed";
pub(super) const RANGE_WARNING_WINDOW: &str = "Rib Warning";

pub(super) fn open_disposable_rib_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy).unwrap_or_else(|error| panic!("rib fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
}

pub(super) fn ensure_sketch_visible(world: &mut ApplicationWorld, name: &str) {
    let document = world.state().document.expect("expected active document");
    let sketch = world
        .state()
        .browser
        .expect("expected browser")
        .sketches
        .into_iter()
        .find(|s| s.name == name)
        .unwrap_or_else(|| panic!("expected sketch {name:?} in browser"));
    if !sketch.visible {
        world
            .app()
            .owner
            .set_sketch_visibility_from_gui(document, sketch.id, true)
            .unwrap();
        world.pump(Vec::new());
        world.pump(Vec::new());
    }
}

pub(super) fn click_rib_ribbon(world: &mut ApplicationWorld) {
    if !world.shows("Rib") {
        let create = world.run("CREATE▼").center();
        world.click(super::super::super::diagnostics::pointer_pixels(create));
        world.pump(Vec::new());
    }
    world.click_text("Rib");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_rib_control(world: &mut ApplicationWorld, label: &str) {
    let center = world.run_in_window(RIB_WINDOW, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_modal_control(world: &mut ApplicationWorld, window: &str, label: &str) {
    let center = world.run_in_window(window, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn select_rib_combo(world: &mut ApplicationWorld, label: &str, popup_text: &str) {
    let row = world.run_in_window(RIB_WINDOW, label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(RIB_WINDOW).unwrap();
    let combo = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        row.center()[1],
    ]);
    world.click(combo);
    world.pump(Vec::new());
    let squeezed_target = super::super::super::world::squeezed(popup_text);
    let item = world
        .frame
        .runs
        .iter()
        .find(|run| {
            world.frame.is_popup_row(run)
                && super::super::super::world::squeezed(&run.text) == squeezed_target
        })
        .unwrap_or_else(|| {
            panic!(
                "expected {popup_text:?} in Rib {label} dropdown, saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn replace_rib_thickness(world: &mut ApplicationWorld, value: &str) {
    let label = world.run_in_window(RIB_WINDOW, "Thickness").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry(RIB_WINDOW).unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        label.center()[1],
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

pub(super) fn shell_block_top_face(world: &mut ApplicationWorld) {
    for candidate in [[-2.0, -1.25, 2.0], [-1.0, -1.25, 2.0], [-3.0, -1.25, 2.0]] {
        let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, candidate).unwrap();
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world.state().selected_model_face.is_some() {
            break;
        }
    }
    assert!(
        world.state().selected_model_face.is_some(),
        "expected selecting the block's top face before shelling"
    );
    world.click_text("Shell");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry("Shell").is_some(),
        "expected Shell dialog to open, saw {}",
        world.text()
    );
    let ok_enabled = world.run_is_enabled("OK");
    let ok = world.run_in_window("Shell", "OK").center();
    world.click(super::super::super::diagnostics::pointer_pixels(ok));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry("Shell").is_none(),
        "expected Shell dialog to close after shelling top face; ok_enabled={ok_enabled}, status={}, shell_text={}, faces={:?}, notice={:?}",
        world.window_text("##status-bar"),
        world.window_text("Shell"),
        world.state().selected_face_appearances,
        world.app().owner.take_command_notice()
    );
    let browser = world.state().browser.unwrap();
    assert!(
        browser.features.iter().any(|f| f.name.starts_with("Shell")),
        "expected Shell feature in browser, saw {:?}",
        browser.features.iter().map(|f| &f.name).collect::<Vec<_>>()
    );
}
