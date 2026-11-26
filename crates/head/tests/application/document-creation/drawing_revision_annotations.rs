// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    json::EmptyObject,
    wire::{AddRevisionTableArgs, AddRevisionTagArgs, DrawingAnnotationInfo, RevisionTableRow},
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
use oblikovati_model::drawing::DrawingAnnotationDefinition;

#[test]
fn drawing_revision_table_and_tag_have_independent_history_and_persist() {
    let (mut owner, _) = owner();
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
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
