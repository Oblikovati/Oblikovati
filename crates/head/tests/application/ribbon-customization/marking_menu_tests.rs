// SPDX-License-Identifier: GPL-2.0-only
//! Native checks for sketch marking-menu commands.
use super::*;
use oblikovati_api::{
    client::Client,
    types,
    wire::{MarkingMenuItem, MarkingMenuView, SetMarkingMenuArgs},
};

#[test]
fn marking_menu_horizontal_constrains_a_drawn_line() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.click_text("Line");
    world.sketch_click([620, 300]);
    world.sketch_click([840, 420]);
    world.escape();
    let before = world.edited_geometry().constraints().len();
    let menu = MarkingMenuView {
        environment: types::SKETCH_ENVIRONMENT,
        quadrants: Some(vec![MarkingMenuItem {
            quadrant: types::QUADRANT_NORTH,
            command_id: "Sketch.Horizontal".into(),
        }]),
        overflow: None,
    };
    Client::new(world.app().owner.document_host())
        .ui()
        .set_marking_menu(&SetMarkingMenuArgs { menu })
        .unwrap();
    world.right_click([900, 450]);
    world.pump(Vec::new());
    let entry = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Horizontal" && run.center()[0] > 700. && run.center()[1] > 250.)
        .expect("expected Horizontal in sketch marking menu")
        .center();
    world.click(diagnostics::pointer_pixels(entry));
    world.sketch_click([730, 360]);
    assert_eq!(world.edited_geometry().constraints().len(), before + 1);
    world.close();
}
