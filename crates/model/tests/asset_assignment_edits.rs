// SPDX-License-Identifier: GPL-2.0-only
#[path = "assets/support.rs"]
mod support;
use oblikovati_api::types::{ASSET_BUILTIN, ASSET_DOCUMENT, ASSET_PROJECT};
use oblikovati_model::assets::*;
fn physical_record(
    source: oblikovati_api::types::AssetSource,
) -> oblikovati_api::wire::MaterialInfo {
    let mut record = support::material_record(source);
    record.magnetic.class = oblikovati_api::types::SOFT_MAGNETIC;
    record
}
fn project() -> ProjectAssetLibrary {
    ProjectAssetLibrary::from_records(
        vec![MaterialAsset::from_info(physical_record(ASSET_PROJECT)).unwrap()],
        vec![AppearanceAsset::from_info(support::appearance_record(ASSET_PROJECT)).unwrap()],
    )
    .unwrap()
}
fn builtin() -> BuiltinAssetCatalog {
    let record = physical_record(ASSET_BUILTIN);
    let visual = AppearanceAsset::from_info(support::appearance_record(ASSET_BUILTIN)).unwrap();
    BuiltinAssetCatalog::from_records(
        vec![MaterialAsset::from_info(record).unwrap()],
        vec![visual, neutral()],
    )
    .unwrap()
}
fn neutral() -> AppearanceAsset {
    let mut record = support::appearance_record(ASSET_BUILTIN);
    record.id = DEFAULT_APPEARANCE_ID.into();
    AppearanceAsset::from_info(record).unwrap()
}
fn neutral_catalog() -> BuiltinAssetCatalog {
    BuiltinAssetCatalog::from_records(vec![], vec![neutral()]).unwrap()
}
fn body(path: &str) -> AssignmentReference {
    AssignmentReference::new("body", vec![path.into()]).unwrap()
}
#[test]
fn physical_assignment_copies_the_full_material_and_its_visual_dependency() {
    let original = DocumentAssets::default();
    let next = original
        .with_material_assignment(
            &project(),
            &builtin(),
            &MaterialAssignmentTarget::Part,
            "physical",
        )
        .unwrap();
    assert!(original.is_empty());
    assert_eq!(next.assignments.part_material_id(), Some("physical"));
    assert_eq!(
        next.materials().get("physical").unwrap().info(),
        physical_record(ASSET_DOCUMENT)
    );
    assert_eq!(
        next.appearances().get("visual").unwrap().info(),
        support::appearance_record(ASSET_DOCUMENT)
    );
}
#[test]
fn missing_material_or_dependency_refuses_the_whole_candidate() {
    let original = DocumentAssets::default();
    let missing = neutral_catalog();
    assert!(
        original
            .with_material_assignment(
                &project(),
                &missing,
                &MaterialAssignmentTarget::Part,
                "unknown"
            )
            .is_err()
    );
    let physical_only = ProjectAssetLibrary::from_records(
        vec![MaterialAsset::from_info(physical_record(ASSET_PROJECT)).unwrap()],
        vec![],
    )
    .unwrap();
    assert!(
        original
            .with_material_assignment(
                &physical_only,
                &missing,
                &MaterialAssignmentTarget::Part,
                "physical"
            )
            .is_err()
    );
    assert!(original.is_empty());
}
#[test]
fn builtin_physics_still_embeds_its_project_visual_dependency() {
    let visual_only = ProjectAssetLibrary::from_records(
        vec![],
        vec![AppearanceAsset::from_info(support::appearance_record(ASSET_PROJECT)).unwrap()],
    )
    .unwrap();
    let next = DocumentAssets::default()
        .with_material_assignment(
            &visual_only,
            &builtin(),
            &MaterialAssignmentTarget::Part,
            "physical",
        )
        .unwrap();
    assert!(next.materials().get("physical").is_none());
    assert!(next.appearances().get("visual").is_some());
}
#[test]
fn builtin_definitions_are_references_and_empty_material_appearance_needs_no_copy() {
    let next = DocumentAssets::default()
        .with_material_assignment(
            &ProjectAssetLibrary::default(),
            &builtin(),
            &MaterialAssignmentTarget::Part,
            "physical",
        )
        .unwrap();
    assert!(next.materials().ordered().next().is_none());
    assert!(next.appearances().ordered().next().is_none());
    let mut record = physical_record(ASSET_PROJECT);
    record.appearance_id.clear();
    let physical_only =
        ProjectAssetLibrary::from_records(vec![MaterialAsset::from_info(record).unwrap()], vec![])
            .unwrap();
    let next = next
        .with_material_assignment(
            &physical_only,
            &builtin(),
            &MaterialAssignmentTarget::Part,
            "physical",
        )
        .unwrap();
    assert_eq!(
        next.materials()
            .get("physical")
            .unwrap()
            .info()
            .appearance_id,
        ""
    );
}
#[test]
fn existing_document_snapshots_win_over_project_edits() {
    let original = DocumentAssets::default()
        .with_material_assignment(
            &project(),
            &builtin(),
            &MaterialAssignmentTarget::Part,
            "physical",
        )
        .unwrap();
    let mut changed = physical_record(ASSET_PROJECT);
    changed.density = 100.;
    let project = project()
        .with_material(MaterialAsset::from_info(changed).unwrap())
        .unwrap();
    let next = original
        .with_material_assignment(
            &project,
            &builtin(),
            &MaterialAssignmentTarget::Body(body("one")),
            "physical",
        )
        .unwrap();
    assert_eq!(
        next.materials().get("physical"),
        original.materials().get("physical")
    );
    assert_eq!(
        next.assignments.effective_material_id(&body("one")),
        Some("physical")
    );
}
#[test]
fn visual_targets_preserve_physics_and_occurrence_isolation() {
    let original = DocumentAssets::default()
        .with_material_assignment(
            &project(),
            &builtin(),
            &MaterialAssignmentTarget::Body(body("one")),
            "physical",
        )
        .unwrap();
    for target in [
        AppearanceAssignmentTarget::Part,
        AppearanceAssignmentTarget::Body(body("one")),
        AppearanceAssignmentTarget::Face(
            AssignmentReference::new("face", vec!["one".into()]).unwrap(),
        ),
    ] {
        let next = original
            .with_appearance_assignment(&project(), &builtin(), &target, "visual")
            .unwrap();
        assert_eq!(
            next.assignments.effective_material_id(&body("one")),
            Some("physical")
        );
        assert_eq!(next.assignments.effective_material_id(&body("two")), None);
        let cleared = next
            .with_appearance_assignment(&project(), &builtin(), &target, "")
            .unwrap();
        assert_eq!(cleared, original);
    }
}
#[test]
fn clearing_lost_physics_requires_no_catalog_and_keeps_portable_definitions() {
    let original = DocumentAssets::default()
        .with_material_assignment(
            &project(),
            &builtin(),
            &MaterialAssignmentTarget::Part,
            "physical",
        )
        .unwrap();
    let empty = neutral_catalog();
    let next = original
        .with_material_assignment(
            &ProjectAssetLibrary::default(),
            &empty,
            &MaterialAssignmentTarget::Part,
            "",
        )
        .unwrap();
    assert_eq!(next.assignments.part_material_id(), None);
    assert_eq!(
        next.materials().get("physical"),
        original.materials().get("physical")
    );
}

#[test]
fn direct_visual_assignment_embeds_only_visuals_and_refuses_missing_assets() {
    let original = DocumentAssets::default();
    let next = original
        .with_appearance_assignment(
            &project(),
            &builtin(),
            &AppearanceAssignmentTarget::Part,
            "visual",
        )
        .unwrap();
    assert!(next.materials().ordered().next().is_none());
    assert_eq!(
        next.appearances().get("visual").unwrap().info(),
        support::appearance_record(ASSET_DOCUMENT)
    );
    assert!(
        original
            .with_appearance_assignment(
                &project(),
                &builtin(),
                &AppearanceAssignmentTarget::Part,
                "missing"
            )
            .is_err()
    );
    assert!(original.is_empty());
}
#[test]
fn body_clear_removes_only_the_selected_occurrence_assignment() {
    let target = MaterialAssignmentTarget::Body(body("one"));
    let first = DocumentAssets::default()
        .with_material_assignment(&project(), &builtin(), &target, "physical")
        .unwrap();
    let second = first
        .with_material_assignment(
            &project(),
            &builtin(),
            &MaterialAssignmentTarget::Body(body("two")),
            "physical",
        )
        .unwrap();
    let cleared = second
        .with_material_assignment(
            &ProjectAssetLibrary::default(),
            &neutral_catalog(),
            &target,
            "",
        )
        .unwrap();
    assert_eq!(
        cleared.assignments.effective_material_id(&body("one")),
        None
    );
    assert_eq!(
        cleared.assignments.effective_material_id(&body("two")),
        Some("physical")
    );
    assert!(cleared.materials().get("physical").is_some());
}
