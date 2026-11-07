// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn texel(color: [u8; 4]) -> GuiAtlas {
    GuiAtlas {
        extent: [1; 2],
        rgba: Arc::new(vec![color]),
    }
}
#[test]
fn portable_keys_and_exact_bindings_refuse_null_overflow_and_unknown_values() {
    assert!(GuiTextureKey::from_raw(0).is_err());
    if let Ok(overflow) = usize::try_from(u64::from(u32::MAX) + 1) {
        assert!(GuiTextureKey::from_raw(overflow).is_err());
    }
    let bank = GuiTextureResources::new(texel([255; 4]), []).unwrap();
    assert_eq!(bank.binding(1).unwrap(), GuiTextureKey::FONT);
    assert!(bank.binding(2).unwrap_err().contains("GUI texture 2"));
    assert_eq!(bank.iter().count(), 1);
}
#[test]
fn duplicate_reserved_and_malformed_images_are_refused_before_publication() {
    let key = GuiTextureKey::from_raw(2).unwrap();
    for images in [
        vec![(GuiTextureKey::FONT, texel([0; 4]))],
        vec![(key, texel([0; 4])), (key, texel([1; 4]))],
    ] {
        assert!(
            GuiTextureResources::new(texel([255; 4]), images)
                .unwrap_err()
                .contains("unique binding")
        );
    }
    for extent in [[0, 1], [1, 0], [2, 1], [u32::MAX; 2]] {
        let bad = GuiAtlas {
            extent,
            rgba: Arc::new(vec![[255; 4]]),
        };
        assert!(
            GuiTextureResources::new(bad, [])
                .unwrap_err()
                .contains("width*height")
        );
    }
    assert!(
        GuiTextureResources::new(texel([0, 0, 0, 255]), [])
            .unwrap_err()
            .contains("white RGB")
    );
}
#[test]
fn new_font_generations_keep_color_images_and_old_pixels_immutable() {
    let key = GuiTextureKey::from_raw(2).unwrap();
    let mut original = texel([128, 0, 0, 128]);
    let first = GuiTextureResources::new(texel([255; 4]), [(key, original.clone())]).unwrap();
    Arc::make_mut(&mut original.rgba)[0] = [0, 255, 0, 255];
    let second = first.with_font(texel([255, 255, 255, 0])).unwrap();
    assert_eq!(
        first.source(key).unwrap().pixels().rgba[0],
        [128, 0, 0, 128]
    );
    assert_eq!(second.source(key), first.source(key));
    assert_ne!(first, second);
    assert!(matches!(
        second.source(GuiTextureKey::FONT).unwrap(),
        GuiTextureSource::Font(_)
    ));
}
