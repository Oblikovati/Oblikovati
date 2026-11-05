// SPDX-License-Identifier: GPL-2.0-only
//! Crossing-curve evidence; the sampled diagnostic classifier is not production.
use super::*;
use std::collections::BTreeSet;

const CURVED_CASES: [(&str, usize); 8] = [
    ("circle-boundary", 3),
    ("ellipse-boundary", 3),
    ("circle-divider", 4),
    ("overlap-circles", 4),
    ("circle-tangent", 2),
    ("pair-separated", 2),
    ("pair-tangent", 2),
    ("pair-near-overlap", 3),
];

#[test]
fn curved_intersections_match_every_occt_face() {
    for (case, count) in CURVED_CASES {
        let source = fixture::build_region_fixture(case).unwrap();
        let arrangement = arrange_region_fixture(&source).unwrap();
        history::require_region_history_coverage(&arrangement).unwrap();
        let builder = build_fixture_regions(&arrangement).unwrap();
        let reports = bounded_region_reports(&builder, &arrangement).unwrap();
        assert_eq!(reports.len(), count, "case {case}");
        verify_curved_case(case, &reports);
        curved_history::verify_curved_lineage(case, &reports);
        curved_prism::verify_curved_prisms(case, &builder);
    }
}

fn verify_curved_case(case: &str, reports: &[RegionReport]) {
    let rows = curved_oracle_rows(case);
    assert_eq!(reports.len(), rows.len(), "case {case}");
    let mut used = BTreeSet::new();
    for report in reports {
        let index = unique_region_row(case, report, &rows);
        assert!(used.insert(index), "case {case}: duplicate region match");
        verify_occt_region(report, &rows[index]);
    }
}

fn unique_region_row(case: &str, report: &RegionReport, rows: &[Vec<f64>]) -> usize {
    let candidates: Vec<usize> = rows
        .iter()
        .enumerate()
        .filter_map(|(index, row)| same_oracle_region(report, row).then_some(index))
        .collect();
    assert_eq!(
        candidates.len(),
        1,
        "case {case}: {report:?}, oracle {rows:?}"
    );
    candidates[0]
}

fn same_oracle_region(report: &RegionReport, row: &[f64]) -> bool {
    // Independent integrators can reverse the order of mathematically equal areas.
    // Matching is diagnostic only and must be unique; it never assigns source identity.
    (report.area - row[0]).abs() < 1e-8
        && report
            .centroid
            .iter()
            .zip(&row[1..4])
            .all(|(actual, expected)| (actual - expected).abs() < 1e-8)
}

fn curved_oracle_rows(case: &str) -> Vec<Vec<f64>> {
    include_str!("occt-8.0.0-curved.csv")
        .lines()
        .filter_map(|line| line.strip_prefix(&format!("{case},")))
        .map(|line| {
            line.split(',')
                .map(|value| value.parse().unwrap())
                .collect()
        })
        .collect()
}
