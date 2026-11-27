// SPDX-License-Identifier: GPL-2.0-only
//! Exercise shared asset browser controls in a native Part workspace.
use super::{ApplicationWorld, diagnostics};
use crate::HeadWork as _;
use cucumber::{given, then, when};

const TITLE: &str = "Material Browser";
const SEARCH: &str = "Searchassets";
const TERM: &str = "aluminum";

#[given("an asset browser is open over the workspace")]
pub(super) fn open_browser(world: &mut ApplicationWorld) {
    let document = world.active_id();
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let oak = catalog
        .materials()
        .get("oak-red")
        .unwrap()
        .duplicate(
            "document-oak",
            "Document Oak",
            oblikovati_api::types::ASSET_DOCUMENT,
        )
        .unwrap();
    let owner = &mut world.app().owner;
    owner
        .document_commands(document)
        .run("Seed document material", |snapshot, _| {
            snapshot.assets.embed_material(&oak)
        })
        .unwrap();
    owner
        .assign_part_asset_from_gui(
            document,
            oblikovati_gui::GuiPartAssetKind::Material,
            "steel",
        )
        .unwrap();
    world.pump(Vec::new());
    open_tool(world, TITLE);
}

pub(super) fn open_tool(world: &mut ApplicationWorld, title: &str) {
    if let Some((minimum, maximum)) = diagnostics::window_dock_tab_bounds(title) {
        world.click(diagnostics::pointer_pixels([
            (minimum[0] + maximum[0]) * 0.5,
            (minimum[1] + maximum[1]) * 0.5,
        ]));
    } else if diagnostics::window_geometry(title).is_none() {
        let button = super::quick_access_icons::standard_icon_center(title).unwrap();
        world.click(diagnostics::pointer_pixels(button));
    }
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(title).is_some());
}

#[then("the browser has a title, close control and search field")]
fn title_and_search(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(TITLE).is_some());
    label(world, SEARCH);
    label(world, "Clear");
}

#[then("its upper region contains document assets")]
#[then("the document assets return")]
fn document_assets(world: &mut ApplicationWorld) {
    let heading = label(world, "Documentassets").center()[1];
    let library = label(world, "Alllibraries").center()[1];
    for asset in ["DocumentOak", "Steel"] {
        assert!(
            world
                .frame
                .runs
                .iter()
                .any(|run| run.text.starts_with(asset)
                    && heading < run.center()[1]
                    && run.center()[1] < library),
            "expected {asset} above the library in {}",
            world.window_text(TITLE)
        );
    }
}

#[then("its lower region contains a library selector, category tree and asset view")]
#[then("the lower region returns to library browsing")]
pub(super) fn library_assets(world: &mut ApplicationWorld) {
    label(world, "Alllibraries");
    let categories = label(world, "Allcategories");
    let names = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Name")
        .collect::<Vec<_>>();
    assert!(
        names
            .iter()
            .any(|name| name.minimum[0] > categories.minimum[0])
    );
    world.expect_text("Built-in");
}

#[then("each region has a display-options control")]
fn independent_display_controls(world: &mut ApplicationWorld) {
    let document = label(world, "Documentview");
    let library = label(world, "Libraryview");
    assert!(document.center()[1] < library.center()[1]);
}

#[then("the model workspace remains visible behind the browser")]
#[then("the part document and modeling workspace remain open")]
fn workspace_visible(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Model").is_some());
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[given("the current document has no asset matching a known library search term")]
fn no_document_match(world: &mut ApplicationWorld) {
    let rows = world.state().browser.unwrap().materials;
    assert!(
        !rows
            .iter()
            .any(|row| row.document && row.name.to_lowercase().contains(TERM))
    );
}

#[given("the active library has matching assets")]
fn library_has_matches(world: &mut ApplicationWorld) {
    let rows = world.state().browser.unwrap().materials;
    assert!(
        rows.iter()
            .any(|row| row.library.is_some() && row.name.to_lowercase().contains(TERM))
    );
}

#[when("I enter the search term in the browser search field")]
#[given("an asset search is active")]
fn search(world: &mut ApplicationWorld) {
    click_label(world, SEARCH);
    world.type_text(TERM);
    world.pump(Vec::new());
}

#[then("the document region reports that the term was not found in the document")]
fn document_has_no_match(world: &mut ApplicationWorld) {
    label(world, "Nomatchingdocumentassets");
}

#[then("the library region shows matching assets")]
fn library_results(world: &mut ApplicationWorld) {
    let library = label(world, "Alllibraries");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text.starts_with("Aluminum") && run.center()[1] > library.center()[1])
    );
}

#[then("the library tree shows results grouped by library and category")]
fn category_groups(world: &mut ApplicationWorld) {
    let rows = world.window_text(TITLE);
    assert!(
        rows.contains("Built-in") && rows.contains("Aluminum"),
        "{rows}"
    );
}

#[when("I clear the search field")]
fn clear_search(world: &mut ApplicationWorld) {
    click_label(world, "Clear");
    label(world, SEARCH);
}

#[when("I open the document region display menu")]
pub(super) fn document_display(world: &mut ApplicationWorld) {
    click_label(world, "Documentview");
}

pub(super) fn inspect_options(world: &mut ApplicationWorld, current: &str, expected: &[&str]) {
    world.click_text(current);
    world.pump(Vec::new());
    for label in expected {
        world.expect_text(label);
    }
    world.escape();
}

#[then("it offers Show All, Show Used and Show Unused")]
fn usage_choices(world: &mut ApplicationWorld) {
    inspect_options(world, "Show All", &["Show All", "Show Used", "Show Unused"]);
}

#[then("it offers Thumbnail View, List View and Text View")]
fn view_choices(world: &mut ApplicationWorld) {
    inspect_options(
        world,
        "List View",
        &["Thumbnail View", "List View", "Text View"],
    );
}

#[then("it offers sort and thumbnail-size choices")]
fn sort_and_size(world: &mut ApplicationWorld) {
    inspect_options(world, "Name", &["Name", "Material Color", "Category"]);
    inspect_options(
        world,
        "16 px",
        &["16 px", "24 px", "32 px", "64 px", "256 px"],
    );
}

#[when("I close the asset browser")]
fn close_browser(world: &mut ApplicationWorld) {
    let close = if let Some((minimum, maximum)) = diagnostics::window_dock_tab_bounds(TITLE) {
        let height = maximum[1] - minimum[1];
        [maximum[0] - height * 0.5, minimum[1] + height * 0.5]
    } else {
        let (position, size) = diagnostics::window_geometry(TITLE).unwrap();
        [position[0] + size[0] - 13., position[1] + 10.]
    };
    world.click(diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
}

#[then("its window closes")]
fn browser_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(TITLE).is_none());
}

pub(super) fn label<'a>(world: &'a ApplicationWorld, name: &str) -> &'a diagnostics::RenderedRun {
    world.run_in_window(TITLE, name)
}

fn click_label(world: &mut ApplicationWorld, name: &str) {
    world.click(diagnostics::pointer_pixels(label(world, name).center()));
    world.pump(Vec::new());
}
