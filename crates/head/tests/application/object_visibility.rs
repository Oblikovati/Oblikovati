// SPDX-License-Identifier: GPL-2.0-only
//! Drive the published Object Visibility checklist through the View ribbon.
use super::{ApplicationWorld, diagnostics, world::squeezed};
use cucumber::{given, then, when};
use oblikovati_persistence::package::DocumentPackage;

const MASTER: &str = "All Workfeatures";
const ORIGIN_PLANES: &str = "Origin Planes";

#[given("the MCAD Software has a part document open")]
#[given("the MCAD Software has a solid part document open")]
fn open_part(world: &mut ApplicationWorld) {
    let stored = include_bytes!(concat!(
        env!("CARGO_MANIFEST_DIR"),
        "/../../test-utilities/fixtures/document-saving/SaveAsFolder/ExistingSolidPart.opd"
    ));
    let package = DocumentPackage::decode(stored).unwrap();
    let mut restored = package.restore_sketches().unwrap();
    let profile = restored.sketches.sketches()[0].id();
    restored.sketches.set_visible(profile, true).unwrap();
    let package = package
        .with_sketches(
            &restored.sketches,
            restored.sketch_ids.high_water(),
            restored.entity_ids.high_water(),
        )
        .unwrap();
    let path = world.scratch_folder().join("visibility-solid.opd");
    std::fs::write(&path, package.encode().unwrap()).unwrap();
    world.open_fixture(&path);
    let state = world.state();
    assert_eq!(
        state.document_storage[&world.active_id()].path.as_deref(),
        path.to_str()
    );
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[given("the View ribbon is selected")]
fn view_selected(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.expect_text("Object Visibility");
}

fn click(world: &mut ApplicationWorld, label: &str) {
    let at = diagnostics::pointer_pixels(world.run(label).center());
    #[cfg(target_os = "linux")]
    super::desktop_input::click(world, at);
    #[cfg(not(target_os = "linux"))]
    world.click(at);
}

#[when("I open Object Visibility")]
fn open_checklist(world: &mut ApplicationWorld) {
    click(world, "Object Visibility");
}

fn entries(world: &ApplicationWorld) -> Vec<&diagnostics::RenderedRun> {
    let master = world.run(MASTER);
    assert!(world.frame.is_popup_row(master));
    world
        .frame
        .runs
        .iter()
        .filter(|row| row.list == master.list)
        .collect()
}

#[then("a checklist opens beneath the ribbon control")]
fn opens_beneath_trigger(world: &mut ApplicationWorld) {
    let trigger = world.run("Object Visibility");
    let first = entries(world)[0];
    assert!(first.minimum[1] > trigger.maximum[1]);
    assert!((trigger.minimum[0] - first.minimum[0]).abs() < 40.);
}

#[then("the viewport and model browser remain visible behind the checklist")]
fn workspace_remains(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Model").is_some());
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}

fn table_entries(step: &cucumber::gherkin::Step) -> impl Iterator<Item = String> + '_ {
    step.table
        .as_ref()
        .expect("expected checklist entries")
        .rows
        .iter()
        .flatten()
        .map(|name| squeezed(name))
}

#[then("the checklist contains the following entries in order:")]
fn lists_in_order(world: &mut ApplicationWorld, step: &cucumber::gherkin::Step) {
    let shown: Vec<_> = entries(world).iter().map(|row| row.text.clone()).collect();
    assert_eq!(shown, table_entries(step).collect::<Vec<_>>());
}

#[then("a separator appears before the UCS entries")]
fn separates_ucs(world: &mut ApplicationWorld) {
    let center = |label| world.run(label).center()[1];
    let normal = center("3D Annotations") - center("Sketch Dimensions");
    let separator = center("UCS Triad") - center("3D Annotations");
    assert!(
        separator > normal + 1.,
        "UCS gap {separator}, row pitch {normal}"
    );
}

fn expect_checks(world: &ApplicationWorld, want: impl Fn(&str) -> bool) {
    for row in entries(world) {
        assert_eq!(world.checked_run(row), want(&row.text), "{}", row.text);
    }
}

#[given("all object visibility entries are checked")]
fn starts_checked(world: &mut ApplicationWorld) {
    open_checklist(world);
    expect_checks(world, |_| true);
}

#[given("All Workfeatures and its related categories are unchecked")]
fn master_starts_unchecked(world: &mut ApplicationWorld) {
    starts_checked(world);
    click(world, MASTER);
    assert!(!world.checked(MASTER));
}

#[when("I uncheck All Workfeatures")]
#[when("I check All Workfeatures")]
fn toggle_master(world: &mut ApplicationWorld) {
    click(world, MASTER);
}

#[then("the menu remains open")]
fn menu_open(world: &mut ApplicationWorld) {
    entries(world);
    assert!(!world.checked(MASTER));
}

#[then("the following entries are unchecked:")]
fn listed_unchecked(world: &mut ApplicationWorld, step: &cucumber::gherkin::Step) {
    for name in table_entries(step) {
        assert!(!world.checked(&name), "{name}");
    }
}

#[then("the following entries remain checked:")]
fn listed_checked(world: &mut ApplicationWorld, step: &cucumber::gherkin::Step) {
    for name in table_entries(step) {
        assert!(world.checked(&name), "{name}");
    }
}

#[then("the related origin, user work-feature, and UCS entries are checked")]
#[then("the other category checks remain unchanged")]
fn restored(world: &mut ApplicationWorld) {
    expect_checks(world, |_| true);
}

#[when("I uncheck Origin Planes")]
fn toggle_planes(world: &mut ApplicationWorld) {
    click(world, ORIGIN_PLANES);
}

#[then("Origin Planes is unchecked")]
fn planes_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked(ORIGIN_PLANES));
}

#[then("All Workfeatures remains checked")]
fn master_stays_checked(world: &mut ApplicationWorld) {
    assert!(world.checked(MASTER));
}

#[then("every other entry remains checked")]
fn only_planes_unchecked(world: &mut ApplicationWorld) {
    expect_checks(world, |name| name != "OriginPlanes");
}

#[test]
fn object_visibility_hides_and_restores_origin_axis_and_sketch_dimensions() {
    use super::workspace_layout::browser_feature::colored_pixels;
    let mut world = ApplicationWorld::new();
    open_part(&mut world);
    view_selected(&mut world);
    world.click_text("X Axis");
    open_checklist(&mut world);
    let blue_axis = [64, 153, 255];
    assert!(colored_pixels(&mut world, blue_axis) > 0);
    click(&mut world, "Origin Axes");
    assert_eq!(colored_pixels(&mut world, blue_axis), 0);
    click(&mut world, "Origin Axes");
    assert!(colored_pixels(&mut world, blue_axis) > 0);
    world.expect_text("40 mm");
    click(&mut world, "Sketch Dimensions");
    world.expect_no_text("40 mm");
    click(&mut world, "Sketch Dimensions");
    world.expect_text("40 mm");
    workspace_remains(&mut world);
    world.close();
}

#[test]
fn object_visibility_filters_sketches_without_editing_stored_visibility() {
    use crate::HeadSketchWork as _;
    let mut world = ApplicationWorld::new();
    open_part(&mut world);
    view_selected(&mut world);
    let document = world.active_id();
    let before = world.state();
    let sketch = before.browser.as_ref().unwrap().sketches[0].id;
    assert_eq!(published_sketches(&mut world), [sketch]);

    open_checklist(&mut world);
    click(&mut world, "2D Sketches");
    assert!(published_sketches(&mut world).is_empty());
    world.expect_no_text("40 mm");
    let filtered = world.state();
    assert!(filtered.browser.unwrap().sketches[0].visible);
    assert_eq!(filtered.undo_labels, before.undo_labels);
    assert!(!filtered.document_storage[&document].unsaved_edits);
    workspace_remains(&mut world);
    click(&mut world, "2D Sketches");
    assert_eq!(published_sketches(&mut world), [sketch]);
    world.expect_text("40 mm");

    world.escape();
    world
        .app()
        .owner
        .set_sketch_visibility_from_gui(document, sketch, false)
        .unwrap();
    world.pump(Vec::new());
    open_checklist(&mut world);
    click(&mut world, "2D Sketches");
    click(&mut world, "2D Sketches");
    assert!(published_sketches(&mut world).is_empty());
    assert!(!world.state().browser.unwrap().sketches[0].visible);

    click(&mut world, "2D Sketches");
    world.escape();
    world
        .app()
        .owner
        .edit_existing_sketch_from_gui(document, sketch)
        .unwrap();
    world.pump(Vec::new());
    assert_eq!(published_sketches(&mut world), [sketch]);
    world.app().owner.edit_sketch_from_gui(false).unwrap();
    world.pump(Vec::new());
    assert!(published_sketches(&mut world).is_empty());
    world.close();
}

fn published_sketches(
    world: &mut ApplicationWorld,
) -> Vec<oblikovati_model::document_sketch::DocumentSketchId> {
    world
        .app()
        .window
        .native
        .sketch_scene_status()
        .unwrap()
        .sketches
        .iter()
        .map(|entry| entry.sketch)
        .collect()
}
