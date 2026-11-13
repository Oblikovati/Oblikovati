// SPDX-License-Identifier: GPL-2.0-only
//! Saved drawing with a resolvable model and a real projected view.
use super::*;
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
    drawing.add_base_view_definition(0, view).unwrap();
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n")
        .unwrap()
        .with_drawing(Some(&drawing))
        .unwrap();
    std::fs::write(path, package.encode().unwrap()).unwrap();
}
#[given("a saved drawing with a sheet, border and view is available and not already open")]
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
    write_drawing(&folder.join(DRAWING_FILE), &missing);
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
    assert!(world.document_is_open(original));
    browser_has_view(&mut world);
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
