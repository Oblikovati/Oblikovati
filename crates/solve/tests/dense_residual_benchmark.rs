// SPDX-License-Identifier: GPL-2.0-only
//! Release-only diagnostic for equations that use every sketch coordinate.
use oblikovati_solve::{
    ad::DualNumber,
    residual::{ResidualBlock, ResidualError, ResidualSource, evaluate_residuals},
};
use std::time::{Duration, Instant};

struct Coordinate;
impl ResidualSource for Coordinate {
    fn evaluate(&self, variables: &[DualNumber]) -> Result<Vec<DualNumber>, ResidualError> {
        Ok(vec![variables[0].clone()])
    }
}

#[test]
#[ignore = "measure dense residual evaluation in release mode"]
fn measure_dense_residual_evaluation() {
    for lines in [200, 1000] {
        let values = vec![1.0; lines * 4];
        let indices: Vec<[usize; 1]> = (0..values.len()).map(|index| [index]).collect();
        let blocks: Vec<ResidualBlock<'_>> = indices
            .iter()
            .map(|index| ResidualBlock {
                source: &Coordinate,
                variables: index,
                equations: 1,
            })
            .collect();
        let mut samples = Vec::new();
        let measured = if lines == 200 { 50 } else { 10 };
        for iteration in 0..measured + 3 {
            let started = Instant::now();
            let evaluation = evaluate_residuals(&values, &blocks).unwrap();
            std::hint::black_box(evaluation);
            if iteration >= 3 {
                samples.push(started.elapsed());
            }
        }
        samples.sort();
        report(lines, &samples);
    }
}
fn report(lines: usize, samples: &[Duration]) {
    println!(
        "dense_residual lines={lines} n={} min_ms={:.3} p50_ms={:.3} max_ms={:.3}",
        samples.len(),
        samples[0].as_secs_f64() * 1000.0,
        samples[samples.len() / 2].as_secs_f64() * 1000.0,
        samples[samples.len() - 1].as_secs_f64() * 1000.0
    );
}
