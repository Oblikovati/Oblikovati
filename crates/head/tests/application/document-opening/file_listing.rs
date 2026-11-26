// SPDX-License-Identifier: GPL-2.0-only
//! Find and select a row through the file picker's visible list and wheel input.
use super::super::diagnostics::{ChildScroll, child_scroll, pointer_pixels};
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

fn visible_file_row(world: &ApplicationWorld, name: &str, list: usize) -> Option<[f32; 2]> {
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == list)
        .find_map(|run| {
            let rendered = squeezed(&run.text);
            let suffix = rendered.strip_prefix(name)?;
            // The draw reader may join the filename and the modification date into one run.
            (suffix.is_empty() || starts_with_date(suffix))
                .then(|| [run.minimum[0] + 1., run.center()[1]])
        })
}

fn starts_with_date(text: &str) -> bool {
    text.as_bytes()
        .get(..5)
        .is_some_and(|prefix| prefix[4] == b'-' && prefix[..4].iter().all(u8::is_ascii_digit))
}
