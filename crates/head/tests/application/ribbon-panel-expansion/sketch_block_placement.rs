// SPDX-License-Identifier: GPL-2.0-only
//! Drive block placement through the native browser and sketch viewport.
use super::{ApplicationWorld, diagnostics, sketch_constrain::segment_color_changed};
use crate::{HeadWindow, HeadWork};
use oblikovati_viewport::{ViewportKey, ViewportPhysicalKey};

#[test]
fn place_block_context_menu_repeats_and_escape_cancels() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = world.edited_geometry().lines()[0].id;
    world
        .app()
        .owner
        .document_commands(edit.document)
        .run("Capture block", |snapshot, _| {
            snapshot
                .sketches
                .capture_block_definition("Rib", edit.sketch, &[line])
        })
        .unwrap();
    world.pump(Vec::new());
    start_block_from_browser(&mut world);
    assert!(!world.state().sketch_at_ready);
    verify_preview(&mut world, edit);
    let positions = [[1100, 550], [1150, 600]].map(|pixel| {
        world
            .app()
            .window
            .sketch_point(pixel, edit.plane)
            .unwrap()
            .unwrap()
    });
    world.click([1100, 550]);
    world.expect_text("Rib:1");
    world.click([1150, 600]);
    world.expect_text("Rib:2");
    let instances = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot()
        .sketches
        .get(edit.sketch)
        .unwrap()
        .block_instances()
        .to_vec();
    for (instance, expected) in instances.iter().zip(positions) {
        for (actual, expected) in instance.position().into_iter().zip(expected) {
            assert!((actual - expected).abs() < 1e-9);
        }
    }
    world.escape();
    assert!(world.state().sketch_at_ready);
    assert!(world.app().owner.sketch_preview().is_none());
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    world.expect_no_text("Rib:2");
    world.command_key(ViewportPhysicalKey::Y, ViewportKey::Redo);
    world.expect_text("Rib:2");
    start_block_from_browser(&mut world);
    world.escape();
    assert_eq!(world.state().browser.unwrap().block_instances.len(), 2);
    world.close();
}
fn verify_preview(world: &mut ApplicationWorld, edit: crate::HeadSketchEdit) {
    let before = super::super::capture_framebuffer(&mut world.app().window.native);
    world.hover([1100, 550]);
    let preview = world.app().owner.sketch_preview().unwrap();
    assert_eq!(preview.member_count(), 1);
    let oblikovati_viewport::ViewportSketchPreview::Resolved { geometry, .. } = preview else {
        panic!("expected placed block preview geometry");
    };
    let segment = geometry.lines()[0].endpoints.map(|id| {
        let point = geometry.point(id).unwrap().position;
        world.app().window.sketch_pixel(edit.plane, point).unwrap()
    });
    let after = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(segment_color_changed(&before, &after, segment));
}
fn start_block_from_browser(world: &mut ApplicationWorld) {
    let name = world.run("Rib").center();
    world.right_click(diagnostics::pointer_pixels(name));
    world.click_text("Place Block");
    world.pump(Vec::new());
}
