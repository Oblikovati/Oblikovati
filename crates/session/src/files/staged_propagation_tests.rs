// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::workspace::WorkspaceObserver;
use oblikovati_api::wire::TransactionEventPayload;
use oblikovati_model::identity::{DocumentGuid, IdentityRegistration, IdentityTransitionError};
use oblikovati_persistence::file_access::NativeDocumentFiles;

struct QuietDocument;
impl TransactionObserver for QuietDocument {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}
    fn changed(&mut self, _: u64, _: u64) {}
}

struct QuietWorkspace;
impl WorkspaceObserver for QuietWorkspace {
    fn interest_client_present(&self, _: &str) -> bool {
        false
    }
    fn unavailable_interest_clients(&mut self, _: u64, _: &str, _: &[String]) {}
    fn created(&mut self, _: u64, _: IdentityRegistration) {}
    fn path_changed(&mut self, _: u64, _: &Path, _: &Path) {}
    fn before_open(&mut self, _: &Path) -> Result<(), String> {
        Ok(())
    }
    fn before_save(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn saved(&mut self, _: u64) {}
    fn opened(&mut self, _: u64, _: IdentityRegistration) {}
    fn activated(&mut self, _: u64) {}
    fn before_close(&mut self, _: u64, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }
    fn closed(&mut self, _: u64, _: Option<u64>) {}
}

struct SequenceGuids(u64);
impl GuidSource for SequenceGuids {
    fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
        self.0 += 1;
        Ok(format!("00000000-0000-4000-8000-{:012x}", self.0)
            .parse()
            .unwrap())
    }
}

type Host = FileWorkspace<QuietDocument, QuietWorkspace, NativeDocumentFiles>;
fn package(name: &str, parameters: &str) -> DocumentPackage {
    let yaml = format!(
        "schemaVersion: 2\ndocumentType: 1\nidentity: {{internalName: 00000000-0000-4000-8000-000000000001}}\ndata: {{marker: AQID}}\nmodel:\n  parameters: {parameters}\n"
    );
    DocumentPackage::decode(yaml.as_bytes())
        .unwrap()
        .with_display_name(name)
}
fn linked_documents() -> (Host, u64, u64) {
    let mut host = FileWorkspace::new(QuietWorkspace, NativeDocumentFiles);
    let mut guids = SequenceGuids(100);
    let source = host
        .create_unbound(
            package(
                "source",
                "[{name: width, kind: user, expression: '1 cm', export: true}]",
            ),
            QuietDocument,
            "alpha",
            &mut guids,
        )
        .unwrap()
        .session_id;
    let target = host
        .create_unbound(package("target", "[]"), QuietDocument, "alpha", &mut guids)
        .unwrap()
        .session_id;
    host.add_derived_table(target, "source", &["width".into()])
        .unwrap();
    (host, source, target)
}
fn width(snapshot: &DocumentSnapshot) -> f64 {
    let id = snapshot.parameters.find("width").unwrap();
    snapshot
        .parameters
        .get(id)
        .unwrap()
        .current_value()
        .unwrap()
        .value
}

#[test]
fn prepared_reader_uses_candidate_without_updating_the_live_documents() {
    let (host, source, target) = linked_documents();
    let mut candidate = host.workspace.session(source).unwrap().snapshot().clone();
    candidate.parameters.set_expression(1, "2 cm").unwrap();
    let staged = host.staged_derived_snapshots(source, candidate).unwrap();
    assert_eq!(width(staged.get(&target).unwrap()), 2.0);
    assert_eq!(
        width(host.workspace.session(source).unwrap().snapshot()),
        1.0
    );
    assert_eq!(
        width(host.workspace.session(target).unwrap().snapshot()),
        1.0
    );
}

#[test]
fn refused_reader_preparation_leaves_source_and_reader_unchanged() {
    let (mut host, source, target) = linked_documents();
    let mut candidate = host.workspace.session(source).unwrap().snapshot().clone();
    candidate.parameters.set_expression(1, "2 cm").unwrap();
    host.commands(target).unwrap().begin("busy").unwrap();
    assert!(host.staged_derived_snapshots(source, candidate).is_err());
    assert_eq!(
        width(host.workspace.session(source).unwrap().snapshot()),
        1.0
    );
    assert_eq!(
        width(host.workspace.session(target).unwrap().snapshot()),
        1.0
    );
    host.commands(target).unwrap().abort().unwrap();
}

#[test]
fn workspace_publishes_a_linked_source_and_reader_together() {
    let (mut host, source, target) = linked_documents();
    let plan = host.staged_derived_plan(source).unwrap();
    let readers = plan.reader_ids();
    host.workspace
        .run_named_prepared(
            source,
            None,
            readers,
            |snapshot, _| {
                snapshot
                    .parameters
                    .set_expression(1, "2 cm")
                    .map_err(|error| error.to_string())
            },
            |_, (), _| Ok(()),
            |snapshot| {
                plan.evaluate(snapshot.clone())
                    .map_err(|error| error.to_string())
            },
        )
        .unwrap();
    assert_eq!(
        width(host.workspace.session(source).unwrap().snapshot()),
        2.0
    );
    assert_eq!(
        width(host.workspace.session(target).unwrap().snapshot()),
        2.0
    );
    assert_eq!(
        host.workspace.session(source).unwrap().undo_labels(),
        ["Edit Model State Table"]
    );
}

#[test]
fn workspace_refresh_failure_keeps_linked_documents_unchanged() {
    let (mut host, source, target) = linked_documents();
    let plan = host.staged_derived_plan(source).unwrap();
    let readers = plan.reader_ids();
    let source_generation = host.workspace.session(source).unwrap().generation();
    let target_generation = host.workspace.session(target).unwrap().generation();
    let result = host.workspace.run_named_prepared(
        source,
        None,
        readers,
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(1, "2 cm")
                .map_err(|error| error.to_string())
        },
        |_, (), _| Ok(()),
        |_| Err("injected dependent refresh failure".into()),
    );
    assert!(result.is_err());
    assert_eq!(
        host.workspace.session(source).unwrap().generation(),
        source_generation
    );
    assert_eq!(
        host.workspace.session(target).unwrap().generation(),
        target_generation
    );
    assert_eq!(
        width(host.workspace.session(source).unwrap().snapshot()),
        1.0
    );
    assert_eq!(
        width(host.workspace.session(target).unwrap().snapshot()),
        1.0
    );
}

fn prism_package(name: &str, source: bool) -> DocumentPackage {
    let mut snapshot = crate::prepared_command::fixture::selected_prism_document();
    if source {
        snapshot
            .parameters
            .insert_expression(4, "master", oblikovati_api::types::USER_PARAM, "2 cm")
            .unwrap();
        snapshot.parameters.set_expression(3, "master").unwrap();
    }
    package(name, "[]")
        .with_parameters(&snapshot.parameters, Default::default())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap()
}
fn linked_prism_documents(target_expression: &str) -> (Host, u64, u64) {
    let mut host = FileWorkspace::new(QuietWorkspace, NativeDocumentFiles);
    let mut guids = SequenceGuids(100);
    let source = host
        .create_unbound(
            prism_package("source", true),
            QuietDocument,
            "alpha",
            &mut guids,
        )
        .unwrap()
        .session_id;
    let target = host
        .create_unbound(
            prism_package("target", false),
            QuietDocument,
            "alpha",
            &mut guids,
        )
        .unwrap()
        .session_id;
    host.add_derived_table(target, "source", &["master".into()])
        .unwrap();
    host.commands(target)
        .unwrap()
        .run("Link depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, target_expression)
        })
        .unwrap();
    retain_prism(&mut host, source);
    retain_prism(&mut host, target);
    (host, source, target)
}
fn retain_prism(host: &mut Host, document: u64) {
    let guid = host.workspace.identity(document).unwrap().internal_name;
    let completed = {
        let session = host.workspace.session(document).unwrap();
        let mut evaluator =
            oblikovati_martensite::CertifiedNamedSolidEvaluator::with_optional_policy(
                &session.history,
                guid,
                None,
            );
        session
            .prepare_part_evaluation()
            .evaluate_checkpoints(&mut evaluator)
            .unwrap()
    };
    host.commands(document)
        .unwrap()
        .adopt_owned_full_checkpoints(None, completed)
        .unwrap();
}
fn part_volume(host: &Host, document: u64) -> f64 {
    let session = host.workspace.session(document).unwrap();
    let guid = host.workspace.identity(document).unwrap().internal_name;
    session
        .owned_part_slot(None)
        .unwrap()
        .current(session)
        .unwrap()
        .body(oblikovati_model::feature::FeatureId(1))
        .unwrap()
        .volume(&session.history, guid)
        .unwrap()
}

#[test]
fn file_workspace_publishes_source_and_linked_real_solids() {
    let (mut host, source, target) = linked_prism_documents("master");
    host.run_named_prepared(
        source,
        None,
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(4, "3 cm")
                .map_err(|error| error.to_string())
        },
        |_, (), _| Ok(()),
    )
    .unwrap();
    assert!((part_volume(&host, source) - 36.0).abs() < 1e-8);
    assert!((part_volume(&host, target) - 36.0).abs() < 1e-8);
}

#[test]
fn failed_linked_solid_replay_preserves_both_real_solids() {
    let (mut host, source, target) = linked_prism_documents("4 cm - master");
    let source_generation = host.workspace.session(source).unwrap().generation();
    let target_generation = host.workspace.session(target).unwrap().generation();
    let result = host.run_named_prepared(
        source,
        None,
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(4, "4 cm")
                .map_err(|error| error.to_string())
        },
        |_, (), _| Ok(()),
    );
    assert!(result.is_err());
    assert_eq!(
        host.workspace.session(source).unwrap().generation(),
        source_generation
    );
    assert_eq!(
        host.workspace.session(target).unwrap().generation(),
        target_generation
    );
    assert!((part_volume(&host, source) - 24.0).abs() < 1e-8);
    assert!((part_volume(&host, target) - 24.0).abs() < 1e-8);
}

fn change_master(host: &mut Host, source: u64, expression: &str) {
    host.run_named_prepared(
        source,
        None,
        |snapshot, _| {
            snapshot
                .parameters
                .set_expression(4, expression)
                .map_err(|error| error.to_string())
        },
        |_, (), _| Ok(()),
    )
    .unwrap();
}
fn set_target_depth(host: &mut Host, target: u64, expression: &str) {
    host.commands(target)
        .unwrap()
        .run("Target depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, expression)
        })
        .unwrap();
    retain_prism(host, target);
}

#[test]
fn linked_real_solids_follow_atomic_undo_and_redo() {
    use oblikovati_model::history::PreparedNavigation;
    let (mut host, source, target) = linked_prism_documents("master");
    change_master(&mut host, source, "3 cm");
    host.navigate_named_prepared(source, None, PreparedNavigation::Undo)
        .unwrap();
    assert!((part_volume(&host, source) - 24.0).abs() < 1e-8);
    assert!((part_volume(&host, target) - 24.0).abs() < 1e-8);
    host.navigate_named_prepared(source, None, PreparedNavigation::Redo)
        .unwrap();
    assert!((part_volume(&host, source) - 36.0).abs() < 1e-8);
    assert!((part_volume(&host, target) - 36.0).abs() < 1e-8);
}

#[test]
fn refused_linked_undo_preserves_history_and_both_real_solids() {
    use oblikovati_model::history::PreparedNavigation;
    let (mut host, source, target) = linked_prism_documents("master");
    change_master(&mut host, source, "3 cm");
    set_target_depth(&mut host, target, "master - 2 cm");
    let source_generation = host.workspace.session(source).unwrap().generation();
    let target_generation = host.workspace.session(target).unwrap().generation();
    assert!(
        host.navigate_named_prepared(source, None, PreparedNavigation::Undo)
            .is_err()
    );
    assert_eq!(
        host.workspace.session(source).unwrap().generation(),
        source_generation
    );
    assert_eq!(
        host.workspace.session(target).unwrap().generation(),
        target_generation
    );
    assert!((part_volume(&host, source) - 36.0).abs() < 1e-8);
    assert!((part_volume(&host, target) - 12.0).abs() < 1e-8);
    assert_eq!(
        host.workspace
            .session(source)
            .unwrap()
            .undo_labels()
            .last()
            .copied(),
        Some("Edit Model State Table")
    );
}

#[test]
fn refused_linked_redo_preserves_history_and_both_real_solids() {
    use oblikovati_model::history::PreparedNavigation;
    let (mut host, source, target) = linked_prism_documents("master");
    change_master(&mut host, source, "3 cm");
    host.navigate_named_prepared(source, None, PreparedNavigation::Undo)
        .unwrap();
    set_target_depth(&mut host, target, "3 cm - master");
    let source_generation = host.workspace.session(source).unwrap().generation();
    let target_generation = host.workspace.session(target).unwrap().generation();
    assert!(
        host.navigate_named_prepared(source, None, PreparedNavigation::Redo)
            .is_err()
    );
    assert_eq!(
        host.workspace.session(source).unwrap().generation(),
        source_generation
    );
    assert_eq!(
        host.workspace.session(target).unwrap().generation(),
        target_generation
    );
    assert!((part_volume(&host, source) - 24.0).abs() < 1e-8);
    assert!((part_volume(&host, target) - 12.0).abs() < 1e-8);
    assert_eq!(
        host.workspace
            .session(source)
            .unwrap()
            .redo_labels()
            .first()
            .copied(),
        Some("Edit Model State Table")
    );
}

#[path = "staged_verification_tests.rs"]
mod verification;

#[path = "staged_assembly_tests.rs"]
mod assembly;
