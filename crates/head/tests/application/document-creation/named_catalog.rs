// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the active project catalog with the current named native fixtures.
use super::*;
use oblikovati_gui::{GuiProjectTemplate, GuiTemplateKind, GuiTemplateUnits};

const METRIC_TITLES: [&str; 4] = [
    "Sheet Metal (DIN)",
    "Sheet Metal (mm)",
    "Standard (DIN)",
    "Standard (mm)",
];
const INCH_TITLE: &str = "Standard (in)";

#[given(
    "a fresh MCAD Software session loads `fixtures/cad/document-creation-navigation-candidate.opj` as its active project"
)]
fn load_navigation_project(world: &mut ApplicationWorld) {
    let path = super::super::super::scenario_root()
        .join("fixtures/cad/document-creation-navigation-candidate.opj");
    world.app().owner.load_desktop_project(&path).unwrap();
    assert!(
        world
            .state()
            .projects
            .iter()
            .any(|entry| entry.active && entry.name == "DocumentCreationNavigationCandidate")
    );
}

fn is_part(entry: &GuiProjectTemplate) -> bool {
    matches!(
        entry.kind,
        GuiTemplateKind::Part | GuiTemplateKind::SheetMetal
    )
}

#[given("its Metric folder contains exactly the four named native part templates used below")]
fn metric_catalog(world: &mut ApplicationWorld) {
    let state = world.state();
    let mut titles: Vec<_> = state
        .project_templates
        .iter()
        .filter(|entry| entry.folder == GuiTemplateUnits::Metric && is_part(entry))
        .map(|entry| entry.title.as_str())
        .collect();
    titles.sort_unstable();
    assert_eq!(
        titles, METRIC_TITLES,
        "catalog issues: {:?}",
        state.project_template_issue
    );
}

#[given("its English folder contains the standard inch part template")]
fn english_catalog(world: &mut ApplicationWorld) {
    let state = world.state();
    assert!(
        state
            .project_templates
            .iter()
            .any(|entry| entry.folder == GuiTemplateUnits::Inch
                && entry.kind == GuiTemplateKind::Part
                && entry.title == INCH_TITLE
                && entry.length_unit == "in"),
        "catalog: {:?}",
        state.project_templates
    );
}

#[given("the user selects the standard inch part in the English folder")]
fn select_named_inch(world: &mut ApplicationWorld) {
    open_chooser(world);
    world.click_text("English");
    world.click_text(INCH_TITLE);
    inch_details(world);
}

#[given("the user switches to the metric folder and displays grouped Details view")]
fn metric_details(world: &mut ApplicationWorld) {
    world.click_text("Metric");
    grouped_by_default(world);
    choose_template_view(world, "Details");
}

#[then("the details pane still identifies the standard inch part and inch units")]
pub(super) fn inch_details(world: &mut ApplicationWorld) {
    expect_selected_template(world, INCH_TITLE);
    expect_selected_template(world, "Length: inches (in)");
}

fn expect_metric_order(world: &ApplicationWorld, order: impl IntoIterator<Item = &'static str>) {
    let mut last = world.run("Part").center()[1];
    for title in order {
        let row = world.run(title);
        assert!(row.minimum[0] < world.run("Template details").minimum[0]);
        let current = row.center()[1];
        assert!(
            current > last,
            "expected {title} below the previous gallery row"
        );
        last = current;
    }
}

#[given("the visible part section lists two Sheet Metal entries before two Standard entries")]
#[then("the visible part section lists the Sheet Metal entries before the Standard entries")]
fn by_name_ascending(world: &mut ApplicationWorld) {
    expect_metric_order(world, METRIC_TITLES);
}

#[then("the visible part section lists the Standard entries before the Sheet Metal entries")]
fn by_name_descending(world: &mut ApplicationWorld) {
    expect_metric_order(world, METRIC_TITLES.into_iter().rev());
}

#[when("the user clicks the Name column header")]
#[when("the user clicks the Name column header again")]
fn click_name_heading(world: &mut ApplicationWorld) {
    world.click_text("Name");
    world.pump(Vec::new());
}

#[when("the user restores Medium icons and cancels the chooser")]
fn restore_icons_and_cancel(world: &mut ApplicationWorld) {
    choose_template_view(world, "Medium Icons");
    cancel_chooser(world);
}

#[test]
fn create_from_named_template_retains_the_saved_extrusion() {
    let mut world = ApplicationWorld::new();
    let root = world.scratch_folder();
    let metric = root.join("Templates/Metric");
    std::fs::create_dir_all(&metric).unwrap();
    let source = std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/main-window-frame-start.opd");
    let template = metric.join("Shop block.opd");
    std::fs::copy(source, &template).unwrap();
    let original_bytes = std::fs::read(&template).unwrap();
    let project = root.join("shop.opj");
    std::fs::write(
        &project,
        "schemaVersion: 1\nname: Shop\nworkspace: .\ntemplates: Templates\n",
    )
    .unwrap();
    world.app().owner.load_desktop_project(&project).unwrap();
    open_chooser(&mut world);
    world.click_text("Shop block");
    world.click_text("Create");

    let state = world.state();
    let created = state.document.expect("expected a new Part");
    assert_eq!(state.documents.len(), 1);
    assert!(state.document_storage[&created].path.is_none());
    assert!(
        state
            .browser
            .unwrap()
            .features
            .iter()
            .any(|feature| feature.kind == "Extrusion")
    );
    assert_eq!(std::fs::read(template).unwrap(), original_bytes);
    world.close();
}
