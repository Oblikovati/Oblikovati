// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{Rgba, TOKEN_SKETCH_CANDIDATE, TOKEN_SKETCH_GEOMETRY};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId as Id, SketchPlane},
    theme::ThemeDefinition,
};
use oblikovati_viewport::{
    ViewportSketchDisplay, ViewportSketchFeedback, ViewportSketchHover, ViewportSketchPalette,
};

fn parallel_lines(offset: f64) -> ViewportSketchDisplay {
    let mut geometry = Sketch2D::default();
    for (id, point) in [
        (1, [-4., offset]),
        (2, [4., offset]),
        (3, [-4., offset + 3.]),
        (4, [4., offset + 3.]),
    ] {
        geometry.add_defining_point(Id(id), point).unwrap();
    }
    geometry.add_line(Id(5), [Id(1), Id(2)]).unwrap();
    geometry.add_line(Id(6), [Id(3), Id(4)]).unwrap();
    ViewportSketchDisplay {
        working_scale: 1.,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        geometry,
        blocks: Vec::new(),
        line_color: None,
    }
}

fn candidate_palette() -> ViewportSketchPalette {
    let mut theme = ThemeDefinition::dark()
        .unwrap()
        .duplicate("Hover test")
        .unwrap();
    for (token, rgb) in [
        (TOKEN_SKETCH_GEOMETRY, [1., 0., 0.]),
        (TOKEN_SKETCH_CANDIDATE, [0., 1., 0.]),
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

fn green_near(view: &Viewport, image: &ViewportCapture, point: [f64; 2]) -> bool {
    let [x, y] = view.sketch_pixel(SketchPlane::xy(), point).unwrap();
    (-2..=2).any(|dy| {
        (-2..=2).any(|dx| {
            let at = ((y + dy) as usize * image.extent[0] as usize + (x + dx) as usize) * 4;
            image.rgba[at + 1] > 200 && image.rgba[at] < 20
        })
    })
}

#[test]
fn hovering_two_lines_updates_only_transient_presentation() {
    let (runtime, mut view) = super::sketch_display::open_display("Constraint hover");
    let packet = parallel_lines(0.);
    view.set_sketch_display(std::slice::from_ref(&packet))
        .unwrap();
    view.set_sketch_palette(&candidate_palette()).unwrap();
    let baseline = view.redraw().unwrap();
    assert!(!green_near(&view, &baseline, [0., 0.]));
    let hover = ViewportSketchHover {
        sketch: packet.sketch,
        entities: vec![Id(5), Id(6)],
    };
    assert!(view.set_sketch_hover(Some(&hover)).unwrap());
    let highlighted = view.redraw().unwrap();
    assert!(green_near(&view, &highlighted, [0., 0.]));
    assert!(green_near(&view, &highlighted, [0., 3.]));
    assert!(!view.set_sketch_hover(Some(&hover)).unwrap());
    assert_eq!(view.capture().unwrap(), highlighted);
    view.set_sketch_hover(None).unwrap();
    assert_eq!(view.redraw().unwrap().rgba, baseline.rgba);
    super::sketch_display::close_display(&runtime, &mut view);
}

#[test]
fn hovering_survives_geometry_refresh_and_clears_when_sketch_is_removed() {
    let (runtime, mut view) = super::sketch_display::open_display("Constraint hover refresh");
    view.set_sketch_display(&[parallel_lines(0.)]).unwrap();
    view.set_sketch_palette(&candidate_palette()).unwrap();
    let hover = ViewportSketchHover {
        sketch: DocumentSketchId(1),
        entities: vec![Id(5), Id(6)],
    };
    view.set_sketch_hover(Some(&hover)).unwrap();
    view.set_sketch_display(&[parallel_lines(1.)]).unwrap();
    let refreshed = view.redraw().unwrap();
    assert!(green_near(&view, &refreshed, [0., 1.]));
    assert!(green_near(&view, &refreshed, [0., 4.]));
    view.set_sketch_display(&[]).unwrap();
    assert!(!view.set_sketch_hover(None).unwrap());
    super::sketch_display::close_display(&runtime, &mut view);
}

#[test]
fn one_curved_candidate_uses_the_transient_theme_color_and_clears() {
    let (runtime, mut view) = super::sketch_display::open_display("Curve candidate");
    let mut packet = parallel_lines(0.);
    packet.geometry.add_defining_point(Id(7), [0., 2.]).unwrap();
    packet.geometry.add_circle(Id(8), Id(7), 2.).unwrap();
    view.set_sketch_display(&[packet]).unwrap();
    view.set_sketch_palette(&candidate_palette()).unwrap();
    let baseline = view.redraw().unwrap();
    assert!(!green_near(&view, &baseline, [0., 4.]));
    let hover = ViewportSketchHover {
        sketch: DocumentSketchId(1),
        entities: vec![Id(8)],
    };
    assert!(view.set_sketch_hover(Some(&hover)).unwrap());
    let highlighted = view.redraw().unwrap();
    assert!(green_near(&view, &highlighted, [0., 4.]));
    assert!(!view.set_sketch_hover(Some(&hover)).unwrap());
    assert!(view.set_sketch_hover(None).unwrap());
    assert_eq!(view.redraw().unwrap().rgba, baseline.rgba);
    super::sketch_display::close_display(&runtime, &mut view);
}

#[test]
fn derived_midpoint_feedback_is_transient_and_clears_on_source_change() {
    let (runtime, mut view) = super::sketch_display::open_display("Midpoint candidate");
    let packet = parallel_lines(0.);
    view.set_sketch_display(&[packet]).unwrap();
    view.set_sketch_palette(&candidate_palette()).unwrap();
    let baseline = view.redraw().unwrap();
    assert!(!green_near(&view, &baseline, [0., 1.]));
    let hover = ViewportSketchFeedback::Point {
        sketch: DocumentSketchId(1),
        position: [0., 1.],
    };
    assert!(view.set_sketch_feedback(Some(&hover)).unwrap());
    let highlighted = view.redraw().unwrap();
    assert!(green_near(&view, &highlighted, [0., 1.]));
    assert!(!view.set_sketch_feedback(Some(&hover)).unwrap());
    let invalid = ViewportSketchFeedback::Point {
        sketch: DocumentSketchId(1),
        position: [f64::NAN, 1.],
    };
    assert!(view.set_sketch_feedback(Some(&invalid)).is_err());
    assert!(!view.set_sketch_feedback(Some(&hover)).unwrap());
    view.set_sketch_display(&[parallel_lines(1.)]).unwrap();
    assert!(!view.set_sketch_feedback(None).unwrap());
    let changed = view.redraw().unwrap();
    assert!(!green_near(&view, &changed, [0., 1.]));
    super::sketch_display::close_display(&runtime, &mut view);
}
