// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::DOCUMENT_ASSEMBLY;

fn sample() -> (DocumentPackage, SavedAssemblyView) {
    let occurrence = AssemblyOccurrenceRecipe {
        name: "Block:1".into(),
        component: "block.opd".into(),
        ..Default::default()
    };
    let package = DocumentPackage::empty(DOCUMENT_ASSEMBLY, "Assembly")
        .with_owned_assembly_occurrences(
            std::slice::from_ref(&occurrence),
            &std::collections::BTreeSet::from(["block.opd".to_owned()]),
        )
        .unwrap();
    let view = SavedAssemblyView {
        version: SavedAssemblyView::VERSION,
        occurrences: vec![occurrence],
        sources: vec![AssemblySourceRecord {
            occurrence_path: vec![0],
            reference: "block.opd".into(),
            resolved_path: std::env::temp_dir().join("parts").join("block.opd"),
            identity: "00000000-0000-4000-8000-000000000001".into(),
            digest: "a".repeat(64),
        }],
        catalog_digest: "b".repeat(64),
        chord_error: 0.05,
        angular_error: 0.5,
        triangles: vec![SavedAssemblyTriangle {
            positions: [[0., 0., 0.], [1., 0., 0.], [0., 1., 0.]],
            color: [0.2, 0.3, 0.4, 1.],
        }],
    };
    (package, view)
}

#[test]
fn cache_round_trips_with_saved_recipe() {
    let (package, view) = sample();
    let saved = package.with_saved_assembly_view(&view).unwrap();
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    assert_eq!(reopened.restore_saved_assembly_view().unwrap(), Some(view));
}

#[test]
fn cache_refuses_missing_duplicate_or_mismatched_source() {
    let (package, mut view) = sample();
    let source = view.sources[0].clone();
    view.sources.clear();
    assert!(package.with_saved_assembly_view(&view).is_err());
    view.sources = vec![source.clone(), source.clone()];
    assert!(package.with_saved_assembly_view(&view).is_err());
    view.sources = vec![source];
    view.sources[0].reference = "other.opd".into();
    assert!(package.with_saved_assembly_view(&view).is_err());
}

#[test]
fn cache_refuses_nonfinite_geometry_bad_hash_and_recipe_change() {
    let (package, mut view) = sample();
    view.triangles[0].positions[0][0] = f32::NAN;
    assert!(package.with_saved_assembly_view(&view).is_err());
    view.triangles[0].positions[0][0] = 0.;
    view.catalog_digest = "x".repeat(64);
    assert!(package.with_saved_assembly_view(&view).is_err());
    view.catalog_digest = "b".repeat(64);
    view.occurrences[0].name = "changed".into();
    assert!(package.with_saved_assembly_view(&view).is_err());
}

#[test]
fn modified_cache_content_fails_its_checksum() {
    let (package, view) = sample();
    let saved = package.with_saved_assembly_view(&view).unwrap();
    let bytes = saved.streams().get(SECTION).unwrap();
    let mut envelope: AssemblyViewEnvelope = serde_yaml_ng::from_slice(bytes).unwrap();
    envelope.content = envelope.content.replace("block.opd", "other.opd");
    let modified = serde_yaml_ng::to_string(&envelope).unwrap();
    let damaged = saved.with_stream(SECTION, modified.as_bytes());
    let error = damaged.restore_saved_assembly_view().unwrap_err();
    assert!(error.to_string().contains("expected its content checksum"));
}
