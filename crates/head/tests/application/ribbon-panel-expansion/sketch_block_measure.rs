// SPDX-License-Identifier: GPL-2.0-only
//! Measure a synthetic transformed block member through native mouse input.
use super::{ApplicationWorld, diagnostics};
use oblikovati_model::sketch::{SketchPointInput, SketchPrimitiveCreation};
use oblikovati_viewport::{ViewportKey, ViewportPhysicalKey, ViewportSketchFeedback};

#[test]
fn measure_transformed_block_line_without_exploding() {
    let mut world = ApplicationWorld::new();
    prepare_block(&mut world);
    measure_line(&mut world, true);
    world.escape();
    let row = world.run("Rib:1").center();
    world.right_click(diagnostics::pointer_pixels(row));
    world.click_text("Explode");
    world.pump(Vec::new());
    world.expect_no_text("Rib:1");
    measure_line(&mut world, false);
    world.escape();
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    world.expect_text("Rib:1");
    measure_line(&mut world, true);
    world.escape();
    world.command_key(ViewportPhysicalKey::Y, ViewportKey::Redo);
    world.expect_no_text("Rib:1");
    measure_line(&mut world, false);
    world.close();
}

fn prepare_block(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::New([0.0, 0.0]),
            SketchPointInput::New([4.0, 0.0]),
        ],
        construction: false,
        centerline: false,
    };
    world
        .app()
        .owner
        .document_commands(edit.document)
        .create_sketch_primitive(edit.sketch, &line)
        .unwrap();
    let line = world.edited_geometry().lines()[0].id;
    world
        .app()
        .owner
        .document_commands(edit.document)
        .run("Prepare measured block", |snapshot, ids| {
            snapshot
                .sketches
                .capture_block_definition("Rib", edit.sketch, &[line])
                .map_err(|error| error.to_string())?;
            let id = ids
                .allocate_sketch_entity()
                .map_err(|error| error.to_string())?;
            let instance = oblikovati_model::document_sketch::SketchBlockInstance::new(
                id,
                "Rib",
                [4.0, -4.0],
                std::f64::consts::FRAC_PI_2,
                2.0,
            )
            .map_err(|error| error.to_string())?;
            snapshot
                .sketches
                .insert_block_instance(edit.sketch, instance)
                .map_err(|error| error.to_string())
        })
        .unwrap();
    world.pump(Vec::new());
}

fn measure_line(world: &mut ApplicationWorld, grouped: bool) {
    world.key(ViewportPhysicalKey::M, false);
    let target = world.sketch_pixel_of([4.0, -2.0]);
    world.hover(target);
    if grouped {
        assert!(matches!(
            world.app().window.native.tool_hover,
            Some(ViewportSketchFeedback::BlockMember { .. })
        ));
    }
    world.sketch_click(target);
    world.expect_measure("Selection 1: Curve");
    world.expect_measure("Length 80.000 mm");
    assert_eq!(
        world.state().browser.unwrap().block_instances.len(),
        usize::from(grouped)
    );
}
