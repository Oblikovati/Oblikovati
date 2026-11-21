// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::USER_PARAM;
use oblikovati_model::{
    history::SketchDimensionCreation,
    sketch::{MeasuredDimensionOperands, SketchDimensionalConstraintCreation},
};
struct TranslatedCorner;
impl SketchCornerPredicate for TranslatedCorner {
    fn turn(&self, _: [[f64; 2]; 3]) -> Result<SketchCornerTurn, Box<dyn std::error::Error>> {
        Ok(SketchCornerTurn::CounterClockwise)
    }
}
impl SketchCornerConstruction for TranslatedCorner {
    fn construct_corner(
        &self,
        points: [[f64; 2]; 3],
        _: SketchCornerSize,
        _: f64,
    ) -> Result<SketchCornerSeed, Box<dyn std::error::Error>> {
        let [x, y] = points[0];
        Ok(SketchCornerSeed {
            endpoints: [[x + 1., y], [x, y + 1.]],
            arc: Some(([x + 1., y + 1.], false)),
        })
    }
}
fn two_corner_history() -> DocumentEditHistory {
    let mut geometry = Sketch2D::default();
    for (n, p) in [
        (1, [0., 0.]),
        (2, [4., 0.]),
        (3, [0., 5.]),
        (6, [10., 0.]),
        (7, [14., 0.]),
        (8, [10., 5.]),
    ] {
        geometry.add_point(id(n), p).unwrap();
    }
    for (n, ends) in [(4, [2, 1]), (5, [1, 3]), (9, [7, 6]), (10, [6, 8])] {
        geometry.add_line(id(n), ends.map(id)).unwrap();
    }
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .parameters
        .insert_expression(1, "width", USER_PARAM, "1 cm")
        .unwrap();
    snapshot
        .parameters
        .insert_expression(2, "other", USER_PARAM, "1 cm")
        .unwrap();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Corners", SketchPlane::xy(), geometry)
        .unwrap();
    DocumentEditHistory::new(
        snapshot,
        oblikovati_model::identity::RecipeIdAllocator::restore(Some(2), &[1, 2]).unwrap(),
    )
    .unwrap()
}
fn create_driven_corner(
    history: &mut DocumentEditHistory,
    lines: [u64; 2],
    expression: &str,
) -> SketchEntityId {
    let token = history.begin("corner").unwrap();
    let result = history
        .edit(&token, |snapshot, ids| {
            let request = SketchCornerCreation {
                lines: lines.map(id),
                ends: [SketchLineEnd::Last, SketchLineEnd::First],
                size: SketchCornerSize::Fillet(1.),
                construction: false,
            };
            let report = snapshot
                .create_sketch_corner(DocumentSketchId(1), &request, &TranslatedCorner, 0., ids)
                .map_err(|e| e.to_string())?;
            let dimension = snapshot
                .create_sketch_dimension(
                    DocumentSketchId(1),
                    &SketchDimensionCreation {
                        operands: SketchDimensionalConstraintCreation::Measured(
                            MeasuredDimensionOperands::Radius {
                                curve: report.entity,
                            },
                        ),
                        expression: expression.into(),
                        reference: false,
                        text_point: None,
                    },
                    Default::default(),
                    ids,
                )
                .map_err(|e| e.to_string())?;
            snapshot
                .own_corner_relation(DocumentSketchId(1), report.entity, dimension.constraint)
                .map_err(|e| e.to_string())?;
            Ok::<_, String>(report.entity)
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    result
}
#[test]
fn disconnected_corners_solve_independently_without_hiding_failed_targets() {
    let mut history = two_corner_history();
    let first = create_driven_corner(&mut history, [4, 5], "width");
    let second = create_driven_corner(&mut history, [9, 10], "other");
    let mut snapshot = history.snapshot().clone();
    snapshot.parameters.set_expression(1, "2 cm").unwrap();
    snapshot.parameters.set_expression(2, "-1 cm").unwrap();
    let geometry = snapshot
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let solved = geometry
        .solve_corner_component(first, &snapshot.parameters, &[], Default::default())
        .unwrap();
    assert!((solved.arc_radius(first).unwrap() - 2.).abs() < 1e-7);
    assert!((solved.arc_radius(second).unwrap() - 1.).abs() < 1e-7);
    assert!(
        geometry
            .solve_corner_component(second, &snapshot.parameters, &[], Default::default())
            .is_err()
    );
}
