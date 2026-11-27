// SPDX-License-Identifier: GPL-2.0-only
//! Drive the published Appearance Browser tests with native input and a project library.
use super::{ApplicationWorld, asset_browser_shared as shared, diagnostics};
use crate::HeadWork as _;
use cucumber::{given, then, when};
use oblikovati_api::types::ASSET_DOCUMENT;
use oblikovati_model::assets::AppearanceAsset;

const TITLE: &str = "Appearance Browser";
const BRASS: &str = "brass-polished";
const BRASS_CAPTION: &str = "BrassPolished";
const ASSIGN: &str = "Assign to Selection";

#[given("the Appearance Browser is open")]
fn browser_open(world: &mut ApplicationWorld) {
    load_brass_project(world);
    seed_appearances(world);
    shared::open_tool(world, TITLE);
}

fn load_brass_project(world: &mut ApplicationWorld) {
    let folder = world.scratch_folder();
    let design = folder.join("design");
    std::fs::create_dir_all(&design).unwrap();
    let path = folder.join("appearance-browser.opj");
    std::fs::write(
        &path,
        "schemaVersion: 1\nname: Appearance Browser\nworkspace: work\ndesignData: design\n",
    )
    .unwrap();
    std::fs::write(design.join("appearances.yaml"),
        "appearances:\n  - {id: brass-polished, name: Brass Polished, albedo: '#C4A048', metallic: 1.0, roughness: 0.12, opacity: 1}\n  - {id: brass-satin, name: Brass Satin, albedo: '#C4A048', metallic: 1.0, roughness: 0.55, opacity: 1}\n"
    ).unwrap();
    world.app().owner.load_desktop_project(&path).unwrap();
}

fn seed_appearances(world: &mut ApplicationWorld) {
    let document = world.active_id();
    let catalog = oblikovati_persistence::assets::load_builtin_catalog().unwrap();
    let base = catalog.appearances().get("default").unwrap();
    let owner = &mut world.app().owner;
    owner
        .document_commands(document)
        .run("Seed appearance previews", |snapshot, _| {
            for fuzzy in [false, true] {
                snapshot
                    .assets
                    .embed_appearance(&finish_example(base, fuzzy))?;
            }
            Ok::<(), String>(())
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
}

fn finish_example(base: &AppearanceAsset, fuzzy: bool) -> AppearanceAsset {
    let (id, name) = if fuzzy {
        ("red-fuzz", "Red Fuzz")
    } else {
        ("blue-gloss", "Blue Gloss")
    };
    let mut info = base.duplicate(id, name, ASSET_DOCUMENT).unwrap().info();
    if fuzzy {
        [info.base.color.r, info.base.color.g, info.base.color.b] = [0.85, 0.15, 0.10];
        [info.fuzz.color.r, info.fuzz.color.g, info.fuzz.color.b] = [0.1, 0.95, 0.2];
        info.fuzz.weight = 1.;
        info.specular.roughness = 0.82;
    } else {
        [info.base.color.r, info.base.color.g, info.base.color.b] = [0.12, 0.24, 0.9];
        info.specular.roughness = 0.05;
    }
    AppearanceAsset::from_info(info).unwrap()
}

#[then("the upper document region shows labeled appearance previews")]
#[given("document appearances and library appearances are displayed as thumbnails")]
fn document_previews(world: &mut ApplicationWorld) {
    let heading = label(world, "Alllibraries").center()[1];
    assert!(label(world, "BlueGloss").center()[1] < heading);
    assert!(label(world, "RedFuzz").center()[1] < heading);
    assert!(!world.window_text(TITLE).contains("Generic"));
    library_previews(world);
}

#[then("the lower library region shows a category tree and labeled appearance thumbnails")]
#[then("the library appearances remain in their thumbnail grid")]
fn library_previews(world: &mut ApplicationWorld) {
    label(world, "Allcategories");
    let heading = label(world, "Alllibraries").center()[1];
    let first = label(world, "ABS(Black)");
    let white = label(world, "ABS(White)");
    assert!(first.center()[1] > heading && white.center()[1] > heading);
    assert!(white.center()[1] <= first.center()[1] + 100.);
}

#[then("preview tiles can show different colors, finishes and textures")]
fn rendered_finishes(world: &mut ApplicationWorld) {
    let blue = label(world, "BlueGloss").clone();
    let red = label(world, "RedFuzz").clone();
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    assert!(preview_pixels(&capture, &blue).any(|[r, _, b, _]| b > r.saturating_add(80)));
    assert!(preview_pixels(&capture, &red).any(|[r, _, b, _]| r > b.saturating_add(80)));
    assert!(
        preview_pixels(&capture, &red)
            .any(|[r, g, b, _]| g > r.saturating_add(20) && g > b.saturating_add(50))
    );
    let rows = world.state().browser.unwrap().appearances;
    let finish = |id| {
        rows.iter()
            .find(|row| row.id == id)
            .unwrap()
            .appearance_preview
            .unwrap()
    };
    assert!(finish("blue-gloss").roughness < finish("red-fuzz").roughness);
}

fn preview_pixels<'a>(
    capture: &'a oblikovati_viewport::ViewportCapture,
    caption: &diagnostics::RenderedRun,
) -> impl Iterator<Item = [u8; 4]> + 'a {
    let [x, y] = diagnostics::pointer_pixels(caption.minimum).map(|p| p as usize);
    let width = capture.extent[0] as usize;
    (y.saturating_sub(61)..y.saturating_sub(5)).flat_map(move |row| {
        (x..x + 76).map(move |column| {
            let index = (row * width + column) * 4;
            capture.rgba[index..index + 4].try_into().unwrap()
        })
    })
}

#[given(expr = "the document has no appearance matching {string}")]
fn no_document_match(world: &mut ApplicationWorld, term: String) {
    assert!(
        !world
            .state()
            .browser
            .unwrap()
            .appearances
            .iter()
            .any(|row| row.document && row.name.to_lowercase().contains(&term))
    );
}

#[given("the active library includes matching brass appearances")]
#[given("the active library contains a brass appearance not yet in the document")]
fn brass_in_library(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    for id in [BRASS, "brass-satin"] {
        let row = browser.appearances.iter().find(|row| row.id == id).unwrap();
        assert!(row.library.is_some() && !row.document);
    }
}

#[then("the library region shows matching appearance thumbnails")]
fn brass_thumbnails(world: &mut ApplicationWorld) {
    let heading = label(world, "Alllibraries").center()[1];
    assert!(label(world, BRASS_CAPTION).center()[1] > heading);
    assert!(label(world, "BrassSatin").center()[1] > heading);
    assert!(!world.window_text(TITLE).contains("Generic"));
}

#[then("the result tree groups matches by library and category")]
fn search_group(world: &mut ApplicationWorld) {
    label(world, "Projectlibrary");
    label(world, "Uncategorized");
}

#[when("I choose List View from the document appearance display menu")]
fn list_view(world: &mut ApplicationWorld) {
    open_display(world);
    world.click_text("Thumbnail View");
    world.click_text("List View");
    world.escape();
    world.pump(Vec::new());
}

#[then("document appearances appear in rows with Name, Type and Category columns")]
fn document_columns(world: &mut ApplicationWorld) {
    let boundary = label(world, "Alllibraries").center()[1];
    for heading in ["Name", "Type", "Category", "Generic"] {
        assert!(label(world, heading).center()[1] < boundary);
    }
}

#[when("I open the document appearance display menu")]
fn open_display(world: &mut ApplicationWorld) {
    let at = label(world, "Documentview").center();
    world.click(diagnostics::pointer_pixels(at));
    world.pump(Vec::new());
}

#[then("it offers sorting by Name, Type, Material Color and Category")]
fn sort_choices(world: &mut ApplicationWorld) {
    shared::inspect_options(
        world,
        "Name",
        &["Name", "Type", "Material Color", "Category"],
    );
}

#[given("no eligible model object is selected")]
fn no_target(world: &mut ApplicationWorld) {
    assert!(!world.state().browser.unwrap().root_selected);
}

#[when("I open the context menu of a matching library appearance")]
fn open_context_menu(world: &mut ApplicationWorld) {
    super::material_browser_scenarios::search_in_browser(world, TITLE, "brass");
    let point = diagnostics::pointer_pixels(label(world, BRASS_CAPTION).center());
    #[cfg(target_os = "linux")]
    super::desktop_input::right_click(world, point);
    #[cfg(not(target_os = "linux"))]
    world.right_click(point);
}

#[then("Assign to Selection is disabled")]
fn assignment_is_disabled(world: &mut ApplicationWorld) {
    assert!(world.frame.is_popup_row(world.run(ASSIGN)));
    assert!(!world.run_is_enabled(ASSIGN));
}

#[then("the menu includes Rename, Delete, Where Used and Add to")]
fn menu_entries(world: &mut ApplicationWorld) {
    for name in ["Rename", "Delete", "Where Used", "Add to"] {
        assert!(world.frame.is_popup_row(world.run(name)));
    }
}

#[when("I choose Assign to Selection from that appearance's context menu")]
fn assign_brass(world: &mut ApplicationWorld) {
    open_context_menu(world);
    assert!(world.run_is_enabled(ASSIGN));
    world.click_text(ASSIGN);
    world.pump(Vec::new());
}

#[then("the appearance is added to the document appearances")]
fn document_brass(world: &mut ApplicationWorld) {
    let row = world
        .state()
        .browser
        .unwrap()
        .appearances
        .into_iter()
        .find(|row| row.id == BRASS)
        .unwrap();
    assert!(row.document && row.used);
    let boundary = label(world, "Alllibraries").center()[1];
    let footer = label(world, "Appearance").center()[1];
    let (position, size) = diagnostics::window_geometry(TITLE).unwrap();
    let (document, library): (Vec<_>, Vec<_>) = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            let [x, y] = run.center();
            run.text == BRASS_CAPTION
                && x >= position[0]
                && x <= position[0] + size[0]
                && y + 4. < footer
        })
        .partition(|run| run.center()[1] < boundary);
    assert_eq!(document.len(), 1);
    assert_eq!(library.len(), 1);
}

#[then("the appearance selector shows the assigned appearance")]
fn assigned_appearance(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().browser.unwrap().part_appearance.as_deref(),
        Some(BRASS)
    );
    world.run_in_window("##application-ribbon", BRASS_CAPTION);
}

#[then("the material selector retains its previously assigned material")]
fn material_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().browser.unwrap().part_material.as_deref(),
        Some("steel")
    );
    let text = world.window_text("##application-ribbon");
    assert!(text.contains("Steel"), "{text}");
}

fn label<'a>(world: &'a ApplicationWorld, name: &str) -> &'a diagnostics::RenderedRun {
    world.run_in_window(TITLE, name)
}
