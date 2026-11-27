// SPDX-License-Identifier: GPL-2.0-only
//! Synthetic release probe; this does not measure a GUI frame.
use oblikovati_solve::affine_preference::{
    AffineLimits, AffinePreferenceInput, certify_affine_preference,
};
use std::time::Instant;

const LIMITS: AffineLimits = AffineLimits {
    max_order: 128,
    max_bits: 4096,
};
const SAMPLES: usize = 9;

struct ProbeCase {
    rows: Vec<Vec<f64>>,
    rhs: Vec<f64>,
    targets: Vec<(usize, f64)>,
}

fn main() {
    for n in [8, 16, 32, 64] {
        measure("chain", n, &chain(n));
    }
    measure("dense", 32, &dense_case(32));
}

fn chain(n: usize) -> ProbeCase {
    let mut rows = Vec::new();
    for i in 0..n - 1 {
        let mut row = vec![0.; n];
        row[i] = 1.;
        row[i + 1] = -1.;
        rows.push(row);
    }
    ProbeCase {
        rows,
        rhs: vec![0.; n - 1],
        targets: vec![(0, 1.)],
    }
}

fn dense_case(n: usize) -> ProbeCase {
    let mut state = 7_u64;
    let mut rows = Vec::new();
    for _ in 0..n / 2 {
        let mut row = Vec::new();
        for _ in 0..n {
            state = state.wrapping_mul(6364136223846793005).wrapping_add(1);
            row.push(((state >> 32) % 5) as f64 - 2.);
        }
        rows.push(row);
    }
    let targets = (0..n).map(|i| (i, (i % 5) as f64 - 2.)).collect();
    ProbeCase {
        rows,
        rhs: vec![0.; n / 2],
        targets,
    }
}

fn measure(name: &str, n: usize, case: &ProbeCase) {
    let seed = vec![0.; n];
    let input = AffinePreferenceInput {
        equations: &case.rows,
        right_hand_side: &case.rhs,
        initial: &seed,
        targets: &case.targets,
    };
    let mut times = samples(input, n);
    times.sort_by(f64::total_cmp);
    println!(
        "{name} variables={n} equations={} median_ms={:.3} max_ms={:.3}",
        case.rows.len(),
        times[SAMPLES / 2],
        times[SAMPLES - 1]
    );
}

fn samples(input: AffinePreferenceInput<'_>, count: usize) -> Vec<f64> {
    let mut times = Vec::new();
    for _ in 0..SAMPLES {
        let start = Instant::now();
        let result = certify_affine_preference(input, LIMITS, || false).unwrap();
        times.push(start.elapsed().as_secs_f64() * 1000.);
        assert_eq!(result.coordinates().len(), count);
        std::hint::black_box(result);
    }
    times
}
