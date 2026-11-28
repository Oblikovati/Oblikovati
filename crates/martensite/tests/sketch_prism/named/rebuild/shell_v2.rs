// SPDX-License-Identifier: GPL-2.0-only
//! Verify v2 Shell directions (Inside, Outside, Both), per-face thickness overrides, and Rounded join type.
use super::*;
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::feature::{
    FeatureEvaluationStatus,
    shell::{ShellDefinition, ShellDirection, ShellFaceThickness, ShellOptions, ShellType},
};

#[test]
fn shell_v2_directions_overrides_and_rounded_type() {
    let history = owner(cube_document());
    let mut backend =
        CertifiedNamedSolidEvaluator::new(&history, document(), certified_resolution());
    let base = history
        .snapshot()
        .evaluate_part_features(&mut backend)
        .unwrap();
    let solid = base.body(FeatureId(1)).unwrap();
    let base_vol = solid.volume(&history, document()).unwrap();
    let faces = solid.face_reference_keys(&history, document()).unwrap();
    assert_eq!(faces.len(), 6);
    let top = faces[0].clone();
    let side = faces[2].clone();

    // 1) Inside 0.1 cm
    let inside_def = ShellDefinition::new(
        key("body", &[1]),
        vec![top.clone()],
        DatumLength::Constant(0.1),
        ShellDirection::Inside,
        Vec::new(),
    )
    .unwrap();
    let mut h1 = owner(cube_document());
    let tok = h1.begin("ShellIn").unwrap();
    let id1 = h1.create_shell(&tok, "Shell1", inside_def).unwrap();
    h1.commit(&tok, false).unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::new(&h1, document(), certified_resolution());
    let r1 = h1.snapshot().evaluate_part_features(&mut backend).unwrap();
    assert!(
        r1.health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy)
    );
    let v_in = r1.body(id1).unwrap().volume(&h1, document()).unwrap();
    assert!(
        v_in > 0.0 && v_in < base_vol,
        "v_in={v_in}, base={base_vol}"
    );

    // 2) Both 0.2 cm with 0.1 cm side-face override
    let both_def = ShellDefinition::new_with_options(
        key("body", &[1]),
        vec![top.clone()],
        DatumLength::Constant(0.2),
        ShellDirection::Both,
        vec![ShellFaceThickness {
            face: side,
            thickness: DatumLength::Constant(0.1),
        }],
        ShellOptions::default(),
    )
    .unwrap();
    let mut h2 = owner(cube_document());
    let tok = h2.begin("ShellBoth").unwrap();
    let id2 = h2.create_shell(&tok, "Shell1", both_def).unwrap();
    h2.commit(&tok, false).unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::new(&h2, document(), certified_resolution());
    let r2 = h2.snapshot().evaluate_part_features(&mut backend).unwrap();
    assert!(
        r2.health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy),
        "Both shell health: {:?}",
        r2.health()
    );
    let v_both = r2.body(id2).unwrap().volume(&h2, document()).unwrap();
    assert!(v_both > 0.0, "v_both={v_both}");

    // 3) Rounded Shell Type (Outside 0.15 cm)
    let rounded_def = ShellDefinition::new_with_options(
        key("body", &[1]),
        vec![top],
        DatumLength::Constant(0.15),
        ShellDirection::Outside,
        Vec::new(),
        ShellOptions {
            shell_type: ShellType::Rounded,
            ..Default::default()
        },
    )
    .unwrap();
    let mut h3 = owner(cube_document());
    let tok = h3.begin("ShellRounded").unwrap();
    let id3 = h3.create_shell(&tok, "Shell1", rounded_def).unwrap();
    h3.commit(&tok, false).unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::new(&h3, document(), certified_resolution());
    let r3 = h3.snapshot().evaluate_part_features(&mut backend).unwrap();
    assert!(
        r3.health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy),
        "Rounded shell health: {:?}",
        r3.health()
    );
    let v_round = r3.body(id3).unwrap().volume(&h3, document()).unwrap();
    assert!(v_round > 0.0, "v_round={v_round}");

    // 4) Compare Sharp vs Rounded on Both and Outside, and verify Inside+Rounded succeeds
    let out_sharp_def = ShellDefinition::new_with_options(
        key("body", &[1]),
        vec![faces[0].clone()],
        DatumLength::Constant(0.15),
        ShellDirection::Outside,
        Vec::new(),
        ShellOptions::default(),
    )
    .unwrap();
    let mut h4 = owner(cube_document());
    let tok = h4.begin("ShellOutSharp").unwrap();
    let id4 = h4.create_shell(&tok, "Shell1", out_sharp_def).unwrap();
    h4.commit(&tok, false).unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::new(&h4, document(), certified_resolution());
    let r4 = h4.snapshot().evaluate_part_features(&mut backend).unwrap();
    let v_out_sharp = r4.body(id4).unwrap().volume(&h4, document()).unwrap();
    assert!(
        (v_out_sharp - v_round).abs() > 1e-4,
        "expected Sharp ({v_out_sharp}) and Rounded ({v_round}) Outside shells to differ"
    );

    // 4b) Both + Rounded vs Both + Sharp
    let both_round_def = ShellDefinition::new_with_options(
        key("body", &[1]),
        vec![faces[0].clone()],
        DatumLength::Constant(0.2),
        ShellDirection::Both,
        Vec::new(),
        ShellOptions {
            shell_type: ShellType::Rounded,
            ..Default::default()
        },
    )
    .unwrap();
    let mut h4b = owner(cube_document());
    let tok = h4b.begin("ShellBothRound").unwrap();
    let id4b = h4b.create_shell(&tok, "Shell1", both_round_def).unwrap();
    h4b.commit(&tok, false).unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::new(&h4b, document(), certified_resolution());
    let r4b = h4b.snapshot().evaluate_part_features(&mut backend).unwrap();
    assert!(
        r4b.health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy),
        "Both+Rounded health: {:?}",
        r4b.health()
    );
    let v_both_round = r4b.body(id4b).unwrap().volume(&h4b, document()).unwrap();
    assert!(
        (v_both - v_both_round).abs() > 1e-4,
        "expected Both Sharp ({v_both}) and Both Rounded ({v_both_round}) to differ"
    );

    // 5) Tangent face chain expansion on a filleted cube + 3-face shell
    let mut h5 = owner(cube_document());
    let edges = solid.edge_reference_keys(&history, document()).unwrap();
    let tok = h5.begin("Fillet1").unwrap();
    let fdef = oblikovati_model::feature::fillet::FilletDefinition::new(
        key("body", &[1]),
        vec![edges[0].clone()],
        DatumLength::Constant(0.15),
    )
    .unwrap();
    let fid = h5.create_fillet(&tok, "Fillet1", fdef).unwrap();
    h5.commit(&tok, false).unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::new(&h5, document(), certified_resolution());
    let r5 = h5.snapshot().evaluate_part_features(&mut backend).unwrap();
    let filleted = r5.body(fid).unwrap();
    let f_faces = filleted.face_reference_keys(&h5, document()).unwrap();
    let cyl_face = f_faces
        .iter()
        .find(|k| {
            filleted
                .face_topology(&h5, document(), k)
                .is_ok_and(|t| t.surface == oblikovati_martensite::PrismSurfaceKind::Cylinder)
        })
        .expect("expected cylindrical fillet face")
        .clone();
    let chain = filleted
        .shell_tangent_face_chain(&h5, document(), &cyl_face)
        .unwrap();
    assert_eq!(
        chain.len(),
        3,
        "expected cylindrical fillet + 2 tangent planes"
    );
    let planar_only = filleted
        .shell_tangent_face_chain(&h5, document(), &chain[1])
        .unwrap();
    assert_eq!(planar_only.len(), 1);

    let tok = h5.begin("Shell3Faces").unwrap();
    let body_key = filleted
        .body_reference_key(&h5, document())
        .unwrap()
        .to_vec();
    let s3_def = ShellDefinition::new_with_options(
        body_key,
        chain,
        DatumLength::Constant(0.05),
        ShellDirection::Inside,
        Vec::new(),
        ShellOptions {
            allow_approximation: true,
            ..Default::default()
        },
    )
    .unwrap();
    let sid = h5.create_shell(&tok, "Shell1", s3_def).unwrap();
    h5.commit(&tok, false).unwrap();
    let mut backend = CertifiedNamedSolidEvaluator::new(&h5, document(), certified_resolution());
    let r5s = h5.snapshot().evaluate_part_features(&mut backend).unwrap();
    assert!(
        r5s.health()
            .iter()
            .all(|h| h.status == FeatureEvaluationStatus::Healthy),
        "3-face filleted shell health: {:?}",
        r5s.health()
    );
    let v_s3 = r5s.body(sid).unwrap().volume(&h5, document()).unwrap();
    assert!(v_s3 > 0.0, "v_s3={v_s3}");
}
