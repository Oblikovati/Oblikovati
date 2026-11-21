// SPDX-License-Identifier: GPL-2.0-only
//! Persistence and reference-lifetime tests for saved drawings.
use super::*;

#[test]
fn drawing_open_loads_a_closed_model_without_a_visible_part_tab() {
    let mut world = ApplicationWorld::new();
    let original = world.add_part();
    let folder = models_folder(&mut world);
    let source = folder.join("source.opd");
    let drawing = folder.join(DRAWING_FILE);
    write_source_model(&source);
    write_drawing(&drawing, &source);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, DRAWING_FILE);
    choose_open(&mut world);
    assert_eq!(
        world
            .state()
            .documents
            .iter()
            .filter(|row| row.visible)
            .count(),
        2,
        "only the original part and drawing have tabs"
    );
    assert!(world.document_is_open(original));
    sheet_and_border(&mut world);
    browser_has_view(&mut world);
    world.close_active_tab();
    assert_eq!(
        world.state().documents.len(),
        1,
        "closing the last drawing releases its background model"
    );
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, DRAWING_FILE);
    choose_open(&mut world);
    sheet_and_border(&mut world);
    world.close();
}

#[test]
fn drawing_with_a_missing_model_keeps_the_document_open_and_reports_the_path() {
    let mut world = ApplicationWorld::new();
    let original = world.add_part();
    let folder = models_folder(&mut world);
    let missing = folder.join("missing.opd");
    let file = folder.join(DRAWING_FILE);
    write_drawing(&file, &missing);
    let original_bytes = std::fs::read(&file).unwrap();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, DRAWING_FILE);
    choose_open(&mut world);
    let state = world.state();
    let drawing = state.document.unwrap();
    let error = state.drawing_projections[&drawing]
        .error
        .as_deref()
        .unwrap();
    assert!(error.contains("missing.opd"), "reference failure {error:?}");
    let save = Client::new(world.app().owner.document_host())
        .documents()
        .save(&oblikovati_api::wire::SaveDocumentArgs { document: drawing });
    assert!(
        save.is_err(),
        "a missing projection must not be saved as current"
    );
    assert_eq!(std::fs::read(&file).unwrap(), original_bytes);
    assert!(world.document_is_open(original));
    browser_has_view(&mut world);
    world.close();
}

#[test]
fn saving_a_drawing_stores_the_projected_view_geometry() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    drawing_available(&mut world);
    choose_drawing(&mut world);
    choose_open(&mut world);
    let state = world.state();
    let drawing = state.document.unwrap();
    let visible = state.drawing_projections[&drawing].lines.clone();
    assert!(!visible.is_empty(), "expected projected view lines");
    world.press_save_shortcut();
    settle(&mut world);
    assert_saved_view(world.saved_path.as_ref().unwrap(), &visible);
    let folder = models_folder(&mut world);
    let renamed = folder.join("renamed.odd");
    let copy = folder.join("copy.odd");
    Client::new(world.app().owner.document_host())
        .documents()
        .save_as(&oblikovati_api::wire::SaveDocumentAsArgs {
            document: drawing,
            new_full_document_name: renamed.to_string_lossy().into_owned(),
        })
        .unwrap();
    assert_saved_view(&renamed, &visible);
    Client::new(world.app().owner.document_host())
        .documents()
        .save_copy_as(&oblikovati_api::wire::SaveCopyAsArgs {
            document: drawing,
            target_file_name: copy.to_string_lossy().into_owned(),
            metadata: None,
        })
        .unwrap();
    assert_saved_view(&copy, &visible);
    Client::new(world.app().owner.document_host())
        .drawing()
        .set_model_reference(&oblikovati_api::wire::SetModelReferenceArgs {
            full_document_name: "source.opd".into(),
        })
        .unwrap();
    Client::new(world.app().owner.document_host())
        .documents()
        .close(&oblikovati_api::wire::CloseDocumentArgs {
            id: drawing,
            force: false,
        })
        .unwrap();
    assert_saved_view(&renamed, &visible);
    world.close();
}
fn assert_saved_view(path: &Path, visible: &[oblikovati_gui::GuiDrawingLine]) {
    let bytes = std::fs::read(path).unwrap();
    let package = DocumentPackage::decode(&bytes).unwrap();
    let saved = package
        .restore_saved_drawing_views()
        .unwrap()
        .expect("expected saved view geometry");
    assert_eq!(saved.views.len(), 1);
    assert_eq!(saved.views[0].lines.len(), visible.len());
    for (actual, expected) in saved.views[0].lines.iter().zip(visible) {
        assert_eq!(actual.points, expected.points);
        assert_eq!(actual.hidden, expected.hidden);
    }
}

#[test]
fn deferred_drawing_saves_frozen_views_without_the_source_model() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    drawing_available(&mut world);
    open_drawing_picker(&mut world);
    select_deferred_drawing(&mut world);
    choose_open(&mut world);
    acknowledge_deferred_message(&mut world);
    let state = world.state();
    let drawing = state.document.unwrap();
    let visible = state.drawing_projections[&drawing].lines.clone();
    let folder = models_folder(&mut world);
    std::fs::remove_file(folder.join("source.opd")).unwrap();
    world.press_save_shortcut();
    settle(&mut world);
    assert_saved_view(world.saved_path.as_ref().unwrap(), &visible);
    let renamed = folder.join("deferred-renamed.odd");
    let copy = folder.join("deferred-copy.odd");
    let host = world.app().owner.document_host();
    let client = Client::new(host);
    client
        .documents()
        .save_as(&oblikovati_api::wire::SaveDocumentAsArgs {
            document: drawing,
            new_full_document_name: renamed.to_string_lossy().into_owned(),
        })
        .unwrap();
    client
        .documents()
        .save_copy_as(&oblikovati_api::wire::SaveCopyAsArgs {
            document: drawing,
            target_file_name: copy.to_string_lossy().into_owned(),
            metadata: None,
        })
        .unwrap();
    assert_saved_view(&renamed, &visible);
    assert_saved_view(&copy, &visible);
    client
        .drawing()
        .set_model_reference(&oblikovati_api::wire::SetModelReferenceArgs {
            full_document_name: "changed.opd".into(),
        })
        .unwrap();
    let before = std::fs::read(&renamed).unwrap();
    let failure = client
        .documents()
        .save(&oblikovati_api::wire::SaveDocumentArgs { document: drawing })
        .unwrap_err();
    assert!(
        failure
            .to_string()
            .contains("expected the saved drawing recipe")
    );
    assert_eq!(std::fs::read(&renamed).unwrap(), before);
    client
        .documents()
        .close(&oblikovati_api::wire::CloseDocumentArgs {
            id: drawing,
            force: true,
        })
        .unwrap();
    drop(client);
    world.close();
}

#[test]
fn batch_save_reports_a_missing_model_and_saves_the_other_drawing() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    let source = folder.join("source.opd");
    write_source_model(&source);
    write_drawing(&folder.join(DRAWING_FILE), &source);
    write_drawing(
        &folder.join(SECOND_DRAWING_FILE),
        &folder.join("missing.opd"),
    );
    let mut ids = Vec::new();
    for file in [SECOND_DRAWING_FILE, DRAWING_FILE] {
        open_dialog(&mut world);
        look_in(&mut world, &folder);
        select_row(&mut world, file);
        choose_open(&mut world);
        ids.push(world.active_id());
    }
    let targets = [folder.join("bad-copy.odd"), folder.join("good-copy.odd")];
    let items = ids
        .iter()
        .zip(&targets)
        .map(|(&document, target)| oblikovati_api::wire::BatchSaveItem {
            document,
            target_file_name: target.to_string_lossy().into_owned(),
        })
        .collect();
    let result = Client::new(world.app().owner.document_host())
        .documents()
        .batch_save(&oblikovati_api::wire::BatchSaveArgs {
            operation: "saveCopyAs".into(),
            items: Some(items),
        })
        .unwrap();
    let outcomes = result.results.unwrap();
    assert_eq!(result.saved, 1);
    assert!(!outcomes[0].ok && outcomes[0].error.contains("missing.opd"));
    assert!(outcomes[1].ok);
    assert!(!targets[0].exists());
    let package = DocumentPackage::decode(&std::fs::read(&targets[1]).unwrap()).unwrap();
    assert!(package.restore_saved_drawing_views().unwrap().is_some());
    world.close();
}

#[test]
fn drawings_share_a_relative_background_model_until_the_last_one_closes() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    write_source_model(&folder.join("source.opd"));
    for file in [DRAWING_FILE, SECOND_DRAWING_FILE] {
        write_drawing(&folder.join(file), Path::new("source.opd"));
        open_dialog(&mut world);
        look_in(&mut world, &folder);
        select_row(&mut world, file);
        choose_open(&mut world);
        sheet_and_border(&mut world);
    }
    assert_eq!(
        world.state().documents.len(),
        4,
        "one model shared by two drawings"
    );
    world.close_active_tab();
    assert_eq!(
        world.state().documents.len(),
        3,
        "the first drawing retains its model"
    );
    sheet_and_border(&mut world);
    world.close_active_tab();
    assert_eq!(
        world.state().documents.len(),
        1,
        "the last drawing releases its model"
    );
    for file in [DRAWING_FILE, SECOND_DRAWING_FILE] {
        open_dialog(&mut world);
        look_in(&mut world, &folder);
        select_row(&mut world, file);
        choose_open(&mut world);
    }
    let closed = Client::new(world.app().owner.document_host())
        .documents()
        .close_all(&oblikovati_api::wire::CloseAllDocumentsArgs { force: true })
        .unwrap();
    assert_eq!(
        closed.closed, 4,
        "close all counts the shared background model once"
    );
    world.close();
}

#[test]
fn opening_a_drawing_reuses_and_preserves_a_visible_model() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    let source = folder.join("source.opd");
    write_source_model(&source);
    write_drawing(&folder.join(DRAWING_FILE), &source);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, "source.opd");
    choose_open(&mut world);
    let model = world.active_id();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, DRAWING_FILE);
    choose_open(&mut world);
    assert_eq!(
        world.state().documents.len(),
        3,
        "the visible model is reused"
    );
    sheet_and_border(&mut world);
    let close = Client::new(world.app().owner.document_host())
        .documents()
        .close(&oblikovati_api::wire::CloseDocumentArgs {
            id: model,
            force: true,
        });
    assert!(close.is_err(), "a referenced model must remain available");
    sheet_and_border(&mut world);
    world.close_active_tab();
    assert!(
        world.document_is_open(model),
        "the user's model remains open"
    );
    world.close();
}

fn assert_loaded_reference(world: &mut ApplicationWorld, expected: &Path, absent: &Path) {
    let state = world.state();
    let id = state.document.unwrap();
    assert_eq!(
        state.drawing_documents[&id].model_reference(),
        expected.to_str().unwrap()
    );
    let paths: Vec<_> = state
        .document_storage
        .values()
        .filter_map(|storage| storage.path.as_deref())
        .collect();
    assert!(
        paths.contains(&expected.to_str().unwrap()),
        "expected model {expected:?}"
    );
    assert!(
        !paths.contains(&absent.to_str().unwrap()),
        "obsolete model {absent:?} remained open"
    );
    let projection = &state.drawing_projections[&id];
    assert!(
        projection.error.is_none(),
        "projection: {:?}",
        projection.error
    );
    assert!(!projection.lines.is_empty(), "expected projected geometry");
}

#[test]
fn changing_a_drawing_reference_reloads_models_on_undo_and_redo() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    let first = folder.join("first.opd");
    let second = folder.join("second.opd");
    write_source_model(&first);
    write_source_model(&second);
    write_drawing(&folder.join(DRAWING_FILE), &first);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, DRAWING_FILE);
    choose_open(&mut world);
    assert_loaded_reference(&mut world, &first, &second);
    Client::new(world.app().owner.document_host())
        .drawing()
        .set_model_reference(&oblikovati_api::wire::SetModelReferenceArgs {
            full_document_name: second.to_string_lossy().into_owned(),
        })
        .unwrap();
    assert_loaded_reference(&mut world, &second, &first);
    settle(&mut world);
    world.command_key(
        ViewportPhysicalKey::Z,
        oblikovati_viewport::ViewportKey::Undo,
    );
    assert_loaded_reference(&mut world, &first, &second);
    world.command_key(
        ViewportPhysicalKey::Y,
        oblikovati_viewport::ViewportKey::Redo,
    );
    assert_loaded_reference(&mut world, &second, &first);
    let missing = folder.join("missing.opd");
    Client::new(world.app().owner.document_host())
        .drawing()
        .set_model_reference(&oblikovati_api::wire::SetModelReferenceArgs {
            full_document_name: missing.to_string_lossy().into_owned(),
        })
        .unwrap();
    let state = world.state();
    let error = state.drawing_projections[&state.document.unwrap()]
        .error
        .as_deref()
        .unwrap();
    assert!(
        error.contains("missing.opd"),
        "expected missing model path: {error:?}"
    );
    Client::new(world.app().owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_loaded_reference(&mut world, &second, &first);
    Client::new(world.app().owner.document_host())
        .transactions()
        .begin(&oblikovati_api::wire::TransactionBeginArgs {
            label: "Try source".into(),
        })
        .unwrap();
    Client::new(world.app().owner.document_host())
        .drawing()
        .set_model_reference(&oblikovati_api::wire::SetModelReferenceArgs {
            full_document_name: first.to_string_lossy().into_owned(),
        })
        .unwrap();
    assert_loaded_reference(&mut world, &first, &second);
    Client::new(world.app().owner.document_host())
        .transactions()
        .abort()
        .unwrap();
    assert_loaded_reference(&mut world, &second, &first);
    world.close();
}
