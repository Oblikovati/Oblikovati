// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    json::EmptyObject,
    wire::{
        AddRevisionCloudArgs, AddRevisionTableArgs, AddRevisionTagArgs, DeleteAnnotationArgs,
        DrawingAnnotationInfo, RevisionTableRow,
    },
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
use oblikovati_model::drawing::DrawingAnnotationDefinition;

#[test]
fn drawing_revision_table_and_tag_have_independent_history_and_persist() {
    let (mut owner, drawing) = drawing_owner();
    let client = Client::new(owner.document_host());
    let [table, tag] = add_revision_annotations(&client);
    client.transactions().undo().unwrap();
    assert_eq!(listed(&client).as_slice(), std::slice::from_ref(&table));
    client.transactions().redo().unwrap();
    assert_eq!(listed(&client), [table.clone(), tag.clone()]);
    let reopened =
        super::drawing_custom_table::reopen(&client, drawing, "/memory/revision-annotations.odd");
    assert_eq!(listed(&client), [table, tag]);
    drop(client);
    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&reopened].error.is_none());
    let sheet = &state.drawing_documents[&reopened].sheets()[0];
    assert_presentations(&sheet.annotations);
}

fn drawing_owner() -> (TestOwner, u64) {
    let (mut owner, _) = owner();
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    (owner, drawing)
}

#[test]
fn drawing_revision_cloud_has_outward_scallops_and_round_trips() {
    let (mut owner, drawing) = drawing_owner();
    let client = Client::new(owner.document_host());
    let cloud = add_cloud(&client);
    client.transactions().undo().unwrap();
    assert!(listed(&client).is_empty());
    client.transactions().redo().unwrap();
    assert_eq!(listed(&client).as_slice(), std::slice::from_ref(&cloud));
    let reopened =
        super::drawing_custom_table::reopen(&client, drawing, "/memory/revision-cloud.odd");
    assert_eq!(listed(&client).as_slice(), std::slice::from_ref(&cloud));
    drop(client);
    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&reopened].error.is_none());
    let sheet = &state.drawing_documents[&reopened].sheets()[0];
    assert_cloud_presentation(&sheet.annotations[0]);
    let client = Client::new(owner.document_host());
    assert!(
        client
            .drawing_annotations()
            .delete(&DeleteAnnotationArgs { name: cloud.name })
            .unwrap()
            .annotations
            .is_none()
    );
}

fn add_cloud(client: &Client<impl Caller>) -> DrawingAnnotationInfo {
    let annotations = client.drawing_annotations();
    let request = AddRevisionCloudArgs {
        xmm: 60.,
        ymm: 80.,
        width_mm: 60.,
        height_mm: 40.,
        tag: "A".into(),
        ..Default::default()
    };
    for width_mm in [0., -1., 1e30] {
        assert!(
            annotations
                .add_revision_cloud(&AddRevisionCloudArgs {
                    width_mm,
                    ..request.clone()
                })
                .is_err()
        );
    }
    let cloud = annotations.add_revision_cloud(&request).unwrap().annotation;
    assert_eq!(
        (
            cloud.name.as_str(),
            cloud.kind.as_str(),
            cloud.tag.as_str(),
            cloud.curve_count
        ),
        ("NOTE:1", "revisionCloud", "A", 204)
    );
    cloud
}

fn assert_cloud_presentation(annotation: &DrawingAnnotationDefinition) {
    let cloud = annotation.revision_cloud_presentation().unwrap();
    assert_eq!(cloud.tag, "A");
    assert_eq!(cloud.label_center_mm(), [90., 126.]);
    let outline = cloud.visible_segments([420., 297.], 0.25);
    assert_eq!(outline.len(), 204);
    assert_eq!(outline[0][0], [60., 80.]);
    assert!((outline[2][1][0] - 63.).abs() < 1e-10);
    assert_eq!(outline[2][1][1], 77.);
    assert!(
        outline[60..102]
            .iter()
            .flatten()
            .all(|point| point[0] >= 120.)
    );
    assert_eq!(cloud.visible_segments([420., 297.], 10.).len(), 4);
    assert_eq!(cloud.visible_segments([90., 90.], 0.25).len(), 84);
}

fn add_revision_annotations(client: &Client<impl Caller>) -> [DrawingAnnotationInfo; 2] {
    let annotations = client.drawing_annotations();
    assert!(
        annotations
            .add_revision_table(&AddRevisionTableArgs::default())
            .is_err()
    );
    assert!(
        annotations
            .add_revision_tag(&AddRevisionTagArgs::default())
            .is_err()
    );
    let table = annotations
        .add_revision_table(&AddRevisionTableArgs {
            name: "Changes".into(),
            xmm: 50.,
            ymm: 180.,
            rows: Some(vec![
                RevisionTableRow {
                    revision: "A".into(),
                    date: "2026-11-26".into(),
                    description: "Released".into(),
                },
                RevisionTableRow {
                    revision: "B".into(),
                    description: "Bore enlarged".into(),
                    ..Default::default()
                },
            ]),
        })
        .unwrap()
        .annotation;
    let tag = annotations
        .add_revision_tag(&AddRevisionTagArgs {
            name: "B Marker".into(),
            xmm: 190.,
            ymm: 125.,
            revision: "B".into(),
        })
        .unwrap()
        .annotation;
    assert_eq!(
        (table.kind.as_str(), table.row_count, table.curve_count),
        ("revisionTable", 2, 8)
    );
    assert_eq!(
        (tag.kind.as_str(), tag.tag.as_str(), tag.curve_count),
        ("revisionTag", "B", 3)
    );
    [table, tag]
}

fn listed(client: &Client<impl Caller>) -> Vec<DrawingAnnotationInfo> {
    client
        .drawing_annotations()
        .list(&EmptyObject {})
        .unwrap()
        .annotations
        .unwrap_or_default()
}

fn assert_presentations(annotations: &[DrawingAnnotationDefinition]) {
    let table = annotations[0].authored_table_presentation().unwrap();
    assert_eq!(table.lines[0].start, [50., 180.]);
    assert_eq!(table.lines[0].end, [152., 180.]);
    let labels: Vec<_> = table.labels.iter().map(|cell| cell.text.as_str()).collect();
    assert_eq!(
        labels,
        [
            "REV",
            "DATE",
            "DESCRIPTION",
            "A",
            "2026-11-26",
            "Released",
            "B",
            "",
            "Bore enlarged"
        ]
    );
    let tag = annotations[1].revision_tag_presentation().unwrap();
    assert_eq!(tag.text, "B");
    assert_eq!(tag.text_center_mm, [190., 123.]);
    assert_eq!(
        tag.segments_mm,
        [
            [[190., 131.], [184., 119.]],
            [[184., 119.], [196., 119.]],
            [[196., 119.], [190., 131.]],
        ]
    );
}
