// SPDX-License-Identifier: GPL-2.0-only
//! Verify STL and STEP exchange inspection, Martensite shape evaluation, and v50 persistence.
#[path = "../../model/tests/common/document_snapshot.rs"]
mod document_snapshot;

use oblikovati_martensite::{
    CertifiedNamedSolidEvaluator, PrismResolution, PrismResolutionPolicy,
    import::{inspect_step, inspect_stl},
};
use oblikovati_model::{
    feature::{
        FeatureSequence,
        imported_model::{
            ImportMode, ImportedFileKind, ImportedModelDefinition, ImportedModelRecipe,
            StepImportOptions, StlImportOptions, StlImportUnits,
        },
    },
    history::DocumentEditHistory,
    identity::{DocumentGuid, RecipeIdAllocator},
};
use std::path::PathBuf;

fn fixture(relative: &str) -> PathBuf {
    PathBuf::from("/mnt/vininas-projects/MCAD-Scenarios").join(relative)
}

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
fn stl_and_step_fixtures_inspect_evaluate_and_roundtrip_v50() {
    let stl_path = fixture("fixtures/cad/import/tetrahedron.stl");
    if !stl_path.exists() {
        return;
    }
    let stl = inspect_stl(&stl_path).expect("inspect tetrahedron.stl");
    assert_eq!(stl.solid_name, "tetrahedron");
    assert_eq!(stl.triangle_count, 4);
    assert!((stl.bounding_box[0] - 10.0).abs() < 1e-6);

    let step_path = fixture("fixtures/cad/import/single-solid.stp");
    let step = inspect_step(&step_path).expect("inspect single-solid.stp");
    assert_eq!(step.solid_count, 1);
    assert_eq!(step.solid_names, vec!["Solid1"]);

    let mut history = DocumentEditHistory::new(
        document_snapshot::empty_snapshot(),
        RecipeIdAllocator::default(),
    )
    .unwrap();
    let stl_def = ImportedModelDefinition::from_recipe(&ImportedModelRecipe {
        version: 1,
        source_path: stl_path.to_string_lossy().into_owned(),
        source_name: "tetrahedron.stl".into(),
        file_kind: ImportedFileKind::Stl,
        import_mode: ImportMode::ConvertModel,
        body_names: vec!["tetrahedron".into()],
        link_suppressed: false,
        link_broken: false,
        unit_scale: StlImportUnits::Millimeter.scale_to_cm(0.1),
        stl_options: Some(StlImportOptions::default()),
        step_options: None,
    })
    .unwrap();
    let step_def = ImportedModelDefinition::from_recipe(&ImportedModelRecipe {
        version: 1,
        source_path: step_path.to_string_lossy().into_owned(),
        source_name: "single-solid.stp".into(),
        file_kind: ImportedFileKind::Step,
        import_mode: ImportMode::ReferenceModel,
        body_names: vec!["Solid1".into()],
        link_suppressed: false,
        link_broken: false,
        unit_scale: 0.1,
        stl_options: None,
        step_options: Some(StepImportOptions::default()),
    })
    .unwrap();
    let token = history.begin("Import STL and STEP").unwrap();
    let (stl_id, step_id) = history
        .edit(&token, |snap, ids| {
            let f1 = snap.create_imported_model("tetrahedron.stl", stl_def, ids)?;
            let f2 = snap.create_imported_model("single-solid.stp", step_def, ids)?;
            Ok::<_, oblikovati_model::history::HistoryError>((f1, f2))
        })
        .unwrap();
    history.commit(&token, false).unwrap();

    let recipe = history
        .snapshot()
        .work_geometry
        .work_plane_recipe()
        .unwrap();
    assert_eq!(recipe.version, 50);
    let restored = FeatureSequence::from_work_plane_recipe(&recipe).unwrap();
    assert_eq!(restored.features().len(), 2);

    let guid: DocumentGuid = "00000000-0000-4000-8000-000000001a01".parse().unwrap();
    let mut evaluator =
        CertifiedNamedSolidEvaluator::with_optional_policy(&history, guid, Some(resolution()));
    let eval = evaluator
        .evaluate_private_candidate(history.snapshot())
        .expect("evaluate imported STL and STEP solids");
    assert!(eval.body(stl_id).is_some(), "expected STL body");
    assert!(eval.body(step_id).is_some(), "expected STEP body");
}
