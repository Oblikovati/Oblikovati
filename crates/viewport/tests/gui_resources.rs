// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_viewport::*;
use std::sync::Arc;
#[path = "gui_fixture.rs"]
mod gui_fixture;
fn image(color: [u8; 4]) -> GuiAtlas {
    GuiAtlas {
        extent: [1; 2],
        rgba: Arc::new(vec![color]),
    }
}
fn resource_frame() -> GuiFrame {
    let mut frame = gui_fixture::frame([32; 2], [0, 0, 0, 255], [0., 0., 32., 32.]);
    let images = [(2, [128, 0, 0, 255]), (3, [0, 255, 0, 255])]
        .map(|(id, color)| (GuiTextureKey::from_raw(id).unwrap(), image(color)));
    frame.resources = GuiTextureResources::new(gui_fixture::atlas(), images).unwrap();
    for (id, clip) in [(2, [0., 0., 16., 32.]), (3, [16., 0., 32., 32.])] {
        let mut batch = gui_fixture::frame([32; 2], [255; 4], clip)
            .batches
            .remove(0);
        batch.texture = GuiTextureKey::from_raw(id).unwrap();
        frame.batches.push(batch);
    }
    frame
}
#[test]
fn one_frame_resolves_fonts_and_distinct_color_images_in_submission_order() {
    let frame = resource_frame();
    let compiled = GuiFrameCompiler::with_resources(&frame.resources)
        .unwrap()
        .compile(&frame)
        .unwrap();
    assert_eq!(compiled.draws().len(), 3);
    let red = compiled.draws()[1]
        .textured_style()
        .unwrap()
        .image()
        .pixels()[0];
    let green = compiled.draws()[2]
        .textured_style()
        .unwrap()
        .image()
        .pixels()[0];
    assert!((red[0] - 0.2158605).abs() < 1e-7 && red[1..] == [0., 0., 1.]);
    assert_eq!(green, [0., 1., 0., 1.]);
}
#[test]
fn unknown_texture_is_refused_even_when_its_batch_has_no_indices() {
    let mut frame = resource_frame();
    frame.batches[0].texture = GuiTextureKey::from_raw(900).unwrap();
    frame.batches[0].indices.clear();
    let compiler = GuiFrameCompiler::with_resources(&frame.resources).unwrap();
    assert!(
        compiler
            .compile(&frame)
            .unwrap_err()
            .contains("GUI texture 900")
    );
}
#[test]
fn changed_resources_cannot_rebind_a_retained_old_frame() {
    let first = resource_frame();
    let compiler = GuiFrameCompiler::with_resources(&first.resources).unwrap();
    let changed = GuiTextureResources::new(
        gui_fixture::atlas(),
        [(GuiTextureKey::from_raw(2).unwrap(), image([0, 0, 255, 128]))],
    )
    .unwrap();
    let next_compiler = GuiFrameCompiler::with_resources(&changed).unwrap();
    assert!(
        next_compiler
            .compile(&first)
            .unwrap_err()
            .contains("resource generation")
    );
    drop(changed);
    let retained = compiler.compile(&first).unwrap();
    assert_eq!(
        retained.draws()[1]
            .textured_style()
            .unwrap()
            .image()
            .pixels()[0][3],
        1.
    );
}
#[test]
fn color_alpha_remains_straight_in_the_linear_kernel_image() {
    let resources = GuiTextureResources::new(
        gui_fixture::atlas(),
        [(GuiTextureKey::from_raw(2).unwrap(), image([255, 0, 0, 128]))],
    )
    .unwrap();
    let mut frame = gui_fixture::frame([32; 2], [255; 4], [0., 0., 32., 32.]);
    frame.batches[0].texture = GuiTextureKey::from_raw(2).unwrap();
    frame.resources = resources;
    let compiled = compile_gui_frame(&frame, &gui_fixture::atlas()).unwrap();
    assert_eq!(
        compiled.draws()[0]
            .textured_style()
            .unwrap()
            .image()
            .pixels()[0],
        [1., 0., 0., 128. / 255.]
    );
}
#[test]
fn gpu_uses_each_owned_image_for_its_exact_clip_region() {
    use martensite_vulkan::{RenderBackend, VulkanOptions, VulkanRenderer};
    let frame = resource_frame();
    let mut renderer = VulkanRenderer::new(VulkanOptions {
        validation: true,
        ..Default::default()
    })
    .unwrap();
    let compiled = compile_gui_frame(&frame, &gui_fixture::atlas()).unwrap();
    let capture = renderer.render(&compiled).unwrap();
    assert_eq!(capture.pixel(8, 16).unwrap(), [128, 0, 0, 255]);
    assert_eq!(capture.pixel(24, 16).unwrap(), [0, 255, 0, 255]);
    assert!(renderer.validation_errors().unwrap().is_empty());
}
