// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn details_type_labels_are_visible_without_scrolling_at_default_size() {
    let mut world = ApplicationWorld::new();
    open_chooser(&mut world);
    open_details_view(&mut world);
    for label in ["Assembly", "Presentation"] {
        expect_full_type_value(&world, label);
    }
    world.close();
}

pub(super) fn scroll_gallery_right(world: &mut ApplicationWorld) {
    let diag =
        super::super::super::super::diagnostics::child_scroll("New document", "##template-gallery")
            .expect("expected the template gallery scroll region");
    assert!(diag.horizontal_scrollbar);
    let bar_y = diag.origin[1] + diag.size[1] - 5.;
    let start =
        super::super::super::super::diagnostics::pointer_pixels([diag.origin[0] + 40., bar_y]);
    let end = super::super::super::super::diagnostics::pointer_pixels([
        diag.origin[0] + diag.size[0] + 60.,
        bar_y,
    ]);
    world.drag(start, end);
}

pub(super) fn expect_full_type_value(world: &ApplicationWorld, label: &str) {
    let column = world.run("Type").minimum[0];
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text == label && run.minimum[0] >= column - 2.),
        "expected full {label:?} value in the Type column at {column}; got {}",
        world.text()
    );
}
