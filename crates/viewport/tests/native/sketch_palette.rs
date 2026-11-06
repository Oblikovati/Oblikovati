// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{Rgba, TOKEN_SKETCH_GEOMETRY, TOKEN_SKETCH_PREVIEW};
use oblikovati_model::theme::ThemeDefinition;
use oblikovati_viewport::{ViewportNavigation, ViewportSketchPalette};
fn palette(opacity: f32) -> ViewportSketchPalette {
    let mut theme = ThemeDefinition::dark()
        .unwrap()
        .duplicate("Acceptance")
        .unwrap();
    theme
        .set_color(
            &TOKEN_SKETCH_GEOMETRY,
            Rgba {
                r: 1.,
                g: 0.,
                b: 0.,
                a: opacity,
            },
        )
        .unwrap();
    theme
        .set_color(
            &TOKEN_SKETCH_PREVIEW,
            Rgba {
                r: 0.,
                g: 1.,
                b: 0.,
                a: 1.,
            },
        )
        .unwrap();
    ViewportSketchPalette::from_theme(&theme)
}
fn colour_count(capture: &ViewportCapture, expected: [u8; 3]) -> usize {
    capture
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|pixel| {
            pixel[..3]
                .iter()
                .zip(expected)
                .all(|(actual, expected)| actual.abs_diff(expected) <= 1)
        })
        .count()
}
#[test]
fn native_sketch_palette_restyles_existing_curves_points_and_previews_without_changing_geometry() {
    let (runtime, mut view) = super::sketch_display::open_display("Sketch palette acceptance");
    let geometry = super::sketch_display::packet();
    let preview = super::sketch_preview::packet();
    view.set_sketch_display(std::slice::from_ref(&geometry))
        .unwrap();
    view.set_sketch_preview(Some(&preview)).unwrap();
    let before = view.redraw().unwrap();
    assert!(view.set_sketch_palette(&palette(1.)).unwrap());
    assert!(view.capture().is_err());
    let colored = view.redraw().unwrap();
    assert_eq!(colored.scene_generation, before.scene_generation);
    assert!(colored.overlay_generation > before.overlay_generation);
    assert!(colour_count(&colored, [255, 0, 0]) > 10);
    assert!(colour_count(&colored, [0, 255, 0]) > 10);
    assert!(!view.set_sketch_palette(&palette(1.)).unwrap());
    assert_eq!(view.capture().unwrap(), colored);
    verify_palette_refresh(&mut view, &geometry);
    verify_palette_opacity(&mut view);
    verify_srgb_channels(&mut view);
    view.close();
    assert!(view.set_sketch_palette(&palette(1.)).is_err());
    assert!(runtime.validation_errors().unwrap().is_empty());
}
fn verify_palette_refresh(
    view: &mut Viewport,
    geometry: &oblikovati_viewport::ViewportSketchDisplay,
) {
    assert!(
        !view
            .set_sketch_display(std::slice::from_ref(geometry))
            .unwrap()
    );
    view.submit(&ViewportScene {
        background: [0.; 3],
        triangles: vec![],
    })
    .unwrap();
    view.navigate(ViewportNavigation::Zoom {
        factor: 1.5,
        anchor: None,
    })
    .unwrap();
    let refreshed = view.redraw().unwrap();
    assert!(colour_count(&refreshed, [255, 0, 0]) > 10);
    assert!(colour_count(&refreshed, [0, 255, 0]) > 10);
    view.set_sketch_preview(None).unwrap();
    let cleared = view.redraw().unwrap();
    assert!(colour_count(&cleared, [255, 0, 0]) > 10);
    assert_eq!(colour_count(&cleared, [0, 255, 0]), 0);
}
fn verify_palette_opacity(view: &mut Viewport) {
    assert!(view.set_sketch_palette(&palette(0.5)).unwrap());
    let translucent = view.redraw().unwrap();
    // Martensite blends linear-light red over black before its sRGB output conversion.
    assert!(
        colour_count(&translucent, [188, 0, 0]) > 10,
        "expected half-opacity red, observed {:?}",
        translucent
            .rgba
            .as_chunks::<4>()
            .0
            .iter()
            .copied()
            .collect::<std::collections::BTreeSet<_>>()
    );
    assert_eq!(colour_count(&translucent, [255, 0, 0]), 0);
    assert!(view.set_sketch_palette(&palette(0.)).unwrap());
    let transparent = view.redraw().unwrap();
    assert!(
        transparent
            .rgba
            .as_chunks::<4>()
            .0
            .iter()
            .all(|pixel| pixel[..3] == [0; 3])
    );
}

fn verify_srgb_channels(view: &mut Viewport) {
    let mut theme = ThemeDefinition::light()
        .unwrap()
        .duplicate("Midtones")
        .unwrap();
    theme
        .set_color(
            &TOKEN_SKETCH_GEOMETRY,
            Rgba {
                r: 0.5,
                g: 0.25,
                b: 0.75,
                a: 1.,
            },
        )
        .unwrap();
    view.set_sketch_palette(&ViewportSketchPalette::from_theme(&theme))
        .unwrap();
    assert!(colour_count(&view.redraw().unwrap(), [128, 64, 191]) > 10);
}

#[test]
fn native_selection_restyles_only_committed_sketches_and_survives_refresh() {
    let (runtime, mut view) = super::sketch_display::open_display("Selected sketch acceptance");
    let geometry = super::sketch_display::packet();
    view.set_sketch_palette(&selection_palette([0., 0., 1.]))
        .unwrap();
    view.set_sketch_display(std::slice::from_ref(&geometry))
        .unwrap();
    view.set_sketch_preview(Some(&super::sketch_preview::packet()))
        .unwrap();
    let before = view.redraw().unwrap();
    assert!(colour_count(&before, [255, 0, 0]) > 10);
    assert!(
        view.set_sketch_selection(&oblikovati_viewport::ViewportSketchSelection {
            sketches: vec![geometry.sketch, geometry.sketch],
            ..Default::default()
        })
        .unwrap()
    );
    let selected = view.redraw().unwrap();
    assert_eq!(selected.scene_generation, before.scene_generation);
    assert!(colour_count(&selected, [0, 0, 255]) > 10);
    assert!(colour_count(&selected, [0, 255, 0]) > 10);
    assert_eq!(colour_count(&selected, [255, 0, 0]), 0);
    verify_selected_refresh(&mut view, &geometry);
    super::sketch_display::close_display(&runtime, &mut view);
    assert!(
        view.set_sketch_selection(&oblikovati_viewport::ViewportSketchSelection {
            sketches: vec![],
            ..Default::default()
        })
        .is_err()
    );
}
fn selection_palette(selected: [f32; 3]) -> ViewportSketchPalette {
    let mut theme = ThemeDefinition::dark()
        .unwrap()
        .duplicate("Selection")
        .unwrap();
    for (token, rgb) in [
        (TOKEN_SKETCH_GEOMETRY, [1., 0., 0.]),
        (TOKEN_SKETCH_PREVIEW, [0., 1., 0.]),
        (oblikovati_api::types::TOKEN_SKETCH_SELECTED, selected),
    ] {
        theme
            .set_color(
                &token,
                Rgba {
                    r: rgb[0],
                    g: rgb[1],
                    b: rgb[2],
                    a: 1.,
                },
            )
            .unwrap();
    }
    ViewportSketchPalette::from_theme(&theme)
}
fn verify_selected_refresh(
    view: &mut Viewport,
    geometry: &oblikovati_viewport::ViewportSketchDisplay,
) {
    assert!(
        !view
            .set_sketch_selection(&oblikovati_viewport::ViewportSketchSelection {
                sketches: vec![geometry.sketch],
                ..Default::default()
            })
            .unwrap()
    );
    assert!(
        !view
            .set_sketch_display(std::slice::from_ref(geometry))
            .unwrap()
    );
    view.navigate(ViewportNavigation::Zoom {
        factor: 1.5,
        anchor: None,
    })
    .unwrap();
    assert!(colour_count(&view.redraw().unwrap(), [0, 0, 255]) > 10);
    view.set_sketch_palette(&selection_palette([0., 0., 1.]))
        .unwrap();
    view.set_sketch_selection(&oblikovati_viewport::ViewportSketchSelection {
        sketches: vec![],
        ..Default::default()
    })
    .unwrap();
    assert!(colour_count(&view.redraw().unwrap(), [255, 0, 0]) > 10);
}

#[test]
fn native_selection_uses_sketch_identity_and_restyles_selected_theme_colours() {
    let (runtime, mut view) = super::sketch_display::open_display("Independent sketch selection");
    let first = super::sketch_display::packet();
    let mut second = first.clone();
    second.sketch = oblikovati_model::document_sketch::DocumentSketchId(2);
    second.plane = oblikovati_model::sketch::SketchPlane::new(
        [8., 0., 0.],
        [1., 0., 0.],
        [0., 1., 0.],
        Default::default(),
    )
    .unwrap();
    view.set_sketch_palette(&selection_palette([0., 0., 1.]))
        .unwrap();
    view.set_sketch_display(&[first.clone(), second]).unwrap();
    view.set_sketch_selection(&oblikovati_viewport::ViewportSketchSelection {
        sketches: vec![first.sketch],
        ..Default::default()
    })
    .unwrap();
    assert_selection_colours(&mut view, [0, 0, 255]);
    view.set_sketch_palette(&selection_palette([0., 1., 1.]))
        .unwrap();
    assert_selection_colours(&mut view, [0, 255, 255]);
    super::sketch_display::close_display(&runtime, &mut view);
}
fn assert_selection_colours(view: &mut Viewport, selected: [u8; 3]) {
    let capture = view.redraw().unwrap();
    assert!(colour_count(&capture, [255, 0, 0]) > 10);
    assert!(colour_count(&capture, selected) > 10);
}

#[test]
fn individual_curve_and_hidden_endpoint_highlights_keep_other_entities_unselected() {
    use oblikovati_model::sketch::SketchEntityId as Id;
    use oblikovati_viewport::ViewportSketchSelection;
    let (runtime, mut view) = super::sketch_display::open_display("Entity selection");
    let geometry = super::sketch_display::packet();
    view.set_sketch_palette(&selection_palette([0., 0., 1.]))
        .unwrap();
    view.set_sketch_display(std::slice::from_ref(&geometry))
        .unwrap();
    let selected = ViewportSketchSelection {
        sketches: vec![],
        entities: vec![(geometry.sketch, Id(1))],
    };
    assert!(view.set_sketch_selection(&selected).unwrap());
    assert_selection_colours(&mut view, [0, 0, 255]);
    assert!(
        !view
            .set_sketch_selection(&ViewportSketchSelection {
                entities: vec![(geometry.sketch, Id(1)); 2],
                ..Default::default()
            })
            .unwrap()
    );
    view.set_sketch_selection(&ViewportSketchSelection {
        entities: vec![(geometry.sketch, Id(100))],
        ..Default::default()
    })
    .unwrap();
    assert_selection_colours(&mut view, [0, 0, 255]);
    view.set_sketch_selection(&ViewportSketchSelection::default())
        .unwrap();
    assert_eq!(colour_count(&view.redraw().unwrap(), [0, 0, 255]), 0);
    super::sketch_display::close_display(&runtime, &mut view);
}
