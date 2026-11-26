// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    wire::{AddBaseViewArgs, RotateViewArgs, SetModelReferenceArgs},
};
use oblikovati_gui::{GuiDocumentTemplate, GuiDrawingLine, GuiTemplateKind, GuiTemplateUnits};

const CENTER: [f64; 2] = [120., 90.];

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

fn add_top_view(client: &Client<impl Caller>) -> oblikovati_api::wire::DrawingViewInfo {
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
    let path = "/memory/rotated-view.odd";
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
