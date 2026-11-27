// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::DimensionInfo;
use oblikovati_model::sketch::{
    AngleTurn, DistanceMeasure, TangentDistanceContact, TangentDistanceOptions,
};
pub(super) fn dimensions(files: &DiscoveryWorkspace) -> Vec<DimensionInfo> {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .dimensions(&Default::default())
        .unwrap()
        .dimensions
        .unwrap()
}
fn populated(kind: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, source) = fresh(kind);
    files
        .commands(document)
        .unwrap()
        .run("Dimension vocabulary", install_dimensions)
        .unwrap();
    (files, document, source)
}
fn install_dimensions(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let angle = ids.allocate().unwrap();
    snapshot
        .parameters
        .insert_expression(angle, "angle", USER_PARAM, "90 deg")
        .unwrap();
    let sketch = DocumentSketchId(1);
    let mut geometry = snapshot.sketches.get(sketch).unwrap().geometry().clone();
    append_fixed_points(&mut geometry, ids);
    for relation in dimension_relations(angle) {
        geometry
            .add_constraint(ids.allocate_sketch_constraint(sketch).unwrap(), relation)
            .unwrap();
    }
    snapshot
        .sketches
        .replace_geometry(sketch, geometry)
        .map_err(|cause| cause.to_string())
}
fn append_fixed_points(geometry: &mut Sketch2D, ids: &mut RecipeReservations<'_>) {
    for point in geometry.points().to_vec() {
        geometry
            .add_constraint(
                ids.allocate_sketch_constraint(DocumentSketchId(1)).unwrap(),
                Fix {
                    point: point.id,
                    position: point.position,
                },
            )
            .unwrap();
    }
}
fn dimension_relations(angle: u64) -> Vec<PlanarConstraint> {
    distance_relations()
        .into_iter()
        .chain(radial_relations())
        .chain(offset_relations())
        .chain(angle_relations(angle))
        .chain(tangent_relations())
        .collect()
}
fn distance_relations() -> Vec<PlanarConstraint> {
    [
        DistanceMeasure::Aligned,
        DistanceMeasure::Horizontal,
        DistanceMeasure::Vertical,
    ]
    .into_iter()
    .map(|measure| PlanarConstraint::Distance {
        points: [entity(1), entity(5)],
        measure,
        parameter: 1,
    })
    .collect()
}
fn radial_relations() -> Vec<PlanarConstraint> {
    vec![
        Radius {
            curve: entity(9),
            parameter: 1,
        },
        Diameter {
            curve: entity(10),
            parameter: 1,
        },
        PlanarConstraint::ArcLength {
            arc: entity(11),
            parameter: 1,
        },
        Radius {
            curve: entity(11),
            parameter: 1,
        },
    ]
}
fn offset_relations() -> Vec<PlanarConstraint> {
    [false, true]
        .into_iter()
        .map(|diameter| PlanarConstraint::Offset {
            point: entity(3),
            line: entity(7),
            parameter: 1,
            side: LineContactSide::Left,
            linear_diameter: diameter,
        })
        .collect()
}
fn angle_relations(parameter: u64) -> Vec<PlanarConstraint> {
    vec![
        PlanarConstraint::Angle {
            lines: [entity(7), entity(8)],
            parameter,
            turn: AngleTurn::CounterClockwise,
        },
        PlanarConstraint::ThreePointAngle {
            vertex: entity(1),
            points: [entity(2), entity(3)],
            parameter,
            turn: AngleTurn::CounterClockwise,
        },
    ]
}
fn tangent_relations() -> Vec<PlanarConstraint> {
    let mut relations = Vec::new();
    for diameter in [false, true] {
        for far_side in [false, true] {
            for curve in [entity(9), entity(11)] {
                relations.push(tangent_relation(curve, diameter, far_side));
            }
        }
    }
    relations
}
#[test]
fn every_dimension_form_returns_actual_measurement_source_and_canonical_metadata() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (files, document, _) = populated(kind);
        let rows = dimensions(&files);
        assert_rows(&rows);
        assert_nonconverged_geometry(&files, document);
    }
}
fn assert_rows(rows: &[DimensionInfo]) {
    assert_eq!(rows.len(), 19);
    for (index, (row, (kind, value, orientation, diameter))) in
        rows.iter().zip(expected_rows()).enumerate()
    {
        assert_eq!(row.index, index as i64);
        assert_eq!(row.kind, kind);
        let tolerance = SolveOptions::default().relative_tolerance * value.abs().max(1.0);
        assert!(
            (row.value - value).abs() <= tolerance,
            "{row:?}, expected {value}"
        );
        assert_eq!(row.orientation, orientation);
        assert_eq!(row.linear_diameter, diameter);
        assert!(!row.driven);
        assert!(row.text_point.is_none());
        assert_source_fields(row, kind);
    }
}
fn expected_rows() -> Vec<(&'static str, f64, &'static str, bool)> {
    let mut expected = vec![
        ("distance", 20.0_f64.sqrt(), "", false),
        ("distance", 4.0, "horizontal", false),
        ("distance", 2.0, "vertical", false),
        ("radius", 1.0, "", false),
        ("diameter", 2.0, "", false),
        ("arcLength", std::f64::consts::FRAC_PI_2, "", false),
        ("radius", 1.0, "", false),
        ("offsetDim", 1.0, "", false),
        ("offsetDim", 2.0, "", true),
        ("angle", 0.0, "", false),
        ("threePointAngle", std::f64::consts::FRAC_PI_2, "", false),
    ];
    expected.extend(expected_tangent_rows());
    expected
}
fn assert_nonconverged_geometry(files: &DiscoveryWorkspace, document: u64) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let health = snapshot.sketches.sketches()[0].health(&snapshot.parameters);
    let PlacedSketchHealth::Current(SketchEvaluation::Numerical(result)) = health else {
        panic!("{health:?}");
    };
    assert_ne!(
        result.termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );
}
#[test]
fn both_hosts_read_the_same_measurements_without_owner_or_evaluation_changes() {
    let (mut files, document, mut source) = populated(DOCUMENT_PART);
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    let focus = files.workspace().active_document();
    let expected = dimensions(&files);
    let actual = Client::new(FileHost::new(&mut files, &mut source, "test"))
        .sketch()
        .dimensions(&Default::default())
        .unwrap()
        .dimensions
        .unwrap();
    assert_eq!(actual, expected);
    assert_eq!(owner_revision(&files, document), before);
    assert_eq!(files.workspace().active_document(), focus);
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
}

fn right_contact() -> TangentDistanceContact {
    TangentDistanceContact {
        side: LineContactSide::Right,
        lean: None,
    }
}

fn assert_source_fields(row: &DimensionInfo, kind: &str) {
    let angular = kind == "angle" || kind == "threePointAngle";
    assert_eq!(row.name, if angular { "angle" } else { "width" });
    assert_eq!(row.expression, if angular { "90 deg" } else { "1 cm" });
}

#[path = "dimensions/lifecycle.rs"]
mod lifecycle;

fn tangent_relation(curve: SketchEntityId, diameter: bool, far_side: bool) -> PlanarConstraint {
    PlanarConstraint::TangentDistance {
        line: entity(8),
        curve,
        parameter: 1,
        options: TangentDistanceOptions {
            far_side,
            linear_diameter: diameter,
        },
        contact: right_contact(),
    }
}

fn expected_tangent_rows() -> Vec<(&'static str, f64, &'static str, bool)> {
    vec![
        ("tangentDistance", 1.0, "", false),
        ("tangentDistance", 1.0, "", false),
        ("tangentDistance", 3.0, "", false),
        ("tangentDistance", 3.0, "", false),
        ("tangentDistance", 2.0, "", true),
        ("tangentDistance", 2.0, "", true),
        ("tangentDistance", 6.0, "", true),
        ("tangentDistance", 6.0, "", true),
    ]
}

#[path = "dimensions/placement.rs"]
mod placement;

#[path = "dimensions/drive.rs"]
mod drive;
