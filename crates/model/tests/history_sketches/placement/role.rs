// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn duplicate_locked_fields_follow_the_document_redundancy_policy() {
    use oblikovati_api::types::OVER_CONSTRAINED_APPLY_DRIVING;
    for driving in [false, true] {
        let mut history = super::super::polyline::polyline_history();
        let mut recipe = placement(false);
        recipe.fields.push(recipe.fields[0].clone());
        let locks = [
            lock("10 cm"),
            None,
            None,
            lock(if driving { "10 cm" } else { "20 cm" }),
        ];
        let token = history.begin("Redundancy policy").unwrap();
        let report = history
            .edit(&token, |snapshot, ids| {
                if driving {
                    let mut settings = snapshot.effective_sketch_settings();
                    settings.over_constrained_behavior = OVER_CONSTRAINED_APPLY_DRIVING;
                    snapshot.sketch_settings = Some(settings);
                }
                snapshot.create_sketch_placement(
                    DocumentSketchId(1),
                    &recipe,
                    &locks,
                    Default::default(),
                    ids,
                )
            })
            .unwrap();
        history
            .snapshot()
            .require_converged_sketch(report.sketch)
            .unwrap();
        history.commit(&token, false).unwrap();
        let geometry = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry();
        let second = &report.dimensions[1].1;
        assert_eq!(
            geometry.dimension_is_reference(second.constraint).unwrap(),
            !driving
        );
        assert_eq!(
            history
                .snapshot()
                .analyze_document_sketch(report.sketch, SolveOptions::default().rank_tolerance)
                .unwrap()
                .redundant_equations,
            usize::from(driving)
        );
        assert!(
            (geometry
                .measure_dimension(report.dimensions[0].1.constraint)
                .unwrap()
                .value
                - 10.0)
                .abs()
                < 1e-7
        );
    }
}
#[test]
fn redundant_reference_lock_preserves_the_accepted_geometry_bit_exact() {
    let mut baseline = None;
    for duplicate in [false, true] {
        let mut history = super::super::polyline::polyline_history();
        let mut recipe = placement(false);
        recipe.fields.push(recipe.fields[0].clone());
        let locks = [
            lock("10 cm"),
            None,
            None,
            if duplicate { lock("20 cm") } else { None },
        ];
        let token = history.begin("Reference classification").unwrap();
        let report = history
            .edit(&token, |snapshot, ids| {
                snapshot.create_sketch_placement(
                    DocumentSketchId(1),
                    &recipe,
                    &locks,
                    Default::default(),
                    ids,
                )
            })
            .unwrap();
        history
            .snapshot()
            .require_converged_sketch(report.sketch)
            .unwrap();
        let points = history
            .snapshot()
            .sketches
            .get(report.sketch)
            .unwrap()
            .geometry()
            .points()
            .iter()
            .map(|point| (point.id, point.position.map(f64::to_bits)))
            .collect::<Vec<_>>();
        if let Some(expected) = &baseline {
            assert_eq!(&points, expected);
        } else {
            baseline = Some(points);
        }
    }
}
