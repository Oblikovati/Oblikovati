// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    json::EmptyObject,
    wire::{
        AddBaseViewArgs, AddCenterlinesArgs, DeleteViewArgs, DrawingAnnotationInfo, RotateViewArgs,
        SetModelReferenceArgs,
    },
};
use oblikovati_gui::{GuiDocumentTemplate, GuiDrawingLine, GuiTemplateKind, GuiTemplateUnits};

const CENTER: [f64; 2] = [120., 90.];

#[test]
fn drawing_centerlines_track_view_bounds_and_restore_with_history() {
    let (mut owner, _) = owner();
    super::super::part_scene::open_prism(&mut owner, false);
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let client = Client::new(owner.document_host());
    add_top_view(&client);
    assert!(
        client
            .drawing_annotations()
            .add_centerlines(&AddCenterlinesArgs {
                view_name: "Missing".into(),
                ..Default::default()
            })
            .is_err()
    );
    let info = client
        .drawing_annotations()
        .add_centerlines(&AddCenterlinesArgs {
            name: "Axes".into(),
            view_name: "Top".into(),
        })
        .unwrap()
        .annotation;
    assert_eq!(
        (info.kind.as_str(), info.view_name.as_str()),
        ("centerline", "Top")
    );
    assert!(info.curve_count > 2);
    drop(client);
    let initial = centerline_segments(&mut owner, drawing);
    assert_eq!(initial.len() as i64, info.curve_count);
    let client = Client::new(owner.document_host());
    client
        .drawing_views()
        .rotate(&RotateViewArgs {
            name: "Top".into(),
            angle_deg: 45.,
        })
        .unwrap();
    drop(client);
    let rotated = centerline_segments(&mut owner, drawing);
    assert_ne!(rotated, initial);
    let client = Client::new(owner.document_host());
    client.transactions().undo().unwrap();
    drop(client);
    assert_eq!(centerline_segments(&mut owner, drawing), initial);
    let client = Client::new(owner.document_host());
    client.transactions().redo().unwrap();
    verify_view_delete(&client, rotated.len() as i64);
    let path = document_fixture_path!("/memory/centerlines.odd");
    let reopened = super::drawing_custom_table::reopen(&client, drawing, path);
    assert_eq!(listed_centerline(&client).curve_count, rotated.len() as i64);
    drop(client);
    assert_eq!(centerline_segments(&mut owner, reopened), rotated);
    let client = Client::new(owner.document_host());
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: reopened,
            force: true,
        })
        .unwrap();
    drop(client);
    let deferred = owner
        .document_host()
        .open_deferred_drawing_path(std::path::Path::new(path))
        .unwrap();
    assert_eq!(
        listed_centerline(&Client::new(owner.document_host())).curve_count,
        rotated.len() as i64
    );
    assert_eq!(centerline_segments(&mut owner, deferred), rotated);
}

fn verify_view_delete(client: &Client<impl Caller>, curves: i64) {
    client
        .drawing_views()
        .delete(&DeleteViewArgs { name: "Top".into() })
        .unwrap();
    assert_eq!(listed_centerline(client).curve_count, 0);
    client.transactions().undo().unwrap();
    assert_eq!(listed_centerline(client).curve_count, curves);
}

fn listed_centerline(client: &Client<impl Caller>) -> DrawingAnnotationInfo {
    client
        .drawing_annotations()
        .list(&EmptyObject {})
        .unwrap()
        .annotations
        .unwrap()
        .remove(0)
}

fn centerline_segments(owner: &mut TestOwner, drawing: u64) -> Vec<[[f64; 2]; 2]> {
    let state = owner.gui_command_state().unwrap();
    let bounds = state.drawing_projections[&drawing].view_bounds_mm["Top"];
    oblikovati_model::drawing::DrawingCenterlines::from_view_bounds(bounds)
        .visible_segments([420., 297.], 0.25)
}

#[test]
fn drawing_view_rotation_preserves_visibility_and_round_trips() {
    let (mut owner, _) = owner();
    super::super::part_scene::open_prism(&mut owner, false);
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let client = Client::new(owner.document_host());
    let original = add_top_view(&client);
    drop(client);
    let unrotated = lines(&mut owner, drawing);
    let client = Client::new(owner.document_host());
    assert!(
        client
            .drawing_views()
            .rotate(&RotateViewArgs {
                name: "Missing".into(),
                angle_deg: 90.
            })
            .is_err()
    );
    let views = client
        .drawing_views()
        .rotate(&RotateViewArgs {
            name: "Top".into(),
            angle_deg: 90.,
        })
        .unwrap()
        .views
        .unwrap();
    assert_eq!(
        views,
        [oblikovati_api::wire::DrawingViewInfo {
            rotation_deg: 90.,
            ..original
        }]
    );
    drop(client);
    let rotated = lines(&mut owner, drawing);
    assert_quarter_turn(&unrotated, &rotated);
    let reopened = verify_history_and_reopen(&mut owner, drawing, &unrotated, &rotated);
    assert_eq!(lines(&mut owner, reopened), rotated);
}

pub(super) fn add_top_view(client: &Client<impl Caller>) -> oblikovati_api::wire::DrawingViewInfo {
    let model = std::env::temp_dir()
        .join("head-scene.opd")
        .to_string_lossy()
        .into_owned();
    client
        .drawing()
        .set_model_reference(&SetModelReferenceArgs {
            full_document_name: model,
        })
        .unwrap();
    client
        .drawing_views()
        .add_base(&AddBaseViewArgs {
            name: "Top".into(),
            orientation: "top".into(),
            center_xmm: CENTER[0],
            center_ymm: CENTER[1],
            ..Default::default()
        })
        .unwrap()
        .view
}

fn lines(owner: &mut TestOwner, drawing: u64) -> Vec<GuiDrawingLine> {
    let mut state = owner.gui_command_state().unwrap();
    state.drawing_projections.remove(&drawing).unwrap().lines
}

fn assert_quarter_turn(original: &[GuiDrawingLine], rotated: &[GuiDrawingLine]) {
    assert!(!original.is_empty());
    assert_eq!(original.len(), rotated.len());
    for (before, after) in original.iter().zip(rotated) {
        assert_eq!(before.hidden, after.hidden);
        assert_eq!(before.points.len(), after.points.len());
        for ([x, y], [rotated_x, rotated_y]) in before.points.iter().zip(&after.points) {
            assert!((rotated_x - (CENTER[0] - (y - CENTER[1]))).abs() < 1e-9);
            assert!((rotated_y - (CENTER[1] + (x - CENTER[0]))).abs() < 1e-9);
        }
    }
}

fn verify_history_and_reopen(
    owner: &mut TestOwner,
    drawing: u64,
    unrotated: &[GuiDrawingLine],
    rotated: &[GuiDrawingLine],
) -> u64 {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(lines(owner, drawing), unrotated);
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(lines(owner, drawing), rotated);
    let client = Client::new(owner.document_host());
    let path = document_fixture_path!("/memory/rotated-view.odd");
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: drawing,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: drawing,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id
}
