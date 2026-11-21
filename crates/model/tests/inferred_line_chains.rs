// SPDX-License-Identifier: GPL-2.0-only
#[path = "common/document_snapshot.rs"]
mod document_snapshot;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::{DocumentEditHistory, SketchLineChainCreation, SketchLineChainCreationReport},
    identity::RecipeIdAllocator,
    sketch::{Sketch2D, SketchEntityId, SketchInferencePolicy, SketchPlane, SketchRecipe},
    sketch_settings::SketchSettings,
};
fn owner(settings: Option<SketchSettings>) -> DocumentEditHistory {
    let mut snapshot = document_snapshot::empty_snapshot();
    let mut sketch = Sketch2D::default();
    for (id, position) in [(1, [0.0, 0.0]), (2, [3.0, 2.0]), (4, [0.0, 0.0])] {
        sketch.add_point(SketchEntityId(id), position).unwrap();
        if id == 2 {
            sketch
                .add_line(SketchEntityId(3), [SketchEntityId(1), SketchEntityId(2)])
                .unwrap();
        }
    }
    snapshot
        .sketches
        .add(DocumentSketchId(1), "Chain", SketchPlane::xy(), sketch)
        .unwrap();
    snapshot.sketch_settings = settings;
    DocumentEditHistory::new(snapshot, RecipeIdAllocator::default()).unwrap()
}
fn input(points: &[[f64; 2]], closed: bool) -> SketchLineChainCreation {
    SketchLineChainCreation {
        points: points.to_vec(),
        closed,
        construction: false,
        centerline: false,
        joins: Default::default(),
        locks: Default::default(),
    }
}
fn create(
    history: &mut DocumentEditHistory,
    request: &SketchLineChainCreation,
) -> SketchLineChainCreationReport {
    let token = history.begin("Inferred chain").unwrap();
    let report = history
        .edit(&token, |snapshot, ids| {
            let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
            snapshot.create_inferred_sketch_line_chain(
                DocumentSketchId(1),
                request,
                policy,
                Default::default(),
                ids,
            )
        })
        .unwrap();
    history.commit(&token, false).unwrap();
    report
}
fn geometry(history: &DocumentEditHistory) -> &Sketch2D {
    history
        .snapshot()
        .sketches
        .get(DocumentSketchId(1))
        .unwrap()
        .geometry()
}
#[path = "inferred_line_chains/failures.rs"]
mod failures;
#[path = "inferred_line_chains/go_cases.rs"]
mod go_cases;
#[test]
fn history_and_serialization_preserve_visible_shared_vertices_and_role_flags() {
    for (construction, centerline) in [(false, false), (true, false), (false, true), (true, true)] {
        let mut history = owner(None);
        let before = geometry(&history).clone();
        let mut request = input(&[[10.0, 10.0], [14.0, 10.0], [14.0, 13.0]], true);
        request.construction = construction;
        request.centerline = centerline;
        let report = create(&mut history, &request);
        verify_roles(geometry(&history), &report, construction, centerline);
        verify_history(&mut history, &before);
    }
}
fn verify_roles(
    sketch: &Sketch2D,
    report: &SketchLineChainCreationReport,
    construction: bool,
    centerline: bool,
) {
    for point in &report.points {
        assert!(sketch.entity_ids().contains(point));
        assert!(!sketch.is_construction(*point).unwrap());
    }
    for line in &report.lines {
        assert_eq!(
            sketch.is_construction(line.primitive.entity).unwrap(),
            construction || centerline
        );
        assert_eq!(
            sketch.is_centerline(line.primitive.entity).unwrap(),
            centerline
        );
    }
}
fn verify_history(history: &mut DocumentEditHistory, before: &Sketch2D) {
    let after = geometry(history).clone();
    let marks = history.sketch_reservations();
    let encoded = serde_json::to_vec(&after.recipe()).unwrap();
    let decoded: SketchRecipe = serde_json::from_slice(&encoded).unwrap();
    assert!(after.same_recipe(&Sketch2D::from_recipe(&decoded).unwrap()));
    history.undo().unwrap();
    assert!(before.same_recipe(geometry(history)));
    history.redo().unwrap();
    assert!(after.same_recipe(geometry(history)));
    assert_eq!(history.sketch_reservations(), marks);
}

#[test]
fn nonpersistent_direction_aligns_without_saving_a_relation() {
    let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
    settings.persist_inferred_constraints = false;
    let mut history = owner(Some(settings));
    let before_dof = geometry(&history)
        .analyze(&history.snapshot().parameters, 1e-9)
        .unwrap()
        .degrees_of_freedom;
    let report = create(&mut history, &input(&[[10.0, 10.0], [14.0, 10.01]], false));
    let line = &report.lines[0];
    assert!(line.inference.direction.is_some());
    assert!(
        line.direction.is_none(),
        "temporary direction must not be saved"
    );
    let sketch = geometry(&history);
    let first = sketch.point(line.primitive.points[0]).unwrap().position;
    let second = sketch.point(line.primitive.points[1]).unwrap().position;
    assert!((first[1] - second[1]).abs() < 1e-9);
    assert!(sketch.constraints().is_empty());
    let after_dof = sketch
        .analyze(&history.snapshot().parameters, 1e-9)
        .unwrap()
        .degrees_of_freedom;
    assert_eq!(after_dof, before_dof + 4);
}

#[test]
fn joined_source_endpoint_is_not_resnapped_to_another_existing_point() {
    let mut settings = oblikovati_model::sketch_settings::default_sketch_settings();
    settings.persist_inferred_constraints = false;
    let mut history = owner(Some(settings));
    let before = geometry(&history).clone();
    let mut request = input(&[[0.0, 0.0], [4.0, 0.01]], false);
    request.joins[0] = Some(SketchEntityId(1));
    let report = create(&mut history, &request);
    assert!(report.lines[0].point_constraints.is_empty());
    assert!(report.lines[0].inference.points.is_empty());
    assert_eq!(
        geometry(&history)
            .point(SketchEntityId(1))
            .unwrap()
            .position,
        before.point(SketchEntityId(1)).unwrap().position
    );
}
