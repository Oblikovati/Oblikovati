// SPDX-License-Identifier: GPL-2.0-only
//! Measures the host projection boundary used by analytic sketch picking (#4035).
use super::*;
use oblikovati_model::sketch::SketchPlane;

#[test]
#[ignore = "compare individual and cached glyph projections"]
fn measure_constraint_glyph_projection_reuse() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime
        .open_window("Glyph projection benchmark", [1440, 900])
        .unwrap();
    view.poll_events(64).unwrap();
    assert_eq!(view.drawable_extent().unwrap(), [1440, 900]);
    eprintln!(
        "glyph_projection device={} extent=1440x900",
        runtime.device_name()
    );
    let points: Vec<_> = (0..201).map(|i| [i as f64 * 0.04 - 4., 0.1]).collect();
    let mut single = Vec::new();
    let mut cached = Vec::new();
    for index in 0..140 {
        let start = Instant::now();
        let expected: Vec<_> = points
            .iter()
            .map(|point| view.sketch_pixel(SketchPlane::xy(), *point).unwrap())
            .collect();
        let single_time = start.elapsed();
        let start = Instant::now();
        let actual = view.sketch_pixels(SketchPlane::xy(), &points).unwrap();
        let cached_time = start.elapsed();
        assert_eq!(actual, expected);
        if index >= 20 {
            single.push(single_time);
            cached.push(cached_time);
        }
    }
    report_projection("glyph-individual", points.len(), &single);
    report_projection("glyph-cached", points.len(), &cached);
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

#[test]
#[ignore = "measure checked sketch projection in an optimized build"]
fn measure_sketch_projection_batches() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut offscreen = runtime.open_offscreen([1440, 900]).unwrap();
    let mut native = runtime
        .open_window("Projection benchmark", [1440, 900])
        .unwrap();
    native.poll_events(64).unwrap();
    assert_eq!(native.drawable_extent().unwrap(), [1440, 900]);
    assert_eq!(offscreen.drawable_extent().unwrap(), [1440, 900]);
    eprintln!(
        "projection device={}, extent=1440x900",
        runtime.device_name()
    );
    for count in [5_usize, 201, 1001] {
        measure_projection(&offscreen, "offscreen", count);
        measure_projection(&native, "native", count);
    }
    offscreen.close();
    native.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

fn measure_projection(view: &Viewport, target: &str, count: usize) {
    let points: Vec<_> = (0..count)
        .map(|i| [i as f64 * 0.04 - 4., (i % 2) as f64 * 0.1])
        .collect();
    let mut samples = Vec::with_capacity(120);
    for sample in 0..140 {
        let start = Instant::now();
        let projected = view.sketch_pixels(SketchPlane::xy(), &points).unwrap();
        std::hint::black_box(projected);
        if sample >= 20 {
            samples.push(start.elapsed());
        }
    }
    report_projection(target, count, &samples);
}

fn report_projection(target: &str, count: usize, samples: &[Duration]) {
    for (index, elapsed) in samples.iter().enumerate() {
        eprintln!(
            "sketch_projection_sample target={target} points={count} sample={index} elapsed_ns={}",
            elapsed.as_nanos()
        );
    }
    let mut sorted = samples.to_vec();
    sorted.sort_unstable();
    eprintln!(
        "sketch_projection target={target} points={count} samples={} p50_us={:.3} p95_us={:.3} max_us={:.3}",
        sorted.len(),
        sorted[60].as_secs_f64() * 1e6,
        sorted[113].as_secs_f64() * 1e6,
        sorted[119].as_secs_f64() * 1e6
    );
}

#[test]
#[ignore = "measure alternating fixed and changing projection batches"]
fn measure_alternating_sketch_projection_batches() {
    let runtime = ViewportRuntime::new(Default::default()).unwrap();
    let mut view = runtime
        .open_window("Alternating projection benchmark", [1440, 900])
        .unwrap();
    view.poll_events(64).unwrap();
    assert_eq!(view.drawable_extent().unwrap(), [1440, 900]);
    eprintln!(
        "projection device={}, extent=1440x900",
        runtime.device_name()
    );
    for count in [201_usize, 1001] {
        measure_alternating(&view, count);
    }
    view.close();
    assert!(runtime.validation_errors().unwrap().is_empty());
}

fn measure_alternating(view: &Viewport, count: usize) {
    let points: Vec<_> = (0..count)
        .map(|i| [i as f64 * 0.04 - 4., (i % 2) as f64 * 0.1])
        .collect();
    let mut curves = points.clone();
    let mut samples = Vec::with_capacity(120);
    for sample in 0..140 {
        for curve in &mut curves {
            curve[1] += 0.000_001;
        }
        let start = Instant::now();
        let first = view.sketch_pixels(SketchPlane::xy(), &points).unwrap();
        let second = view.sketch_pixels(SketchPlane::xy(), &curves).unwrap();
        std::hint::black_box((first, second));
        if sample >= 20 {
            samples.push(start.elapsed());
        }
    }
    report_projection("native-alternating", count, &samples);
}
