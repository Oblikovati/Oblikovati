// SPDX-License-Identifier: GPL-2.0-only
//! Real persisted assembly for the express-open journey.
use super::*;

const COMPONENT: &str = "visible-component.opd";
const OCCURRENCE: &str = "Visible:1";

fn write_assembly(folder: &Path) {
    let part = super::super::workspace_layout::browser_feature::extruded_part_bytes(
        "Extrude1", "Profile", 2.,
    );
    std::fs::write(folder.join(COMPONENT), part).unwrap();
    let assembly = format!(
        "schemaVersion: 2\ndocumentType: 2\nmodel: {{occurrences: [{{name: '{OCCURRENCE}', component: {COMPONENT}}}]}}\n"
    );
    std::fs::write(folder.join(ASSEMBLY_FILE), assembly).unwrap();
}

fn qualify_saved_assembly(world: &mut ApplicationWorld, folder: &Path, drawing: u64) {
    use oblikovati_api::wire::{CloseDocumentArgs, SaveDocumentArgs};
    open_dialog(world);
    look_in(world, folder);
    select_row(world, ASSEMBLY_FILE);
    choose_open(world);
    settle(world);
    let assembly = world.active_id();
    let part = world
        .app()
        .owner
        .documents()
        .document_reference(assembly, COMPONENT)
        .unwrap();
    for id in [part, assembly] {
        let result = Client::new(world.app().owner.document_host())
            .documents()
            .save(&SaveDocumentArgs { document: id })
            .unwrap();
        assert!(result.warnings.unwrap_or_default().is_empty());
    }
    Client::new(world.app().owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: assembly,
            force: true,
        })
        .unwrap();
    world.activate_through_tab(drawing);
    let package = oblikovati_persistence::package::DocumentPackage::decode(
        &std::fs::read(folder.join(ASSEMBLY_FILE)).unwrap(),
    )
    .unwrap();
    assert!(package.restore_saved_assembly_view().unwrap().is_some());
}
#[given("a saved assembly with visible components is available and not already open")]
fn saved_assembly(world: &mut ApplicationWorld) {
    let drawing = world.active_id();
    assert_eq!(
        world.state().document_type,
        Some(oblikovati_api::types::DOCUMENT_DRAWING)
    );
    let folder = models_folder(world);
    write_assembly(&folder);
    qualify_saved_assembly(world, &folder, drawing);
    let path = folder.join(ASSEMBLY_FILE);
    let state = world.state();
    assert!(
        state
            .document_storage
            .values()
            .all(|storage| { storage.path.as_deref() != path.to_str() }),
        "fixture assembly {path:?}: expected it not already open"
    );
    assert_eq!(world.active_id(), drawing);
    world.other_document = Some(drawing);
}
#[when("the user opens the file selection dialog")]
fn file_selection_dialog(world: &mut ApplicationWorld) {
    open_dialog(world);
}
#[when("the user selects the assembly and enable Load in Express Mode")]
fn select_express(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    look_in(world, &folder);
    select_row(world, ASSEMBLY_FILE);
    world.click_text("Load in Express Mode");
    assert!(world.checked("Load in Express Mode"));
}
#[then("the assembly opens in a new active document tab")]
fn assembly_opens(world: &mut ApplicationWorld) {
    world.expect_no_text("Look in");
    let state = world.state();
    assert_eq!(
        state.document_type,
        Some(oblikovati_api::types::DOCUMENT_ASSEMBLY)
    );
    assert_ne!(state.document, world.other_document);
}
#[then("its component geometry is visible in the viewport")]
fn component_visible(world: &mut ApplicationWorld) {
    let status = world.app().window.native.scene_status().unwrap();
    assert!(
        status.cached_assembly || status.visible_bodies == 1,
        "scene {status:?}"
    );
    assert!(
        status.scene_errors.is_empty(),
        "scene errors {:?}",
        status.scene_errors
    );
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let windows = super::super::diagnostics::window_occlusions();
    assert!(super::super::solid_visibility::visible_solid_fill(
        &capture, &windows
    ));
}
#[then("the assembly browser lists its components")]
fn assembly_components(world: &mut ApplicationWorld) {
    world.expect_text(OCCURRENCE);
    world.expect_no_text("[unresolved]");
}
#[then("the assembly ribbon offers Load Full")]
fn offers_load_full(world: &mut ApplicationWorld) {
    world.expect_text("Load Full");
    assert!(world.run_is_enabled("Load Full"));
}
#[when("the user chooses Load Full")]
fn load_full(world: &mut ApplicationWorld) {
    world.click_text("Load Full");
    settle(world);
}
#[then("the assembly remains active with its component geometry displayed")]
fn still_visible(world: &mut ApplicationWorld) {
    assembly_opens(world);
    component_visible(world);
}
#[then("Load Full is no longer displayed in the ribbon")]
fn no_load_full(world: &mut ApplicationWorld) {
    world.expect_no_text("Load Full");
}
#[then("the Finish and Simplify commands are available")]
#[then("Finish and Simplify are visible and enabled as ribbon commands; neither is invoked")]
fn full_commands(world: &mut ApplicationWorld) {
    for label in ["Finish", "Simplify"] {
        world.expect_text(label);
        assert!(world.run_is_enabled(label), "expected {label:?} enabled");
    }
}
#[test]
fn saved_assembly_fixture_opens_its_real_visible_component() {
    let mut world = ApplicationWorld::new();
    world.add_drawing();
    saved_assembly(&mut world);
    file_selection_dialog(&mut world);
    let folder = models_folder(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    assembly_opens(&mut world);
    component_visible(&mut world);
    assembly_components(&mut world);
    world.close();
}

#[test]
fn saving_assembly_stages_unselected_cache_and_reports_stale_source() {
    use oblikovati_api::wire::{SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE, SaveDocumentArgs};
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    write_assembly(&folder);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    settle(&mut world);
    let assembly = world.active_id();
    let part = world
        .app()
        .owner
        .documents()
        .document_reference(assembly, COMPONENT)
        .unwrap();
    Client::new(world.app().owner.document_host())
        .documents()
        .save(&SaveDocumentArgs { document: part })
        .unwrap();
    world.click_text(OCCURRENCE);
    let result = Client::new(world.app().owner.document_host())
        .documents()
        .save(&SaveDocumentArgs { document: assembly })
        .unwrap();
    assert!(result.warnings.unwrap_or_default().is_empty());
    let saved = std::fs::read(folder.join(ASSEMBLY_FILE)).unwrap();
    let package = oblikovati_persistence::package::DocumentPackage::decode(&saved).unwrap();
    let view = package
        .restore_saved_assembly_view()
        .unwrap()
        .expect("expected a qualified saved view");
    use sha2::Digest;
    let executable = std::fs::read("/proc/self/exe").unwrap();
    assert_eq!(
        view.producer_digest,
        format!("{:x}", sha2::Sha256::digest(executable))
    );
    assert!(!view.triangles.is_empty());
    assert!(
        view.triangles
            .iter()
            .all(|triangle| triangle.color != [0.05, 0.32, 1., 1.])
    );
    assert_eq!(view.sources.len(), 1);
    std::fs::write(
        folder.join(COMPONENT),
        b"schemaVersion: 2\ndocumentType: 1\n",
    )
    .unwrap();
    let result = Client::new(world.app().owner.document_host())
        .documents()
        .save(&SaveDocumentArgs { document: assembly })
        .unwrap();
    let warning = result.warnings.unwrap().pop().unwrap();
    assert_eq!(warning.code, SAVE_WARNING_ASSEMBLY_CACHE_UNAVAILABLE);
    assert!(warning.detail.contains("expected the package loaded"));
    let saved = std::fs::read(folder.join(ASSEMBLY_FILE)).unwrap();
    let package = oblikovati_persistence::package::DocumentPackage::decode(&saved).unwrap();
    assert_eq!(package.restore_saved_assembly_view().unwrap(), None);
    world.key(ViewportPhysicalKey::S, true);
    settle(&mut world);
    assert!(
        matches!(world.app().window.native.command_notice.as_ref(), Some(crate::DesktopCommandNotice::SavedWithWarning { detail, .. }) if detail.contains("expected the package loaded"))
    );
    world.click_text("+");
    world.click_text("Log Panel");
    settle(&mut world);
    world.expect_text("Application messages");
    let visible: String = world
        .text()
        .chars()
        .filter(|ch| !ch.is_whitespace())
        .collect();
    let expected: String = warning
        .detail
        .chars()
        .filter(|ch| !ch.is_whitespace())
        .collect();
    assert!(
        visible.contains(&expected),
        "expected the complete warning in the rendered log: {:?}",
        world.text()
    );
    world.close();
}

fn assert_qualified_cache(path: &Path) {
    let bytes = std::fs::read(path).unwrap();
    let package = oblikovati_persistence::package::DocumentPackage::decode(&bytes).unwrap();
    let view = package
        .restore_saved_assembly_view()
        .unwrap()
        .expect("expected cached presentation");
    assert!(!view.triangles.is_empty());
    assert_eq!(view.sources.len(), 1);
}

#[test]
fn save_as_copy_and_batch_store_a_qualified_assembly_cache() {
    use oblikovati_api::wire::{
        BatchSaveArgs, BatchSaveItem, SaveCopyAsArgs, SaveDocumentArgs, SaveDocumentAsArgs,
    };
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    write_assembly(&folder);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    settle(&mut world);
    let id = world.active_id();
    let part = world
        .app()
        .owner
        .documents()
        .document_reference(id, COMPONENT)
        .unwrap();
    Client::new(world.app().owner.document_host())
        .documents()
        .save(&SaveDocumentArgs { document: part })
        .unwrap();
    let copy = folder.join("copy.oad");
    let response = Client::new(world.app().owner.document_host())
        .documents()
        .save_copy_as(&SaveCopyAsArgs {
            document: id,
            target_file_name: copy.to_str().unwrap().into(),
            metadata: None,
        })
        .unwrap();
    assert!(response.warnings.unwrap_or_default().is_empty());
    assert_qualified_cache(&copy);
    let renamed = folder.join("renamed.oad");
    let response = Client::new(world.app().owner.document_host())
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: id,
            new_full_document_name: renamed.to_str().unwrap().into(),
        })
        .unwrap();
    assert!(response.warnings.unwrap_or_default().is_empty());
    assert_qualified_cache(&renamed);
    let batch_copy = folder.join("batch-copy.oad");
    let response = Client::new(world.app().owner.document_host())
        .documents()
        .batch_save(&BatchSaveArgs {
            operation: "saveCopyAs".into(),
            items: Some(vec![BatchSaveItem {
                document: id,
                target_file_name: batch_copy.to_str().unwrap().into(),
            }]),
        })
        .unwrap();
    let item = response.results.unwrap().pop().unwrap();
    assert!(item.ok, "{}", item.error);
    assert!(item.warnings.unwrap_or_default().is_empty());
    assert_qualified_cache(&batch_copy);
    world.close();
}
