// SPDX-License-Identifier: GPL-2.0-only
//! Phase timings for the analytic candidate path (#4035).
use oblikovati_martensite::planar_curve::{PlanarPickOptionsV1, sketch::SketchPickQuery};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
use std::time::{Duration, Instant};

const WARMUP: usize = 20;
const SAMPLES: usize = 120;

#[test]
#[ignore = "measure analytic picking in an optimized build"]
fn measure_analytic_pick_phases() {
    for count in [4_u64, 200, 1000] {
        let sketch = line_chain(count);
        let samples: Vec<_> = (0..WARMUP + SAMPLES)
            .map(|sample| measure_one(&sketch, sample))
            .skip(WARMUP)
            .collect();
        report(count, &samples);
    }
}

fn line_chain(count: u64) -> Sketch2D {
    let mut sketch = Sketch2D::default();
    for i in 0..=count {
        sketch
            .add_defining_point(
                SketchEntityId(i + 1),
                [i as f64 * 0.04 - 4., (i % 2) as f64 * 0.1],
            )
            .unwrap();
    }
    for i in 0..count {
        sketch
            .add_line(
                SketchEntityId(count + 2 + i),
                [SketchEntityId(i + 1), SketchEntityId(i + 2)],
            )
            .unwrap();
    }
    sketch
}

fn measure_one(sketch: &Sketch2D, sample: usize) -> [Duration; 3] {
    let cursor = [(sample % 50) as f64 * 0.04 - 1., 0.2];
    let start = Instant::now();
    let query = SketchPickQuery::new(sketch, cursor, PlanarPickOptionsV1::default()).unwrap();
    let catalogue = start.elapsed();
    let points = measure_candidates(&query, sketch.points().iter().map(|point| point.id));
    let curves = measure_candidates(&query, sketch.lines().iter().map(|line| line.id));
    [catalogue, points, curves]
}

fn measure_candidates(
    query: &SketchPickQuery<'_>,
    ids: impl Iterator<Item = SketchEntityId>,
) -> Duration {
    let start = Instant::now();
    for id in ids {
        std::hint::black_box(query.candidate(id).unwrap());
    }
    start.elapsed()
}

fn report(count: u64, samples: &[[Duration; 3]]) {
    for (index, sample) in samples.iter().enumerate() {
        eprintln!(
            "analytic_pick_sample lines={count} sample={index} catalogue_ns={} points_ns={} curves_ns={}",
            sample[0].as_nanos(),
            sample[1].as_nanos(),
            sample[2].as_nanos()
        );
    }
    for (phase, index) in [("catalogue", 0), ("points", 1), ("curves", 2)] {
        let mut values: Vec<_> = samples.iter().map(|sample| sample[index]).collect();
        values.sort_unstable();
        eprintln!(
            "analytic_pick lines={count} phase={phase} samples={} p50_us={:.3} p95_us={:.3} max_us={:.3}",
            values.len(),
            micros(values[60]),
            micros(values[113]),
            micros(values[119])
        );
    }
}

fn micros(value: Duration) -> f64 {
    value.as_secs_f64() * 1_000_000.
}
