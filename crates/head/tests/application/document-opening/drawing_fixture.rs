// SPDX-License-Identifier: GPL-2.0-only
//! Saved drawing with a resolvable model and a real projected view.
use super::*;
use oblikovati_automation::DrawingViewProjectionAccess;
use oblikovati_model::drawing::{DrawingBaseViewDefinition, DrawingDocument};
use oblikovati_persistence::package::DocumentPackage;

fn write_source_model(path: &Path) {
    let snapshot = crate::scene_fixture::selected_prism_document();
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, snapshot.parameters.units().clone())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap();
    std::fs::write(path, package.encode().unwrap()).unwrap();
}
fn write_drawing(path: &Path, source: &Path) {
    let mut drawing = DrawingDocument::new();
    drawing
        .set_model_reference(source.to_str().unwrap())
        .unwrap();
    let view = DrawingBaseViewDefinition::new(
        "VIEW:1",
        oblikovati_api::types::BASE_VIEW_FRONT,
        1.,
        [210., 148.5],
    )
    .unwrap();
    drawing.add_base_view_definition(0, view.clone()).unwrap();
    let snapshot = crate::scene_fixture::selected_prism_document();
    let ids =
        oblikovati_model::identity::RecipeIdAllocator::restore(None, snapshot.parameters.ids())
            .unwrap();
    let history = oblikovati_model::history::DocumentEditHistory::new(snapshot, ids).unwrap();
    let guid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
    let lines = oblikovati_drawing::DrawingProjectionService
        .saved_lines(&history, guid, &view)
        .unwrap();
    let views = oblikovati_persistence::package::SavedDrawingViews {
        version: oblikovati_persistence::package::SavedDrawingViews::VERSION,
        drawing_recipe: drawing.recipe().clone(),
        views: vec![oblikovati_persistence::package::SavedDrawingView {
            sheet: drawing.sheets()[0].name.clone(),
            view: view.name.clone(),
            lines,
        }],
    };
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n")
        .unwrap()
        .with_drawing(Some(&drawing))
        .unwrap()
        .with_saved_drawing_views(&views)
        .unwrap();
    std::fs::write(path, package.encode().unwrap()).unwrap();
}
#[given("a saved drawing with a sheet, border and view is available and not already open")]
#[given("a saved drawing with a sheet, border and views is available and not already open")]
fn drawing_available(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    let source = folder.join("source.opd");
    write_source_model(&source);
    let drawing = folder.join(DRAWING_FILE);
    write_drawing(&drawing, &source);
    world.saved_path = Some(drawing);
    world.observed_counts = vec![
        world
            .state()
            .documents
            .iter()
            .filter(|row| row.visible)
            .count(),
    ];
}
#[given("the deferred-update information message is enabled")]
fn deferred_message_enabled(world: &mut ApplicationWorld) {
    // ApplicationWorld gives each scenario a new preferences directory and dialog state.
    world.expect_no_text("Drawing updates are deferred");
}
#[when("the user opens the drawing's file selection dialog")]
fn open_drawing_picker(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    open_dialog(world);
    look_in(world, &folder);
}
#[when("the user selects the drawing and enable Load in Defer Mode")]
fn select_deferred_drawing(world: &mut ApplicationWorld) {
    select_row(world, DRAWING_FILE);
    assert!(!world.checked("Load in Defer Mode"));
    world.click_text("Load in Defer Mode");
    assert!(world.checked("Load in Defer Mode"));
}
#[then("an information message states that drawing updates are deferred")]
fn deferred_message(world: &mut ApplicationWorld) {
    world.expect_text("Drawing updates are deferred");
}
#[then(
    "it indicates that defer status must change to update views and associate added annotations"
)]
fn deferred_explanation(world: &mut ApplicationWorld) {
    assert!(holds_text(world, "Change the defer status to update views"));
    assert!(holds_text(world, "associate added annotations"));
}
#[then("the message offers a display preference and an OK control")]
fn deferred_controls(world: &mut ApplicationWorld) {
    world.expect_text("Do not show this message again");
    world.expect_text("OK");
}
#[when("the user acknowledges the message")]
fn acknowledge_deferred_message(world: &mut ApplicationWorld) {
    world.click_text("OK");
    settle(world);
}
#[then("the drawing opens in a new active document tab")]
fn deferred_drawing_active(world: &mut ApplicationWorld) {
    drawing_active(world);
}
#[then("the sheet, border and saved view geometry are displayed")]
fn deferred_sheet(world: &mut ApplicationWorld) {
    sheet_and_border(world);
    let path = world.saved_path.as_ref().unwrap();
    let package = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    let saved = package.restore_saved_drawing_views().unwrap().unwrap();
    let state = world.state();
    let projection = &state.drawing_projections[&state.document.unwrap()];
    let expected: Vec<_> = saved.views[0]
        .lines
        .iter()
        .map(|line| (&line.points, line.hidden))
        .collect();
    let actual: Vec<_> = projection
        .lines
        .iter()
        .map(|line| (&line.points, line.hidden))
        .collect();
    assert_eq!(actual, expected);
    assert!(!state.documents.iter().any(|row| row.name == "source"));
}
#[then("the drawing browser shows the sheet and a view")]
fn deferred_browser(world: &mut ApplicationWorld) {
    browser_has_view(world);
}
#[then("drawing view creation commands in the ribbon are unavailable")]
fn deferred_commands_off(world: &mut ApplicationWorld) {
    for command in ["Base View", "Projected View"] {
        assert!(
            !world.run_is_enabled(command),
            "{command} should be unavailable"
        );
    }
}
#[then("the original part tab remains available")]
fn deferred_part_remains(world: &mut ApplicationWorld) {
    part_tab_remains(world);
}
#[when("the user presses Control+O and selects the drawing")]
fn choose_drawing(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    open_dialog(world);
    look_in(world, &folder);
    select_row(world, DRAWING_FILE);
}
#[then("the drawing becomes active in a new document tab")]
fn drawing_active(world: &mut ApplicationWorld) {
    let state = world.state();
    assert_eq!(
        state.documents.iter().filter(|row| row.visible).count(),
        world.observed_counts[0] + 1
    );
    let active = state.document.unwrap();
    assert_eq!(
        state.document_storage[&active].path.as_deref(),
        world.saved_path.as_ref().and_then(|path| path.to_str())
    );
    let name = world.document_name(active);
    world.expect_tab(&name);
}
#[then("the drawing sheet and border are displayed")]
fn sheet_and_border(world: &mut ApplicationWorld) {
    let state = world.state();
    let drawing = state.document.unwrap();
    assert!(state.drawing_documents[&drawing].sheets()[0].border);
    let projection = &state.drawing_projections[&drawing];
    assert!(
        projection.error.is_none(),
        "projection: {:?}",
        projection.error
    );
    assert!(
        !projection.lines.is_empty(),
        "expected projected model geometry"
    );
    world.expect_text("Scale 1:1");
    let capture = super::super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    let height = capture.extent[1] as usize;
    let dark_pixels = (height * 40 / 100..height * 65 / 100)
        .flat_map(|y| (width * 50 / 100..width * 70 / 100).map(move |x| (y * width + x) * 4))
        .filter(|&at| {
            capture.rgba[at..at + 3]
                .iter()
                .all(|channel| *channel < 150)
        })
        .count();
    assert!(
        dark_pixels > 20,
        "expected visible view lines, found {dark_pixels} dark pixels"
    );
}
#[then("the drawing browser lists the sheet and a view")]
fn browser_has_view(world: &mut ApplicationWorld) {
    world.expect_text("Sheet:1");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| { run.text.contains("VIEW:1") && run.center()[0] < 300. }),
        "expected the view in the model browser"
    );
}
#[then("the part document tab remains available")]
fn part_tab_remains(world: &mut ApplicationWorld) {
    let part = world.previous_document.unwrap();
    let name = world.document_name(part);
    world.expect_tab(&name);
}

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
