// SPDX-License-Identifier: GPL-2.0-only
//! Model-only diagnostics; these timings do not establish desktop frame rate.
use super::*;
use std::time::{Duration, Instant};

#[test]
#[ignore = "measure release-mode transient inference latency"]
fn measure_transient_inference_cost() {
    let mode = std::env::var("OBK_TRANSIENT_BENCH_MODE").unwrap_or_else(|_| "both".into());
    assert!(["both", "manual", "transient"].contains(&mode.as_str()));
    for count in [200_u64, 1000] {
        let (source, candidate, point, line) = fixture(count);
        let parameters = ParameterCollection::new(Default::default());
        if mode == "both" {
            let original = manual_guidance(&source, &candidate, point, line, &parameters);
            let revised = transient_guidance(&source, &candidate, point, line, &parameters);
            assert!(original.0.same_recipe(&revised.0));
            assert_eq!(original.1, revised.1);
        }
        let mut baseline = Vec::new();
        let mut proposed = Vec::new();
        for iteration in 0..60 {
            if mode != "transient" {
                let start = Instant::now();
                std::hint::black_box(manual_guidance(
                    &source,
                    &candidate,
                    point,
                    line,
                    &parameters,
                ));
                if iteration >= 10 {
                    baseline.push(start.elapsed());
                }
            }
            if mode != "manual" {
                let start = Instant::now();
                std::hint::black_box(transient_guidance(
                    &source,
                    &candidate,
                    point,
                    line,
                    &parameters,
                ));
                if iteration >= 10 {
                    proposed.push(start.elapsed());
                }
            }
        }
        if !baseline.is_empty() {
            report(count, "manual", &mut baseline);
        }
        if !proposed.is_empty() {
            report(count, "transient", &mut proposed);
        }
    }
}
fn manual_guidance(
    source: &Sketch2D,
    candidate: &Sketch2D,
    point: SketchEntityId,
    line: SketchEntityId,
    parameters: &ParameterCollection<u64>,
) -> (Sketch2D, oblikovati_solve::diagnostics::ConstraintStructure) {
    use oblikovati_model::sketch::SketchConstraintId;
    let mut guided = candidate.clone();
    guided
        .add_constraint(
            SketchConstraintId(u64::MAX),
            PlanarConstraint::Midpoint { point, line },
        )
        .unwrap();
    let result = guided
        .solve_with_grounded_points(parameters, source.points(), SolveOptions::default(), || {
            false
        })
        .unwrap();
    let mut solved = result.into_candidate();
    solved
        .remove_geometric_constraint(SketchConstraintId(u64::MAX))
        .unwrap();
    let structure = solved
        .analyze(parameters, SolveOptions::default().rank_tolerance)
        .unwrap();
    (solved, structure)
}
fn transient_guidance(
    source: &Sketch2D,
    candidate: &Sketch2D,
    point: SketchEntityId,
    line: SketchEntityId,
    parameters: &ParameterCollection<u64>,
) -> (Sketch2D, oblikovati_solve::diagnostics::ConstraintStructure) {
    let source_state = SketchInferenceFreeze::capture(source).unwrap();
    let relation = PlanarConstraint::Midpoint { point, line };
    let context = SketchTransientInference {
        source: &source_state,
        durable_anchors: &[],
        temporary_anchors: &[],
        temporary_relations: &[relation],
    };
    let result = candidate
        .solve_transient_inference(parameters, &context, SolveOptions::default(), || false)
        .unwrap();
    (
        result.candidate().clone(),
        result.durable_structure().clone(),
    )
}
fn fixture(count: u64) -> (Sketch2D, Sketch2D, SketchEntityId, SketchEntityId) {
    let mut source = Sketch2D::default();
    for index in 0..count {
        let a = SketchEntityId(3 * index + 1);
        let b = SketchEntityId(3 * index + 2);
        source.add_point(a, [index as f64 * 3.0, 0.0]).unwrap();
        source
            .add_point(b, [index as f64 * 3.0 + 2.0, 0.0])
            .unwrap();
        source
            .add_line(SketchEntityId(3 * index + 3), [a, b])
            .unwrap();
    }
    let mut candidate = source.clone();
    let point = SketchEntityId(3 * count + 1);
    candidate.add_point(point, [1.0, 0.1]).unwrap();
    (source, candidate, point, SketchEntityId(3))
}
fn report(count: u64, method: &str, samples: &mut [Duration]) {
    samples.sort_unstable();
    println!(
        "transient lines={count} method={method} n={} p50_ms={:.3} p95_ms={:.3} max_ms={:.3}",
        samples.len(),
        ms(samples[25]),
        ms(samples[47]),
        ms(samples[49])
    );
}
fn ms(duration: Duration) -> f64 {
    duration.as_secs_f64() * 1000.0
}
