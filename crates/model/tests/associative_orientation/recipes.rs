// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn absent_orientation_keeps_legacy_layout_and_oriented_records_require_version_three() {
    let mut snapshot = snapshot();
    let legacy = snapshot.sketches.recipe();
    assert_eq!(legacy.version, 1);
    assert!(
        !serde_json::to_string(&legacy)
            .unwrap()
            .contains("orientation")
    );
    assign(&mut snapshot, SketchAxisReference::OriginX);
    let mut recipe = snapshot.sketches.recipe();
    for version in [1, 2, 5] {
        recipe.version = version;
        assert!(DocumentSketches::from_recipe(&recipe).is_err());
    }
    recipe.version = 3;
    let restored = DocumentSketches::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&snapshot.sketches));
}

#[test]
fn malformed_authored_orientation_is_refused_before_mutation_or_restore() {
    let mut snapshot = snapshot();
    let before = snapshot.sketches.clone();
    let mut definitions = [
        orientation(SketchAxisReference::Feature(FeatureId(0))),
        orientation(SketchAxisReference::OriginX),
    ];
    definitions[1].origin = Some([f64::NAN; 3]);
    for definition in definitions {
        assert!(assign_definition(&mut snapshot, Some(definition)).is_err());
        assert!(before.same_evaluated_state(&snapshot.sketches));
        let mut recipe = before.recipe();
        recipe.version = 3;
        recipe.sketches[0].orientation = Some(definition);
        assert!(DocumentSketches::from_recipe(&recipe).is_err());
    }
}
#[test]
fn unknown_sketch_orientation_edits_are_refused() {
    let mut snapshot = snapshot();
    let result = snapshot
        .sketches
        .set_orientation(DocumentSketchId(999), None);
    assert!(result.unwrap_err().to_string().contains("999"));
}

#[test]
fn exact_orientation_comparison_distinguishes_signed_origins_and_all_authored_options() {
    let mut snapshot = snapshot();
    assign(&mut snapshot, SketchAxisReference::OriginX);
    let plain = snapshot.sketches.clone();
    let mut definition = orientation(SketchAxisReference::OriginX);
    definition.origin = Some([0.0; 3]);
    assign_definition(&mut snapshot, Some(definition)).unwrap();
    let positive = snapshot.sketches.clone();
    definition.origin = Some([-0.0, 0.0, 0.0]);
    assign_definition(&mut snapshot, Some(definition)).unwrap();
    assert!(!positive.same_recipe(&snapshot.sketches));
    assert!(!plain.same_recipe(&snapshot.sketches));
    assign_definition(&mut snapshot, None).unwrap();
    assert!(!plain.same_recipe(&snapshot.sketches));
    assert!(snapshot.sketches.same_recipe(&snapshot.sketches.clone()));
}

fn changed_options(base: SketchOrientationDefinition) -> [SketchOrientationDefinition; 3] {
    [
        SketchOrientationDefinition {
            reference: SketchAxisReference::OriginY,
            ..base
        },
        SketchOrientationDefinition {
            sketch_axis: SketchFrameAxis::X,
            ..base
        },
        SketchOrientationDefinition {
            reverse: true,
            ..base
        },
    ]
}
#[test]
fn source_axis_choice_and_direction_changes_are_authored_edits() {
    let mut snapshot = snapshot();
    let base = orientation(SketchAxisReference::OriginX);
    assign_definition(&mut snapshot, Some(base)).unwrap();
    let before = snapshot.sketches.clone();
    for changed in changed_options(base) {
        assign_definition(&mut snapshot, Some(changed)).unwrap();
        assert!(!before.same_recipe(&snapshot.sketches));
    }
}
