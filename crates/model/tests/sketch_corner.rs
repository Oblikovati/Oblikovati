// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentEditHistory, SketchCornerCreation},
    sketch::{
        PlanarConstraint, Sketch2D, SketchConstraintId, SketchCornerConstruction,
        SketchCornerPredicate, SketchCornerSeed, SketchCornerSize, SketchCornerTurn,
        SketchEntityId, SketchLineEnd, SketchPlane,
    },
};
struct FakeCornerConstruction;
impl SketchCornerPredicate for FakeCornerConstruction {
    fn turn(&self, _: [[f64; 2]; 3]) -> Result<SketchCornerTurn, Box<dyn std::error::Error>> {
        Ok(SketchCornerTurn::CounterClockwise)
    }
}
impl SketchCornerConstruction for FakeCornerConstruction {
    fn construct_corner(
        &self,
        _: [[f64; 2]; 3],
        size: SketchCornerSize,
        _: f64,
    ) -> Result<SketchCornerSeed, Box<dyn std::error::Error>> {
        Ok(SketchCornerSeed {
            endpoints: [[1., 0.], [0., 1.]],
            arc: matches!(size, SketchCornerSize::Fillet(_)).then_some(([1., 1.], false)),
        })
    }
}
fn id(n: u64) -> SketchEntityId {
    SketchEntityId(n)
}
fn fixture() -> DocumentEditHistory {
    let mut geometry = Sketch2D::default();
    for (n, p) in [(1, [0., 0.]), (2, [4., 0.]), (3, [0., 5.])] {
        geometry.add_point(id(n), p).unwrap();
    }
    geometry.add_line(id(4), [id(2), id(1)]).unwrap();
    geometry.add_line(id(5), [id(1), id(3)]).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Fix {
                point: id(2),
                position: [4., 0.],
            },
        )
        .unwrap();
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Corner", SketchPlane::xy(), geometry)
        .unwrap();
    DocumentEditHistory::new(snapshot, Default::default()).unwrap()
}
fn input(arc: bool) -> SketchCornerCreation {
    SketchCornerCreation {
        lines: [id(4), id(5)],
        ends: [SketchLineEnd::Last, SketchLineEnd::First],
        construction: false,
        size: if arc {
            SketchCornerSize::Fillet(1.)
        } else {
            SketchCornerSize::Chamfer([1., 1.])
        },
    }
}
#[test]
fn certified_fillet_rewires_stable_lines_and_keeps_virtual_corner_and_tangency() {
    let mut history = fixture();
    let token = history.begin("Fillet").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_corner(
                DocumentSketchId(1),
                &input(true),
                &FakeCornerConstruction,
                0.,
                ids,
            )
        })
        .unwrap();
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert_eq!(
        geometry.line(id(4)).unwrap().endpoints,
        [id(2), report.trimmed[0]]
    );
    assert_eq!(
        geometry.line(id(5)).unwrap().endpoints,
        [report.trimmed[1], id(3)]
    );
    assert_eq!(report.virtual_corners, [id(1), id(1)]);
    let solved = geometry
        .solve(&history.snapshot().parameters, Default::default(), || false)
        .unwrap();
    assert_eq!(
        solved.numerical().termination,
        oblikovati_solve::nonlinear::SolveTermination::Converged
    );

    assert_eq!(
        geometry.arc(report.entity).unwrap().start,
        report.trimmed[0]
    );
    assert_eq!(
        geometry
            .constraints()
            .iter()
            .filter(|c| matches!(c.relation, PlanarConstraint::LineTangent { .. }))
            .count(),
        2
    );
    assert_eq!(
        geometry
            .constraints()
            .iter()
            .filter(|c| matches!(c.relation, PlanarConstraint::PointOnLine { .. }))
            .count(),
        2
    );
}
#[test]
fn rejected_corner_keeps_original_geometry() {
    let mut history = fixture();
    let before = history.snapshot().sketches.clone();
    let mut invalid = input(false);
    invalid.lines = [id(4), id(4)];
    let token = history.begin("Invalid").unwrap();
    assert!(
        history
            .edit(&token, |snapshot, ids| snapshot.create_sketch_corner(
                DocumentSketchId(1),
                &invalid,
                &FakeCornerConstruction,
                0.,
                ids
            ))
            .is_err()
    );
    assert_eq!(history.snapshot().sketches, before);
}
#[test]
fn deleting_a_fillet_removes_only_its_owned_relations_after_storage_round_trip() {
    let mut history = fixture();
    let token = history.begin("Fillet").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_corner(
                DocumentSketchId(1),
                &input(true),
                &FakeCornerConstruction,
                0.,
                ids,
            )
        })
        .unwrap();
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    let recipe = geometry.recipe();
    assert_eq!(recipe.version, 19);
    let mut restored = Sketch2D::from_recipe(&recipe).unwrap();
    assert!(restored.same_recipe(geometry));
    let owned = recipe.owned_relations.as_ref().unwrap();
    assert_eq!(owned.len(), 4);
    assert!(
        restored
            .remove_geometric_constraint(owned[0].constraint)
            .is_err()
    );
    let deleted = restored.delete_entities(&[report.entity]).unwrap();
    assert_eq!(deleted.constraints.len(), 4);
    assert_eq!(restored.constraints().len(), 1);
    assert!(matches!(
        restored.constraints()[0].relation,
        PlanarConstraint::Fix { .. }
    ));
    assert!(restored.line(id(4)).is_ok());
    assert!(restored.line(id(5)).is_ok());
    assert!(restored.point(id(1)).is_ok());
}
#[test]
fn deleting_source_line_also_retires_its_dependent_blend() {
    let mut history = fixture();
    let token = history.begin("Fillet").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_corner(
                DocumentSketchId(1),
                &input(true),
                &FakeCornerConstruction,
                0.,
                ids,
            )
        })
        .unwrap();
    let mut geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    let deleted = geometry.delete_entities(&[id(4)]).unwrap();
    assert!(deleted.entities.contains(&report.entity));
    assert_eq!(deleted.constraints.len(), 4);
    assert!(geometry.line(id(5)).is_ok());
}
#[test]
fn blend_deletion_undo_redo_restores_owned_relations_and_unrelated_constraints() {
    let mut history = fixture();
    let token = history.begin("Fillet").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_corner(
                DocumentSketchId(1),
                &input(true),
                &FakeCornerConstruction,
                0.,
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    let before = history.snapshot().sketches.clone();
    let token = history.begin("Delete").unwrap();
    history
        .edit(&token, |snapshot, _| {
            snapshot.delete_sketch_entities(DocumentSketchId(1), &[report.entity])
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    assert_eq!(
        history
            .snapshot()
            .sketches
            .get(DocumentSketchId(1))
            .unwrap()
            .geometry()
            .constraints()
            .len(),
        1
    );
    history.undo().unwrap();
    assert_eq!(history.snapshot().sketches, before);
    history.redo().unwrap();
    let geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry();
    assert!(geometry.arc(report.entity).is_err());
    assert!(geometry.line(id(4)).is_ok());
    assert!(geometry.line(id(5)).is_ok());
}
#[test]
fn structural_relation_schema_rejects_older_versions_missing_owners_and_duplicate_owners() {
    let mut history = fixture();
    let token = history.begin("Fillet").unwrap();
    history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_corner(
                DocumentSketchId(1),
                &input(true),
                &FakeCornerConstruction,
                0.,
                ids,
            )
        })
        .unwrap();
    let recipe = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .recipe();
    let mut old = recipe.clone();
    old.version = 18;
    assert!(Sketch2D::from_recipe(&old).is_err());
    let mut missing = recipe.clone();
    missing.owned_relations = None;
    assert!(Sketch2D::from_recipe(&missing).is_err());
    let mut duplicate = recipe.clone();
    let owners = duplicate.owned_relations.as_mut().unwrap();
    owners.push(owners[0]);
    assert!(Sketch2D::from_recipe(&duplicate).is_err());
    let mut unknown = recipe;
    unknown.owned_relations.as_mut().unwrap()[0].owner = id(99999);
    assert!(Sketch2D::from_recipe(&unknown).is_err());
}
#[test]
fn existing_coincidence_cannot_be_removed_while_a_blend_depends_on_it() {
    let mut geometry = Sketch2D::default();
    for (n, p) in [(1, [0., 0.]), (2, [4., 0.]), (3, [0., 0.]), (4, [0., 5.])] {
        geometry.add_point(id(n), p).unwrap();
    }
    geometry.add_line(id(5), [id(2), id(1)]).unwrap();
    geometry.add_line(id(6), [id(3), id(4)]).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(1),
            PlanarConstraint::Coincident {
                points: [id(1), id(3)],
            },
        )
        .unwrap();
    let mut snapshot = document_snapshot::empty_snapshot();
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Corner", SketchPlane::xy(), geometry)
        .unwrap();
    let mut history = DocumentEditHistory::new(snapshot, Default::default()).unwrap();
    let mut request = input(true);
    request.lines = [id(5), id(6)];
    let token = history.begin("Fillet").unwrap();
    let blend = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_corner(
                DocumentSketchId(1),
                &request,
                &FakeCornerConstruction,
                0.,
                ids,
            )
        })
        .unwrap();
    let mut geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    assert!(
        geometry
            .remove_geometric_constraint(SketchConstraintId(1))
            .is_err()
    );
    assert!(
        geometry
            .delete_selection(&oblikovati_model::sketch::SketchDeletionSelection {
                geometric_constraints: vec![SketchConstraintId(1)],
                ..Default::default()
            })
            .is_err()
    );
    geometry
        .delete_selection(&oblikovati_model::sketch::SketchDeletionSelection {
            entities: vec![blend.entity],
            geometric_constraints: vec![SketchConstraintId(1)],
            ..Default::default()
        })
        .unwrap();
    assert!(geometry.arc(blend.entity).is_err());
}
#[test]
fn corner_dependency_map_follows_pattern_point_offsets_and_radius_links() {
    use oblikovati_model::sketch::{
        SketchPatternOffset, SketchPatternOffsetTerm, SketchPatternPointLink,
        SketchPatternRadiusLink,
    };
    let mut history = fixture();
    let token = history.begin("Fillet").unwrap();
    let blend = history
        .edit(&token, |snapshot, ids| {
            snapshot.create_sketch_corner(
                DocumentSketchId(1),
                &input(true),
                &FakeCornerConstruction,
                0.,
                ids,
            )
        })
        .unwrap();
    let mut geometry = history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
        .clone();
    geometry.add_point(id(50), [-1., 0.]).unwrap();
    geometry
        .add_pattern_point_link(SketchPatternPointLink {
            source: id(50),
            member: id(2),
            offset: [5., 0.],
        })
        .unwrap();
    geometry
        .set_pattern_offset(SketchPatternOffset {
            member: id(2),
            terms: vec![SketchPatternOffsetTerm {
                parameter: 7,
                coefficient: [1., 0.],
            }],
        })
        .unwrap();
    geometry.add_point(id(51), [20., 0.]).unwrap();
    geometry.add_point(id(52), [21., 0.]).unwrap();
    geometry
        .add_pattern_point_link(SketchPatternPointLink {
            source: id(51),
            member: id(52),
            offset: [1., 0.],
        })
        .unwrap();
    geometry
        .set_pattern_offset(SketchPatternOffset {
            member: id(52),
            terms: vec![SketchPatternOffsetTerm {
                parameter: 8,
                coefficient: [1., 0.],
            }],
        })
        .unwrap();
    geometry.add_circle(id(53), id(51), 2.).unwrap();
    geometry.add_circle(id(54), id(2), 2.).unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(98),
            PlanarConstraint::EqualRadius {
                curves: [blend.entity, id(54)],
            },
        )
        .unwrap();
    geometry
        .add_pattern_radius_link(SketchPatternRadiusLink {
            source: id(53),
            member: id(54),
        })
        .unwrap();
    let anchors = [id(2), id(51)].map(|point| *geometry.point(point).unwrap());
    geometry
        .add_constraint(
            SketchConstraintId(97),
            PlanarConstraint::Ground {
                points: anchors.to_vec(),
            },
        )
        .unwrap();
    geometry
        .add_constraint(
            SketchConstraintId(99),
            PlanarConstraint::Radius {
                curve: id(53),
                parameter: 9,
            },
        )
        .unwrap();
    let related = geometry.corner_related_parameters(blend.entity).unwrap();
    assert!(related.contains(&7));
    assert!(related.contains(&9));
    assert!(!related.contains(&8));
    let points = geometry.corner_related_points(blend.entity).unwrap();
    assert!(points.contains(&id(2)));
    assert!(!points.contains(&id(52)));
}

#[path = "sketch_corner/components.rs"]
mod components;
