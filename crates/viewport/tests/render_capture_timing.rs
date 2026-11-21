// SPDX-License-Identifier: GPL-2.0-only
//! Split native sketch redraw and readback timings for the frame-budget investigation (#4031).
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
};
use std::time::{Duration, Instant};

#[test]
#[ignore = "measure native sketch redraw and capture on the selected GPU"]
fn measure_native_sketch_redraw_and_capture() {
    let runtime = ViewportRuntime::new(ViewportOptions {
        validation: false,
        physical_device: None,
    })
    .unwrap();
    let mut view = runtime
        .open_window("Sketch redraw timing", [1440, 900])
        .unwrap();
    view.poll_events(64).unwrap();
    assert_eq!(view.drawable_extent().unwrap(), [1440, 900]);
    view.set_sketch_display(&[sketch_display()]).unwrap();
    eprintln!(
        "device={} extent=1440x900 lines=200 warmup=20 samples=120",
        runtime.device_name()
    );
    let samples = measure_frames(&mut view);
    report_samples(&samples);
    view.close();
}
fn sketch_display() -> crate::ViewportSketchDisplay {
    let mut geometry = Sketch2D::default();
    for index in 0..200_u64 {
        let a = SketchEntityId(index * 2 + 1);
        let b = SketchEntityId(index * 2 + 2);
        let x = index as f64 * 0.04 - 4.;
        geometry.add_point(a, [x, -2.]).unwrap();
        geometry.add_point(b, [x, 2.]).unwrap();
        geometry
            .add_line(SketchEntityId(index + 1001), [a, b])
            .unwrap();
    }
    crate::ViewportSketchDisplay {
        text_declines: Vec::new(),
        working_scale: 1.,
        sketch: DocumentSketchId(1),
        plane: SketchPlane::xy(),
        geometry,
        blocks: vec![],
        curvature: Vec::new(),
        line_color: None,
    }
}
fn measure_frames(view: &mut Viewport) -> Vec<(Duration, Duration)> {
    let mut samples = Vec::with_capacity(120);
    for index in 0..140 {
        let start = Instant::now();
        assert!(view.draw_current_surface().unwrap());
        let draw = start.elapsed();
        let start = Instant::now();
        let ViewportPresentation::Completed(frame) = view.capture_completed_surface().unwrap()
        else {
            panic!("capture: expected a completed frame");
        };
        std::hint::black_box(&frame);
        drop(frame);
        let capture = start.elapsed();
        if index >= 20 {
            samples.push((draw, capture));
        }
    }
    samples
}
fn report_samples(samples: &[(Duration, Duration)]) {
    for (index, (draw, capture)) in samples.iter().enumerate() {
        eprintln!(
            "sample={index} draw_ns={} capture_ns={}",
            draw.as_nanos(),
            capture.as_nanos()
        );
    }
    for (name, mut values) in [
        (
            "draw",
            samples.iter().map(|sample| sample.0).collect::<Vec<_>>(),
        ),
        (
            "capture",
            samples.iter().map(|sample| sample.1).collect::<Vec<_>>(),
        ),
    ] {
        values.sort_unstable();
        eprintln!(
            "{name} p50_ms={:.3} p95_ms={:.3} max_ms={:.3}",
            values[60].as_secs_f64() * 1000.,
            values[113].as_secs_f64() * 1000.,
            values[119].as_secs_f64() * 1000.
        );
    }
}
