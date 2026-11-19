// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the block definition scenario through the browser menu.
use super::super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_model::{
    document_sketch::SketchPlacement, history::DocumentSketchCreation, sketch::SketchPlane,
};

#[given("a block definition with an instance")]
fn block_definition_with_instance(world: &mut ApplicationWorld) {
    let document = world.add_part();
    let request = DocumentSketchCreation {
        name: "Sketch3".into(),
        placement: SketchPlacement::Fixed(SketchPlane::xy()),
        orientation: None,
        project_origin: false,
    };
    world
        .app()
        .owner
        .create_sketch_for_edit(document, &request)
        .unwrap();
    world.pump(Vec::new());
    world.draw_rectangle();
    world.escape();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let lines: Vec<_> = world
        .edited_geometry()
        .lines()
        .iter()
        .map(|line| line.id.0)
        .collect();
    let circle = add_inner_circle(world, edit);
    let mut entities = lines;
    entities.push(circle);
    let args = oblikovati_api::wire::CreateBlockDefinitionArgs {
        name: "Block1".into(),
        source_sketch_index: 0,
        entity_refs: Some(entities),
    };
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .sketch()
        .create_block_definition(&args)
        .unwrap();
    world.pump(Vec::new());
    world.expect_text("Sketch3");
    world.expect_text("Block1:1");
}

fn add_inner_circle(world: &mut ApplicationWorld, edit: crate::HeadSketchEdit) -> u64 {
    world
        .app()
        .owner
        .document_commands(edit.document)
        .run("Prepare block fixture", |snapshot, ids| {
            let mut geometry = snapshot
                .sketches
                .get(edit.sketch)
                .unwrap()
                .geometry()
                .clone();
            let points: Vec<_> = geometry
                .lines()
                .iter()
                .flat_map(|line| line.endpoints)
                .map(|id| geometry.point(id).unwrap().position)
                .collect();
            let bounds = [0, 1].map(|axis| {
                points
                    .iter()
                    .map(|point| point[axis])
                    .fold([f64::INFINITY, f64::NEG_INFINITY], |[low, high], value| {
                        [low.min(value), high.max(value)]
                    })
            });
            let position = bounds.map(|[low, high]| (low + high) / 2.0);
            let radius = (bounds[0][1] - bounds[0][0]).min(bounds[1][1] - bounds[1][0]) / 4.0;
            let center = ids.allocate_sketch_entity().unwrap();
            let circle = ids.allocate_sketch_entity().unwrap();
            geometry.add_point(center, position).unwrap();
            geometry.add_circle(circle, center, radius).unwrap();
            snapshot
                .sketches
                .replace_geometry(edit.sketch, geometry)
                .unwrap();
            Ok::<_, String>(circle.0)
        })
        .unwrap()
}

#[when("the user explode the instance")]
fn explode_instance(world: &mut ApplicationWorld) {
    let row = world.run("Block1:1").center();
    world.right_click(diagnostics::pointer_pixels(row));
    world.click_text("Explode");
    world.pump(Vec::new());
}

#[then("Block1:1 is absent from Sketch3")]
fn instance_absent(world: &mut ApplicationWorld) {
    world.expect_text("Sketch3");
    world.expect_no_text("Block1:1");
    assert_eq!(world.edited_geometry().lines().len(), 4);
    assert_eq!(world.edited_geometry().circles().len(), 1);
}

#[then("definition Block1 remains in the document Blocks folder")]
fn definition_remains(world: &mut ApplicationWorld) {
    world.expect_text("Blocks");
    world.expect_text("Block1");
    assert_eq!(world.state().sketch_block_names, ["Block1"]);
}
