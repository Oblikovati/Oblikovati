// SPDX-License-Identifier: GPL-2.0-only
use super::super::parameter_value::numeric;
use super::super::sketch_updates::{circle_radius, create_circle};
use super::*;
use oblikovati_automation::{FILE_QUERY_METHODS, FileHost, SKETCH_COMMAND_METHODS};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, PlacedSketchHealth, SketchEvaluation},
    sketch::{PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId},
};
use oblikovati_solve::nonlinear::SolveOptions;

fn create_kind(files: &mut DiscoveryWorkspace, kind: DocumentType, source: &mut QueryGuids) -> u64 {
    let contents = format!(
        "schemaVersion: 2\ndocumentType: {}\nmodel:\n  parameters: [{{name: width, kind: user, expression: '1 cm'}}]\n",
        kind.0
    );
    let package = DocumentPackage::decode(contents.as_bytes()).unwrap();
    files
        .create(
            Path::new("/fixtures/sketch.obk"),
            package,
            QueryObserver::default(),
            "test",
            source,
        )
        .unwrap()
        .session_id
}
pub(super) fn configured_files(
    kind: DocumentType,
    options: SolveOptions,
) -> (DiscoveryWorkspace, u64, QueryGuids) {
    let mut files = DiscoveryWorkspace::new_with_sketch_options(
        QueryObserver::default(),
        DiscoveryFiles::default(),
        options,
    );
    let mut source = QueryGuids(0);
    let document = create_kind(&mut files, kind, &mut source);
    (files, document, source)
}
fn files_with_circle(
    options: SolveOptions,
) -> (DiscoveryWorkspace, u64, DocumentSketchId, QueryGuids) {
    let (mut files, document, source) = configured_files(DOCUMENT_PART, options);
    let sketch = insert_circle(&mut files, document);
    (files, document, sketch, source)
}
fn insert_circle(files: &mut DiscoveryWorkspace, document: u64) -> DocumentSketchId {
    files
        .commands(document)
        .unwrap()
        .run("Create circle", create_circle)
        .unwrap()
}
fn solve_call(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    index: i64,
) -> Result<SolveSketchResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .solve(&SketchArgs {
            sketch_index: index,
        })
}
fn status_call(
    files: &DiscoveryWorkspace,
    index: i64,
) -> Result<ConstraintStatusResult, oblikovati_api::Error> {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .constraint_status(&SketchArgs {
            sketch_index: index,
        })
}
pub(super) fn owner_revision(
    files: &DiscoveryWorkspace,
    document: u64,
) -> (u64, Vec<String>, bool) {
    let session = files.workspace().session(document).unwrap();
    (
        session.generation(),
        session
            .undo_labels()
            .into_iter()
            .map(str::to_owned)
            .collect(),
        files.workspace().needs_save(document).unwrap(),
    )
}
fn point_geometry(point: SketchEntityId, redundant: bool) -> Sketch2D {
    let mut geometry = Sketch2D::default();
    geometry.add_point(point, [2.0, 3.0]).unwrap();
    if redundant {
        for id in [1, 2] {
            geometry
                .add_constraint(
                    SketchConstraintId(id),
                    PlanarConstraint::Fix {
                        point,
                        position: [2.0, 3.0],
                    },
                )
                .unwrap();
        }
    }
    geometry
}
fn replace_with_point(
    files: &mut DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
    redundant: bool,
) {
    files
        .commands(document)
        .unwrap()
        .run("Point constraints", |snapshot, ids| {
            let point = ids
                .allocate_sketch_entity()
                .map_err(|cause| cause.to_string())?;
            snapshot
                .sketches
                .replace_geometry(sketch, point_geometry(point, redundant))
                .map_err(|cause| cause.to_string())
        })
        .unwrap();
}

#[test]
fn typed_query_reads_structure_without_changing_geometry_history_or_saved_state() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    files.save(document, "test", &mut source).unwrap();
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    assert_circle_structure(&status_call(&files, 0).unwrap());
    assert_eq!(before, owner_revision(&files, document));
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
    assert!(SKETCH_COMMAND_METHODS.contains(&METHOD_SKETCH_SOLVE));
    assert!(FILE_QUERY_METHODS.contains(&METHOD_SKETCH_CONSTRAINT_STATUS));
}

#[test]
fn typed_solve_distinguishes_convergence_freedom_and_redundancy_health() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let well = solve_call(&mut files, &mut source, 0).unwrap();
    assert_eq!(well.sketch_index, 0);
    assert_wire_solve(&well, "well", 0, true, true);
    for (redundant, status, dof, healthy) in [(false, "under", 2, true), (true, "over", 0, false)] {
        replace_with_point(&mut files, document, sketch, redundant);
        let solved = solve_call(&mut files, &mut source, 0).unwrap();
        assert_wire_solve(&solved, status, dof, true, healthy);
        let structure = status_call(&files, 0).unwrap();
        assert_eq!((structure.status.as_str(), structure.dof), (status, dof));
        assert_eq!(structure.redundant, if redundant { 2 } else { 0 });
    }
}

#[test]
fn invalid_indices_fail_before_any_owner_command() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let before = owner_revision(&files, document);
    for index in [-1, 1, i64::MAX] {
        expect_api_decline(status_call(&files, index), &index.to_string());
        expect_api_decline(
            solve_call(&mut files, &mut source, index),
            &index.to_string(),
        );
        assert_eq!(before, owner_revision(&files, document));
    }
}

#[test]
fn absent_active_document_and_wrong_kinds_are_refused() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    expect_api_decline(status_call(&files, 0), "active part");
    expect_api_decline(solve_call(&mut files, &mut source, 0), "active part");
    for kind in [DOCUMENT_DRAWING, DOCUMENT_PRESENTATION] {
        let (mut files, document, mut source) = configured_files(kind, Default::default());
        let before = owner_revision(&files, document);
        expect_api_decline(status_call(&files, 0), "kind");
        expect_api_decline(solve_call(&mut files, &mut source, 0), "kind");
        assert_eq!(before, owner_revision(&files, document));
    }
}

#[test]
fn empty_part_and_read_only_host_refuse_invalid_solve_requests() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let before = owner_revision(&files, document);
    assert!(
        status_call(&files, 0)
            .unwrap_err()
            .to_string()
            .contains("0..0")
    );
    assert!(
        solve_call(&mut files, &mut source, 0)
            .unwrap_err()
            .to_string()
            .contains("0..0")
    );
    let client = Client::new(FileQueryHost::new(&files));
    assert!(client.sketch().solve(&SketchArgs::default()).is_err());
    assert_eq!(before, owner_revision(&files, document));
}

#[test]
fn injected_iteration_limit_is_visible_and_does_not_adopt_unconverged_geometry() {
    let options = SolveOptions {
        max_iterations: 0,
        ..Default::default()
    };
    let (mut files, document, sketch, mut source) = files_with_circle(options);
    change_target(&mut files, document, "3 cm");
    let before = owner_revision(&files, document);
    let result = solve_call(&mut files, &mut source, 0).unwrap();
    assert_wire_solve(&result, "well", 0, false, false);
    let placed = owned_geometry(&files, document).get(sketch).unwrap();
    assert_eq!(placed.geometry().circles()[0].radius, 1.0);
    let after = owner_revision(&files, document);
    assert_eq!((before.1, before.2), (after.1, after.2));
    assert!(after.0 > before.0);
}

#[test]
fn local_solver_error_reaches_the_client_and_keeps_failed_health_without_an_undo_step() {
    let options = SolveOptions {
        max_trial_steps: 0,
        ..Default::default()
    };
    let (mut files, document, sketch, mut source) = files_with_circle(options);
    verify_with_default_policy(&mut files, document, sketch);
    let labels = owner_revision(&files, document).1;
    expect_api_decline(solve_call(&mut files, &mut source, 0), "max_trial_steps 0");
    assert_failed_health(&files, document, sketch);
    assert_eq!(labels, owner_revision(&files, document).1);
}

#[test]
fn injected_rank_policy_errors_do_not_change_read_only_query_state() {
    let options = SolveOptions {
        rank_tolerance: 0.0,
        ..Default::default()
    };
    let (files, document, _, _) = files_with_circle(options);
    let before = owner_revision(&files, document);
    assert!(status_call(&files, 0).is_err());
    assert_eq!(before, owner_revision(&files, document));
}

#[test]
fn an_existing_empty_sketch_is_well_constrained_and_healthy() {
    let (mut files, document, sketch, mut source) = files_with_circle(Default::default());
    let mut commands = files.commands(document).unwrap();
    commands
        .run("Empty sketch", |snapshot, _| {
            snapshot
                .sketches
                .replace_geometry(sketch, Sketch2D::default())
        })
        .unwrap();
    let result = solve_call(&mut files, &mut source, 0).unwrap();
    assert_wire_solve(&result, "well", 0, true, true);
    let structure = status_call(&files, 0).unwrap();
    assert_eq!(structure.variables, 0);
    assert_eq!(structure.equations, 0);
    assert_eq!(structure.redundant, 0);
}

#[path = "sketch_api/history.rs"]
mod history;

#[path = "sketch_api/delete.rs"]
mod delete;

fn owned_geometry(
    files: &DiscoveryWorkspace,
    document: u64,
) -> &oblikovati_model::document_sketch::DocumentSketches {
    &files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
}
fn expect_api_decline<T: std::fmt::Debug>(
    response: Result<T, oblikovati_api::Error>,
    expected: &str,
) {
    let message = response.unwrap_err().to_string();
    assert!(
        message.contains(expected),
        "decline {message:?}: expected {expected:?}"
    );
}
fn assert_wire_solve(
    result: &SolveSketchResult,
    status: &str,
    dof: i64,
    converged: bool,
    healthy: bool,
) {
    assert_eq!(
        (
            result.status.as_str(),
            result.dof,
            result.converged,
            result.healthy
        ),
        (status, dof, converged, healthy)
    );
}
fn assert_circle_structure(result: &ConstraintStatusResult) {
    assert_eq!(
        result,
        &ConstraintStatusResult {
            status: "well".into(),
            dof: 0,
            variables: 3,
            equations: 3,
            redundant: 0,
        }
    );
}
fn change_target(files: &mut DiscoveryWorkspace, document: u64, expression: &str) {
    files
        .commands(document)
        .unwrap()
        .run("Target", |snapshot, _| {
            snapshot.parameters.set_expression(1, expression)
        })
        .unwrap();
}
fn verify_with_default_policy(
    files: &mut DiscoveryWorkspace,
    document: u64,
    sketch: DocumentSketchId,
) {
    files
        .commands(document)
        .unwrap()
        .recompute_sketches(Default::default(), || false)
        .unwrap();
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    assert!(
        snapshot
            .sketches
            .get(sketch)
            .unwrap()
            .require_fully_constrained(&snapshot.parameters)
            .is_ok()
    );
}
fn assert_failed_health(files: &DiscoveryWorkspace, document: u64, sketch: DocumentSketchId) {
    let snapshot = files.workspace().session(document).unwrap().snapshot();
    let placed = snapshot.sketches.get(sketch).unwrap();
    assert!(matches!(
        placed.health(&snapshot.parameters),
        PlacedSketchHealth::Current(SketchEvaluation::Failed(_))
    ));
}

#[test]
fn ordered_indices_select_stable_sketches_after_an_earlier_sketch_is_removed() {
    let (mut files, document, first, mut source) = files_with_circle(Default::default());
    let second = insert_circle(&mut files, document);
    replace_with_point(&mut files, document, second, false);
    let result = solve_call(&mut files, &mut source, 1).unwrap();
    assert_eq!(result.sketch_index, 1);
    assert_wire_solve(&result, "under", 2, true, true);
    assert_eq!(status_call(&files, 0).unwrap().status, "well");
    remove_sketch(&mut files, document, first);
    let result = solve_call(&mut files, &mut source, 0).unwrap();
    assert_eq!(result.sketch_index, 0);
    assert_wire_solve(&result, "under", 2, true, true);
    assert_eq!(owned_geometry(&files, document).sketches()[0].id(), second);
}
fn remove_sketch(files: &mut DiscoveryWorkspace, document: u64, sketch: DocumentSketchId) {
    files
        .commands(document)
        .unwrap()
        .run("Remove sketch", |snapshot, _| {
            snapshot.sketches.remove(sketch)
        })
        .unwrap();
}
pub(super) fn reopen_with_observer(
    files: &mut DiscoveryWorkspace,
    document: u64,
    source: &mut QueryGuids,
    observer: QueryObserver,
) -> u64 {
    files.save(document, "test", source).unwrap();
    files.close_discarding_changes(document).unwrap();
    files
        .open(
            Path::new("/fixtures/sketch.obk"),
            observer,
            "test",
            source,
            OpenActivation::Activate,
        )
        .unwrap()
        .session_id()
}
#[test]
fn typed_solve_respects_group_veto_and_restores_geometry() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let observer = QueryObserver {
        veto: true,
        ..Default::default()
    };
    let document = reopen_with_observer(&mut files, document, &mut source, observer);
    let before = owner_revision(&files, document);
    let geometry = owned_geometry(&files, document).clone();
    vetoed_api_group(&mut files, &mut source);
    let after = owner_revision(&files, document);
    assert_eq!((before.1, before.2), (after.1, after.2));
    assert!(geometry.same_evaluated_state(owned_geometry(&files, document)));
}
#[test]
fn malformed_shared_requests_fail_before_mutation() {
    let (mut files, document, _, mut source) = files_with_circle(Default::default());
    let before = owner_revision(&files, document);
    {
        let host = FileHost::new(&mut files, &mut source, "test");
        expect_api_decline(
            host.call(METHOD_SKETCH_SOLVE, br#"{"sketchIndex":"invalid"}"#),
            METHOD_SKETCH_SOLVE,
        );
        expect_api_decline(
            host.call(METHOD_SKETCH_CONSTRAINT_STATUS, b"{"),
            METHOD_SKETCH_CONSTRAINT_STATUS,
        );
    }
    assert_eq!(before, owner_revision(&files, document));
}

fn vetoed_api_group(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) {
    let client = Client::new(FileHost::new(files, source, "test"));
    client
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Veto solve group".into(),
        })
        .unwrap();
    client.parameters().set(&numeric("width", "4 cm")).unwrap();
    assert!(
        client
            .sketch()
            .solve(&SketchArgs::default())
            .unwrap()
            .converged
    );
    expect_api_decline(client.transactions().end(), "locked");
}

#[path = "sketch_api/create.rs"]
mod create;

#[path = "sketch_api/references.rs"]
mod references;

#[path = "sketch_api/entities.rs"]
mod entities;

#[path = "sketch_api/construction.rs"]
mod construction;

#[path = "sketch_api/primitive.rs"]
mod primitive;

#[path = "sketch_api/translation.rs"]
mod translation;
