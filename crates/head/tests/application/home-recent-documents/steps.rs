// SPDX-License-Identifier: GPL-2.0-only
use super::ApplicationWorld;
use super::diagnostics::{RenderedRun, window_geometry};
use crate::HeadWork;
use cucumber::{given, then, when};

const HOME: &str = "##home-workspace";
const OTHERS: [&str; 2] = ["Bracket", "Housing"];
const SEARCHED: &str = "Part26";
const ASSEMBLY: &str = "Frame";
const NO_MATCH: &str = "zzq_no_match_7319";
const SELECTED_ACTIONS: [&str; 2] = ["Open Selected", "Remove Selected"];
const ROW_REACH: f32 = 4.;
#[path = "date_filter.rs"]
mod date_filter;
#[path = "layout_bounds.rs"]
mod layout_bounds;
#[path = "lists.rs"]
mod lists;
#[path = "native_input.rs"]
mod native_input;
#[path = "published_profile.rs"]
mod published_profile;
use published_profile::{assembly_name, other_recent_names, recent_name};
#[path = "startup_observation.rs"]
mod startup_observation;
#[path = "workspace_chrome.rs"]
mod workspace_chrome;

/// Real files through a real save: each lands on the recent list as a person's would.
fn save_recent_files(world: &mut ApplicationWorld) -> [u64; 4] {
    world.app().owner.forget_recent_documents();
    let [searched, bracket, housing] = [SEARCHED, OTHERS[0], OTHERS[1]].map(|name| {
        let part = world.add_part();
        world.save_part_to_file(part, &format!("{name}.opd"));
        part
    });
    let assembly = world.add_template(oblikovati_gui::GuiTemplateKind::Assembly);
    world.save_part_to_file(assembly, &format!("{ASSEMBLY}.oad"));
    [searched, bracket, housing, assembly]
}
fn in_home(run: &RenderedRun) -> bool {
    let Some((position, size)) = window_geometry(HOME) else {
        return false;
    };
    let center = run.center();
    (0..2).all(|axis| center[axis] >= position[axis] && center[axis] <= position[axis] + size[axis])
}
/// A word Home shares with the browser or the ribbon is found inside Home's own window.
fn home_run(world: &ApplicationWorld, text: &str) -> Option<RenderedRun> {
    let wanted = text.replace(' ', "");
    let exact = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| in_home(run) && run.text == wanted);
    exact
        .or_else(|| {
            world
                .frame
                .runs
                .iter()
                .rev()
                .find(|run| in_home(run) && run.text.contains(&wanted))
        })
        .cloned()
}
fn click_home(world: &mut ApplicationWorld, text: &str) {
    let run = home_run(world, text)
        .unwrap_or_else(|| panic!("expected {text:?} in Home: {:?}", world.window_text(HOME)));
    world.click(run.center().map(|value| value as i32));
    world.pump(Vec::new());
}
/// Check Home for a rendered label, for example a recent file name.
pub(super) fn home_shows(world: &ApplicationWorld, name: &str) -> bool {
    home_run(world, name).is_some()
}
/// Find a clipped Home label, for example the visible prefix of a recent file name.
pub(super) fn home_ellipsized_run<'a>(
    world: &'a ApplicationWorld,
    full: &str,
) -> Option<&'a RenderedRun> {
    world.frame.runs.iter().find(|run| {
        in_home(run)
            && run
                .text
                .strip_suffix("...")
                .is_some_and(|prefix| !prefix.is_empty() && full.starts_with(prefix))
    })
}
/// Open the Home workspace using its visible control, for example before checking recent files.
pub(super) fn open_home(world: &mut ApplicationWorld) {
    world.click_text("Home");
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        window_geometry(HOME).is_some(),
        "expected the Home workspace"
    );
}
fn enter_search(world: &mut ApplicationWorld, text: &str) {
    let hint = home_run(world, "Search recent").expect("expected an empty search field");
    world.click(hint.center().map(|value| value as i32));
    world.type_text(text);
    world.pump(Vec::new());
}
fn search_for(world: &mut ApplicationWorld, text: &str) {
    enter_search(world, text);
    click_home(world, "Search");
}
/// The panel stays up between choices, so opening it again must not close it.
fn open_filter(world: &mut ApplicationWorld) {
    if !world.shows("File Types") {
        click_home(world, "Filter");
    }
}
fn select_assembly_filter(world: &mut ApplicationWorld) {
    open_filter(world);
    world.click_text("Assembly");
    world.pump(Vec::new());
}
fn expect_filter_count(world: &ApplicationWorld, count: usize) {
    world.expect_text(&format!("Filter ({count})"));
}
#[given("Home is active in the MCAD Software")]
#[given("the Home workspace is active in the MCAD Software")]
fn home_active(world: &mut ApplicationWorld) {
    if !published_profile::uses_published_profile(world) {
        save_recent_files(world);
    }
    open_home(world);
}
#[given(regex = r#"^the recent list contains a part named "Part26" and other documents$"#)]
#[given(regex = r#"^the recent list includes a part named "Part26" and other document types$"#)]
fn list_contains(world: &mut ApplicationWorld) {
    for name in std::iter::once(SEARCHED).chain(other_recent_names(world).iter().copied()) {
        assert!(
            recent_name(world, name).is_some(),
            "expected {name} on the list"
        );
    }
}
#[given("the recent list contains documents")]
fn recent_list_filled(world: &mut ApplicationWorld) {
    // Keep the seed files in Recent without adding their tabs to the Part under edit.
    let part = world.active_id();
    if published_profile::uses_published_profile(world) {
        published_profile::expect_recent_files(world);
        world.previous_document = Some(part);
        return;
    }
    for id in save_recent_files(world) {
        let request = oblikovati_api::wire::CloseDocumentArgs { id, force: false };
        assert!(world.app().owner.close_document_from_gui(&request).unwrap());
    }
    world.pump(Vec::new());
    world.activate_through_tab(part);
    world.previous_document = Some(part);
}
#[then("the Home workspace displays Open and New actions in a left navigation rail")]
fn rail_actions(world: &mut ApplicationWorld) {
    let recent = home_run(world, "Recent").expect("expected the Recent area");
    for action in ["Open", "New"] {
        let run = home_run(world, action)
            .unwrap_or_else(|| panic!("expected {action:?} in {:?}", world.window_text(HOME)));
        assert!(
            run.center()[0] < recent.center()[0],
            "{action}: expected the rail left of Recent"
        );
    }
}
#[then("a Recent area appears to the right of the rail")]
fn recent_area(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Recent"), "expected the Recent area");
}
#[then("the Recent area offers list view, grid view, search and filter controls")]
fn recent_controls(world: &mut ApplicationWorld) {
    for control in ["List View", "Grid View", "Search recent", "Filter"] {
        assert!(home_shows(world, control), "expected {control} in Home");
    }
}
#[given("the recent documents are displayed as a list")]
#[given("recent documents are displayed in List View")]
#[given("recent documents are shown in List View")]
#[then("the Home recent-documents view remains visible around the menu")]
fn home_behind_menu(_: &mut ApplicationWorld) {
    assert!(
        window_geometry(HOME).is_some(),
        "expected the Home workspace"
    );
}
#[given("recent documents are in List View")]
#[given("several recent documents are displayed in List View")]
#[given("recent documents with different names are displayed in List View")]
#[given("two recent parts are displayed in List View")]
#[given("recent parts and assemblies are displayed in List View")]
#[given("a recent part is displayed in List View")]
fn displayed_as_list(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Location"), "expected the list's columns");
}
#[when("the user enters \"Part26\" in Search recent")]
#[when(regex = r#"^I enter "Part26" in Search recent$"#)]
fn enter_part26(world: &mut ApplicationWorld) {
    enter_search(world, SEARCHED);
}
#[when("the user activates the Search control")]
#[when("I activate the Search control")]
#[when("the user activates the Search control again")]
#[when("I activate the Search control again")]
fn activate_search(world: &mut ApplicationWorld) {
    click_home(world, "Search");
}
#[then("the recent list shows the matching part")]
#[then("the recent list displays the matching part")]
#[then("the matching part is displayed")]
#[then("the matching part is displayed again")]
fn shows_match(world: &mut ApplicationWorld) {
    assert!(
        matched_part_is_visible(world),
        "expected {SEARCHED} on the list: {:?}",
        world.window_text(HOME)
    );
}
fn matched_part_is_visible(world: &ApplicationWorld) -> bool {
    let search = home_run(world, "Search").expect("expected the Search button");
    world
        .frame
        .runs
        .iter()
        .any(|run| in_home(run) && run.text == SEARCHED && run.minimum[1] > search.maximum[1])
}
#[then("the other previously visible documents are not displayed")]
fn others_hidden(world: &mut ApplicationWorld) {
    for name in other_recent_names(world) {
        assert!(recent_name(world, name).is_none(), "expected no {name}");
    }
}
#[when("the user clears the search field with its clear control")]
#[when("I clear the search field with its clear control")]
pub(super) fn clear_field(world: &mut ApplicationWorld) {
    let run = world
        .frame
        .runs
        .iter()
        .rev()
        .find(|run| in_home(run) && run.text == "×")
        .cloned()
        .expect("expected a clear control");
    world.click(run.center().map(|value| value as i32));
    world.pump(Vec::new());
}
#[then("the search field is empty")]
fn field_empty(world: &mut ApplicationWorld) {
    assert!(
        home_shows(world, "Search recent"),
        "expected the field's hint"
    );
}
#[then("the filtered result remains displayed")]
fn filtered_remains(world: &mut ApplicationWorld) {
    shows_match(world);
    others_hidden(world);
}
#[then("the other recent documents are displayed again")]
#[then("the broader recent-document list is displayed again")]
#[then("the broader recent list is displayed")]
fn others_back(world: &mut ApplicationWorld) {
    list_contains(world);
}
#[given("the user has searched recent documents for \"Part26\"")]
#[given(regex = r#"^I have searched recent documents for "Part26"$"#)]
fn searched(world: &mut ApplicationWorld) {
    search_for(world, SEARCHED);
}
#[then("a Search History panel appears beneath the field")]
fn history_panel(world: &mut ApplicationWorld) {
    let hint = home_run(world, "Search recent")
        .expect("expected the field")
        .center()[1];
    let panel = home_run(world, "Search History")
        .expect("expected the panel")
        .center()[1];
    assert!(
        panel > hint,
        "panel at {panel}: expected below the field at {hint}"
    );
}
#[then(regex = r#"^it includes "Part26"$"#)]
#[then("it includes \"Part26\" preceded by a history symbol")]
fn history_includes(world: &mut ApplicationWorld) {
    let panel = home_run(world, "Search History")
        .expect("expected the panel")
        .center()[1];
    let term = world
        .frame
        .runs
        .iter()
        .any(|run| in_home(run) && run.text == "(h)Part26" && run.center()[1] > panel);
    assert!(term, "expected {SEARCHED} under the panel title");
}
#[when("the user selects Grid View")]
#[when("I select Grid View")]
#[when("the user switches to Grid View")]
#[when("I switch to Grid View")]
fn select_grid(world: &mut ApplicationWorld) {
    click_home(world, "Grid View");
}
#[then("recent documents appear as a grid of thumbnail cards")]
fn grid_cards(world: &mut ApplicationWorld) {
    assert!(!home_shows(world, "Location"), "expected no list columns");
    let cards: Vec<_> = std::iter::once(SEARCHED)
        .chain(other_recent_names(world).iter().copied())
        .map(|name| {
            recent_name(world, name)
                .unwrap_or_else(|| panic!("expected the {name:?} card"))
                .center()
        })
        .collect();
    assert!(
        cards
            .iter()
            .enumerate()
            .any(|(index, first)| cards[index + 1..]
                .iter()
                .any(|second| (first[1] - second[1]).abs() < ROW_REACH && first[0] != second[0])),
        "expected at least two cards on one row, found {cards:?}"
    );
    world.save_capture("home-recent-grid");
}
#[then("each visible card shows a document name and date")]
fn card_facts(world: &mut ApplicationWorld) {
    let rows = world.state().recent_documents;
    for name in std::iter::once(SEARCHED).chain(other_recent_names(world).iter().copied()) {
        let card = recent_name(world, name).unwrap_or_else(|| panic!("expected {name:?} card"));
        let modified = &rows
            .iter()
            .find(|row| row.name == name)
            .unwrap()
            .modified_text;
        let day = modified
            .split_whitespace()
            .next()
            .expect("expected the file modification date");
        let date = world.frame.runs.iter().any(|row| {
            let below = row.center()[1] - card.center()[1];
            in_home(row)
                && row.text.starts_with(day)
                && (row.minimum[0] - card.minimum[0]).abs() < ROW_REACH
                && below > 0.
                && below < 32.
        });
        assert!(date, "expected a date under {name:?}");
    }
}
#[then("a Sort by control is displayed")]
fn sort_shown(world: &mut ApplicationWorld) {
    assert!(home_shows(world, "Sort by"), "expected Sort by");
}
#[when("the user selects List View")]
#[when("I select List View")]
#[when("the user switches back to List View")]
#[when("I switch back to List View")]
fn select_list(world: &mut ApplicationWorld) {
    click_home(world, "List View");
}
#[then("recent documents appear as rows with metadata columns")]
fn list_rows(world: &mut ApplicationWorld) {
    for column in ["Name", "Location", "Date Modified", "Last Opened", "Size"] {
        assert!(home_shows(world, column), "expected the {column} column");
    }
}
#[given("recent parts and assemblies are available")]
fn parts_and_assemblies(world: &mut ApplicationWorld) {
    let assembly = assembly_name(world);
    assert!(
        recent_name(world, assembly).is_some(),
        "expected {assembly}"
    );
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
#[given("no other rows are checked and no recent-document filters are active")]
#[given("no recent-document filters are active")]
fn no_filters(world: &mut ApplicationWorld) {
    world.expect_no_text("Filter (");
}
#[when("the user opens Filter and selects Assembly")]
#[when("I open Filter and select Assembly")]
fn filter_assembly(world: &mut ApplicationWorld) {
    select_assembly_filter(world);
}
#[then("the recent list displays assembly documents")]
fn assemblies_shown(world: &mut ApplicationWorld) {
    let assembly = assembly_name(world);
    assert!(
        recent_name(world, assembly).is_some(),
        "expected {assembly}"
    );
    assert!(!home_shows(world, SEARCHED), "expected no {SEARCHED}");
    for name in other_recent_names(world)
        .iter()
        .filter(|&&name| name != assembly)
    {
        assert!(recent_name(world, name).is_none(), "expected no {name}");
    }
}
#[then("the filter control indicates one active filter")]
fn one_filter(world: &mut ApplicationWorld) {
    expect_filter_count(world, 1);
}
#[when("the user chooses Clear filters")]
#[when("I choose Clear filters")]
fn clear_filters(world: &mut ApplicationWorld) {
    if !world.shows("Clear filters") {
        open_filter(world);
    }
    world.click_text("Clear filters");
    world.pump(Vec::new());
}
#[then("the Assembly filter is unchecked")]
fn assembly_unchecked(world: &mut ApplicationWorld) {
    open_filter(world);
    assert!(!world.checked("Assembly"), "expected Assembly unchecked");
}
#[then("recent parts and assemblies are displayed again")]
#[then("the broader recent list includes parts and assemblies")]
fn both_again(world: &mut ApplicationWorld) {
    let assembly = assembly_name(world);
    assert!(
        recent_name(world, assembly).is_some(),
        "expected {assembly}"
    );
    assert!(home_shows(world, SEARCHED), "expected {SEARCHED}");
}
