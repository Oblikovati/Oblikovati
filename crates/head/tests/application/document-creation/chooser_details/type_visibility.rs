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
