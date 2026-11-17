// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchPlane;
use std::cell::Cell;

#[derive(Default)]
struct ProjectionWindow {
    batches: Cell<usize>,
    single: Cell<usize>,
    fail_batch: bool,
    short_batch: bool,
}
impl HeadWindow for ProjectionWindow {
    fn sketch_pixel(&self, _: SketchPlane, point: [f64; 2]) -> Pixel {
        self.single.set(self.single.get() + 1);
        if point[0] < 0. {
            return Err(format!("point {point:?}: expected nonnegative x"));
        }
        Ok([point[0] as i32, point[1] as i32])
    }
    fn sketch_pixel_results(
        &self,
        _: SketchPlane,
        points: &[[f64; 2]],
    ) -> Result<Vec<Pixel>, String> {
        self.batches.set(self.batches.get() + 1);
        if self.fail_batch {
            return Err("batch: expected a valid extent".into());
        }
        if self.short_batch {
            return Ok(Vec::new());
        }
        Ok(points
            .iter()
            .map(|point| {
                if point[0] < 0. {
                    Err(format!("point {point:?}: expected nonnegative x"))
                } else {
                    Ok([point[0] as i32, point[1] as i32])
                }
            })
            .collect())
    }
    fn events(&mut self, _: usize) -> Result<Vec<oblikovati_viewport::ViewportEvent>, String> {
        Ok(Vec::new())
    }
    fn present(&mut self) -> Result<crate::HeadPresentation, String> {
        Ok(crate::HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<std::time::Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
fn annotation(
    plane: SketchPlane,
    position: [f64; 2],
    line: Option<[[f64; 2]; 2]>,
) -> HeadDimensionAnnotation {
    HeadDimensionAnnotation {
        plane,
        position,
        line,
        text: "distance".into(),
        role: oblikovati_gui::GuiDimensionLabelRole::Driving,
        edit: None,
    }
}
#[test]
fn same_plane_annotations_use_one_batch_and_preserve_order() {
    let window = ProjectionWindow::default();
    let values = [
        annotation(SketchPlane::xy(), [1., 2.], Some([[3., 4.], [5., 6.]])),
        annotation(SketchPlane::xy(), [7., 8.], None),
        annotation(SketchPlane::xz(), [9., 10.], None),
    ];
    let output = project(&window, &values);
    assert_eq!(window.batches.get(), 2);
    assert_eq!(window.single.get(), 0);
    assert_eq!(output[0].position, Ok([1, 2]));
    assert_eq!(output[0].line, Some(Ok([[3, 4], [5, 6]])));
    assert_eq!(output[1].position, Ok([7, 8]));
    assert_eq!(output[2].position, Ok([9, 10]));
}
#[test]
fn batch_preserves_individual_errors_and_other_points() {
    let window = ProjectionWindow::default();
    let values = [
        annotation(SketchPlane::xy(), [1., 2.], Some([[-1., 4.], [5., 6.]])),
        annotation(SketchPlane::xy(), [7., 8.], None),
    ];
    let output = project(&window, &values);
    assert_eq!(window.batches.get(), 1);
    assert_eq!(window.single.get(), 0);
    assert_eq!(output[0].position, Ok([1, 2]));
    assert!(output[0].line.as_ref().unwrap().is_err());
    assert_eq!(output[1].position, Ok([7, 8]));
}

#[test]
fn unavailable_or_incomplete_batch_reports_projection_failure() {
    for window in [
        ProjectionWindow {
            fail_batch: true,
            ..Default::default()
        },
        ProjectionWindow {
            short_batch: true,
            ..Default::default()
        },
    ] {
        let values = [
            annotation(SketchPlane::xy(), [1., 2.], None),
            annotation(SketchPlane::xy(), [-1., 2.], None),
        ];
        let output = project(&window, &values);
        assert_eq!(window.batches.get(), 1);
        assert_eq!(window.single.get(), 0);
        assert!(output[0].position.is_err());
        assert!(output[1].position.is_err());
    }
}

#[test]
#[ignore = "measure native dimension projection batches in an optimized build"]
fn measure_dimension_projection_batches() {
    let mut window = crate::NativeHeadWindow::open_with_extent([1440, 900]).unwrap();
    let annotations: Vec<_> = (0..200)
        .map(|index| {
            let x = f64::from(index) * 0.01;
            annotation(SketchPlane::xy(), [x, 1.], Some([[x, 0.], [x, 2.]]))
        })
        .collect();
    let (single, batch) = measure_paths(&window, &annotations);
    report_timing("single", single);
    report_timing("batch", batch);
    window.finish().unwrap();
}
fn measure_paths(
    window: &dyn HeadWindow,
    annotations: &[HeadDimensionAnnotation],
) -> (Vec<std::time::Duration>, Vec<std::time::Duration>) {
    let mut single = Vec::new();
    let mut batch = Vec::new();
    for index in 0..50 {
        let started = std::time::Instant::now();
        let expected = individual_projection(window, annotations);
        let single_elapsed = started.elapsed();
        let started = std::time::Instant::now();
        let actual = project(window, annotations);
        let batch_elapsed = started.elapsed();
        for (actual, expected) in actual.iter().zip(expected) {
            assert_eq!(actual.position, expected.position);
            assert_eq!(actual.line, expected.line);
        }
        if index >= 10 {
            single.push(single_elapsed);
            batch.push(batch_elapsed);
        }
    }
    (single, batch)
}
fn individual_projection(
    window: &dyn HeadWindow,
    annotations: &[HeadDimensionAnnotation],
) -> Vec<AnnotationPixels> {
    annotations
        .iter()
        .map(|annotation| AnnotationPixels {
            position: window.sketch_pixel(annotation.plane, annotation.position),
            line: annotation.line.map(|[from, to]| {
                window
                    .sketch_pixel(annotation.plane, from)
                    .and_then(|first| {
                        window
                            .sketch_pixel(annotation.plane, to)
                            .map(|second| [first, second])
                    })
            }),
        })
        .collect()
}
fn report_timing(label: &str, mut samples: Vec<std::time::Duration>) {
    samples.sort_unstable();
    eprintln!(
        "dimension_projection {label} n={} p50={:?} p95={:?} max={:?}",
        samples.len(),
        samples[20],
        samples[37],
        samples[39]
    );
}
