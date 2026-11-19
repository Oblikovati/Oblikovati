// SPDX-License-Identifier: GPL-2.0-only
//! Mixed view and table navigation must retain linked solids (#4119, #4122).
use super::*;
use oblikovati_api::types::{PartDesignViewCamera, ViewOrientationTypeEnum};
use oblikovati_automation::PartViewDisplayAccess;
use oblikovati_model::feature::FeatureId;
use oblikovati_persistence::package::DocumentPackage;

struct FakeDisplay(PartDesignViewCamera);
impl PartViewDisplayAccess for FakeDisplay {
    fn camera(&mut self) -> Result<PartDesignViewCamera, String> {
        Ok(self.0.clone())
    }
    fn orient(
        &mut self,
        camera: &PartDesignViewCamera,
        _: ViewOrientationTypeEnum,
    ) -> Result<PartDesignViewCamera, String> {
        let mut target = camera.clone();
        target.scale *= 2.;
        Ok(target)
    }
    fn apply(&mut self, camera: &PartDesignViewCamera) -> Result<(), String> {
        self.0 = camera.clone();
        Ok(())
    }
}
fn fixture_package(source: bool) -> Vec<u8> {
    let mut snapshot = crate::scene_fixture::selected_prism_document();
    if source {
        snapshot
            .parameters
            .insert_expression(4, "master", oblikovati_api::types::USER_PARAM, "2 cm")
            .unwrap();
        snapshot.parameters.set_expression(3, "master").unwrap();
    }
    DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n")
        .unwrap()
        .with_parameters(&snapshot.parameters, Default::default())
        .unwrap()
        .with_sketches(&snapshot.sketches, 1, 104)
        .unwrap()
        .with_work_geometry(&snapshot.work_geometry, 1)
        .unwrap()
        .encode()
        .unwrap()
}
fn open_fixture(owner: &mut TestOwner, name: &str, source: bool) -> (u64, std::path::PathBuf) {
    let path = std::env::temp_dir().join(name);
    owner
        .documents()
        .files()
        .saved
        .borrow_mut()
        .insert(path.clone(), fixture_package(source));
    let id = Client::new(owner.document_host())
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.to_str().unwrap().into(),
            ..Default::default()
        })
        .unwrap()
        .id;
    (id, path)
}
fn setup() -> (TestOwner, u64, u64) {
    let (mut owner, _) = owner();
    let (source, source_path) = open_fixture(&mut owner, "view-table-source.opd", true);
    let (target, _) = open_fixture(&mut owner, "view-table-target.opd", false);
    owner
        .files
        .add_derived_table(target, source_path.to_str().unwrap(), &["master".into()])
        .unwrap();
    owner.files.activate(source).unwrap();
    (owner, source, target)
}
fn edit_master(owner: &mut TestOwner, expression: &str, member: u64) {
    use oblikovati_api::wire::{
        ModelStateTableParameter, ModelStateTableParameterCell, SetModelStateTableArgs,
    };
    let client = Client::new(owner.document_host());
    let view = client
        .model_states()
        .get_table(&Default::default())
        .unwrap();
    let mut parameters = view.parameters.clone().unwrap_or_default();
    if parameters.iter().all(|column| column.parameter_id != 4) {
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
    let cell = parameters
        .iter_mut()
        .find(|column| column.parameter_id == 4)
        .unwrap()
        .cells
        .as_mut()
        .unwrap()
        .iter_mut()
        .find(|cell| cell.row.member_id == member)
        .unwrap();
    cell.expression = Some(Box::new(expression.into()));
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
fn volume(owner: &TestOwner, document: u64) -> f64 {
    let session = owner.files.workspace().session(document).unwrap();
    let policy = session.owned_checkpoint_prefix().unwrap().0;
    let guid = owner
        .files
        .workspace()
        .identity(document)
        .unwrap()
        .internal_name;
    session
        .owned_part_slot(policy)
        .unwrap()
        .current(session)
        .unwrap()
        .body(FeatureId(1))
        .unwrap()
        .volume(session.history(), guid)
        .unwrap()
}
#[test]
fn typed_jump_restores_view_then_stops_before_failed_linked_replay() {
    let (mut owner, source, target) = setup();
    let before = owner
        .files
        .workspace()
        .session(source)
        .unwrap()
        .undo_labels()
        .len() as i64;
    let original = oblikovati_viewport::ViewportCamera::default().part_view_camera();
    let mut display = FakeDisplay(original.clone());
    edit_master(&mut owner, "3 cm", 1);
    Client::new(owner.document_host().with_part_view_display(&mut display))
        .part_design_views()
        .activate(&PartDesignViewRef { id: 3 })
        .unwrap();
    edit_master(&mut owner, "4 cm", 1);
    owner
        .files
        .commands(target)
        .unwrap()
        .run("Dependent depth", |snapshot, _| {
            snapshot.parameters.set_expression(3, "master - 2 cm")
        })
        .unwrap();
    // A current target solid is required to distinguish retained geometry from stale results.
    use oblikovati_automation::{PartReplayService, PartUpdateAccess};
    let mut replay =
        PartReplayService::with_policy(crate::part_scene::resolution_policy::desktop_policy())
            .unwrap();
    replay.ensure_part(&mut owner.files, target).unwrap();
    let result = Client::new(owner.document_host().with_part_view_display(&mut display))
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document: source,
            position: before,
        });
    assert!(
        result.is_err(),
        "the second dependent replay must fail at zero depth"
    );
    assert_eq!(
        owner
            .files
            .workspace()
            .session(source)
            .unwrap()
            .undo_labels()
            .len() as i64,
        before + 1
    );
    assert_eq!(display.0, original);
    assert!((volume(&owner, source) - 36.).abs() < 1e-8);
    assert!((volume(&owner, target) - 12.).abs() < 1e-8);
}
#[test]
fn inactive_table_undo_does_not_bypass_a_busy_linked_reader() {
    let (mut owner, source, target) = setup();
    let second = Client::new(owner.document_host())
        .model_states()
        .create(&CreateModelStateArgs {
            name: "Other".into(),
            ..Default::default()
        })
        .unwrap()
        .model_state
        .id;
    edit_master(&mut owner, "3 cm", second);
    let before = owner
        .files
        .workspace()
        .session(source)
        .unwrap()
        .generation();
    owner
        .files
        .commands(target)
        .unwrap()
        .begin("Busy reader")
        .unwrap();
    let mut display =
        FakeDisplay(oblikovati_viewport::ViewportCamera::default().part_view_camera());
    assert!(
        Client::new(owner.document_host().with_part_view_display(&mut display))
            .transactions()
            .undo()
            .is_err()
    );
    assert_eq!(
        owner
            .files
            .workspace()
            .session(source)
            .unwrap()
            .generation(),
        before
    );
    owner.files.commands(target).unwrap().abort().unwrap();
}
