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
    let part = oblikovati_drawing::evaluate_part(&history, guid).unwrap();
    let lines = oblikovati_drawing::DrawingProjectionService
        .saved_lines_current(&history, guid, &view, &part)
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

#[given("the MCAD Software has an existing document open")]
fn existing_document(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_part());
}
#[given(
    "two saved drawings with available references and visible saved view geometry are not already open"
)]
fn two_closed_drawings(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    let source = folder.parent().unwrap().join("source.opd");
    write_source_model(&source);
    for name in [DRAWING_FILE, SECOND_DRAWING_FILE] {
        write_drawing(&folder.join(name), &source);
    }
}
#[given("the Open dialog displays both drawings as thumbnails")]
fn two_drawing_thumbnails(world: &mut ApplicationWorld) {
    let folder = models_folder(world);
    open_dialog(world);
    look_in(world, &folder);
    super::selections::switch_to_tiles(world);
}
#[then("both drawings are available in separate document tabs")]
fn both_drawing_tabs(world: &mut ApplicationWorld) {
    let state = world.state();
    for name in [DRAWING_FILE, SECOND_DRAWING_FILE] {
        let row = state
            .document_storage
            .iter()
            .find(|(_, item)| {
                item.path
                    .as_deref()
                    .is_some_and(|path| path.ends_with(name))
            })
            .unwrap();
        let title = world.document_name(*row.0);
        world.expect_tab(&title);
    }
}
#[then("one of the drawings is active with its sheet and saved geometry displayed")]
fn one_drawing_with_geometry(world: &mut ApplicationWorld) {
    sheet_and_border(world);
}
#[then("the existing document tab remains available")]
fn existing_tab_remains(world: &mut ApplicationWorld) {
    let name = world.document_name(world.previous_document.unwrap());
    world.expect_tab(&name);
}
#[when("the user selects the other drawing tab")]
fn select_other_drawing_tab(world: &mut ApplicationWorld) {
    let state = world.state();
    let current = state.document.unwrap();
    let other = state
        .drawing_documents
        .keys()
        .copied()
        .find(|id| *id != current)
        .unwrap();
    world.other_document = Some(other);
    world.click_tab(other);
    settle(world);
}
#[then("that drawing becomes active and its sheet is displayed")]
fn other_drawing_active(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.other_document);
    sheet_and_border(world);
}

#[given("an assembly document is active in the MCAD Software")]
fn active_assembly(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_template(GuiTemplateKind::Assembly));
}
#[given("the Open dialog displays two saved drawings in a folder")]
fn two_drawing_rows(world: &mut ApplicationWorld) {
    two_closed_drawings(world);
    let folder = models_folder(world);
    open_dialog(world);
    look_in(world, &folder);
}
#[given("the first drawing is selected")]
fn first_drawing_selected(world: &mut ApplicationWorld) {
    select_row(world, DRAWING_FILE);
}
#[given("the second drawing has a sheet, border and saved views and is not already open")]
fn second_drawing_has_saved_views(world: &mut ApplicationWorld) {
    let path = models_folder(world).join(SECOND_DRAWING_FILE);
    let package = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    let drawing = package.restore_drawing().unwrap().unwrap();
    assert!(drawing.sheets()[0].border);
    assert!(
        !package
            .restore_saved_drawing_views()
            .unwrap()
            .unwrap()
            .views[0]
            .lines
            .is_empty()
    );
    assert!(world.state().drawing_documents.is_empty());
}
#[when("the user double-clicks the second drawing's row")]
fn double_click_second_drawing(world: &mut ApplicationWorld) {
    let position = world
        .run(SECOND_DRAWING_FILE)
        .center()
        .map(|value| value as i32);
    world.pump(super::super::pointer_click_events(position, 2));
    settle(world);
}
#[then("the second drawing opens in a new active document tab")]
fn second_drawing_active(world: &mut ApplicationWorld) {
    let state = world.state();
    let id = state.document.unwrap();
    assert!(
        state.document_storage[&id]
            .path
            .as_deref()
            .unwrap()
            .ends_with(SECOND_DRAWING_FILE)
    );
    let title = world.document_name(id);
    world.expect_tab(&title);
}
#[then("its sheet, border and saved views are displayed")]
fn second_drawing_sheet(world: &mut ApplicationWorld) {
    sheet_and_border(world);
}
#[then("the assembly tab remains available")]
fn assembly_tab_remains(world: &mut ApplicationWorld) {
    existing_tab_remains(world);
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
#[test]
fn double_click_preserves_deferred_drawing_mode() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    drawing_available(&mut world);
    std::fs::remove_file(models_folder(&mut world).join("source.opd")).unwrap();
    open_drawing_picker(&mut world);
    select_deferred_drawing(&mut world);
    let position = world.run(DRAWING_FILE).center().map(|value| value as i32);
    world.pump(super::super::pointer_click_events(position, 2));
    settle(&mut world);
    deferred_message(&mut world);
    assert_eq!(world.active_id(), part);
    acknowledge_deferred_message(&mut world);
    assert!(world.state().drawing_updates_deferred);
    deferred_sheet(&mut world);
    deferred_commands_off(&mut world);
    world.close();
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

#[cfg(test)]
#[path = "drawing_fixture_tests.rs"]
mod tests;
