// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::icons::catalog;
use crate::{GuiCommandState, GuiFrameRequest};
use oblikovati_model::ui_scale::UiScaleOptions;
fn request(font: f32, icon: f32, backing: f32) -> GuiFrameRequest {
    GuiFrameRequest {
        extent: [1440, 900],
        events: Vec::new(),
        commands: GuiCommandState {
            ui_scale: Some(UiScaleOptions::new(font, icon).unwrap()),
            ..Default::default()
        },
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [backing; 2],
    }
}
fn font() -> GuiAtlas {
    GuiAtlas {
        extent: [1; 2],
        rgba: Arc::new(vec![[255; 4]]),
    }
}
#[test]
fn independent_factors_preserve_physical_icon_size_when_only_text_changes() {
    let mut cache = GuiIconCache::new(font()).unwrap();
    for (text, icon, expected) in [(1., 1., 40.), (2., 1., 40.), (1., 2., 80.), (2., 2., 80.)] {
        let request = request(text, icon, 1.);
        cache.prepare(&request, &font()).unwrap();
        assert_eq!(
            cache.presentation.logical[1][0] * request.layout_scale()[0],
            expected
        );
        assert_eq!(
            cache
                .resources
                .source(
                    cache
                        .presentation
                        .key(IconGlyph::Rectangle, IconSize::Large)
                )
                .unwrap()
                .pixels()
                .extent,
            [expected as u32; 2]
        );
    }
}
#[test]
fn stable_preparation_reuses_masks_and_font_changes_preserve_icon_pixels() {
    let mut cache = GuiIconCache::new(font()).unwrap();
    let request = request(1., 1., 1.);
    assert!(cache.prepare(&request, &font()).unwrap());
    let first = cache.resources.clone();
    let key = cache
        .presentation
        .key(IconGlyph::Rectangle, IconSize::Large);
    assert!(!cache.prepare(&request, &font()).unwrap());
    let replaced = GuiAtlas {
        extent: [1; 2],
        rgba: Arc::new(vec![[255, 255, 255, 0]]),
    };
    cache.prepare(&request, &replaced).unwrap();
    assert!(Arc::ptr_eq(
        &first.source(key).unwrap().pixels().rgba,
        &cache.resources.source(key).unwrap().pixels().rgba
    ));
    assert_eq!(cache.masks.len(), catalog::GLYPHS.len() * 2);
}
#[test]
fn theme_changes_recompose_without_rasterizing_the_svg_again() {
    let mut cache = GuiIconCache::new(font()).unwrap();
    let mut request = request(1., 1., 1.);
    cache.prepare(&request, &font()).unwrap();
    let first = cache.resources.clone();
    let library = oblikovati_model::theme::ThemeLibrary::new(Vec::new(), "Light").unwrap();
    request.commands.theme = Some(crate::GuiThemeState::from_library(&library));
    assert!(!cache.prepare(&request, &font()).unwrap());
    assert_eq!(cache.masks.len(), catalog::GLYPHS.len() * 2);
    assert!(cache.resources != first);
}
#[test]
fn invalid_edges_leave_the_published_bank_unchanged() {
    let mut cache = GuiIconCache::new(font()).unwrap();
    let initial = cache.resources.clone();
    let request = request(1., 2.5, 100.);
    assert!(
        cache
            .prepare(&request, &font())
            .unwrap_err()
            .contains("expected finite pixel sizes")
    );
    assert_eq!(cache.resources, initial);
}

#[test]
fn text_only_scale_changes_invalidate_layout_without_recomposing_icons() {
    let mut cache = GuiIconCache::new(font()).unwrap();
    cache.prepare(&request(1., 1., 1.), &font()).unwrap();
    let previous = cache.resources.clone();
    assert!(cache.prepare(&request(2., 1., 1.), &font()).unwrap());
    assert_eq!(cache.presentation.logical[1], [20.; 2]);
    assert_eq!(cache.resources, previous);
    assert!(!cache.prepare(&request(2., 1., 1.), &font()).unwrap());
}
