// SPDX-License-Identifier: GPL-2.0-only
//! Find and select a row through the file picker's visible list and wheel input.
use super::super::diagnostics::{ChildScroll, child_scroll, pointer_pixels, tooltip_draw_list};
use super::super::world::squeezed;
use super::{ApplicationWorld, settle};
use oblikovati_viewport::{ViewportEvent, ViewportInput};

pub(in super::super) fn select_row(world: &mut ApplicationWorld, file: &str) {
    let name = squeezed(file);
    let (picker, initial) = ["Open", "Choose Component File"]
        .into_iter()
        .find_map(|picker| child_scroll(picker, "##folder-listing").map(|list| (picker, list)))
        .expect("expected a visible file picker listing");
    let position = scan_rows(world, picker, &name, -1, initial).or_else(|| {
        // A previous selection may have left earlier rows above the starting point.
        if initial.offset > 0. {
            return scan_rows(world, picker, &name, 1, listing(picker));
        }
        None
    });
    let position =
        position.unwrap_or_else(|| panic!("file listing: expected a selectable row for {file:?}"));
    world.click(pointer_pixels(position));
    settle(world);
}

fn listing(picker: &str) -> ChildScroll {
    child_scroll(picker, "##folder-listing").expect("expected a visible file picker listing")
}

fn scan_rows(
    world: &mut ApplicationWorld,
    picker: &str,
    name: &str,
    direction: i32,
    mut list: ChildScroll,
) -> Option<[f32; 2]> {
    loop {
        if let Some(position) = visible_file_row(world, name, list.list) {
            return Some(position);
        }
        if (direction < 0 && list.offset >= list.limit) || (direction > 0 && list.offset <= 0.) {
            return None;
        }
        let position = pointer_pixels(list.center);
        world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
            notches: direction,
            position,
        })]);
        let next = listing(picker);
        assert_ne!(
            next.offset, list.offset,
            "file listing: wheel did not scroll while looking for {name:?}"
        );
        list = next;
    }
}

fn visible_file_row(world: &mut ApplicationWorld, name: &str, list: usize) -> Option<[f32; 2]> {
    let mut shortened = Vec::new();
    for run in world.frame.runs.iter().filter(|run| run.list == list) {
        let rendered = squeezed(&run.text);
        let position = [run.minimum[0] + 1., run.center()[1]];
        if rendered == name {
            return Some(position);
        }
        if rendered
            .strip_suffix("...")
            .is_some_and(|prefix| !prefix.is_empty() && name.starts_with(prefix))
        {
            shortened.push(position);
        }
    }
    shortened.into_iter().find(|&position| {
        world.hover(pointer_pixels(position));
        tooltip_shows(world, name)
    })
}

fn tooltip_shows(world: &ApplicationWorld, name: &str) -> bool {
    let list = tooltip_draw_list();
    world
        .frame
        .runs
        .iter()
        .any(|run| list == Some(run.list) && squeezed(&run.text) == name)
}

#[test]
fn details_keeps_long_names_clear_of_the_date_and_selects_the_exact_file() {
    use super::{
        choose_open, file_name_shown, look_in, models_folder, open_dialog, saved_document,
    };
    use oblikovati_gui::GuiTemplateKind;
    const FIRST: &str = "production-assembly-revision-with-a-shared-identifier-front-panel.opd";
    const SECOND: &str = "production-assembly-revision-with-a-shared-identifier-rear-panel.opd";
    let mut world = ApplicationWorld::new();
    let folder = models_folder(&mut world);
    let first = saved_document(&mut world, GuiTemplateKind::Part, &folder, FIRST);
    saved_document(&mut world, GuiTemplateKind::Part, &folder, SECOND);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    let date_x = world.run("Date modified").minimum[0];
    let shortened: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text.starts_with("production-assembly") && run.text.ends_with("..."))
        .collect();
    assert_eq!(
        shortened.len(),
        2,
        "expected both filenames to be shortened"
    );
    assert!(shortened.iter().all(|run| run.maximum[0] < date_x));
    select_row(&mut world, SECOND);
    assert!(tooltip_shows(&world, SECOND));
    assert_eq!(file_name_shown(&world), SECOND);
    select_row(&mut world, FIRST);
    assert_eq!(file_name_shown(&world), FIRST);
    choose_open(&mut world);
    world.expect_active(first);
    world.close();
}
