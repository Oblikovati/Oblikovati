// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::document_sketch::SketchBlockInstance;
use oblikovati_viewport::{ViewportSketchBlock, ViewportSketchSelection};

fn block_packet() -> ViewportSketchDisplay {
    let mut geometry = Sketch2D::default();
    geometry.add_defining_point(Id(1), [0., 0.]).unwrap();
    geometry.add_defining_point(Id(2), [2., 0.]).unwrap();
    geometry.add_line(Id(3), [Id(1), Id(2)]).unwrap();
    let instance =
        SketchBlockInstance::new(Id(42), "Rib", [-3., -2.], std::f64::consts::FRAC_PI_2, 2.)
            .unwrap();
    ViewportSketchDisplay {
        text_declines: Vec::new(),
        working_scale: 1.,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        geometry: Sketch2D::default(),
        blocks: vec![ViewportSketchBlock {
            text_declines: Vec::new(),
            instance,
            geometry,
            definition_color: None,
        }],
        curvature: Vec::new(),
        line_color: None,
    }
}
#[test]
fn block_geometry_renders_at_its_placement_and_uses_instance_selection() {
    let (runtime, mut view) = open_display("Block display");
    let packet = block_packet();
    view.set_sketch_display(std::slice::from_ref(&packet))
        .unwrap();
    let normal = view.redraw().unwrap();
    assert!(lit_near(&view, &normal, [-3., 0.]));
    assert!(!lit_near(&view, &normal, [1., 0.]));
    assert!(view.sketch_display_declines().unwrap().is_empty());
    view.set_sketch_selection(&ViewportSketchSelection {
        sketches: vec![],
        entities: vec![(DocumentSketchId(1), Id(42))],
    })
    .unwrap();
    let selected = view.redraw().unwrap();
    assert_ne!(selected.rgba, normal.rgba);
    view.set_sketch_display(&[]).unwrap();
    assert_black(&view.redraw().unwrap());
    close_display(&runtime, &mut view);
}
#[test]
fn block_transform_failure_names_the_instance() {
    let (runtime, mut view) = open_display("Block display failure");
    let mut packet = block_packet();
    packet.blocks[0].instance =
        SketchBlockInstance::new(Id(42), "Rib", [0., 0.], 0., f64::MAX).unwrap();
    view.set_sketch_display(&[packet]).unwrap();
    let declines = view.sketch_display_declines().unwrap();
    assert!(!declines.is_empty());
    assert!(declines.iter().all(|decline| decline.entity == Id(42)));
    close_display(&runtime, &mut view);
}

#[test]
fn invalid_child_does_not_hide_the_other_block_geometry() {
    let (runtime, mut view) = open_display("Block child failure");
    let mut packet = block_packet();
    packet.blocks[0]
        .geometry
        .add_line(Id(4), [Id(1), Id(1)])
        .unwrap();
    view.set_sketch_display(&[packet]).unwrap();
    let capture = view.redraw().unwrap();
    assert!(lit_near(&view, &capture, [-3., 0.]));
    let declines = view.sketch_display_declines().unwrap();
    assert_eq!(declines.len(), 1);
    assert_eq!(declines[0].entity, Id(42));
    assert!(declines[0].cause.contains("4"));
    close_display(&runtime, &mut view);
}
