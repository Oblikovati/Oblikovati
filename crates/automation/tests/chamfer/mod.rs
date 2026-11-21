// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::featureargs::GeomEdgeSel;

#[test]
fn unsupported_chamfer_options_are_refused() {
    let cases = [
        Chamfer {
            chamfer_type: "twoDistances".into(),
            ..Default::default()
        },
        Chamfer {
            concave_strategy: "inward".into(),
            ..Default::default()
        },
        Chamfer {
            distance2: "1 mm".into(),
            ..Default::default()
        },
        Chamfer {
            angle: "45 deg".into(),
            ..Default::default()
        },
        Chamfer {
            reference_face: "face".into(),
            ..Default::default()
        },
        Chamfer {
            partial_start: "1 mm".into(),
            ..Default::default()
        },
        Chamfer {
            partial_length: "1 mm".into(),
            ..Default::default()
        },
        Chamfer {
            edges_geom: Some(vec![GeomEdgeSel::default()]),
            ..Default::default()
        },
    ];
    for input in cases {
        assert!(validate_options(&input).is_err(), "{input:?}");
    }
    assert!(
        validate_options(&Chamfer {
            chamfer_type: "distance".into(),
            concave_strategy: "outward".into(),
            ..Default::default()
        })
        .is_ok()
    );
}

#[test]
fn edge_selection_requires_nonempty_valid_hex_keys() {
    assert!(parse_edges(&Chamfer::default()).is_err());
    assert!(
        parse_edges(&Chamfer {
            edge_refs: Some(vec!["xx".into()]),
            ..Default::default()
        })
        .is_err()
    );
    assert_eq!(
        parse_edges(&Chamfer {
            edge_refs: Some(vec!["0102".into()]),
            ..Default::default()
        })
        .unwrap(),
        vec![vec![1, 2]]
    );
}
