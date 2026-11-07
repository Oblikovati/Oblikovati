// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiTextureKey, GuiTextureResources};
fn one_pixel(color: [u8; 4]) -> GuiAtlas {
    GuiAtlas {
        extent: [1; 2],
        rgba: Arc::new(vec![color]),
    }
}
fn image_frame(owner: &mut GuiFrameEngine, key: GuiTextureKey) -> Result<GuiFrame, String> {
    owner.context.io_mut().display_size = [128.; 2];
    owner.context.io_mut().delta_time = 1. / 60.;
    let ui = owner.context.frame();
    crate::widget_abi::window_geometry(ui, [0.; 2], [128.; 2], imgui::Condition::Always);
    ui.window("Images").build(|| {
        ui.text("Font");
        crate::widget_abi::icon_button(
            ui,
            "Owned Image",
            crate::widget_abi::IconButtonPresentation {
                texture: key,
                size: [96., 58.],
                icon: [32.; 2],
                large: true,
            },
        );
    });
    capture_frame(owner.context.render(), [128; 2], &owner.resources)
}
#[test]
fn snapshot_retains_font_and_colored_bindings_after_context_destruction() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let key = GuiTextureKey::from_raw(2).unwrap();
    owner.resources =
        GuiTextureResources::new(owner.atlas.clone(), [(key, one_pixel([255, 0, 0, 128]))])
            .unwrap();
    let first = image_frame(&mut owner, key).unwrap();
    owner.resources =
        GuiTextureResources::new(owner.atlas.clone(), [(key, one_pixel([0, 0, 255, 255]))])
            .unwrap();
    let second = image_frame(&mut owner, key).unwrap();
    drop(owner);
    assert!(first.batches.iter().any(|batch| batch.texture == key));
    assert!(
        first
            .batches
            .iter()
            .any(|batch| batch.texture == GuiTextureKey::FONT)
    );
    assert_eq!(
        first.resources.source(key).unwrap().pixels().rgba[0],
        [255, 0, 0, 128]
    );
    assert_eq!(
        second.resources.source(key).unwrap().pixels().rgba[0],
        [0, 0, 255, 255]
    );
    #[cfg(target_os = "linux")]
    oblikovati_viewport::GuiFrameCompiler::with_resources(&first.resources)
        .unwrap()
        .compile(&first)
        .unwrap();
}
#[test]
fn an_imgui_image_with_an_unknown_id_is_refused_at_snapshot_creation() {
    let mut owner = GuiFrameEngine::new().unwrap();
    let key = GuiTextureKey::from_raw(900).unwrap();
    // First-frame windows can be clipped while ImGui measures them; the second emits their image.
    let _ = image_frame(&mut owner, key);
    assert!(
        image_frame(&mut owner, key)
            .unwrap_err()
            .contains("GUI texture 900")
    );
}
