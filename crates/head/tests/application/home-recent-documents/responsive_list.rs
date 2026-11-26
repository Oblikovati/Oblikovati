// SPDX-License-Identifier: GPL-2.0-only
//! Read the rendered Recent rows through a native window resize and scroll.
use super::super::{desktop_input, diagnostics};
use super::*;
use oblikovati_model::application_recent::{RecentDocument, RecentDocumentAccess};
use std::fs::{self, File, FileTimes};
use std::path::Path;
use std::time::{Duration, UNIX_EPOCH};

const ROWS: u64 = 24;
const MODIFIED: u64 = 1_738_572_300;
const OPENED: u64 = 1_767_345_480;

#[given("the Home workspace is active with recent documents in List View")]
fn recent_list_with_dates(world: &mut ApplicationWorld) {
    let source = super::super::scenario_root().join("fixtures/cad/home-recent-native/Part2.opd");
    let folder = world.scratch_folder();
    let documents = (0..ROWS)
        .map(|index| copy_recent(&source, &folder, index))
        .collect();
    world.close();
    let mut options =
        crate::native::preferences::load_native_application_options(world.preferences_directory())
            .unwrap();
    options.set_recent_documents(documents).unwrap();
    world.reopen_application();
    assert!(home_shows(world, "List View"));
    assert_date_format(world, "2025-02-03", "2026-01-02", true);
}

fn copy_recent(source: &Path, folder: &Path, index: u64) -> RecentDocument {
    let path = folder.join(format!("RecentPart{index:02}.opd"));
    fs::copy(source, &path).unwrap();
    let modified = UNIX_EPOCH + Duration::from_secs(MODIFIED + index);
    File::options()
        .write(true)
        .open(&path)
        .unwrap()
        .set_times(FileTimes::new().set_modified(modified))
        .unwrap();
    RecentDocument {
        path: path.to_str().unwrap().to_owned(),
        last_opened: OPENED + index,
        pinned: false,
    }
}

#[when("the user narrows or shortens the main application window")]
fn resize_home(world: &mut ApplicationWorld) {
    let [width, height] = world.extent().map(|value| value as i32);
    desktop_input::drag(world, [width - 2, height - 2], [880, 540]);
    world.pump(Vec::new());
    let [narrow, short] = world.extent();
    assert!(
        narrow.abs_diff(880) <= 4 && short.abs_diff(540) <= 4,
        "resized to {narrow}x{short}"
    );
}

#[then(
    "every visible row formats Date Modified and Last Opened using the same date format for that width"
)]
fn recent_dates_are_uniform(world: &mut ApplicationWorld) {
    assert_date_format(world, "2025-02-03", "2026-01-02", false);
}

fn visible_names(world: &ApplicationWorld) -> Vec<&RenderedRun> {
    let list = diagnostics::window_draw_list(HOME).expect("expected Home draw list");
    world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.list == list
                && run.text.starts_with("RecentPart")
                && run
                    .text
                    .strip_prefix("RecentPart")
                    .unwrap()
                    .starts_with(|c: char| c.is_ascii_digit())
        })
        .collect()
}

fn assert_date_format(world: &ApplicationWorld, modified: &str, opened: &str, long: bool) {
    let names = visible_names(world);
    assert!(names.len() >= 3, "expected multiple visible Recent rows");
    for name in names {
        let dates: Vec<_> = world
            .frame
            .runs
            .iter()
            .filter(|run| {
                (run.center()[1] - name.center()[1]).abs() < 6.
                    && run.list == name.list
                    && run.minimum[0] > name.maximum[0]
                    && (run.text.starts_with(modified) || run.text.starts_with(opened))
            })
            .collect();
        assert_eq!(dates.len(), 2, "expected both dates for {}", name.text);
        for (cell, prefix) in dates.into_iter().zip([modified, opened]) {
            if long {
                assert!(
                    cell.text.starts_with(prefix) && cell.text.ends_with("UTC"),
                    "{}",
                    cell.text
                );
            } else {
                assert_eq!(cell.text, prefix, "date-only format for {}", name.text);
            }
        }
    }
}

#[then("filenames remain readable without collapsing columns into empty slivers")]
fn recent_names_keep_room(world: &mut ApplicationWorld) {
    let location = world.run_in_window(HOME, "Location");
    for name in visible_names(world) {
        assert_eq!(name.text.len(), "RecentPart00".len(), "{}", name.text);
        assert!(name.maximum[0] < location.minimum[0], "{}", name.text);
    }
    let modified = world.run_in_window(HOME, "DateModified");
    assert!(location.maximum[0] < modified.minimum[0]);
}

#[then(
    "the scrolled list keeps bottom clearance so the last visible row is not clipped behind the document tab strip"
)]
fn last_recent_row_clears_tabs(world: &mut ApplicationWorld) {
    assert!(!home_shows(world, "RecentPart00"));
    let (position, size) = window_geometry(HOME).unwrap();
    let pointer =
        diagnostics::pointer_pixels([position[0] + size[0] * 0.65, position[1] + size[1] * 0.65]);
    for _ in 0..6 {
        desktop_input::scroll(world, pointer, -10);
    }
    let last = home_run(world, "RecentPart00").expect("expected the final Recent row");
    let bottom = world.viewport_bounds()[3] as i32;
    let top_of_gap = diagnostics::pointer_pixels(last.maximum)[1];
    assert!(
        bottom - top_of_gap >= 12,
        "last filename ends at {top_of_gap}; tabs start at {bottom}"
    );
    assert_date_format(world, "2025-02-03", "2026-01-02", false);
}
