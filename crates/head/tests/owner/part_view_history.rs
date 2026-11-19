// SPDX-License-Identifier: GPL-2.0-only
//! Verify typed history navigation updates the part view and its live display (#4118).
use super::*;
use oblikovati_api::contract::PartDesignView as _;
use oblikovati_api::types::{PartDesignViewCamera, ViewOrientationTypeEnum};
use oblikovati_automation::PartViewDisplayAccess;

struct FakePartViewDisplay(PartDesignViewCamera);
impl PartViewDisplayAccess for FakePartViewDisplay {
    fn camera(&mut self) -> Result<PartDesignViewCamera, String> {
        Ok(self.0.clone())
    }
    fn orient(
        &mut self,
        camera: &PartDesignViewCamera,
        _: ViewOrientationTypeEnum,
    ) -> Result<PartDesignViewCamera, String> {
        let mut oriented = camera.clone();
        oriented.scale *= 2.;
        Ok(oriented)
    }
    fn apply(&mut self, camera: &PartDesignViewCamera) -> Result<(), String> {
        self.0 = camera.clone();
        Ok(())
    }
}

#[test]
fn typed_undo_redo_and_jump_restore_part_view_and_display() {
    let (mut owner, _) = owner();
    owner.create_part_document().unwrap();
    let document = owner.files.workspace().active_document().unwrap();
    let before = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len();
    let original = oblikovati_viewport::ViewportCamera::default().part_view_camera();
    let mut display = FakePartViewDisplay(original.clone());
    let client = Client::new(owner.document_host().with_part_view_display(&mut display));
    client
        .part_design_views()
        .activate(&PartDesignViewRef { id: 3 })
        .unwrap();
    client.transactions().undo().unwrap();
    client.transactions().redo().unwrap();
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: before as i64,
        })
        .unwrap();
    drop(client);
    assert_eq!(display.0, original);
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(
        session
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .name(),
        "Primary"
    );
    let client = Client::new(owner.document_host().with_part_view_display(&mut display));
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: before as i64 + 1,
        })
        .unwrap();
    drop(client);
    assert_eq!(display.0.scale, original.scale * 2.);
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(
        session
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .name(),
        "Front"
    );
}

#[test]
fn typed_jump_crosses_view_and_sketch_history_without_losing_camera() {
    let (mut owner, _) = owner();
    owner.create_part_document().unwrap();
    let document = owner.files.workspace().active_document().unwrap();
    let start = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len();
    let original = oblikovati_viewport::ViewportCamera::default().part_view_camera();
    let mut display = FakePartViewDisplay(original.clone());
    Client::new(owner.document_host().with_part_view_display(&mut display))
        .part_design_views()
        .activate(&PartDesignViewRef { id: 3 })
        .unwrap();
    owner
        .create_sketch_for_edit(
            document,
            &super::sketch_edit::request(oblikovati_model::sketch::SketchPlane::xy()),
        )
        .unwrap();
    owner.finish_sketch_edit().unwrap();
    let end = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .undo_labels()
        .len();
    assert!(end > start + 1);
    let client = Client::new(owner.document_host().with_part_view_display(&mut display));
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: start as i64,
        })
        .unwrap();
    assert!(
        client
            .sketch()
            .list()
            .unwrap()
            .sketches
            .unwrap_or_default()
            .is_empty()
    );
    client
        .transactions()
        .jump_to(&TransactionJumpToArgs {
            document,
            position: end as i64,
        })
        .unwrap();
    assert_eq!(
        client
            .sketch()
            .list()
            .unwrap()
            .sketches
            .unwrap_or_default()
            .len(),
        1
    );
    client.transactions().undo().unwrap();
    drop(client);
    assert_eq!(display.0.scale, original.scale * 2.);
    let session = owner.files.workspace().session(document).unwrap();
    assert_eq!(
        session
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .name(),
        "Front"
    );
}

#[test]
fn ordinary_geometry_history_preserves_locked_view_transient_display() {
    let (mut owner, _) = owner();
    let document = super::part_scene::open_prism(&mut owner, false);
    let original = oblikovati_viewport::ViewportCamera::default().part_view_camera();
    let mut display = FakePartViewDisplay(original.clone());
    let client = Client::new(owner.document_host().with_part_view_display(&mut display));
    client
        .part_design_views()
        .set_options(&SetPartDesignViewOptionsArgs {
            id: 1,
            locked: Some(Box::new(true)),
            ..Default::default()
        })
        .unwrap();
    client
        .part_design_views()
        .set_body_visibility(&SetPartDesignViewBodyVisibilityArgs {
            id: 1,
            body_keys: None,
            visible: false,
        })
        .unwrap();
    client
        .parameters()
        .set(&ParameterSetArgs {
            name: "width".into(),
            expression: "6 cm".into(),
            ..Default::default()
        })
        .unwrap();
    client.transactions().undo().unwrap();
    let bodies = client
        .body()
        .list(&oblikovati_api::json::EmptyObject::default())
        .unwrap();
    assert!(
        bodies
            .bodies
            .as_ref()
            .is_some_and(|bodies| bodies.len() == 1 && !bodies[0].visible)
    );
    client.transactions().redo().unwrap();
    let bodies = client
        .body()
        .list(&oblikovati_api::json::EmptyObject::default())
        .unwrap();
    assert!(
        bodies
            .bodies
            .as_ref()
            .is_some_and(|bodies| bodies.len() == 1 && !bodies[0].visible)
    );
    drop(client);
    assert_eq!(display.0, original);
    assert!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .part_design_views
            .as_ref()
            .unwrap()
            .active()
            .unwrap()
            .locked()
    );
}
