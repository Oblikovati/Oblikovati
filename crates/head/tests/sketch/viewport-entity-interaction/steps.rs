// SPDX-License-Identifier: GPL-2.0-only
//! Native selection and clearing of a sketch line (#4033).
use super::super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_model::sketch::{SketchPointInput, SketchPrimitiveCreation};

#[given("an editable sketch contains an unselected line and the sketch is ready for input")]
fn unselected_line(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = SketchPrimitiveCreation::Line {
        endpoints: [
            SketchPointInput::New([-3., 1.]),
            SketchPointInput::New([3., 1.]),
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
    world.pump(Vec::new());
    assert!(world.state().sketch_selection.entities.is_empty());
    assert!(world.state().sketch_at_ready);
    world.last_capture = Some(super::super::super::capture_framebuffer(
        &mut world.app().window.native,
    ));
}
#[when("the user clicks the line away from its endpoints and constraint glyphs")]
fn click_line(world: &mut ApplicationWorld) {
    let position = world.line_point(0, 0.25);
    native_click(world, position);
}
#[then("the line becomes the selected sketch entity")]
fn line_selected(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = world.edited_geometry().lines()[0].id;
    assert_eq!(
        world.state().sketch_selection.entities,
        [(edit.sketch, line)]
    );
}
#[then("the viewport displays a distinct selected state")]
fn selected_visible(world: &mut ApplicationWorld) {
    use oblikovati_api::types::{TOKEN_SKETCH_GEOMETRY, TOKEN_SKETCH_SELECTED};
    let point = world.line_point(0, 0.75);
    let state = world.state();
    let palette = state.theme.as_ref().expect("active theme").palette();
    let normal = &palette[&TOKEN_SKETCH_GEOMETRY];
    let selected = &palette[&TOKEN_SKETCH_SELECTED];
    let colors = [
        [normal.r, normal.g, normal.b],
        [selected.r, selected.g, selected.b],
    ];
    let after = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let before = world.last_capture.as_ref().unwrap();
    assert!(
        selection_color_changed(before, &after, point, colors),
        "expected theme selection color near {point:?}"
    );
    world.save_capture("sketch-line-selected-4033");
}
fn selection_color_changed(
    before: &oblikovati_viewport::ViewportCapture,
    after: &oblikovati_viewport::ViewportCapture,
    at: [i32; 2],
    colors: [[f32; 3]; 2],
) -> bool {
    let width = after.extent[0] as i32;
    (-2..=2).any(|y| {
        (-2..=2).any(|x| {
            let offset = (((at[1] + y) * width + at[0] + x) * 4) as usize;
            let first = &before.rgba[offset..offset + 3];
            let second = &after.rgba[offset..offset + 3];
            color_distance(first, colors[0]) < color_distance(first, colors[1])
                && color_distance(second, colors[1]) < color_distance(second, colors[0])
        })
    })
}
fn color_distance(pixel: &[u8], color: [f32; 3]) -> f32 {
    pixel
        .iter()
        .zip(color)
        .map(|(channel, target)| (f32::from(*channel) / 255. - target).powi(2))
        .sum()
}
#[when("the user clicks empty sketch space")]
fn click_empty(world: &mut ApplicationWorld) {
    native_click(world, [1100, 650]);
}
fn native_click(world: &mut ApplicationWorld, position: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::super::x11_input::click(position);
    #[cfg(not(target_os = "linux"))]
    world.click(position);
    world.pump(Vec::new());
}
#[then("the line is no longer selected")]
fn selection_cleared(world: &mut ApplicationWorld) {
    assert!(world.state().sketch_selection.entities.is_empty());
}
