// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::*;
use oblikovati_model::sketch_settings::default_sketch_settings;
use oblikovati_persistence::package::DocumentPackage;
fn package(section: &str) -> DocumentPackage {
    DocumentPackage::decode(
        format!("schemaVersion: 2\ndocumentType: 1\nmodel: {{retained: keep}}\n{section}\n")
            .as_bytes(),
    )
    .unwrap()
}
#[test]
fn absent_and_null_records_remain_distinct_from_explicit_zero_records() {
    for section in ["", "sketchSettings: null"] {
        let original = package(section);
        assert_eq!(original.restore_sketch_settings().unwrap(), None);
        assert_eq!(original.with_sketch_settings(None).unwrap(), original);
    }
    let original = package("sketchSettings: {}");
    assert_eq!(
        original.restore_sketch_settings().unwrap(),
        Some(SketchSettings::default())
    );
    assert!(
        original
            .with_sketch_settings(None)
            .unwrap()
            .section_yaml("sketchSettings")
            .unwrap()
            .is_none()
    );
}
#[test]
fn complete_canonical_values_round_trip_with_frozen_numeric_enums() {
    let mut settings = default_sketch_settings();
    settings.infer_constraints = false;
    settings.constraint_priority = PRIORITY_PARALLEL_PERPENDICULAR;
    settings.enable_relax_mode = true;
    let original = package("").with_sketch_settings(Some(&settings)).unwrap();
    let restored = DocumentPackage::decode(&original.encode().unwrap()).unwrap();
    assert_eq!(restored.restore_sketch_settings().unwrap(), Some(settings));
    let yaml =
        String::from_utf8(restored.section_yaml("sketchSettings").unwrap().unwrap()).unwrap();
    assert!(yaml.contains("constraintPriority: 50433"));
    assert!(yaml.contains("xSnapSpacing: 0.1"));
}
#[test]
fn partial_go_records_use_zero_values_without_default_overlay() {
    let original = package(
        "sketchSettings: {inferConstraints: true, autoApplyConstraints: false, constraintPriority: 50435}",
    );
    let restored = original.restore_sketch_settings().unwrap().unwrap();
    assert!(restored.infer_constraints);
    assert!(!restored.auto_apply_constraints);
    assert_eq!(restored.constraint_priority, PRIORITY_NONE);
    assert_eq!(
        (restored.x_snap_spacing, restored.snaps_per_minor_grid),
        (0.0, 0)
    );
    assert!(!restored.persist_inferred_constraints);
    assert!(!restored.keep_dimensions_with_equation_in_relax_mode);
}
#[test]
fn changing_known_fields_preserves_extensions_and_other_sections() {
    let original = package("sketchSettings: {inferConstraints: true, custom: {revision: 7}}");
    let settings = original.restore_sketch_settings().unwrap().unwrap();
    assert_eq!(
        original.with_sketch_settings(Some(&settings)).unwrap(),
        original
    );
    let changed = original
        .with_sketch_settings(Some(&default_sketch_settings()))
        .unwrap();
    let section =
        String::from_utf8(changed.section_yaml("sketchSettings").unwrap().unwrap()).unwrap();
    assert!(section.contains("revision: 7"));
    assert_eq!(
        original.section_yaml("model").unwrap(),
        changed.section_yaml("model").unwrap()
    );
}
#[test]
fn invalid_types_and_nonfinite_spacing_are_refused_before_capture() {
    for contents in [
        "inferConstraints: broken",
        "constraintPriority: text",
        "snapsPerMinorGrid: 0.5",
        "xSnapSpacing: .nan",
        "ySnapSpacing: .inf",
    ] {
        let original = package(&format!("sketchSettings: {{{contents}}}"));
        let error = original.restore_sketch_settings().unwrap_err().to_string();
        assert!(
            error.contains("sketchSettings") && error.contains("expected"),
            "{error}"
        );
    }
}
#[test]
fn nonfinite_snapshot_spacing_is_refused_before_save() {
    for spacing in [f64::NAN, f64::INFINITY, f64::NEG_INFINITY] {
        let mut settings = default_sketch_settings();
        settings.x_snap_spacing = spacing;
        assert!(package("").with_sketch_settings(Some(&settings)).is_err());
    }
}
#[test]
fn unknown_numeric_ids_are_preserved_as_contract_values() {
    let original =
        package("sketchSettings: {constraintPriority: 9876, overConstrainedBehavior: 1234}");
    let settings = original.restore_sketch_settings().unwrap().unwrap();
    assert_eq!(
        settings.constraint_priority,
        ConstraintInferencePriority(9876)
    );
    assert_eq!(
        settings.over_constrained_behavior,
        OverConstrainedDimensionBehavior(1234)
    );
    assert_eq!(
        original.with_sketch_settings(Some(&settings)).unwrap(),
        original
    );
}
