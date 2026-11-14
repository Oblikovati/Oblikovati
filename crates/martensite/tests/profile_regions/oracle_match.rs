// SPDX-License-Identifier: GPL-2.0-only
//! Match every independent oracle face uniquely, without ordering by rounded measurements.
use super::*;
use std::collections::BTreeSet;
pub(super) fn verify_rows(case: &str, reports: &[RegionReport], rows: &[Vec<f64>]) {
    assert!(
        !rows.is_empty(),
        "{case}: expected retained independent OCCT evidence"
    );
    assert_eq!(
        reports.len(),
        rows.len(),
        "{case}: expected complete bounded-region coverage"
    );
    let mut used = BTreeSet::new();
    for report in reports {
        let index = unique_region_row(case, report, rows);
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

#[cfg(test)]
mod tests {
    use super::*;
    fn receipt(center: f64, area: f64) -> RegionReport {
        RegionReport {
            area,
            centroid: [center, 1., 0.],
            loops: 1,
            edges: 3,
            original_region_parents: vec![1, 2],
        }
    }
    fn expected() -> Vec<Vec<f64>> {
        vec![
            vec![6., 4. / 3., 1., 0., 1., 3.],
            vec![6., 8. / 3., 1., 0., 1., 3.],
        ]
    }
    #[test]
    fn equal_area_roundoff_cannot_swap_the_expected_centroids() {
        verify_rows(
            "crossing",
            &[receipt(8. / 3., 6. - 1e-14), receipt(4. / 3., 6. + 1e-14)],
            &expected(),
        );
    }
    #[test]
    #[should_panic(expected = "duplicate region match")]
    fn duplicate_faces_cannot_consume_one_oracle_row_twice() {
        verify_rows(
            "crossing",
            &[receipt(4. / 3., 6.), receipt(4. / 3., 6.)],
            &expected(),
        );
    }
}
