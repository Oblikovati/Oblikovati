// SPDX-License-Identifier: GPL-2.0-only
#![cfg(target_os = "linux")]
use oblikovati_viewport::*;
use std::sync::Arc;
fn atlas() -> GuiAtlas {
    GuiAtlas {
        extent: [1, 1],
        rgba: Arc::new(vec![[255; 4]]),
    }
}
fn colored_quad(color: [u8; 4], clip_rect: [f32; 4]) -> GuiFrame {
    let vertices = [[0., 0.], [32., 0.], [32., 32.], [0., 32.]]
        .map(|position| GuiVertex {
            position,
            uv: [0.5; 2],
            srgb: color,
        })
        .into();
    GuiFrame {
        extent: [32; 2],
        batches: vec![GuiBatch {
            vertices,
            indices: vec![0, 1, 2, 0, 2, 3],
            clip_rect,
        }],
    }
}
#[test]
fn compilation_preserves_order_and_uses_four_intersected_clip_chains() {
    let mut snapshot = colored_quad([20, 80, 160, 255], [4., 8., 24., 28.]);
    snapshot.batches.push(
        colored_quad([255, 0, 0, 128], [0., 0., 32., 32.])
            .batches
            .remove(0),
    );
    let frame = compile_gui_frame(&snapshot, &atlas()).unwrap();
    assert_eq!(frame.extent(), [32; 2]);
    assert_eq!(frame.layers()[0].depth_policy(), (false, false, false));
    assert_eq!(frame.draws().len(), 2);
    assert_eq!(frame.draws()[0].clipping().unwrap().chains().len(), 4);
    assert!(frame.draws()[0].vertices()[0].color[0] < 0.01);
    assert_eq!(frame.draws()[1].vertices()[0].color[3], 128. / 255.);
}
#[test]
fn invalid_geometry_clip_extent_and_font_content_are_refused_before_submission() {
    let mut snapshot = colored_quad([255; 4], [0., 0., 32., 32.]);
    snapshot.batches[0].indices[0] = 20;
    assert!(compile_gui_frame(&snapshot, &atlas()).is_err());
    snapshot.batches[0].indices[0] = 0;
    snapshot.batches[0].clip_rect[0] = f32::NAN;
    assert!(compile_gui_frame(&snapshot, &atlas()).is_err());
    snapshot.batches[0].clip_rect = [32., 0., 0., 32.];
    assert!(compile_gui_frame(&snapshot, &atlas()).is_err());
    snapshot.batches[0].clip_rect = [0., 0., 32., 32.];
    snapshot.extent[0] = 0;
    assert!(compile_gui_frame(&snapshot, &atlas()).is_err());
    require_invalid_font(&snapshot);
}
#[test]
fn gpu_clipping_and_srgb_conversion_reproduce_the_expected_pixels() {
    use martensite_vulkan::{RenderBackend, VulkanOptions, VulkanRenderer};
    let mut renderer = VulkanRenderer::new(VulkanOptions::default()).unwrap();
    let snapshot = colored_quad([20, 80, 160, 255], [4., 8., 24., 28.]);
    let captured = renderer
        .render(&compile_gui_frame(&snapshot, &atlas()).unwrap())
        .unwrap();
    assert_eq!(captured.pixel(3, 16).unwrap(), [0, 0, 0, 255]);
    assert_eq!(captured.pixel(24, 16).unwrap(), [0, 0, 0, 255]);
    assert_eq!(captured.pixel(16, 7).unwrap(), [0, 0, 0, 255]);
    assert_eq!(captured.pixel(16, 28).unwrap(), [0, 0, 0, 255]);
    let pixel = captured.pixel(16, 16).unwrap();
    for (actual, expected) in pixel.into_iter().zip([20, 80, 160, 255]) {
        assert!(
            actual.abs_diff(expected) <= 1,
            "{actual}: expected {expected}"
        );
    }
    assert!(renderer.validation_errors().is_empty());
}
#[test]
fn empty_commands_produce_an_empty_valid_frame() {
    let mut snapshot = colored_quad([255; 4], [0., 0., 32., 32.]);
    snapshot.batches[0].indices.clear();
    assert!(
        compile_gui_frame(&snapshot, &atlas())
            .unwrap()
            .draws()
            .is_empty()
    );
}

#[test]
fn gpu_alpha_blending_keeps_gui_submission_order() {
    use martensite_vulkan::{RenderBackend, VulkanOptions, VulkanRenderer};
    let mut renderer = VulkanRenderer::new(VulkanOptions::default()).unwrap();
    let mut snapshot = colored_quad([0, 0, 255, 255], [0., 0., 32., 32.]);
    let red = colored_quad([255, 0, 0, 128], [0., 0., 32., 32.]);
    snapshot.batches.extend(red.batches);
    let capture = renderer
        .render(&compile_gui_frame(&snapshot, &atlas()).unwrap())
        .unwrap();
    let actual = capture.pixel(16, 16).unwrap();
    for (observed, expected) in actual.into_iter().zip([188_u8, 0, 187, 255]) {
        assert!(
            observed.abs_diff(expected) <= 1,
            "{observed}: expected {expected}"
        );
    }
    assert!(renderer.validation_errors().is_empty());
}

#[test]
fn repeated_compilation_shares_the_prepared_font_resource() {
    let compiler = GuiFrameCompiler::new(&atlas()).unwrap();
    let snapshot = colored_quad([255; 4], [0., 0., 32., 32.]);
    let first = compiler.compile(&snapshot).unwrap();
    let second = compiler.compile(&snapshot).unwrap();
    let original = first.draws()[0].textured_style().unwrap().image().pixels();
    let repeated = second.draws()[0].textured_style().unwrap().image().pixels();
    assert!(std::ptr::eq(original.as_ptr(), repeated.as_ptr()));
}

fn require_invalid_font(snapshot: &GuiFrame) {
    let colored = GuiAtlas {
        extent: [1, 1],
        rgba: Arc::new(vec![[0, 0, 0, 255]]),
    };
    assert!(
        compile_gui_frame(snapshot, &colored)
            .unwrap_err()
            .contains("white RGB")
    );
}
