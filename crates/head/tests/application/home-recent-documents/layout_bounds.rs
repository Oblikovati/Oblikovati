// SPDX-License-Identifier: GPL-2.0-only
//! Long recent-document cells stay clear of adjacent metadata.

use super::super::diagnostics;
use super::*;

#[test]
fn long_recent_labels_stay_in_their_list_columns() {
    let mut world = ApplicationWorld::new();
    let saved = save_long_recent_labels(&mut world);
    open_home(&mut world);
    let location = saved.parent().unwrap().to_str().unwrap();
    let shown = expect_recent_cell_fits(&world, location, "Date Modified");
    let name = saved.file_stem().unwrap().to_str().unwrap();
    expect_recent_cell_fits(&world, name, "Location");
    world.save_capture("home-recent-long-cells");
    world.hover(diagnostics::pointer_pixels(shown.center()));
    world.pump(Vec::new());
    assert!(
        home_run(&world, location).is_some(),
        "expected the full location in a tooltip over Home"
    );
    world.save_capture("home-recent-location-tooltip");
    world.close();
}

fn save_long_recent_labels(world: &mut ApplicationWorld) -> std::path::PathBuf {
    let folder = format!("recent-fixture-{}", "directory".repeat(13));
    std::fs::create_dir(world.scratch_folder().join(&folder)).unwrap();
    let name = format!("{folder}/RecentPart{}.opd", "Name".repeat(18));
    let part = world.add_part();
    world.save_part_to_file(part, &name)
}

fn expect_recent_cell_fits(world: &ApplicationWorld, full: &str, next_column: &str) -> RenderedRun {
    let prefix = &full[..full.floor_char_boundary(12)];
    let cell = home_run(world, prefix).expect("expected the recent row text");
    let next = home_run(world, next_column).expect("expected the adjacent column header");
    assert!(
        cell.maximum[0] < next.minimum[0],
        "{} ends at {}, beyond the {next_column} column at {}",
        cell.text,
        cell.maximum[0],
        next.minimum[0]
    );
    cell
}
