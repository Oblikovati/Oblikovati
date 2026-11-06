// SPDX-License-Identifier: GPL-2.0-only
use super::{recompute::saved_driven_sketch, *};
use oblikovati_model::document_sketch::{
    PlacedSketchHealth, SketchEvaluation, SketchReadinessCause,
};
use oblikovati_model::sketch::SketchEntityId;
use oblikovati_solve::nonlinear::SolveTermination;

fn replace_disk_parameters(host: &Host, expression: &str) {
    let bytes = host.files().contents.borrow()[&path("a.obk")].clone();
    let package = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = package.restore_document_parameters().unwrap();
    restored.parameters.set_expression(1, expression).unwrap();
    let package = package
        .with_parameters(&restored.parameters, restored.parameters.units().clone())
        .unwrap();
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), package.encode().unwrap());
}
fn reopen_saved(host: &mut Host, source: &mut SequenceGuids, id: u64) -> u64 {
    host.close_discarding_changes(id).unwrap();
    open_file(host, "a.obk", source, OpenActivation::Activate)
}
fn assert_ready_without_undo(host: &Host, id: u64, sketch: DocumentSketchId) {
    let session = host.workspace().session(id).unwrap();
    session.snapshot().require_resolved_sketch(sketch).unwrap();
    assert!(session.undo_labels().is_empty());
    assert!(session.redo_labels().is_empty());
    assert!(session.observer().observations.is_empty());
}
#[test]
fn repaired_coordinates_keep_the_loaded_recipe_dirty_until_save() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    replace_disk_parameters(&host, "6 cm");
    let before = host.files().contents.borrow()[&path("a.obk")].clone();
    let id = reopen_saved(&mut host, &mut source, id);
    assert_ready_without_undo(&host, id, sketch);
    assert!((point_height(&host, id, sketch) - 6.0).abs() < 1e-8);
    assert!(host.workspace().needs_save(id).unwrap());
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], before);
    host.save(id, "alpha", &mut source).unwrap();
    assert!(!host.workspace().needs_save(id).unwrap());
    let id = reopen_saved(&mut host, &mut source, id);
    assert_ready_without_undo(&host, id, sketch);
    assert!(!host.workspace().needs_save(id).unwrap());
}
#[test]
fn parameter_failure_is_local_health_and_keeps_authored_file_state_clean() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    replace_disk_parameters(&host, "missing");
    let id = reopen_saved(&mut host, &mut source, id);
    assert_failed_parameter(&host, id, sketch);
}
fn assert_failed_parameter(host: &Host, id: u64, sketch: DocumentSketchId) {
    let session = host.workspace().session(id).unwrap();
    let snapshot = session.snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    let health = placed.health(&snapshot.parameters);
    assert!(matches!(
        health,
        PlacedSketchHealth::Current(SketchEvaluation::Failed(_))
    ));
    let error = placed
        .require_fully_constrained(&snapshot.parameters)
        .unwrap_err();
    assert!(matches!(error.cause, SketchReadinessCause::Failed(_)));
    assert!(!host.workspace().needs_save(id).unwrap());
    assert!(session.undo_labels().is_empty());
}
fn policy_host(bytes: Vec<u8>, options: SolveOptions) -> Host {
    let files = WorkspaceFiles::default();
    files.contents.borrow_mut().insert(path("a.obk"), bytes);
    FileWorkspace::new_with_sketch_options(RecordingWorkspace::default(), files, options)
}
#[test]
fn injected_bounded_policy_is_used_for_open_and_later_reader_edits() {
    let (original, _, _, sketch) = saved_driven_sketch();
    replace_disk_parameters(&original, "6 cm");
    let bytes = original.files().contents.borrow()[&path("a.obk")].clone();
    let options = SolveOptions {
        max_iterations: 0,
        ..Default::default()
    };
    let mut host = policy_host(bytes, options);
    let mut source = SequenceGuids(2000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    assert_iteration_limit(&host, id, sketch);
    assert!(!host.workspace().needs_save(id).unwrap());
    change_width(&mut host, id, "7 cm");
    assert_iteration_limit(&host, id, sketch);
    assert!((point_height(&host, id, sketch) - 1.0).abs() < 1e-8);
}
fn assert_iteration_limit(host: &Host, id: u64, sketch: DocumentSketchId) {
    let snapshot = host.workspace().session(id).unwrap().snapshot();
    let error = snapshot
        .sketches
        .get(sketch)
        .unwrap()
        .require_fully_constrained(&snapshot.parameters)
        .unwrap_err();
    assert_eq!(
        error.cause,
        SketchReadinessCause::Unconverged(SolveTermination::IterationLimit)
    );
}
#[test]
fn created_package_is_verified_before_the_owner_becomes_active() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    let package = DocumentPackage::decode(&host.files().contents.borrow()[&path("a.obk")]).unwrap();
    let created = host
        .create_unbound(package, RecordingObserver::default(), "alpha", &mut source)
        .unwrap()
        .session_id;
    assert_ready_without_undo(&host, created, sketch);
    assert_eq!(host.workspace().active_document(), Some(created));
    assert!(host.workspace().needs_save(created).unwrap());
    assert!(!host.workspace().needs_save(id).unwrap());
}
fn change_width(host: &mut Host, id: u64, expression: &str) {
    host.commands(id)
        .unwrap()
        .run("Width", |snapshot, _| {
            snapshot.parameters.set_expression(1, expression)
        })
        .unwrap();
}
#[test]
fn invalid_numerical_policy_is_retained_as_a_local_verification_failure() {
    let (original, _, _, sketch) = saved_driven_sketch();
    let bytes = original.files().contents.borrow()[&path("a.obk")].clone();
    let options = SolveOptions {
        rank_tolerance: -1.0,
        ..Default::default()
    };
    let mut host = policy_host(bytes, options);
    let mut source = SequenceGuids(2000);
    let id = open_file(&mut host, "a.obk", &mut source, OpenActivation::Activate);
    let snapshot = host.workspace().session(id).unwrap().snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    let error = placed
        .require_fully_constrained(&snapshot.parameters)
        .unwrap_err();
    assert!(matches!(error.cause, SketchReadinessCause::Failed(_)));
    assert!(error.to_string().contains("rank_tolerance -1"));
    assert!(!host.workspace().needs_save(id).unwrap());
}
#[test]
fn template_creation_rebuilds_verification_without_editing_the_source_file() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    let before = host.files().contents.borrow()[&path("a.obk")].clone();
    let metadata = verification_template();
    let created = host
        .create_from_template(
            &metadata,
            RecordingObserver::default(),
            "alpha",
            &mut source,
        )
        .unwrap()
        .session_id;
    assert_ready_without_undo(&host, created, sketch);
    assert_eq!(host.files().contents.borrow()[&path("a.obk")], before);
    assert!(!host.workspace().needs_save(id).unwrap());
}

fn verification_template() -> oblikovati_api::types::NewFileMetadata {
    oblikovati_api::types::NewFileMetadata {
        file_name: path("verified-template.obk").to_str().unwrap().into(),
        template_file: path("a.obk").to_str().unwrap().into(),
        ..Default::default()
    }
}
fn replace_disk_geometry(host: &Host, sketch: DocumentSketchId, geometry: Sketch2D) {
    let bytes = host.files().contents.borrow()[&path("a.obk")].clone();
    let package = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = package.restore_sketches().unwrap();
    restored
        .sketches
        .replace_geometry(sketch, geometry)
        .unwrap();
    let package = package
        .with_sketches(
            &restored.sketches,
            restored.sketch_ids.high_water(),
            restored.entity_ids.high_water(),
        )
        .unwrap();
    host.files()
        .contents
        .borrow_mut()
        .insert(path("a.obk"), package.encode().unwrap());
}
#[test]
fn converged_free_geometry_is_verified_but_refused_by_checked_consumers() {
    let (mut host, mut source, id, sketch) = saved_driven_sketch();
    let mut geometry = Sketch2D::default();
    geometry.add_point(SketchEntityId(1), [2.0, 3.0]).unwrap();
    replace_disk_geometry(&host, sketch, geometry);
    let id = reopen_saved(&mut host, &mut source, id);
    let snapshot = host.workspace().session(id).unwrap().snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    assert!(matches!(
        placed.health(&snapshot.parameters),
        PlacedSketchHealth::Current(SketchEvaluation::Numerical(_))
    ));
    let error = placed
        .require_fully_constrained(&snapshot.parameters)
        .unwrap_err();
    assert_eq!(error.cause, SketchReadinessCause::FreeVariables(2));
    assert!(!host.workspace().needs_save(id).unwrap());
}
