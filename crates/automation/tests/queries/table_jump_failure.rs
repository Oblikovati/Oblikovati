// SPDX-License-Identifier: GPL-2.0-only
//! A multi-step typed jump leaves the last accepted real solids synchronized (#4122).
use super::*;
use oblikovati_automation::FileHost;
use oblikovati_model::feature::FeatureId;

#[allow(dead_code)]
#[path = "../../../../test-utilities/rust/selected_sketch_prism.rs"]
mod prism_fixture;

fn prism_package(name: &str, source: bool) -> DocumentPackage {
    let mut snapshot = prism_fixture::selected_prism_document();
    if source {
        snapshot
            .parameters
            .insert_expression(4, "master", USER_PARAM, "2 cm")
            .unwrap();
        snapshot.parameters.set_expression(3, "master").unwrap();
    }
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_display_name(name)
        .with_parameters(&snapshot.parameters, Default::default())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap()
}
fn retain(files: &mut DiscoveryWorkspace, id: u64) {
    let guid = files.workspace().identity(id).unwrap().internal_name;
    let completed = {
        let session = files.workspace().session(id).unwrap();
        let mut evaluator =
            oblikovati_martensite::CertifiedNamedSolidEvaluator::with_optional_policy(
                session.history(),
                guid,
                None,
            );
        session
            .prepare_part_evaluation()
            .evaluate_checkpoints(&mut evaluator)
            .unwrap()
    };
    files
        .commands(id)
        .unwrap()
        .adopt_owned_full_checkpoints(None, completed)
        .unwrap();
}
fn volume(files: &DiscoveryWorkspace, id: u64) -> f64 {
    let session = files.workspace().session(id).unwrap();
    let guid = files.workspace().identity(id).unwrap().internal_name;
    session
        .owned_part_slot(None)
        .unwrap()
        .current(session)
        .unwrap()
        .body(FeatureId(1))
        .unwrap()
        .volume(session.history(), guid)
        .unwrap()
}
fn set_master(files: &mut DiscoveryWorkspace, guids: &mut QueryGuids, expression: &str) {
    let client = Client::new(FileHost::new(files, guids, "test"));
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let mut parameters = view.parameters.clone().unwrap_or_default();
    if parameters.iter().all(|item| item.parameter_id != 4) {
        parameters.push(ModelStateTableParameter {
            parameter_id: 4,
            base: "2 cm".into(),
            cells: Some(
                view.rows
                    .as_ref()
                    .unwrap()
                    .iter()
                    .map(|row| ModelStateTableParameterCell {
                        row: row.ref_.clone(),
                        expression: None,
                    })
                    .collect(),
            ),
        });
    }
    parameters
        .iter_mut()
        .find(|item| item.parameter_id == 4)
        .unwrap()
        .cells
        .as_mut()
        .unwrap()[0]
        .expression = Some(std::boxed::Box::new(expression.into()));
    client
        .model_states()
        .set_table(&SetModelStateTableArgs {
            expected_generation: view.generation,
            rows: view.rows,
            columns: view.columns,
            properties: view.properties,
            suppressions: view.suppressions,
            parameters: Some(parameters),
        })
        .unwrap();
}
#[test]
fn typed_jump_stops_at_last_published_step_when_dependent_replay_fails() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut guids = QueryGuids(0);
    let source = files
        .create(
            Path::new("/source.opd"),
            prism_package("source", true),
            QueryObserver::default(),
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    let target = files
        .create(
            Path::new("/target.opd"),
            prism_package("target", false),
            QueryObserver::default(),
            "test",
            &mut guids,
        )
        .unwrap()
        .session_id;
    files
        .add_derived_table(target, "/source.opd", &["master".into()])
        .unwrap();
    files.activate(source).unwrap();
    let initial = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .history(&TransactionHistoryArgs { document: source })
        .unwrap()
        .position;
    set_master(&mut files, &mut guids, "3 cm");
    set_master(&mut files, &mut guids, "4 cm");
    files
        .commands(target)
        .unwrap()
        .run("Target depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, "master - 2 cm")
        })
        .unwrap();
    retain(&mut files, source);
    retain(&mut files, target);
    let result = Client::new(FileHost::new(&mut files, &mut guids, "test"))
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document: source,
            position: initial,
        });
    assert!(
        result.is_err(),
        "the second dependent replay must fail at zero depth"
    );
    assert_eq!(
        files
            .workspace()
            .session(source)
            .unwrap()
            .undo_labels()
            .len() as i64,
        initial + 1
    );
    assert!((volume(&files, source) - 36.0).abs() < 1e-8);
    assert!((volume(&files, target) - 12.0).abs() < 1e-8);
}
