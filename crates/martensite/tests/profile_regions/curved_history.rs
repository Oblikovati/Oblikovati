// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn verify_curved_lineage(case: &str, reports: &[RegionReport]) {
    let mut actual: Vec<Vec<usize>> = reports
        .iter()
        .map(|report| report.original_region_parents.clone())
        .collect();
    let mut expected = expected_region_parents(case);
    actual.sort();
    expected.sort();
    assert_eq!(
        actual, expected,
        "case {case}: expected exact fixture source sets"
    );
}

fn expected_region_parents(case: &str) -> Vec<Vec<usize>> {
    // Source indices are fixture inputs, never persistent keys. Distinct regions
    // can share every parent; the repeated sets intentionally test that limit.
    match case {
        "circle-boundary" | "ellipse-boundary" => vec![vec![1, 3], vec![1, 3], vec![1, 2, 3, 4, 5]],
        "circle-divider" => vec![
            vec![1, 6],
            vec![1, 6],
            vec![1, 2, 4, 5, 6],
            vec![1, 2, 3, 4, 6],
        ],
        "overlap-circles" => vec![vec![1, 2], vec![1, 2], vec![1, 2], vec![1, 2, 3, 4, 5, 6]],
        "circle-tangent" => vec![vec![1], vec![2, 3, 4, 5]],
        "pair-separated" | "pair-tangent" => vec![vec![1], vec![2]],
        "pair-near-overlap" => vec![vec![1, 2], vec![1, 2], vec![1, 2]],
        _ => panic!("case {case}: expected a curved-intersection fixture"),
    }
}
