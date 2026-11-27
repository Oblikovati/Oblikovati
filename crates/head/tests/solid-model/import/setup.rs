// SPDX-License-Identifier: GPL-2.0-only
//! Shared desktop BDD helpers for Part Import scenarios (import-01..import-12).
use super::super::super::*;
use oblikovati_gui::GuiImportPreview;

pub(super) const PICKER_WINDOW: &str = "Import File";
pub(super) const STL_OPTIONS_WINDOW: &str = "STL Import Options";
pub(super) const IMPORT_WINDOW: &str = "Import";

pub(super) fn open_disposable_import_fixture(world: &mut ApplicationWorld, relative: &str) {
    let source = super::super::super::scenario_root().join(relative);
    let copy = world
        .scratch_folder()
        .join(source.file_name().unwrap().to_str().unwrap());
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("import fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
}

pub(super) fn stage_import_file(world: &mut ApplicationWorld, relative: &str) -> String {
    let source = super::super::super::scenario_root().join(relative);
    let file_name = source
        .file_name()
        .and_then(|s| s.to_str())
        .expect("expected UTF-8 exchange filename")
        .to_owned();
    let copy = world.scratch_folder().join(&file_name);
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("exchange fixture {source:?}: {error}"));
    world.saved_path = Some(copy);
    file_name
}

pub(super) fn click_import_ribbon(world: &mut ApplicationWorld) {
    if !world.shows("Import") {
        world.click_text("Manage");
        world.pump(Vec::new());
    }
    world.click_text("Import");
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn open_picker_and_select(world: &mut ApplicationWorld, relative: &str) {
    let file_name = stage_import_file(world, relative);
    click_import_ribbon(world);
    assert!(
        super::super::super::diagnostics::window_geometry(PICKER_WINDOW).is_some(),
        "expected {PICKER_WINDOW} picker to open, saw {}",
        world.text()
    );
    let row = world
        .run_in_window(PICKER_WINDOW, &file_name.replace(' ', ""))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_window_control(world: &mut ApplicationWorld, window: &str, label: &str) {
    let center = world.run_in_window(window, label).center();
    world.click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn wait_for_import_ready(world: &mut ApplicationWorld) {
    for _ in 0..4 {
        if matches!(
            world.state().import_preview.as_ref().map(|p| &p.result),
            Some(GuiImportPreview::Ready)
        ) {
            return;
        }
        world.pump(Vec::new());
    }
    assert_eq!(
        world.state().import_preview.as_ref().map(|p| &p.result),
        Some(&GuiImportPreview::Ready),
        "expected Import preview to reach Ready"
    );
}

pub(super) fn reference_feature_name(world: &mut ApplicationWorld) -> String {
    world
        .state()
        .browser
        .expect("expected browser")
        .features
        .iter()
        .find(|f| f.kind == "Imported Reference")
        .expect("expected Imported Reference feature in browser")
        .name
        .clone()
}

pub(super) fn right_click_reference_entry(world: &mut ApplicationWorld) {
    let name = reference_feature_name(world);
    let target = name.replace(' ', "");
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let squeezed = super::super::super::world::squeezed(&run.text);
            squeezed.starts_with(&target)
                && super::super::super::diagnostics::window_geometry("Model").is_some_and(
                    |(pos, size)| {
                        let c = run.center();
                        c[0] >= pos[0]
                            && c[0] <= pos[0] + size[0]
                            && c[1] >= pos[1]
                            && c[1] <= pos[1] + size[1]
                    },
                )
        })
        .unwrap_or_else(|| panic!("expected reference row {name:?} in Model browser"))
        .center();
    world.right_click(super::super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn click_popup_item(world: &mut ApplicationWorld, label: &str) {
    let target = super::super::super::world::squeezed(label);
    let item = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| {
            world.frame.is_popup_row(run)
                && super::super::super::world::squeezed(&run.text) == target
        })
        .unwrap_or_else(|| panic!("expected popup item {label:?}, saw {}", world.text()))
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
    world.pump(Vec::new());
}
