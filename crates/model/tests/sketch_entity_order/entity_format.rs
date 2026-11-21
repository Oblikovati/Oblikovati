// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{Color, OVERRIDE_COLOR_SOURCE, SKETCH_LINE_DASHED, SketchEntityFormat};
use oblikovati_model::sketch::SketchSelectionTransform;

fn red_format() -> SketchEntityFormat {
    SketchEntityFormat {
        line_type: SKETCH_LINE_DASHED,
        override_color: Color {
            r: 255,
            g: 0,
            b: 0,
            opacity: 1.0,
            source: OVERRIDE_COLOR_SOURCE,
        },
        line_weight: 0.5,
    }
}

#[test]
fn entity_format_survives_copy_capture_and_reload_and_clears_with_default() {
    let mut source = mixed_geometry();
    let line = SketchEntityId(11);
    source.set_entity_format(line, red_format()).unwrap();
    assert_eq!(source.entity_format_overrides().len(), 1);
    let recipe = source.recipe();
    assert_eq!(recipe.version, 21);
    let encoded = serde_json::to_vec(&recipe).unwrap();
    let restored = Sketch2D::from_recipe(&serde_json::from_slice(&encoded).unwrap()).unwrap();
    assert!(source.same_recipe(&restored));
    let captured = source.capture_selection(&[line]).unwrap();
    let captured_line = captured.lines()[0].id;
    assert_eq!(
        captured.entity_format(captured_line).unwrap(),
        Some(&red_format())
    );
    let mut target = Sketch2D::default();
    let mut next = 100;
    let report = target
        .copy_transformed_selection(
            &captured,
            &[captured_line],
            SketchSelectionTransform::Move([2.0, 3.0]),
            &mut || {
                next += 1;
                Ok(SketchEntityId(next))
            },
        )
        .unwrap();
    assert_eq!(
        target.entity_format(report.copies[0]).unwrap(),
        Some(&red_format())
    );
    target
        .set_entity_format(report.copies[0], SketchEntityFormat::default())
        .unwrap();
    assert_eq!(target.entity_format(report.copies[0]).unwrap(), None);
    assert!(target.recipe().entity_formats.is_none());
}

#[test]
fn invalid_or_missing_entity_formats_leave_recipe_unchanged() {
    let mut sketch = mixed_geometry();
    let before = sketch.clone();
    let mut invalid = red_format();
    invalid.line_weight = f64::NAN;
    assert!(
        sketch
            .set_entity_format(SketchEntityId(11), invalid)
            .is_err()
    );
    assert!(
        sketch
            .set_entity_format(SketchEntityId(900), red_format())
            .is_err()
    );
    assert!(sketch.same_recipe(&before));
    let mut recipe = sketch.recipe();
    recipe.entity_formats = Some(Vec::new());
    assert!(Sketch2D::from_recipe(&recipe).is_err());
}

#[test]
fn formats_survive_complex_geometry_reload_and_are_removed_with_their_entity() {
    let mut sketch = super::defining_transforms::hidden_geometry();
    sketch
        .set_entity_format(SketchEntityId(20), red_format())
        .unwrap();
    let recipe = sketch.recipe();
    let restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(&sketch));
    sketch
        .delete_selection(&oblikovati_model::sketch::SketchDeletionSelection {
            entities: vec![SketchEntityId(20)],
            ..Default::default()
        })
        .unwrap();
    assert!(sketch.recipe().entity_formats.is_none());
    assert!(Sketch2D::from_recipe(&sketch.recipe()).is_ok());
}
