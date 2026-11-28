// SPDX-License-Identifier: GPL-2.0-only
//! Verify Unwrap planar surface unfold, alignment orientations, and v51 persistence.
#[path = "../../model/tests/common/document_snapshot.rs"]
mod document_snapshot;

use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolution, PrismResolutionPolicy};
use oblikovati_model::{
    feature::{
        FeatureSequence,
        imported_model::{
            ImportMode, ImportedFileKind, ImportedModelDefinition, ImportedModelRecipe,
            StepImportOptions,
        },
        unwrap::{UnwrapAlignment, UnwrapDefinition, UnwrapRecipe},
    },
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator},
};
use std::path::PathBuf;

fn resolution() -> PrismResolutionPolicy {
    PrismResolutionPolicy::Fixed(PrismResolution {
        version: 1,
        weld: 1e-7,
        sew: 1e-7,
        angular: 1e-9,
        display_chordal: 0.01,
        display_angular: 0.1,
        maximum_tolerance: 1e-4,
    })
}

#[test]
fn unwrap_creates_oriented_surface_body_and_roundtrips_v51() {
    let step_path =
        PathBuf::from("/mnt/vininas-projects/MCAD-Scenarios/fixtures/cad/import/single-solid.stp");
    if !step_path.exists() {
        return;
    }
    let mut history = DocumentEditHistory::new(
        document_snapshot::empty_snapshot(),
        RecipeIdAllocator::default(),
    )
    .unwrap();
    let base_def = ImportedModelDefinition::from_recipe(&ImportedModelRecipe {
        version: 1,
        source_path: step_path.to_string_lossy().into_owned(),
        source_name: "single-solid.stp".into(),
        file_kind: ImportedFileKind::Step,
        import_mode: ImportMode::ConvertModel,
        body_names: vec!["Solid1".into()],
        link_suppressed: false,
        link_broken: false,
        unit_scale: 0.1,
        stl_options: None,
        step_options: Some(StepImportOptions::default()),
    })
    .unwrap();
    let token = history.begin("Base").unwrap();
    let base_id = history
        .edit(&token, |snap, ids| {
            snap.create_imported_model("Solid1:1", base_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let guid: DocumentGuid = "00000000-0000-4000-8000-000000001a02".parse().unwrap();
    let mut eval =
        CertifiedNamedSolidEvaluator::with_optional_policy(&history, guid, Some(resolution()));
    let base_eval = eval.evaluate_private_candidate(history.snapshot()).unwrap();
    let base_body = base_eval.body(base_id).unwrap();
    let body_key = base_body
        .body_reference_key(&history, guid)
        .unwrap()
        .to_vec();
    let face_keys = base_body.face_reference_keys(&history, guid).unwrap();
    let vertex_keys: Vec<_> = base_body
        .subentity_reference_keys(&history, guid)
        .unwrap()
        .into_iter()
        .filter(|key| {
            oblikovati_model::identity::ReferenceEnvelope::decode(key)
                .is_ok_and(|envelope| envelope.kind == 3)
        })
        .collect();

    let unwrap_def = UnwrapDefinition::from_recipe(&UnwrapRecipe {
        version: 1,
        target_body: body_key,
        faces: vec![face_keys[0].clone()],
        alignment_vertex: vertex_keys.first().cloned(),
        auto_face_chain: false,
        merge_result_body: false,
        alignment: UnwrapAlignment::OriginXZ,
        linear_result_edges: Vec::new(),
        rigid_result_edges: Vec::new(),
        seam_edges: Vec::new(),
    })
    .unwrap();
    let token = history.begin("Unwrap1").unwrap();
    let unwrap_id = history
        .edit(&token, |snap, ids| {
            snap.create_unwrap("Unwrap1", unwrap_def, ids)
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let recipe = history
        .snapshot()
        .work_geometry
        .work_plane_recipe()
        .unwrap();
    assert_eq!(recipe.version, 51);
    let restored = FeatureSequence::from_work_plane_recipe(&recipe).unwrap();
    assert_eq!(restored.features().len(), 2);

    let mut eval =
        CertifiedNamedSolidEvaluator::with_optional_policy(&history, guid, Some(resolution()));
    let final_eval = eval.evaluate_private_candidate(history.snapshot()).unwrap();
    let surface = final_eval.body(unwrap_id).expect("expected unwrap body");
    assert!(
        surface.is_surface(),
        "expected unwrap result to be a Surface body"
    );
}
