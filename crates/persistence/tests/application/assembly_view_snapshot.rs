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
        producer_digest: "c".repeat(64),
        chord_error: 0.05,
        angular_error: 0.5,
        triangles: vec![SavedAssemblyTriangle {
            face_index: None,
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
fn saved_mesh_must_meet_both_current_display_limits() {
    let (_, mut view) = sample();
    assert!(view.validate_display_quality(0.05, 0.5).is_ok());
    assert!(view.validate_display_quality(0.1, 1.0).is_ok());
    assert!(
        view.validate_display_quality(0.04, 0.5)
            .unwrap_err()
            .to_string()
            .contains("chord")
    );
    assert!(
        view.validate_display_quality(0.05, 0.4)
            .unwrap_err()
            .to_string()
            .contains("angular")
    );
    for limit in [0.0, -1.0, f64::NAN, f64::INFINITY] {
        assert!(view.validate_display_quality(limit, 0.5).is_err());
        assert!(view.validate_display_quality(0.05, limit).is_err());
    }
    view.chord_error = f64::NAN;
    assert!(view.validate_display_quality(0.05, 0.5).is_err());
}

#[test]
fn cache_requires_the_same_nonempty_producer_identity() {
    let (_, view) = sample();
    assert!(view.validate_producer(&"c".repeat(64)).is_ok());
    for producer in [String::new(), "d".repeat(64), "z".repeat(64)] {
        let error = view.validate_producer(&producer).unwrap_err();
        assert!(error.to_string().contains("producer"), "{error}");
    }
}

#[test]
fn old_cache_without_producer_identity_is_refused() {
    let (package, view) = sample();
    let saved = package.with_saved_assembly_view(&view).unwrap();
    let bytes = saved.streams().get(SECTION).unwrap();
    let mut envelope: AssemblyViewEnvelope = serde_yaml_ng::from_slice(bytes).unwrap();
    let mut old: serde_yaml_ng::Value = serde_yaml_ng::from_str(&envelope.content).unwrap();
    let mapping = old.as_mapping_mut().unwrap();
    mapping.remove(serde_yaml_ng::Value::String("producerDigest".into()));
    mapping.insert("version".into(), 1.into());
    envelope.content = serde_yaml_ng::to_string(&old).unwrap();
    envelope.digest = RecipeDigest::of_recipe(envelope.content.as_bytes()).to_string();
    let encoded = serde_yaml_ng::to_string(&envelope).unwrap();
    let old_package = saved.with_stream(SECTION, encoded.as_bytes());
    let error = old_package.restore_saved_assembly_view().unwrap_err();
    assert!(error.to_string().contains("producerDigest"), "{error}");
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
