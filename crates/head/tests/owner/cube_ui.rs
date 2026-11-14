// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::deliver;
use super::*;
use crate::{HeadWork, placement_test_inputs::*};
use oblikovati_gui::GuiSketchTool;

#[test]
fn mouse_drawn_parameter_circle_extrudes_through_the_desktop_owner() {
    let (owner, _) = owner();
    let mut owner = owner
        .with_extrusion_policy(crate::part_scene::resolution_policy::desktop_policy())
        .unwrap();
    owner.create_part_document().unwrap();
    add_size(&mut owner);
    owner.edit_sketch_from_gui(true).unwrap();
    let edit = owner.active_sketch_edit().unwrap();
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Circle)
        .unwrap();
    deliver(&mut owner, &[button([0, 0], true), button([0, 0], false)]);
    deliver(&mut owner, &[motion([10, 0]), button([10, 0], true)]);
    let circle = owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot()
        .sketches
        .get(edit.sketch)
        .unwrap()
        .geometry()
        .circles()[0];
    constrain_circle(&mut owner, circle.id.0);
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(edit.document)
            .unwrap()
            .analyze_sketch(edit.sketch)
            .unwrap()
            .degrees_of_freedom,
        0
    );
    owner.edit_sketch_from_gui(false).unwrap();
    owner
        .extrude_from_gui(edit.document, edit.sketch, 0, "size")
        .unwrap();
    let scene = owner.take_scene().unwrap().unwrap();
    assert_eq!(scene.status.visible_bodies, 1);
    assert!(!scene.scene.triangles.is_empty());
}

fn add_size(owner: &mut TestOwner) {
    Client::new(owner.document_host())
        .parameters()
        .add(&ParameterSetArgs {
            name: "size".into(),
            expression: "10 mm".into(),
            ..Default::default()
        })
        .unwrap();
}

fn constrain_circle(owner: &mut TestOwner, circle: u64) {
    Client::new(owner.document_host())
        .dimension()
        .add(&AddDimensionArgs {
            sketch_index: 0,
            kind: "radius".into(),
            entities: Some(vec![circle]),
            expression: "size".into(),
            ..Default::default()
        })
        .unwrap();
}
