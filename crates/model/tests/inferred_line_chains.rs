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
