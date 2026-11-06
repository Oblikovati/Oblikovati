// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchCurveFormat::{Centerline, Construction};
#[test]
fn construction_inverts_mixed_curve_flags_in_authored_order() {
    let mut sketch = mixed_geometry();
    sketch
        .set_curve_construction(&[SketchEntityId(7)], true)
        .unwrap();
    let original = sketch.clone();
    let selected = [15, 9, 7, 11].map(SketchEntityId);
    let report = sketch.toggle_curve_format(&selected, Construction).unwrap();
    assert_eq!(report.applicable, construction::curve_ids());
    assert_eq!(report.changed, report.applicable);
    assert!(!sketch.is_construction(SketchEntityId(7)).unwrap());
    assert!(sketch.is_construction(SketchEntityId(11)).unwrap());
    sketch
        .toggle_curve_format(&report.applicable, Construction)
        .unwrap();
    assert!(sketch.same_recipe(&original));
}
#[test]
fn centerline_applies_only_to_lines_and_excludes_them_from_profiles() {
    let mut sketch = mixed_geometry();
    let report = sketch
        .toggle_curve_format(&construction::curve_ids(), Centerline)
        .unwrap();
    assert_eq!(report.applicable, [SketchEntityId(11)]);
    assert_eq!(sketch.centerline_axis().unwrap(), Some(sketch.lines()[0]));
    assert_eq!(
        sketch.profile_curve_ids(),
        [SketchEntityId(7), SketchEntityId(15)]
    );
}
#[test]
fn construction_toggle_keeps_an_independent_axis_and_does_not_claim_a_change() {
    let mut sketch = mixed_geometry();
    let selected = [SketchEntityId(11)];
    sketch.toggle_curve_format(&selected, Centerline).unwrap();
    let report = sketch.toggle_curve_format(&selected, Construction).unwrap();
    assert_eq!(report.applicable, selected);
    assert!(report.changed.is_empty());
    assert!(sketch.is_centerline(selected[0]).unwrap());
    sketch.toggle_curve_format(&selected, Centerline).unwrap();
    assert!(!sketch.is_construction(selected[0]).unwrap());
}
#[test]
fn invalid_format_batches_cannot_partially_change_roles() {
    let original = mixed_geometry();
    for ids in [
        [SketchEntityId(11), SketchEntityId(99)],
        [SketchEntityId(11); 2],
    ] {
        let mut sketch = original.clone();
        assert!(sketch.toggle_curve_format(&ids, Construction).is_err());
        assert!(sketch.same_recipe(&original));
    }
}
#[test]
fn inapplicable_format_picks_are_no_ops() {
    let original = mixed_geometry();
    let mut sketch = original.clone();
    let cases = [
        (vec![SketchEntityId(9)], Construction),
        (vec![SketchEntityId(7)], Centerline),
        (vec![], Construction),
    ];
    for (ids, mode) in cases {
        let report = sketch.toggle_curve_format(&ids, mode).unwrap();
        assert!(report.applicable.is_empty());
        assert!(sketch.same_recipe(&original));
    }
}
#[test]
fn format_conversion_keeps_health_coordinates_and_names_without_recomputation() {
    let parameters = ParameterCollection::new(Default::default());
    let mut sketches = collection()
        .solve(&parameters, Default::default(), || false)
        .candidate;
    let before = sketches.clone();
    let sketch_id = DocumentSketchId(42);
    let curves = construction::curve_ids();
    let report = sketches
        .toggle_curve_format(sketch_id, &curves, Construction)
        .unwrap();
    assert_eq!(report.changed, construction::curve_ids());
    construction::assert_unchanged_numerical_state(&sketches, &before, &parameters);
    let names = DocumentSketchReferenceIndex::new(&sketches, guid()).unwrap();
    let key = names.entity_key(sketch_id, SketchEntityId(11)).unwrap();
    assert_eq!(key, guid().sketch_entity_key(11).to_string());
}
#[test]
fn curve_format_captions_match_the_existing_format_controls() {
    assert_eq!(Construction.label(), "Construction");
    assert_eq!(Centerline.label(), "Centerline");
}
