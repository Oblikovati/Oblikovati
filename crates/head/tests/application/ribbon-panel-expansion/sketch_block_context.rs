// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the placed block context menu on the native sketch canvas.
use super::ApplicationWorld;
use oblikovati_viewport::{ViewportKey, ViewportPhysicalKey};

#[test]
fn viewport_context_explodes_only_the_hit_block_and_supports_undo() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    let geometry = world.edited_geometry();
    let lines: Vec<_> = geometry.lines().iter().map(|line| line.id.0).collect();
    let hit = world.line_point(0, 0.4);
    let request = oblikovati_api::wire::CreateBlockDefinitionArgs {
        name: "Rib".into(),
        source_sketch_index: 0,
        entity_refs: Some(lines),
    };
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .sketch()
        .create_block_definition(&request)
        .unwrap();
    world.pump(Vec::new());
    world.expect_text("Rib:1");
    assert!(world.state().sketch_selection.entities.is_empty());
    world.right_click(hit);
    world.click_text("Explode");
    world.pump(Vec::new());
    world.expect_no_text("Rib:1");
    assert_eq!(world.edited_geometry().lines().len(), 4);
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    world.expect_text("Rib:1");
    assert!(world.edited_geometry().lines().is_empty());
    world.close();
}
