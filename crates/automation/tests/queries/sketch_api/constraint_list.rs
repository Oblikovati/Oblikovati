// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::ConstraintInfo;
use oblikovati_model::sketch::PlanarConstraint::*;
use oblikovati_model::{
    history::RecipeReservations,
    sketch::{CircleContact, LineContactSide, LineRelation, SketchPlane},
};
fn list(files: &DiscoveryWorkspace) -> Vec<ConstraintInfo> {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .constraints(&SketchArgs::default())
        .unwrap()
        .constraints
        .unwrap()
}
fn entity(id: u64) -> SketchEntityId {
    SketchEntityId(id)
}
fn installed(kind: DocumentType) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let (mut files, document, source) = configured_files(kind, Default::default());
    files
        .commands(document)
        .unwrap()
        .run("Constraint vocabulary", install_vocabulary)
        .unwrap();
    (files, document, source)
}
fn install_vocabulary(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(), String> {
    let sketch = ids.allocate_sketch().unwrap();
    for expected in 1..=11 {
        assert_eq!(ids.allocate_sketch_entity().unwrap(), entity(expected));
    }
    let mut geometry = vocabulary_geometry();
    for relation in vocabulary_relations() {
        geometry
            .add_constraint(ids.allocate_sketch_constraint(sketch).unwrap(), relation)
            .unwrap();
    }
    snapshot
        .sketches
        .add(sketch, "Vocabulary", SketchPlane::xy(), geometry)
        .map_err(|cause| cause.to_string())
}
fn vocabulary_geometry() -> Sketch2D {
    let mut geometry = Sketch2D::default();
    append_points(&mut geometry);
    geometry
        .add_line(entity(7), [entity(1), entity(2)])
        .unwrap();
    geometry
        .add_line(entity(8), [entity(4), entity(5)])
        .unwrap();
    geometry.add_circle(entity(9), entity(1), 1.0).unwrap();
    geometry.add_circle(entity(10), entity(4), 1.0).unwrap();
    geometry
        .add_arc(entity(11), entity(1), [entity(2), entity(3)], true)
        .unwrap();
    geometry
}
fn vocabulary_relations() -> Vec<PlanarConstraint> {
    let mut relations = vec![Radius {
        curve: entity(9),
        parameter: 99,
    }];
    relations.extend(point_relations());
    relations.push(Radius {
        curve: entity(10),
        parameter: 99,
    });
    relations.extend(line_relations());
    relations.extend(circular_relations());
    // Dimension sources intentionally reference an unavailable parameter; list must still work.
    relations.push(Radius {
        curve: entity(9),
        parameter: 99,
    });
    relations
}
fn point_relations() -> Vec<PlanarConstraint> {
    let mut relations = alignment_relations();
    relations.push(Fix {
        point: entity(1),
        position: [0.0; 2],
    });
    relations.extend(incidence_relations());
    relations
}
fn alignment_relations() -> Vec<PlanarConstraint> {
    vec![
        Coincident {
            points: [entity(1), entity(4)],
        },
        Horizontal {
            points: [entity(2), entity(4)],
        },
        Vertical {
            points: [entity(3), entity(6)],
        },
    ]
}
fn incidence_relations() -> Vec<PlanarConstraint> {
    vec![
        Midpoint {
            point: entity(6),
            line: entity(7),
        },
        ArcMidpoint {
            point: entity(6),
            arc: entity(11),
        },
        PointOnLine {
            point: entity(6),
            line: entity(8),
        },
    ]
}
fn line_relations() -> Vec<PlanarConstraint> {
    let mut relations = vec![
        LineHorizontal { line: entity(7) },
        LineVertical { line: entity(8) },
    ];
    for relation in [
        LineRelation::Parallel,
        LineRelation::Perpendicular,
        LineRelation::Collinear,
        LineRelation::EqualLength,
    ] {
        relations.push(Lines {
            lines: [entity(8), entity(7)],
            relation,
        });
    }
    relations
}
fn circular_relations() -> Vec<PlanarConstraint> {
    let mut relations = circular_properties();
    relations.extend(contact_relations());
    relations
}
fn circular_properties() -> Vec<PlanarConstraint> {
    vec![
        PointOnCircularCurve {
            point: entity(3),
            curve: entity(9),
        },
        Concentric {
            curves: [entity(9), entity(11)],
        },
        EqualRadius {
            curves: [entity(10), entity(9)],
        },
    ]
}
fn contact_relations() -> Vec<PlanarConstraint> {
    vec![
        LineTangent {
            line: entity(8),
            curve: entity(9),
            side: LineContactSide::Left,
        },
        CircularTangent {
            curves: [entity(10), entity(9)],
            contact: CircleContact::External,
        },
    ]
}
#[test]
fn typed_client_lists_every_supported_geometric_relation_and_ordered_operand() {
    for kind in [DOCUMENT_PART, DOCUMENT_ASSEMBLY] {
        let (files, _, _) = installed(kind);
        let actual = list(&files);
        let expected = expected_rows();
        assert_eq!(actual.len(), expected.len());
        for (index, (row, (kind, entities))) in actual.iter().zip(expected).enumerate() {
            assert_eq!(row.index, index as i64);
            assert_eq!(row.kind, kind);
            assert_eq!(row.entities.as_ref(), Some(&entities));
            assert!(row.deletable);
            assert!(row.client_id.is_empty() && row.name.is_empty());
        }
    }
}
fn expected_rows() -> Vec<(&'static str, Vec<u64>)> {
    let mut rows = expected_point_rows();
    rows.extend(expected_line_rows());
    rows.extend(expected_circular_rows());
    rows
}
fn expected_point_rows() -> Vec<(&'static str, Vec<u64>)> {
    vec![
        ("coincident", vec![1, 4]),
        ("horizontalAlign", vec![2, 4]),
        ("verticalAlign", vec![3, 6]),
        ("fix", vec![1]),
        ("midpoint", vec![6, 7]),
        ("midpoint", vec![6, 11]),
        ("pointOnLine", vec![6, 8]),
    ]
}
fn expected_line_rows() -> Vec<(&'static str, Vec<u64>)> {
    vec![
        ("horizontal", vec![7]),
        ("vertical", vec![8]),
        ("parallel", vec![8, 7]),
        ("perpendicular", vec![8, 7]),
        ("collinear", vec![8, 7]),
        ("equalLength", vec![8, 7]),
    ]
}
fn expected_circular_rows() -> Vec<(&'static str, Vec<u64>)> {
    vec![
        ("pointOnCircle", vec![3, 9]),
        ("concentric", vec![9, 11]),
        ("equalRadius", vec![10, 9]),
        ("tangent", vec![8, 9]),
        ("tangent", vec![10, 9]),
    ]
}
#[path = "constraint_list/lifecycle.rs"]
mod lifecycle;

fn append_points(geometry: &mut Sketch2D) {
    let coordinates = [
        [0.0, 0.0],
        [1.0, 0.0],
        [0.0, 1.0],
        [2.0, 0.0],
        [3.0, 0.0],
        [2.0, 1.0],
    ];
    for (index, point) in coordinates.into_iter().enumerate() {
        geometry.add_point(entity(index as u64 + 1), point).unwrap();
    }
}

#[path = "constraint_list/delete.rs"]
mod delete;
