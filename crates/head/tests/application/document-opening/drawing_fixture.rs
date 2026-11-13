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
    Client::new(world.app().owner.document_host())
        .documents()
        .open(&oblikovati_api::wire::OpenDocumentArgs {
            full_document_name: source.to_string_lossy().into_owned(),
            ..Default::default()
        })
        .unwrap();
    settle(world);
    let drawing = folder.join(DRAWING_FILE);
    write_drawing(&drawing, &source);
    world.saved_path = Some(drawing);
    world.observed_counts = vec![world.state().documents.len()];
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
    assert_eq!(state.documents.len(), world.observed_counts[0] + 1);
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
