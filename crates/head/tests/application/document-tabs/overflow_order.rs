// SPDX-License-Identifier: GPL-2.0-only
//! Order must not change when a tab returns to the visible strip.
use super::super::ApplicationWorld;
use crate::HeadWork;

fn visible_parts(world: &mut ApplicationWorld) -> Vec<u64> {
    let mut positions: Vec<_> = world
        .parts
        .clone()
        .into_iter()
        .filter_map(|id| {
            let name = format!("{}<", world.document_name(id));
            world.tab_run(&name).map(|run| (id, run.center()[0]))
        })
        .collect();
    positions.sort_by(|left, right| left.1.total_cmp(&right.1));
    positions.into_iter().map(|(id, _)| id).collect()
}

#[test]
fn overflow_scroll_preserves_document_order_in_both_directions() {
    let mut world = ApplicationWorld::new();
    for _ in 0..14 {
        world.add_part();
    }
    world.pump(Vec::new());
    let before = visible_parts(&mut world);
    world.click_text("<");
    world.pump(Vec::new());
    let after = visible_parts(&mut world);
    assert!(
        after.iter().any(|id| !before.contains(id)),
        "expected a newly visible part"
    );
    let expected: Vec<_> = world
        .parts
        .iter()
        .copied()
        .filter(|id| after.contains(id))
        .collect();
    assert_eq!(after, expected, "left scroll must preserve document order");
    world.save_capture("tab-order-after-left");
    world.click_text(">");
    world.pump(Vec::new());
    assert_eq!(
        visible_parts(&mut world),
        before,
        "right scroll must restore visible order"
    );
    world.save_capture("tab-order-after-right");
    world.close();
}

fn drag_before(world: &mut ApplicationWorld, source: &str, target: &str) {
    let from = world.expect_tab(source).center().map(|value| value as i32);
    let target = world.expect_tab(target).center().map(|value| value as i32);
    world.drag(from, [target[0] - 30, target[1]]);
    world.pump(Vec::new());
}
fn scroll_to_start(world: &mut ApplicationWorld) {
    for _ in 0..16 {
        world.click_text("<");
        world.pump(Vec::new());
    }
}
#[test]
fn deliberate_document_order_survives_scrolling_and_close_open() {
    let mut world = ApplicationWorld::new();
    for _ in 0..4 {
        world.add_part();
    }
    let ids = world.parts.clone();
    let source = world.document_name(ids[3]);
    let target = world.document_name(ids[1]);
    drag_before(&mut world, &source, &target);
    let expected = vec![ids[0], ids[3], ids[1], ids[2]];
    assert_eq!(
        visible_parts(&mut world),
        expected,
        "expected the user's drag order"
    );
    for _ in 0..10 {
        world.add_part();
    }
    scroll_to_start(&mut world);
    assert_eq!(
        visible_parts(&mut world)[..4],
        expected,
        "scroll must retain the drag order"
    );
    world.save_capture("tab-order-after-drag-and-scroll");
    world
        .app()
        .owner
        .close_document_from_gui(&oblikovati_api::wire::CloseDocumentArgs {
            id: ids[1],
            force: true,
        })
        .unwrap();
    world.parts.retain(|id| *id != ids[1]);
    world.pump(Vec::new());
    world.add_part();
    scroll_to_start(&mut world);
    let actual = visible_parts(&mut world);
    assert_eq!(
        &actual[..3],
        &[ids[0], ids[3], ids[2]],
        "close/open must retain surviving order"
    );
    world.close();
}
#[test]
fn home_drag_order_survives_scrolling() {
    let mut world = ApplicationWorld::new();
    for _ in 0..4 {
        world.add_part();
    }
    let target = world.document_name(world.parts[2]);
    drag_before(&mut world, "Home", &target);
    let second = world.document_name(world.parts[1]);
    assert!(world.expect_tab("Home").center()[0] > world.expect_tab(&second).center()[0]);
    for _ in 0..10 {
        world.add_part();
    }
    scroll_to_start(&mut world);
    assert!(
        world.expect_tab("Home").center()[0] > world.expect_tab(&second).center()[0],
        "Home must retain its dragged position"
    );
    world.save_capture("home-order-after-drag-and-scroll");
    world.close();
}

#[test]
fn surviving_extra_window_keeps_its_dragged_position() {
    let mut world = ApplicationWorld::new();
    for _ in 0..3 {
        world.add_part();
    }
    let ids = world.parts.clone();
    world.click_tab(ids[0]);
    world.app().owner.new_document_window_from_gui().unwrap();
    world.pump(Vec::new());
    let extra = world.state().active_window.unwrap();
    let first_name = world.document_name(ids[0]);
    let source = format!("{first_name}:2");
    let second = world.document_name(ids[1]);
    let third = world.document_name(ids[2]);
    drag_before(&mut world, &source, &third);
    assert!(world.expect_tab(&source).center()[0] > world.expect_tab(&second).center()[0]);
    world
        .app()
        .owner
        .document_window_from_gui(ids[0], true)
        .unwrap();
    world.pump(Vec::new());
    world.hover([900, 500]);
    world.pump(Vec::new());
    let state = world.state();
    assert_eq!(state.active_window, Some(extra));
    assert_eq!(
        state
            .document_windows
            .iter()
            .filter(|window| window.document == ids[0])
            .count(),
        1
    );
    assert!(
        world.expect_tab(&first_name).center()[0] > world.expect_tab(&second).center()[0],
        "surviving window must keep its reordered position"
    );
    world.save_capture("surviving-window-keeps-drag-order");
    world.close();
}
