// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    Error,
    types::DOCUMENT_PART,
    wire::{
        self, AddCustomTableArgs, AddDrawingNoteArgs, AddRevisionTableArgs, AddRevisionTagArgs,
        AnnotationResult, DeleteAnnotationArgs, DrawingAnnotationInfo,
        ListDrawingAnnotationsResult,
    },
};
use oblikovati_model::drawing::{
    DrawingAnnotationDefinition, DrawingDocument, DrawingPartsListPresentation,
    DrawingPartsListRow, DrawingPartsListSource, DrawingRevisionRow,
};
use oblikovati_model::property::{DESIGN_TRACKING, DocumentProperties};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(super) const ANNOTATION_METHODS: [&str; 6] = [
    wire::METHOD_DRAWING_ANNOTATIONS_ADD_NOTE,
    wire::METHOD_DRAWING_ANNOTATIONS_ADD_CUSTOM_TABLE,
    wire::METHOD_DRAWING_ANNOTATIONS_ADD_REV_TABLE,
    wire::METHOD_DRAWING_ANNOTATIONS_ADD_REV_TAG,
    wire::METHOD_DRAWING_ANNOTATIONS_LIST,
    wire::METHOD_DRAWING_ANNOTATIONS_DELETE,
];

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let (owner, drawing) = crate::drawing_sheets::active_drawing(files)?;
    if method == wire::METHOD_DRAWING_ANNOTATIONS_LIST {
        if !request.is_empty() {
            crate::decode::<oblikovati_api::json::EmptyObject>(method, request)?;
        }
        return list(files, owner, drawing);
    }
    if files
        .deferred_drawing_views(owner)
        .map_err(api_error)?
        .is_some()
    {
        return Err(Error(format!(
            "drawing {owner}: reopen with updates enabled before editing annotations"
        )));
    }
    match method {
        wire::METHOD_DRAWING_ANNOTATIONS_ADD_NOTE => {
            let args: AddDrawingNoteArgs = crate::decode(method, request)?;
            add_annotation(files, owner, "Add Drawing Note", |drawing| {
                drawing.add_note(
                    &args.name,
                    [args.xmm, args.ymm],
                    &args.text,
                    [args.leader_xmm, args.leader_ymm],
                )
            })
        }
        wire::METHOD_DRAWING_ANNOTATIONS_ADD_CUSTOM_TABLE => {
            let args: AddCustomTableArgs = crate::decode(method, request)?;
            add_annotation(files, owner, "Add Custom Drawing Table", |drawing| {
                drawing.add_custom_table(
                    &args.name,
                    [args.xmm, args.ymm],
                    args.headers.unwrap_or_default(),
                    args.rows.unwrap_or_default(),
                )
            })
        }
        wire::METHOD_DRAWING_ANNOTATIONS_ADD_REV_TABLE => {
            let args: AddRevisionTableArgs = crate::decode(method, request)?;
            let rows = args
                .rows
                .unwrap_or_default()
                .into_iter()
                .map(|row| DrawingRevisionRow {
                    revision: row.revision,
                    date: row.date,
                    description: row.description,
                })
                .collect();
            add_annotation(files, owner, "Add Drawing Revision Table", |drawing| {
                drawing.add_revision_table(&args.name, [args.xmm, args.ymm], rows)
            })
        }
        wire::METHOD_DRAWING_ANNOTATIONS_ADD_REV_TAG => {
            let args: AddRevisionTagArgs = crate::decode(method, request)?;
            add_annotation(files, owner, "Add Drawing Revision Tag", |drawing| {
                drawing.add_revision_tag(&args.name, [args.xmm, args.ymm], &args.revision)
            })
        }
        wire::METHOD_DRAWING_ANNOTATIONS_DELETE => {
            let args: DeleteAnnotationArgs = crate::decode(method, request)?;
            delete(files, owner, drawing.clone(), &args.name)
        }
        _ => Err(Error(format!(
            "method {method:?}: expected a drawing annotation route"
        ))),
    }
}

fn add_annotation<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: u64,
    command: &str,
    add: impl FnOnce(
        &mut DrawingDocument,
    )
        -> Result<DrawingAnnotationDefinition, oblikovati_model::drawing::DrawingError>,
) -> Result<Vec<u8>, Error> {
    files
        .commands(owner)
        .map_err(api_error)?
        .run_projected(
            command,
            |snapshot, _| {
                let drawing = snapshot
                    .drawing
                    .as_mut()
                    .ok_or_else(|| Error(format!("document {owner}: expected drawing content")))?;
                add(drawing).map_err(api_error)
            },
            |_, annotation| {
                crate::encode(&AnnotationResult {
                    annotation: annotation_info(&annotation, None)?,
                })
            },
        )
        .map_err(api_error)
}

fn delete<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: u64,
    mut drawing: DrawingDocument,
    name: &str,
) -> Result<Vec<u8>, Error> {
    drawing
        .remove_annotation(drawing.active_sheet(), name)
        .map_err(api_error)?;
    let response = list(files, owner, &drawing)?;
    files
        .commands(owner)
        .map_err(api_error)?
        .run("Delete Drawing Annotation", |snapshot, _| {
            snapshot.drawing = Some(drawing);
            Ok::<(), Error>(())
        })
        .map_err(api_error)?;
    Ok(response)
}

fn list<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    drawing: &DrawingDocument,
) -> Result<Vec<u8>, Error> {
    let sheet = &drawing.sheets()[drawing.active_sheet()];
    if sheet.annotations.is_empty() {
        return crate::encode(&ListDrawingAnnotationsResult { annotations: None });
    }
    let tables = current_parts_lists(files, owner, drawing)?;
    let annotations = sheet
        .annotations
        .iter()
        .map(|annotation| {
            let table = tables
                .iter()
                .find(|(name, _)| name == annotation.name())
                .map(|(_, table)| table);
            annotation_info(annotation, table)
        })
        .collect::<Result<Vec<_>, _>>()?;
    crate::encode(&ListDrawingAnnotationsResult {
        annotations: Some(annotations),
    })
}

fn annotation_info(
    annotation: &DrawingAnnotationDefinition,
    table: Option<&DrawingPartsListPresentation>,
) -> Result<DrawingAnnotationInfo, Error> {
    let (kind, tag, curves, rows) = annotation_details(annotation, table);
    let count = |value| i64::try_from(value).map_err(api_error);
    Ok(DrawingAnnotationInfo {
        name: annotation.name().into(),
        kind: kind.into(),
        tag: tag.into(),
        curve_count: count(curves)?,
        row_count: count(rows)?,
        ..Default::default()
    })
}

fn annotation_details<'a>(
    annotation: &'a DrawingAnnotationDefinition,
    table: Option<&DrawingPartsListPresentation>,
) -> (&'static str, &'a str, usize, usize) {
    match annotation {
        DrawingAnnotationDefinition::RevisionTag { revision, .. } => {
            ("revisionTag", revision, 3, 0)
        }
        DrawingAnnotationDefinition::RevisionTable { revisions, .. } => {
            ("revisionTable", "", revisions.len() + 6, revisions.len())
        }
        DrawingAnnotationDefinition::CustomTable { headers, rows, .. } => (
            "customTable",
            "",
            headers.len() + rows.len() + 3,
            rows.len(),
        ),
        DrawingAnnotationDefinition::PartsList { .. } => {
            let (curves, rows) = table.map_or((0, 0), |table| (table.lines.len(), table.row_count));
            ("partsList", "", curves, rows)
        }
        DrawingAnnotationDefinition::Note { text, .. } => {
            let curves = annotation
                .note_presentation()
                .map_or(0, |note| note.leader_segments_mm.len());
            ("drawingNote", text, curves, 0)
        }
    }
}

fn current_parts_lists<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    drawing: &DrawingDocument,
) -> Result<Vec<(String, DrawingPartsListPresentation)>, Error> {
    let sheet = &drawing.sheets()[drawing.active_sheet()];
    if !sheet
        .annotations
        .iter()
        .any(DrawingAnnotationDefinition::requires_model)
    {
        return Ok(Vec::new());
    }
    let name = drawing.model_reference();
    let source = match files.drawing_reference(owner, name) {
        Some(source) => Some(source),
        None => files.resolve_parameter_source(name).map_err(api_error)?,
    };
    if let Some(source) = source
        && files.document_type(source).map_err(api_error)? == DOCUMENT_PART
    {
        return part_drawing_parts_lists(files, source, drawing);
    }
    Ok(Vec::new())
}

/// Evaluate active-sheet tables from a part, e.g. `part_drawing_parts_lists(files, part, drawing)?`.
pub fn part_drawing_parts_lists<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    part: u64,
    drawing: &DrawingDocument,
) -> Result<Vec<(String, DrawingPartsListPresentation)>, Error> {
    let workspace = files.workspace();
    let session = workspace.session(part).map_err(api_error)?;
    let name = workspace.display_name(part).map_err(api_error)?;
    let properties = &session.snapshot().properties;
    let row = PartListRow(DrawingPartsListRow {
        item: 1,
        part_number: text_property(properties, "Part Number")
            .unwrap_or(name)
            .into(),
        description: text_property(properties, "Description")
            .unwrap_or_default()
            .into(),
        quantity: 1,
    });
    drawing
        .evaluate_parts_lists(drawing.active_sheet(), &row)
        .map_err(api_error)
}

struct PartListRow(DrawingPartsListRow);
impl DrawingPartsListSource for PartListRow {
    fn parts_list_rows(&self, _reference: &str) -> Result<Vec<DrawingPartsListRow>, String> {
        Ok(vec![self.0.clone()])
    }
}

fn text_property<'a>(properties: &'a DocumentProperties, name: &str) -> Option<&'a str> {
    use oblikovati_api::types::VariantValue;
    match &properties.get(DESIGN_TRACKING, name)?.value {
        VariantValue::String(text) if !text.trim().is_empty() => Some(text),
        _ => None,
    }
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
