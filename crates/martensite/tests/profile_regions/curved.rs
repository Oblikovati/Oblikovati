// SPDX-License-Identifier: GPL-2.0-only
//! Crossing-curve evidence; the sampled diagnostic classifier is not production.
use super::*;

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
    oracle_match::verify_rows(case, reports, &curved_oracle_rows(case));
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
