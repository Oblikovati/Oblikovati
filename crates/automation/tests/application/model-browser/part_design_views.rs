// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::json::EmptyObject;
use oblikovati_automation::SessionHost;

#[test]
fn typed_part_view_copy_rename_and_options_preserve_active_view() {
    let mut workspace = WorkspaceSession::new(QueryObserver::default());
    let id = register(
        &mut workspace,
        UnitsOfMeasure::default(),
        &mut QueryGuids(0),
    );
    workspace
        .commands(id)
        .unwrap()
        .run("Initialize part views", |snapshot, _| {
            snapshot.part_design_views = Some(Default::default());
            Ok::<_, String>(())
        })
        .unwrap();
    let client = Client::new(SessionHost::new(&mut workspace));
    let views = client
        .part_design_views()
        .list(&EmptyObject::default())
        .unwrap()
        .views
        .unwrap();
    assert_eq!(
        views
            .iter()
            .map(|view| view.name.as_str())
            .collect::<Vec<_>>(),
        ["Primary", "Isometric", "Front", "Top", "Right"]
    );
    let copied = client
        .part_design_views()
        .copy(&CopyPartDesignViewArgs {
            source_id: 1,
            name: String::new(),
        })
        .unwrap()
        .view;
    assert_eq!(copied.name, "View1");
    assert!(!copied.active);
    assert!(
        client
            .part_design_views()
            .camera(&PartDesignViewRef { id: copied.id })
            .is_err()
    );
    let camera = PartDesignViewCamera {
        eye: Point {
            x: 0.,
            y: 0.,
            z: 5.,
        },
        target: Point::default(),
        up: Vector {
            x: 0.,
            y: 1.,
            z: 0.,
        },
        scale: 12.,
        near: 0.5,
        far: 250.,
        projection: ORTHOGRAPHIC_PROJECTION,
    };
    client
        .part_design_views()
        .capture_camera(&CapturePartDesignViewCameraArgs {
            id: copied.id,
            camera: camera.clone(),
        })
        .unwrap();
    assert_eq!(
        client
            .part_design_views()
            .camera(&PartDesignViewRef { id: copied.id })
            .unwrap(),
        camera
    );
    let renamed = client
        .part_design_views()
        .rename(&RenamePartDesignViewArgs {
            id: copied.id,
            name: "Review".into(),
        })
        .unwrap()
        .view;
    assert_eq!(renamed.id, copied.id);
    let locked = client
        .part_design_views()
        .set_options(&SetPartDesignViewOptionsArgs {
            id: copied.id,
            locked: Some(std::boxed::Box::new(true)),
            ..Default::default()
        })
        .unwrap()
        .view;
    assert!(locked.locked);
    let mut changed_camera = camera.clone();
    changed_camera.scale = 30.;
    assert!(
        client
            .part_design_views()
            .capture_camera(&CapturePartDesignViewCameraArgs {
                id: copied.id,
                camera: changed_camera
            })
            .is_err()
    );
    assert_eq!(
        client
            .part_design_views()
            .camera(&PartDesignViewRef { id: copied.id })
            .unwrap(),
        camera
    );
    let before = client
        .part_design_views()
        .list(&EmptyObject::default())
        .unwrap();
    assert!(
        client
            .part_design_views()
            .rename(&RenamePartDesignViewArgs {
                id: copied.id,
                name: "Front".into()
            })
            .is_err()
    );
    assert_eq!(
        client
            .part_design_views()
            .list(&EmptyObject::default())
            .unwrap(),
        before
    );
    assert!(before.views.unwrap()[0].active);
}
