// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::SketchPointProjection,
    feature::{work_geometry::WorkPointReference, work_plane::WorkPlaneOrigin},
    sketch::{DistanceMeasure, SketchEntityId},
};

fn add_rectangle(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<(DocumentSketchId, SketchEntityId), String> {
    let sketch = ids.allocate_sketch().map_err(|e| e.to_string())?;
    let points = (0..5)
        .map(|_| ids.allocate_sketch_entity().unwrap())
        .collect::<Vec<_>>();
    let mut shape = Sketch2D::default();
    for (id, xy) in points
        .iter()
        .zip([[0., 0.], [0., 0.], [1., 0.], [1., 1.], [0., 1.]])
    {
        shape.add_point(*id, xy).unwrap();
    }
    for [a, b] in [[1, 2], [2, 3], [3, 4], [4, 1]] {
        shape
            .add_line(
                ids.allocate_sketch_entity().unwrap(),
                [points[a], points[b]],
            )
            .unwrap();
    }
    let relations = [
        PlanarConstraint::Coincident {
            points: [points[0], points[1]],
        },
        PlanarConstraint::Horizontal {
            points: [points[1], points[2]],
        },
        PlanarConstraint::Horizontal {
            points: [points[3], points[4]],
        },
        PlanarConstraint::Vertical {
            points: [points[2], points[3]],
        },
        PlanarConstraint::Vertical {
            points: [points[4], points[1]],
        },
        PlanarConstraint::Distance {
            points: [points[1], points[2]],
            measure: DistanceMeasure::Horizontal,
            parameter: 1,
        },
        PlanarConstraint::Distance {
            points: [points[1], points[4]],
            measure: DistanceMeasure::Vertical,
            parameter: 1,
        },
    ];
    for relation in relations {
        shape
            .add_constraint(ids.allocate_sketch_constraint(sketch).unwrap(), relation)
            .unwrap();
    }
    snapshot
        .sketches
        .add_on_origin_plane(sketch, "Rectangle", WorkPlaneOrigin::XY, shape)
        .map_err(|e| e.to_string())?;
    snapshot
        .sketches
        .set_point_projections(
            sketch,
            vec![SketchPointProjection {
                anchor: points[0],
                reference: WorkPointReference::Origin,
            }],
        )
        .map_err(|e| e.to_string())?;
    Ok((sketch, points[0]))
}
fn assert_current(
    host: &mut Host,
    document: u64,
    sketch: DocumentSketchId,
    anchor: SketchEntityId,
    size: f64,
) {
    let before = host.workspace().session(document).unwrap();
    let recipe = before.snapshot().sketches.recipe();
    let generation = before.generation();
    let labels = before
        .undo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect::<Vec<_>>();
    let dirty = host.workspace().needs_save(document).unwrap();
    {
        let mut commands = host.commands(document).unwrap();
        let evidence = commands
            .conditional_canonical_sketch(sketch, anchor)
            .unwrap();
        let projection = evidence.projection().unwrap();
        let points = projection
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .points();
        let bounds = points
            .iter()
            .flat_map(|p| p.position)
            .map(|v| [v - 0.01, v + 0.01])
            .collect::<Vec<_>>();
        let root = projection.conditional_root_auto(&bounds, &[]).unwrap();
        let values = [0., 0., 0., 0., size, 0., size, size, 0., size];
        for (value, bound) in values.into_iter().zip(root.root().enclosure()) {
            assert!(bound[0] <= value && value <= bound[1]);
        }
        assert_eq!(root.canonical_segment_errors().unwrap().len(), 4);
    }
    let after = host.workspace().session(document).unwrap();
    assert_eq!(after.snapshot().sketches.recipe(), recipe);
    assert_eq!(after.generation(), generation);
    assert_eq!(after.undo_labels(), labels);
    assert_eq!(host.workspace().needs_save(document).unwrap(), dirty);
}
#[test]
fn session_rederives_evidence_after_edit_history_and_reopen() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let doc = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let (sketch, anchor) = host
        .commands(doc)
        .unwrap()
        .run("Rectangle", add_rectangle)
        .unwrap();
    assert_current(&mut host, doc, sketch, anchor, 1.);
    host.commands(doc)
        .unwrap()
        .run("Edit size", |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    assert_current(&mut host, doc, sketch, anchor, 2.);
    host.commands(doc).unwrap().undo().unwrap();
    assert_current(&mut host, doc, sketch, anchor, 1.);
    host.commands(doc).unwrap().redo().unwrap();
    assert_current(&mut host, doc, sketch, anchor, 2.);
    host.save(doc, "alpha", &mut source).unwrap();
    host.close_discarding_changes(doc).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_current(&mut host, reopened, sketch, anchor, 2.);
    host.commands(reopened)
        .unwrap()
        .run("Same size source", |snapshot, _| {
            snapshot.parameters.set_expression(1, "2 cm")
        })
        .unwrap();
    assert_current(&mut host, reopened, sketch, anchor, 2.);
    let other = open_file(&mut host, "b.obk", &mut source, OpenActivation::Activate);
    assert!(
        host.commands(other)
            .unwrap()
            .conditional_canonical_sketch(sketch, anchor)
            .is_err()
    );
    assert_current(&mut host, reopened, sketch, anchor, 2.);
    host.save_copy(
        reopened,
        &path("precision-copy.obk"),
        None,
        "alpha",
        &mut source,
    )
    .unwrap();
    let copy = open_file(
        &mut host,
        "precision-copy.obk",
        &mut source,
        OpenActivation::Background,
    );
    assert_current(&mut host, copy, sketch, anchor, 2.);
}

#[test]
fn changed_candidate_is_refused_without_changing_session_or_dirty_state() {
    use oblikovati_model::{history::DocumentEditHistory, identity::RecipeIdAllocator};
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let (sketch, anchor) = host
        .commands(id)
        .unwrap()
        .run("Rectangle", add_rectangle)
        .unwrap();
    let mut snapshot = host.workspace().session(id).unwrap().snapshot().clone();
    let placed = snapshot.sketches.get(sketch).unwrap();
    let mut geometry = placed.geometry().clone();
    let point = geometry.points()[2].id;
    geometry.set_position(point, [1.01, 0.]).unwrap();
    snapshot
        .sketches
        .replace_geometry(sketch, geometry)
        .unwrap();
    let loose = SolveOptions {
        relative_tolerance: 0.1,
        ..Default::default()
    };
    let loose_solve = snapshot
        .solve_one_document_sketch(sketch, loose, || false)
        .unwrap();
    snapshot.sketches = loose_solve.candidate;
    snapshot.require_converged_resolved_sketch(sketch).unwrap();
    assert_eq!(
        snapshot
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .point(point)
            .unwrap()
            .position,
        [1.01, 0.]
    );
    let strict = snapshot
        .solve_one_document_sketch(sketch, SolveOptions::default(), || false)
        .unwrap();
    let strict_x = strict
        .candidate
        .get(sketch)
        .unwrap()
        .geometry()
        .point(point)
        .unwrap()
        .position[0];
    assert!(
        (strict_x - 1.0).abs() < 1e-8,
        "candidate x={strict_x:?}, expected 1 cm"
    );
    assert!(!snapshot.sketches.same_recipe(&strict.candidate));
    let allocator = RecipeIdAllocator::restore(Some(1), &[1]).unwrap();
    let history = DocumentEditHistory::new(snapshot, allocator).unwrap();
    let mut workspace = TestWorkspace::new(RecordingWorkspace::default());
    let mut guid = WorkspaceGuidSource {
        result: Ok(guid(999)),
        calls: 0,
    };
    let document = workspace
        .register(
            history,
            identity(998),
            RecordingObserver::default(),
            &mut guid,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id;
    workspace
        .commands(document)
        .unwrap()
        .set_sketch_visibility(sketch, false)
        .unwrap();
    workspace
        .commands(document)
        .unwrap()
        .set_sketch_visibility(sketch, true)
        .unwrap();
    workspace.commands(document).unwrap().undo().unwrap();
    assert!(workspace.needs_save(document).unwrap());
    assert_eq!(workspace.session(document).unwrap().undo_labels().len(), 1);
    assert_eq!(workspace.session(document).unwrap().redo_labels().len(), 1);
    let before = workspace.session(document).unwrap();
    let recipe = before.snapshot().sketches.recipe();
    let evaluated = before.snapshot().sketches.clone();
    let generation = before.generation();
    let high_water = before.high_water();
    let undo = before
        .undo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect::<Vec<_>>();
    let redo = before
        .redo_labels()
        .into_iter()
        .map(str::to_owned)
        .collect::<Vec<_>>();
    let dirty = workspace.needs_save(document).unwrap();
    let result = workspace
        .commands(document)
        .unwrap()
        .conditional_canonical_sketch(sketch, anchor)
        .err()
        .unwrap();
    assert!(
        result.contains("fresh solve changed authored geometry; explicit recompute required"),
        "unexpected decline: {result}"
    );
    let after = workspace.session(document).unwrap();
    assert_eq!(after.snapshot().sketches.recipe(), recipe);
    assert!(after.snapshot().sketches.same_evaluated_state(&evaluated));
    assert_eq!(after.high_water(), high_water);
    assert_eq!(
        after
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .point(point)
            .unwrap()
            .position,
        [1.01, 0.]
    );
    assert_eq!(after.generation(), generation);
    assert_eq!(after.undo_labels(), undo);
    assert_eq!(after.redo_labels(), redo);
    assert_eq!(workspace.needs_save(document).unwrap(), dirty);
}

#[test]
fn real_prism_feature_reports_non_admitting_precision_obligations_across_history() {
    use oblikovati_model::feature::{
        sketch_prism::{SketchDirectedCurve, SketchPrismDefinition, SketchPrismSelection},
        work_geometry::DatumLength,
    };
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(2000);
    let doc = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let (sketch, anchor) = host
        .commands(doc)
        .unwrap()
        .run("Rectangle", add_rectangle)
        .unwrap();
    let lines = host
        .workspace()
        .session(doc)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .lines()
        .iter()
        .map(|line| SketchDirectedCurve {
            curve: line.id,
            reversed: false,
        })
        .collect();
    let definition = SketchPrismDefinition::new(
        sketch,
        SketchPrismSelection {
            outer: lines,
            holes: vec![],
        },
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(1),
        ],
    )
    .unwrap();
    let feature = host
        .commands(doc)
        .unwrap()
        .create_sketch_prism("Cube", definition)
        .unwrap();
    assert_prism_obligations(&mut host, doc, sketch, anchor, feature);
    host.commands(doc)
        .unwrap()
        .run("Size", |snapshot, _| {
            snapshot.parameters.set_expression(1, "20 mm")
        })
        .unwrap();
    assert_prism_obligations(&mut host, doc, sketch, anchor, feature);
    host.commands(doc).unwrap().undo().unwrap();
    assert_prism_obligations(&mut host, doc, sketch, anchor, feature);
    host.commands(doc).unwrap().redo().unwrap();
    host.save(doc, "alpha", &mut source).unwrap();
    host.close_discarding_changes(doc).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_prism_obligations(&mut host, reopened, sketch, anchor, feature);
}
fn assert_prism_obligations(
    host: &mut Host,
    doc: u64,
    sketch: DocumentSketchId,
    anchor: SketchEntityId,
    feature: oblikovati_model::feature::FeatureId,
) {
    use oblikovati_session::PrismPrecisionObligation::*;
    let mut commands = host.commands(doc).unwrap();
    let evidence = commands
        .conditional_canonical_sketch(sketch, anchor)
        .unwrap();
    let projection = evidence.projection().unwrap();
    let points = projection
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .points();
    let bounds = points
        .iter()
        .flat_map(|p| p.position)
        .map(|v| [v - 0.01, v + 0.01])
        .collect::<Vec<_>>();
    let result = evidence
        .prism_precision_obligations(feature, &bounds)
        .unwrap();
    assert_eq!(
        result,
        [
            ArithmeticPlatform,
            ExtrusionSource,
            ProfileIdentityMapping,
            ExactCoincidenceAdmission,
            OperandToleranceBudget,
            AdmissionCoverage
        ]
    );
    assert!(
        evidence
            .prism_precision_obligations(oblikovati_model::feature::FeatureId(u64::MAX), &bounds)
            .is_err()
    );
    assert!(evidence.prism_precision_obligations(feature, &[]).is_err());
}

#[test]
fn prism_obligations_refuse_a_selected_non_line_identity() {
    use oblikovati_model::feature::{
        sketch_prism::{SketchDirectedCurve, SketchPrismDefinition, SketchPrismSelection},
        work_geometry::DatumLength,
    };
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(3000);
    let doc = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let (sketch, anchor) = host
        .commands(doc)
        .unwrap()
        .run("Rectangle", add_rectangle)
        .unwrap();
    let definition = SketchPrismDefinition::new(
        sketch,
        SketchPrismSelection {
            outer: vec![SketchDirectedCurve {
                curve: anchor,
                reversed: false,
            }],
            holes: vec![],
        },
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(1),
        ],
    )
    .unwrap();
    let feature = host
        .commands(doc)
        .unwrap()
        .create_sketch_prism("Invalid selection", definition)
        .unwrap();
    let mut commands = host.commands(doc).unwrap();
    let evidence = commands
        .conditional_canonical_sketch(sketch, anchor)
        .unwrap();
    let projection = evidence.projection().unwrap();
    let points = projection
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .points();
    let bounds = points
        .iter()
        .flat_map(|p| p.position)
        .map(|v| [v - 0.01, v + 0.01])
        .collect::<Vec<_>>();
    let error = evidence
        .prism_precision_obligations(feature, &bounds)
        .unwrap_err();
    assert!(error.contains("expected a supported line"), "{error}");
}
