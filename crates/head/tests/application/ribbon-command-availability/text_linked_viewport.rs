// SPDX-License-Identifier: GPL-2.0-only
//! Follow a committed linked Text through native redraw, picking, and Undo.
use super::*;
use oblikovati_api::{
    client::Client,
    types::{SKETCH_TEXT_FIELD_MODEL_PARAMETER, SKETCH_TEXT_FIELD_PLACEHOLDER, SketchTextField},
    wire::{AddTextArgs, ParameterSetArgs},
};
use oblikovati_martensite::planar_curve::sketch::{SketchSamplingOptionsV1, sample_sketch_text};
use text_format_choices::glyph_pixels;

#[test]
fn native_sketch_text_linked_parameter_redraws_and_remains_pickable() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let position = world.sketch_point_of([900, 580]);
    insert_linked_text(&mut world, position);
    let source = world.edited_geometry();
    let first = glyph_pixels(&mut world);
    assert_eq!(displayed_text(&mut world), "10.000");
    change_dimension(&mut world);
    assert_eq!(displayed_text(&mut world), "12.000");
    assert_ne!(glyph_pixels(&mut world), first);
    assert_eq!(world.edited_geometry(), source);
    open_resolved_glyph(&mut world);
    click_editor_button(&mut world, "Cancel");
    x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
    assert_eq!(displayed_text(&mut world), "10.000");
    assert_eq!(glyph_pixels(&mut world), first);
    assert_eq!(world.edited_geometry(), source);
    world.close();
}

fn insert_linked_text(world: &mut ApplicationWorld, position: [f64; 2]) {
    let client = Client::new(world.app().owner.document_host());
    client.parameters().add(&dimension("10 mm")).unwrap();
    client
        .sketch()
        .add_text(&AddTextArgs {
            anchor: Some(position.to_vec()),
            text: SKETCH_TEXT_FIELD_PLACEHOLDER.into(),
            height: "5 mm".into(),
            font: "DejaVu Sans".into(),
            linked_fields: Some(vec![SketchTextField {
                at: 0,
                kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
                parameter: "d0".into(),
                ..Default::default()
            }]),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}

fn change_dimension(world: &mut ApplicationWorld) {
    let client = Client::new(world.app().owner.document_host());
    client.parameters().set(&dimension("12 mm")).unwrap();
    world.pump(Vec::new());
}

/// Build the fixture's `d0` request, for example from `"10 mm"`.
pub(super) fn dimension(expression: &str) -> ParameterSetArgs {
    ParameterSetArgs {
        name: "d0".into(),
        kind: "model".into(),
        expression: expression.into(),
        ..Default::default()
    }
}

fn displayed_text(world: &mut ApplicationWorld) -> String {
    let view = &world.app().window.native;
    assert!(view.sketch_display_declines().unwrap().is_empty());
    view.sketch_scene_status().unwrap().sketches[0]
        .geometry
        .texts()[0]
        .style
        .content
        .clone()
}

fn open_resolved_glyph(world: &mut ApplicationWorld) {
    let at = rendered_glyph_pixel(world);
    x11_input::right_click(at);
    world.pump(Vec::new());
    let editor = diagnostics::pointer_pixels(world.run("Edit Text").center());
    x11_input::click(editor);
    world.pump(Vec::new());
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the linked text picked at its visible glyph");
    };
    assert_eq!(style.content, "12.000");
    assert_eq!(
        world.state().sketch_text.unwrap().content,
        SKETCH_TEXT_FIELD_PLACEHOLDER
    );
}

fn rendered_glyph_pixel(world: &mut ApplicationWorld) -> [i32; 2] {
    let native = &world.app().window.native;
    let packet = &native.sketch_scene_status().unwrap().sketches[0];
    let text = &packet.geometry.texts()[0];
    let anchor = packet.geometry.point(text.anchor).unwrap().position;
    let paths = sample_sketch_text(
        &text.style,
        anchor,
        SketchSamplingOptionsV1 {
            working_scale: packet.working_scale,
            deflection: 0.001,
            maximum_curve_points: 4096,
        },
    )
    .unwrap();
    native.sketch_pixel(packet.plane, paths[0][0]).unwrap()
}
