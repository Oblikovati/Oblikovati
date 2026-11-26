// SPDX-License-Identifier: GPL-2.0-only
//! Drive the published Material Browser feature through its native window.
use super::{ApplicationWorld, asset_browser_shared as shared, diagnostics};
use cucumber::{given, then, when};
use oblikovati_api::contract::{Appearance, Material};

const TITLE: &str = "Material Browser";
const SEARCH: &str = "aluminum";
const MATERIAL: &str = "aluminum-2011";
const ROW: &str = "Aluminum2011";
const ASSIGN: &str = "Assign to Selection";

#[given("the Material Browser is open")]
fn browser_open(world: &mut ApplicationWorld) {
    shared::open_browser(world);
}

#[then("the upper document region lists materials with Name and Category columns")]
fn document_columns(world: &mut ApplicationWorld) {
    table_headers(world, true, &["Name", "Category"]);
}

#[then("the lower region shows the selected library, category tree and material list")]
fn library_list(world: &mut ApplicationWorld) {
    shared::library_assets(world);
    table_headers(world, false, &["Name", "Category"]);
}

#[given(expr = "the document has no material matching {string}")]
fn no_document_match(world: &mut ApplicationWorld, query: String) {
    let browser = world.state().browser.unwrap();
    assert!(
        !browser
            .materials
            .iter()
            .any(|row| row.document && row.name.to_lowercase().contains(&query))
    );
}

#[given("the active library contains matching aluminum materials")]
#[given("the selected library contains an aluminum material not yet in the document")]
fn aluminum_in_library(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    let row = browser
        .materials
        .iter()
        .find(|row| row.id == MATERIAL)
        .unwrap();
    assert!(row.library.is_some() && !row.document);
}

#[when(expr = "I search for {string}")]
fn search(world: &mut ApplicationWorld, query: String) {
    let at = shared::label(world, "Searchassets").center();
    world.click(diagnostics::pointer_pixels(at));
    world.type_text(&query);
    world.pump(Vec::new());
}

#[then("the document region reports no document match")]
fn no_results(world: &mut ApplicationWorld) {
    shared::label(world, "Nomatchingdocumentassets");
}

#[then("the library region lists matching materials with Name, Library and Category columns")]
fn search_results(world: &mut ApplicationWorld) {
    table_headers(world, false, &["Name", "Library", "Category"]);
    material_row(world, false);
}

#[given("the part is selected in the model browser")]
fn select_part(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let point =
        super::solid_visibility::visible_fixture_point(&capture, &diagnostics::window_occlusions())
            .expect("expected a visible solid before assigning aluminum");
    world.observed_view = Some(point);
    let root = format!("{}<Primary>", world.state().browser.unwrap().name);
    world.click(diagnostics::pointer_pixels(
        world.run_in_window("Model", &root).center(),
    ));
    assert!(world.state().browser.unwrap().root_selected);
    world.workspace_color = Some(world.pixel_at(point.map(|axis| axis as usize)));
}

#[when("I open that library material's context menu")]
fn open_assignment_menu(world: &mut ApplicationWorld) {
    search(world, SEARCH.into());
    let point = diagnostics::pointer_pixels(material_row(world, false).center());
    #[cfg(target_os = "linux")]
    super::desktop_input::right_click(world, point);
    #[cfg(not(target_os = "linux"))]
    world.right_click(point);
}

#[then("Assign to Selection is available")]
fn assign_is_enabled(world: &mut ApplicationWorld) {
    assert!(world.frame.is_popup_row(world.run(ASSIGN)));
    assert!(world.run_is_enabled(ASSIGN));
}

#[then("the menu includes Rename, Delete and Add to")]
fn menu_entries(world: &mut ApplicationWorld) {
    for item in ["Rename", "Delete", "Add to"] {
        assert!(world.frame.is_popup_row(world.run(item)));
    }
}

#[when("I choose Assign to Selection")]
fn assign(world: &mut ApplicationWorld) {
    world.click_text(ASSIGN);
    world.pump(Vec::new());
}

#[then("the material appears in the document materials results")]
fn document_result(world: &mut ApplicationWorld) {
    material_row(world, true);
    material_row(world, false);
}

#[then("the top material selector shows the assigned material")]
fn top_material(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().browser.unwrap().part_material.as_deref(),
        Some(MATERIAL)
    );
    let caption = world.run_in_window("##application-ribbon", "Material");
    assert!(world.frame.runs.iter().any(|run| {
        run.text.starts_with(ROW)
            && run.center()[0] > caption.center()[0]
            && (run.center()[1] - caption.center()[1]).abs() < 4.
    }));
}

#[then("the top appearance selector and rendered solid reflect the assignment")]
fn top_appearance_and_solid(world: &mut ApplicationWorld) {
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let material = catalog.materials().get(MATERIAL).unwrap();
    let appearance = catalog
        .appearances()
        .get(&material.appearance_id())
        .unwrap();
    let name = super::world::squeezed(&appearance.display_name());
    world.run_in_window("##application-ribbon", &name);
    let color = appearance.base().color;
    let submitted = &world.app().window.scene_colors;
    assert!(!submitted.is_empty());
    assert!(
        submitted
            .iter()
            .all(|rgba| *rgba == [color.r, color.g, color.b, 1.])
    );
    let point = world.observed_view.unwrap().map(|axis| axis as usize);
    assert_ne!(world.pixel_at(point), world.workspace_color.unwrap());
}

#[then("the browser remains open")]
fn remains_open(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(TITLE).is_some());
}

#[when("I open the document materials display menu")]
fn open_document_menu(world: &mut ApplicationWorld) {
    shared::document_display(world);
}

#[then("it offers sorting by Name, Material Color and Category")]
fn material_sort_choices(world: &mut ApplicationWorld) {
    shared::inspect_options(world, "Name", &["Name", "Material Color", "Category"]);
}

#[then("it offers thumbnail sizes 16, 24, 32, 64 and 256 pixels")]
fn thumbnail_sizes(world: &mut ApplicationWorld) {
    shared::inspect_options(
        world,
        "16 px",
        &["16 px", "24 px", "32 px", "64 px", "256 px"],
    );
}

fn table_headers(world: &ApplicationWorld, document: bool, headers: &[&str]) {
    let rows: Vec<_> = region(world, document).collect();
    let name = rows
        .iter()
        .find(|row| row.text == "Name")
        .expect("expected the Name column");
    for header in headers {
        assert!(
            rows.iter()
                .any(|row| row.text == *header && (row.center()[1] - name.center()[1]).abs() < 2.),
            "expected {header}: {}",
            world.window_text(TITLE)
        );
    }
}

fn material_row(world: &ApplicationWorld, document: bool) -> &diagnostics::RenderedRun {
    region(world, document)
        .find(|run| run.text.starts_with(ROW))
        .unwrap_or_else(|| {
            panic!(
                "expected {ROW} in {} (document={document})",
                world.window_text(TITLE)
            )
        })
}

fn region(
    world: &ApplicationWorld,
    document: bool,
) -> impl Iterator<Item = &diagnostics::RenderedRun> {
    let (position, size) = diagnostics::window_geometry(TITLE).unwrap();
    let heading = shared::label(world, "Documentassets").center()[1];
    let library = shared::label(world, "Alllibraries").center()[1];
    let footer = world.run_in_window(TITLE, "Assigntopart").center()[1];
    let (top, bottom) = if document {
        (heading, library)
    } else {
        (library, footer)
    };
    world.frame.runs.iter().filter(move |run| {
        let [x, y] = run.center();
        x >= position[0] && x <= position[0] + size[0] && y > top && y < bottom
    })
}
