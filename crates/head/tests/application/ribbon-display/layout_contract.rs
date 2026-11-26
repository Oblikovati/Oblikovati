// SPDX-License-Identifier: GPL-2.0-only
//! Check full ribbon geometry and compact group artwork in the native window.
use super::super::diagnostics::{self, RenderedRun, child_scroll};
use super::super::ribbon_appearance::icon_quads::icon_quads_inside;
use super::*;

const BAND: &str = "##application-ribbon";
const TABS: &str = "##ribbon-tab-region";
pub(super) const COMMANDS: &str = "##ribbon-panels";
pub(super) const TAB_GROUPS: [(&str, &[&str]); 6] = [
    (
        "3D Model",
        &[
            "SKETCH",
            "CREATE",
            "MODIFY",
            "EXPLORE",
            "WORK FEATURES",
            "PATTERN",
            "SIMPLIFICATION",
            "SIMULATION",
            "3D PRINT",
        ],
    ),
    (
        "Sketch",
        &["CREATE", "CONSTRAIN", "MODIFY", "INSERT", "FORMAT", "EXIT"],
    ),
    (
        "Tools",
        &[
            "MEASURE",
            "MATERIAL AND APPEARANCE",
            "OPTIONS",
            "CLIPBOARD",
            "FIND",
            "TEAM WEB",
        ],
    ),
    (
        "Manage",
        &[
            "UPDATE",
            "PARAMETERS",
            "MANAGE",
            "STYLES AND STANDARDS",
            "INSERT",
            "POINT CLOUD",
            "LAYOUT",
            "AUTHOR",
            "ILOGIC",
            "CONTENT CENTER",
        ],
    ),
    ("View", &["VISIBILITY", "APPEARANCE", "WINDOWS", "NAVIGATE"]),
    ("Annotate", &["NOTES"]),
];

pub(in super::super) struct RibbonSample {
    tab: &'static str,
    vertical_geometry: [[f32; 2]; 3],
    scrollbar: bool,
    content_width: f32,
    available_width: f32,
    collapsed: Vec<&'static str>,
}

#[given("`fixtures/cad/docking-visible-solid-current.opd` is open in a wide restored window")]
pub(super) fn wide_restored(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join("fixtures/cad/docking-visible-solid-current.opd");
    world.open_fixture(&path);
    resize(world, 1920);
    assert!(!world.app().window.native.view.window_maximized().unwrap());
}

pub(super) fn resize(world: &mut ApplicationWorld, width: u32) {
    let height = world.extent()[1];
    world
        .app()
        .window
        .native
        .view
        .resize([width, height])
        .unwrap();
    for _ in 0..4 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent()[0], width);
}

#[when("the user switches across the 3D Model, Sketch, Tools, Manage, View, and Annotate tabs")]
fn inspect_tabs(world: &mut ApplicationWorld) {
    for (tab, groups) in TAB_GROUPS {
        select_tab(world, tab);
        let sample = observe_ribbon(world, tab, groups);
        world.ribbon_layout_samples.push(sample);
    }
}

pub(super) fn select_tab(world: &mut ApplicationWorld, tab: &str) {
    let tabs = child_scroll(BAND, TABS).expect("expected the tab strip");
    let run = labeled_run(world, tabs.list, tab);
    world.click(diagnostics::pointer_pixels(run.center()));
    world.pump(Vec::new());
}

pub(super) fn labeled_run<'a>(
    world: &'a ApplicationWorld,
    list: usize,
    name: &str,
) -> &'a RenderedRun {
    let label = super::super::world::squeezed(name);
    world
        .frame
        .runs
        .iter()
        .find(|run| run.list == list && super::super::world::squeezed(&run.text) == label)
        .unwrap_or_else(|| panic!("expected {name:?} in draw list {list}: {:?}", world.text()))
}

fn observe_ribbon(
    world: &ApplicationWorld,
    tab: &'static str,
    groups: &[&'static str],
) -> RibbonSample {
    let (origin, size) = window_geometry(BAND).expect("expected the ribbon band");
    let tabs = child_scroll(BAND, TABS).expect("expected the tab strip");
    let row = child_scroll(BAND, COMMANDS).expect("expected the command area");
    let collapsed = groups
        .iter()
        .copied()
        .filter(|caption| {
            !world.frame.runs.iter().any(|run| {
                run.list == row.list
                    && run.center()[1] > row.origin[1] + row.size[1] * 0.7
                    && diagnostics::panel_caption_matches(&run.text, caption)
            })
        })
        .collect();
    RibbonSample {
        tab,
        vertical_geometry: [
            [origin[1], size[1]],
            [tabs.origin[1], tabs.size[1]],
            [row.origin[1], row.size[1]],
        ],
        scrollbar: tabs.horizontal_scrollbar || row.horizontal_scrollbar,
        content_width: row.content_width,
        available_width: row.size[0],
        collapsed,
    }
}

#[then(
    "the ribbon tab strip and command area retain the same height on every tab without vertical shift or a horizontal scrollbar"
)]
fn consistent_height(world: &mut ApplicationWorld) {
    let samples = &world.ribbon_layout_samples;
    assert_eq!(samples.len(), TAB_GROUPS.len());
    for sample in samples {
        assert_eq!(
            sample.vertical_geometry, samples[0].vertical_geometry,
            "{} ribbon geometry",
            sample.tab
        );
        assert!(
            !sample.scrollbar,
            "{} ribbon horizontal scrollbar",
            sample.tab
        );
    }
}

#[then(
    "ribbon groups remain expanded whenever their combined width fits within the available ribbon width"
)]
fn expanded_when_fit(world: &mut ApplicationWorld) {
    for sample in &world.ribbon_layout_samples {
        assert!(
            sample.content_width <= sample.available_width,
            "{} content width {} exceeds command area {}",
            sample.tab,
            sample.content_width,
            sample.available_width
        );
        assert!(
            sample.collapsed.is_empty(),
            "{} missing expanded captions: {:?}",
            sample.tab,
            sample.collapsed
        );
    }
}

#[when("the user narrows the window until groups must compact")]
fn narrow(world: &mut ApplicationWorld) {
    select_tab(world, "3D Model");
    resize(world, 700);
}

#[then(
    "compacted groups render as icon-and-label buttons with a dropdown indicator rather than blank placeholder boxes"
)]
fn compact_button_artwork(world: &mut ApplicationWorld) {
    let row = child_scroll(BAND, COMMANDS).expect("expected the resized command area");
    let label = labeled_run(world, row.list, "Modify");
    let [x, y] = label.center();
    assert_eq!(
        row.size[1],
        world.ribbon_layout_samples[0].vertical_geometry[2][1]
    );
    assert!(!row.horizontal_scrollbar);
    assert!(
        !icon_quads_inside(([x - 36., row.origin[1]], [x + 36., y])).is_empty(),
        "expected the Modify icon above its caption"
    );
    assert!(
        world.frame.marks.iter().any(|mark| {
            let [mark_x, mark_y] = mark.position;
            mark.list == row.list
                && mark.bright
                && (label.maximum[0]..label.maximum[0] + 14.).contains(&mark_x)
                && (y - 5. ..y + 5.).contains(&mark_y)
        }),
        "expected the Modify dropdown marker next to its caption"
    );
    world.expect_no_text("Fillet");
}
