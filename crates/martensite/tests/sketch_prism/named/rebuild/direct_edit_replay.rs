// SPDX-License-Identifier: GPL-2.0-only
//! Direct Edit Martensite replay tests: Uniform/NonUniform scale, move, rotate, recipe v52/v62 (#3918).
use super::*;
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::feature::{
    FeatureEvaluationStatus,
    direct_edit::{
        DirectEditCoordinateMode, DirectEditDefinition, DirectEditMode, DirectEditOperation,
        DirectEditOperationKind, DirectEditRecipe, DirectEditScaleMode, DirectEditSelectionFilter,
    },
};

fn add_direct_edit(history: &mut DocumentEditHistory, ops: Vec<DirectEditOperation>) -> FeatureId {
    let definition = DirectEditDefinition::from_recipe(DirectEditRecipe {
        version: 1,
        body: key("body", &[1]),
        operations: ops,
    })
    .unwrap();
    let token = history.begin("Direct Edit").unwrap();
    let id = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_direct_edit("Direct Edit1", definition, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    id
}

#[test]
fn uniform_scale_two_scales_volume_by_eight_and_preserves_lineage() {
    let mut history = owner(cube_document());
    let before_vol = {
        let mut backend =
            CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
        let eval = history
            .snapshot()
            .evaluate_part_features(&mut backend)
            .unwrap();
        eval.body(FeatureId(1))
            .unwrap()
            .volume(&history, document())
            .unwrap()
    };
    let id = add_direct_edit(
        &mut history,
        vec![DirectEditOperation {
            name: "Scale1".into(),
            kind: DirectEditOperationKind::Scale {
                mode: DirectEditScaleMode::Uniform,
                uniform: 2.0,
                factors: [2.0, 2.0, 2.0],
            },
        }],
    );
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    assert!(
        result
            .health()
            .iter()
            .all(|record| record.status == FeatureEvaluationStatus::Healthy),
        "health: {:?}",
        result.health()
    );
    let body = result.body(id).expect("expected Direct Edit scaled body");
    let after_vol = body.volume(&history, document()).unwrap();
    assert!(
        (after_vol - before_vol * 8.0).abs() < 1e-6,
        "expected volume {}, got {after_vol}",
        before_vol * 8.0
    );
    assert_eq!(
        body.body_reference_key(&history, document()).unwrap(),
        key("body", &[1])
    );

    let recipe = history.snapshot().work_geometry.part_recipe().unwrap();
    assert_eq!(recipe.version, 62);
    assert_eq!(recipe.features[1].definition.version, 52);

    let restored = restored_owner(history.snapshot());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&restored, document(), certified_resolution());
    let replayed = restored
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let replayed_vol = replayed
        .body(id)
        .unwrap()
        .volume(&restored, document())
        .unwrap();
    assert!((replayed_vol - before_vol * 8.0).abs() < 1e-6);
}

#[test]
fn non_uniform_scale_applies_independent_axis_factors_and_identity() {
    let mut history = owner(cube_document());
    let id = add_direct_edit(
        &mut history,
        vec![DirectEditOperation {
            name: "Scale1".into(),
            kind: DirectEditOperationKind::Scale {
                mode: DirectEditScaleMode::NonUniform,
                uniform: 2.0,
                factors: [2.0, 1.5, 1.0],
            },
        }],
    );
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let vol = result
        .body(id)
        .unwrap()
        .volume(&history, document())
        .unwrap();
    assert!(
        (vol - 3.0).abs() < 1e-5,
        "expected volume 3.0 for [2, 1.5, 1] scale on unit cube, got {vol}"
    );

    // Edit Scale1 to NonUniform [1, 1, 1] (identity) and confirm exact unit volume.
    let edited = DirectEditDefinition::new(
        key("body", &[1]),
        vec![DirectEditOperation {
            name: "Scale1".into(),
            kind: DirectEditOperationKind::Scale {
                mode: DirectEditScaleMode::NonUniform,
                uniform: 1.0,
                factors: [1.0, 1.0, 1.0],
            },
        }],
    )
    .unwrap();
    let token = history.begin("Edit Direct Edit").unwrap();
    history
        .edit(&token, |snapshot, _| snapshot.edit_direct_edit(id, edited))
        .unwrap();
    history.commit(&token, false).unwrap();

    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let after = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let unit_vol = after
        .body(id)
        .unwrap()
        .volume(&history, document())
        .unwrap();
    assert!(
        (unit_vol - 1.0).abs() < 1e-6,
        "expected unit volume after [1,1,1] edit, got {unit_vol}"
    );
}

#[test]
fn move_and_rotate_sub_operations_preserve_volume_and_shift_centroid() {
    let mut history = owner(cube_document());
    let id = add_direct_edit(
        &mut history,
        vec![
            DirectEditOperation {
                name: DirectEditDefinition::next_operation_name(&[], DirectEditMode::Move),
                kind: DirectEditOperationKind::Move {
                    filter: DirectEditSelectionFilter::Solids,
                    faces: Vec::new(),
                    translation: [3.0, -2.0, 1.0],
                },
            },
            DirectEditOperation {
                name: "Rotate1".into(),
                kind: DirectEditOperationKind::Rotate {
                    filter: DirectEditSelectionFilter::Solids,
                    faces: Vec::new(),
                    coordinate: DirectEditCoordinateMode::World,
                    origin: [3.5, -1.5, 1.5],
                    axis: [0.0, 0.0, 1.0],
                    angle_deg: 90.0,
                },
            },
        ],
    );
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let result = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let body = result.body(id).expect("expected transformed body");
    assert!((body.volume(&history, document()).unwrap() - 1.0).abs() < 1e-6);
    let center = body
        .body_bounds_center(&history, document(), &key("body", &[1]))
        .unwrap();
    assert!((center[0] - 3.5).abs() < 1e-5 && (center[1] + 1.5).abs() < 1e-5);
}
