// SPDX-License-Identifier: GPL-2.0-only
//! Release-only diagnostic for sketches with many unrelated line coordinates.
use oblikovati_solve::{
    ad::DualNumber,
    residual::{ResidualBlock, ResidualError, ResidualSource, evaluate_residuals},
};
use std::time::{Duration, Instant};

struct Midpoint;
impl ResidualSource for Midpoint {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![
            variables[2].minus(&variables[0].plus(&variables[1])?.scaled(0.5)?)?,
        ])
    }
}

#[test]
#[ignore = "measure sparse residual evaluation in release mode"]
fn measure_sparse_residual_evaluation() {
    for lines in [200, 1000] {
        let values = vec![1.0; lines * 4 + 2];
        let block = ResidualBlock {
            source: &Midpoint,
            variables: &[0, 2, lines * 4],
            equations: 1,
        };
        let mut samples = Vec::new();
        for iteration in 0..60 {
            let started = Instant::now();
            let evaluation = evaluate_residuals(&values, std::slice::from_ref(&block)).unwrap();
            std::hint::black_box(evaluation);
            if iteration >= 10 {
                samples.push(started.elapsed());
            }
        }
        samples.sort();
        report(lines, &samples);
    }
}
fn report(lines: usize, samples: &[Duration]) {
    println!(
        "sparse_residual lines={lines} n={} p50_ms={:.3} p95_ms={:.3} max_ms={:.3}",
        samples.len(),
        samples[25].as_secs_f64() * 1000.0,
        samples[47].as_secs_f64() * 1000.0,
        samples[49].as_secs_f64() * 1000.0
    );
}
