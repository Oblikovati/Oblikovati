// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_martensite::PrismProbeReport;
pub(super) fn oracle_definition(row: &[f64; 16]) -> PrismProbeDefinition {
    let mut definition = match row[0] as usize {
        0 => rectangle(),
        1 => with_hole(),
        2 => concave(),
        _ => panic!("expected known oracle profile kind"),
    };
    let mut next = 1_u64;
    for ring in std::iter::once(&mut definition.outer).chain(&mut definition.holes) {
        let PrismProbeLoop::Polygon {
            edge_keys,
            vertices,
        } = ring
        else {
            panic!("expected polygon oracle fixture");
        };
        *edge_keys = (next..next + vertices.len() as u64).collect();
        next += vertices.len() as u64;
        for point in vertices {
            point[0] *= row[1];
            point[1] *= row[1];
        }
    }
    definition.displacement = [row[2], row[3], row[4]];
    definition
}
fn concave() -> PrismProbeDefinition {
    PrismProbeDefinition {
        outer: PrismProbeLoop::polygon(
            vec![1, 2, 3, 4, 5, 6],
            vec![[0., 0.], [3., 0.], [3., 1.], [1., 1.], [1., 3.], [0., 3.]],
        ),
        holes: Vec::new(),
        displacement: [0., 0., 2.],
    }
}
fn expected_role(row: &[f64; 16]) -> PrismFaceRole {
    match row[5] as usize {
        0 => PrismFaceRole::Start,
        1 => PrismFaceRole::End,
        2 => PrismFaceRole::ProfileEdge(row[6] as u64),
        _ => panic!("expected known oracle face role"),
    }
}
pub(super) fn verify_face(row: &[f64; 16], report: &PrismProbeReport) {
    let role = expected_role(row);
    let matches: Vec<_> = report
        .faces
        .iter()
        .filter(|face| face.role == role)
        .collect();
    assert_eq!(matches.len(), 1, "{role:?}");
    let face = matches[0];
    let scale = row[1];
    compare(face.area, row[7], scale * scale);
    for (actual, expected) in face.centroid.into_iter().zip(&row[8..11]) {
        compare(actual, *expected, scale);
    }
    assert_eq!(face.loops, row[11] as usize);
    compare(report.properties.volume, row[12], scale * scale * scale);
    for (actual, expected) in report.properties.centroid.into_iter().zip(&row[13..]) {
        compare(actual, *expected, scale);
    }
}
fn compare(actual: f64, expected: f64, working_scale: f64) {
    let tolerance = expected.abs().max(working_scale) * 1e-11;
    assert!(
        (actual - expected).abs() <= tolerance,
        "actual={actual}, expected={expected}, tolerance={tolerance}"
    );
}
#[test]
fn every_generated_prism_face_matches_the_independent_occt_corpus() {
    let corpus = include_str!("occt-8.0.0.csv");
    assert_eq!(corpus.lines().count(), 216);
    let mut report = None;
    let mut cases = 0;
    for line in corpus.lines() {
        let row = parse_oracle_row(line);
        if row[5] == 0. {
            report = Some(probe_planar_prism(&oracle_definition(&row)).unwrap());
            cases += 1;
        }
        verify_face(
            &row,
            report
                .as_ref()
                .expect("expected a start-cap row before other faces"),
        );
    }
    assert_eq!(cases, 27);
}

pub(super) fn parse_oracle_row(line: &str) -> [f64; 16] {
    line.split(',')
        .map(|value| value.parse().unwrap())
        .collect::<Vec<_>>()
        .try_into()
        .expect("expected sixteen oracle columns")
}
