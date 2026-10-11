// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    history::RecipeReservations,
    sketch::{LineContactSide, PlanarConstraint, Sketch2D, SketchConstraintId, SketchPlane},
};
use oblikovati_solve::nonlinear::{SolveOptions, SolveTermination};
fn add_sketch(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
) -> Result<DocumentSketchId, String> {
    let sketch = ids.allocate_sketch().map_err(|e| e.to_string())?;
    let a = ids.allocate_sketch_entity().map_err(|e| e.to_string())?;
    let b = ids.allocate_sketch_entity().map_err(|e| e.to_string())?;
    let p = ids.allocate_sketch_entity().map_err(|e| e.to_string())?;
    let line = ids.allocate_sketch_entity().map_err(|e| e.to_string())?;
    let mut geometry = Sketch2D::default();
    for (point, xy) in [(a, [0.0, 0.0]), (b, [4.0, 0.0]), (p, [0.0, 0.0])] {
        geometry.add_point(point, xy).unwrap();
    }
    geometry.add_line(line, [a, b]).unwrap();
    for (id, relation) in [
        (
            1,
            PlanarConstraint::Fix {
                point: a,
                position: [0.0, 0.0],
            },
        ),
        (
            2,
            PlanarConstraint::Fix {
                point: b,
                position: [4.0, 0.0],
            },
        ),
        (3, PlanarConstraint::Vertical { points: [a, p] }),
        (
            4,
            PlanarConstraint::Offset {
                point: p,
                line,
                parameter: 1,
                side: LineContactSide::Left,
                linear_diameter: false,
            },
        ),
    ] {
        geometry
            .add_constraint(SketchConstraintId(id), relation)
            .unwrap();
    }
    snapshot
        .sketches
        .add(sketch, "Driven", SketchPlane::xz(), geometry)
        .map_err(|e| e.to_string())?;
    solve_sketch(snapshot, sketch)?;
    Ok(sketch)
}
fn solve_sketch(snapshot: &mut DocumentSnapshot, id: DocumentSketchId) -> Result<(), String> {
    let result = snapshot
        .sketches
        .get(id)
        .unwrap()
        .geometry()
        .solve(&snapshot.parameters, SolveOptions::default(), || false)
        .map_err(|e| e.to_string())?;
    assert_eq!(result.numerical().termination, SolveTermination::Converged);
    assert_eq!(result.numerical().structure.degrees_of_freedom, 0);
    snapshot
        .sketches
        .replace_geometry(id, result.into_candidate())
        .map_err(|e| e.to_string())
}
fn point_height(host: &Host, id: u64, sketch: DocumentSketchId) -> f64 {
    let placed = host
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap();
    placed
        .plane()
        .to_model(placed.geometry().points()[2].position)
        .unwrap()[2]
}
#[test]
fn solved_sketch_save_reopen_parameter_edit_undo_redo_and_copy_keep_placement_and_identity() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let sketch = host
        .commands(id)
        .unwrap()
        .run("Create sketch", add_sketch)
        .unwrap();
    assert!((point_height(&host, id, sketch) - 1.0).abs() < 1e-8);
    assert!(host.workspace().needs_save(id).unwrap());
    host.save(id, "alpha", &mut source).unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    let identity = host.workspace().identity(id).unwrap().internal_name;
    let saved = host
        .workspace()
        .session(id)
        .unwrap()
        .snapshot()
        .sketches
        .clone();
    host.close_discarding_changes(id).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_eq!(
        host.workspace()
            .session(reopened)
            .unwrap()
            .snapshot()
            .sketches,
        saved
    );
    assert_eq!(
        host.workspace().identity(reopened).unwrap().internal_name,
        identity
    );
    assert!(!host.workspace().needs_save(reopened).unwrap());
    host.commands(reopened)
        .unwrap()
        .run("Width", |snapshot, _| -> Result<(), String> {
            snapshot
                .parameters
                .set_expression(1, "5 cm")
                .map_err(|e| e.to_string())?;
            solve_sketch(snapshot, sketch)
        })
        .unwrap();
    assert!((point_height(&host, reopened, sketch) - 5.0).abs() < 1e-8);
    host.commands(reopened).unwrap().undo().unwrap();
    assert!((point_height(&host, reopened, sketch) - 1.0).abs() < 1e-8);
    assert!(!host.workspace().needs_save(reopened).unwrap());
    host.commands(reopened).unwrap().redo().unwrap();
    assert!((point_height(&host, reopened, sketch) - 5.0).abs() < 1e-8);
    let copy = host
        .save_copy(
            reopened,
            &path("sketch-copy.obk"),
            None,
            "alpha",
            &mut source,
        )
        .unwrap();
    assert_ne!(copy.internal_name, identity);
    assert!(host.workspace().needs_save(reopened).unwrap());
    let copied = open_file(
        &mut host,
        "sketch-copy.obk",
        &mut source,
        OpenActivation::Background,
    );
    assert!((point_height(&host, copied, sketch) - 5.0).abs() < 1e-8);
    host.save(reopened, "alpha", &mut source).unwrap();
    assert!(!host.workspace().needs_save(reopened).unwrap());
}
#[test]
fn failed_reservations_are_dirty_and_persist_even_without_a_sketch_or_history_step() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let error = host
        .commands(id)
        .unwrap()
        .run("Failed", |_, ids| -> Result<(), String> {
            ids.allocate_sketch().unwrap();
            ids.allocate_sketch_entity().unwrap();
            Err("refused".into())
        });
    assert!(error.is_err());
    assert!(host.workspace().needs_save(id).unwrap());
    host.save(id, "alpha", &mut source).unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    host.close_discarding_changes(id).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let allocated = host
        .commands(reopened)
        .unwrap()
        .run("Reserve", |_, ids| -> Result<_, String> {
            Ok((
                ids.allocate_sketch().unwrap(),
                ids.allocate_sketch_entity().unwrap(),
            ))
        })
        .unwrap();
    assert_eq!(allocated.0.0, 2);
    assert_eq!(allocated.1.0, 2);
    let encoded = host.files().contents.borrow()[&path("a.obk")].clone();
    let restored = DocumentPackage::decode(&encoded)
        .unwrap()
        .restore_sketches()
        .unwrap();
    assert!(restored.sketches.sketches().is_empty());
    assert_eq!(restored.sketch_ids.high_water(), 1);
}
#[test]
fn deleted_and_aborted_sketch_ids_stay_reserved_and_failed_storage_keeps_the_saved_recipe() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let sketch = host
        .commands(id)
        .unwrap()
        .run("Create", add_sketch)
        .unwrap();
    host.save(id, "alpha", &mut source).unwrap();
    let bytes = host.files().contents.borrow()[&path("a.obk")].clone();
    host.commands(id)
        .unwrap()
        .run("Delete", |snapshot, _| snapshot.sketches.remove(sketch))
        .unwrap();
    host.commands(id).unwrap().undo().unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().redo().unwrap();
    host.save(id, "alpha", &mut source).unwrap();
    host.commands(id).unwrap().begin("Abort").unwrap();
    host.commands(id)
        .unwrap()
        .run("Create", add_sketch)
        .unwrap();
    host.commands(id).unwrap().abort().unwrap();
    assert!(
        host.workspace()
            .session(id)
            .unwrap()
            .snapshot()
            .sketches
            .sketches()
            .is_empty()
    );
    assert!(host.workspace().needs_save(id).unwrap());
    host.save(id, "alpha", &mut source).unwrap();
    let mut restored = DocumentPackage::decode(&host.files().contents.borrow()[&path("a.obk")])
        .unwrap()
        .restore_sketches()
        .unwrap();
    assert_eq!(restored.sketch_ids.allocate().unwrap(), 3);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 9);
    assert_ne!(host.files().contents.borrow()[&path("a.obk")], bytes);
    // A separate injected file owner refuses storage without touching its retained bytes.
    let mut refusing = transitions::refusing_host(RecordingWorkspace::default());
    let rid = open_file(
        &mut refusing,
        "a.obk",
        &mut source,
        OpenActivation::Activate,
    );
    refusing
        .commands(rid)
        .unwrap()
        .run("Create", add_sketch)
        .unwrap();
    let original = refusing.files().contents.borrow()[&path("a.obk")].clone();
    assert!(refusing.save(rid, "alpha", &mut source).is_err());
    assert!(refusing.workspace().needs_save(rid).unwrap());
    assert_eq!(refusing.files().contents.borrow()[&path("a.obk")], original);
}
#[test]
fn changing_an_authored_signed_zero_is_dirty_and_restorable_through_history() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let sketch = host
        .commands(id)
        .unwrap()
        .run("Create", add_sketch)
        .unwrap();
    host.save(id, "alpha", &mut source).unwrap();
    host.commands(id)
        .unwrap()
        .run("Origin bits", |snapshot, _| {
            let plane = SketchPlane::new(
                [-0.0, 0.0, 0.0],
                [1.0, 0.0, 0.0],
                [0.0, 0.0, 1.0],
                Default::default(),
            )
            .unwrap();
            snapshot.sketches.replace_plane(sketch, plane)
        })
        .unwrap();
    assert!(host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().undo().unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    host.commands(id).unwrap().redo().unwrap();
    host.save(id, "alpha", &mut source).unwrap();
    host.close_discarding_changes(id).unwrap();
    let reopened = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let origin = host
        .workspace()
        .session(reopened)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .plane()
        .recipe()
        .origin;
    assert_eq!(origin[0].to_bits(), (-0.0_f64).to_bits());
}

#[test]
fn template_restores_placed_sketches_and_deleted_reservations_under_a_new_identity() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let original = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    host.commands(original)
        .unwrap()
        .run("Sketch", add_sketch)
        .unwrap();
    host.commands(original)
        .unwrap()
        .run::<(), String>("Reserve", |_, ids| {
            ids.allocate_sketch().unwrap();
            ids.allocate_sketch_entity().unwrap();
            Ok(())
        })
        .unwrap();
    host.save(original, "alpha", &mut source).unwrap();
    let saved = host.files().contents.borrow()[&path("a.obk")].clone();
    let metadata = NewFileMetadata {
        file_name: path("new.obk").to_str().unwrap().into(),
        template_file: path("a.obk").to_str().unwrap().into(),
        display_name: "Template copy".into(),
        sub_type: String::new(),
    };
    let copied = host
        .create_from_template(
            &metadata,
            RecordingObserver::default(),
            "alpha",
            &mut source,
        )
        .unwrap()
        .session_id;
    assert!(
        host.workspace()
            .session(original)
            .unwrap()
            .snapshot()
            .sketches
            .same_recipe(
                &host
                    .workspace()
                    .session(copied)
                    .unwrap()
                    .snapshot()
                    .sketches
            )
    );
    assert_ne!(
        host.workspace().identity(original).unwrap().internal_name,
        host.workspace().identity(copied).unwrap().internal_name
    );
    let next = host
        .commands(copied)
        .unwrap()
        .run::<_, String>("Reserve", |_, ids| {
            Ok((
                ids.allocate_sketch().unwrap().0,
                ids.allocate_sketch_entity().unwrap().0,
            ))
        })
        .unwrap();
    assert_eq!(next, (3, 6));
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], saved);
}

#[test]
fn legacy_sketch_load_refuses_before_document_registration_or_identity_entropy() {
    let mut host = host(RecordingWorkspace::default());
    let mut source = SequenceGuids(1000);
    let bytes = String::from_utf8(fixture(1))
        .unwrap()
        .replace("model:\n", "model:\n  sketches: [{id: 1}]\n")
        .into_bytes();
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), bytes.clone());
    assert!(
        host.open(
            &path("a.obk"),
            RecordingObserver::default(),
            "alpha",
            &mut source,
            OpenActivation::Activate
        )
        .is_err()
    );
    assert!(host.workspace().documents().is_empty());
    assert_eq!(source.0, 1000);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], bytes);
}

#[path = "sketches/recompute.rs"]
mod recompute;

#[path = "sketches/health.rs"]
mod health;

#[path = "sketches/automatic.rs"]
mod automatic;
