// SPDX-License-Identifier: GPL-2.0-only
//! Sheet collection operations over the drawing's authored recipe.
use oblikovati_api::{
    Error,
    wire::{
        self, AddDefaultBorderArgs, AddSheetArgs, AddSheetUsingFormatArgs, ListSheetsResult,
        RemoveSheetArgs, SetActiveSheetArgs, SetSheetRevisionArgs, SetTitleBlockArgs, SheetInfo,
        SheetResult,
    },
};
use oblikovati_model::drawing::{DrawingDocument, DrawingSheetRecipe};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

mod reusable_format;
mod title_block;

pub(super) const SHEET_METHODS: [&str; 10] = [
    wire::METHOD_DRAWING_LIST_SHEETS,
    wire::METHOD_DRAWING_ADD_SHEET,
    wire::METHOD_DRAWING_REMOVE_SHEET,
    wire::METHOD_DRAWING_SET_ACTIVE_SHEET,
    wire::METHOD_DRAWING_TITLE_BLOCK_FIELDS,
    wire::METHOD_DRAWING_SET_TITLE_BLOCK,
    wire::METHOD_DRAWING_SET_SHEET_REVISION,
    wire::METHOD_DRAWING_ADD_DEFAULT_BORDER,
    wire::METHOD_DRAWING_DEFINE_SHEET_FORMAT,
    wire::METHOD_DRAWING_ADD_SHEET_USING_FORMAT,
];

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let (owner, drawing) = active_drawing(files)?;
    if method == wire::METHOD_DRAWING_TITLE_BLOCK_FIELDS {
        let args = if request.is_empty() {
            Default::default()
        } else {
            crate::decode(method, request)?
        };
        return title_block::fields(files, owner, drawing, args);
    }
    if method == wire::METHOD_DRAWING_LIST_SHEETS {
        if !request.is_empty() {
            crate::decode::<oblikovati_api::json::EmptyObject>(method, request)?;
        }
        return list_sheets(drawing);
    }
    if files
        .deferred_drawing_views(owner)
        .map_err(api_error)?
        .is_some()
    {
        return Err(Error(format!(
            "drawing {owner}: reopen with updates enabled before editing sheets"
        )));
    }
    edit_sheets(files, owner, drawing.clone(), method, request)
}

fn active_drawing<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<(u64, &DrawingDocument), Error> {
    let workspace = files.workspace();
    let owner = workspace
        .active_document()
        .ok_or_else(|| Error("drawing sheets: expected an active drawing".into()))?;
    let drawing = workspace
        .session(owner)
        .map_err(api_error)?
        .snapshot()
        .drawing
        .as_ref()
        .ok_or_else(|| Error(format!("document {owner}: expected drawing content")))?;
    Ok((owner, drawing))
}

fn edit_sheets<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: u64,
    mut drawing: DrawingDocument,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let (label, sheet_index) = update_sheets(&mut drawing, method, request)?;
    let reply = match sheet_index {
        None => list_sheets(&drawing)?,
        Some(index) => crate::encode(&SheetResult {
            sheet: sheet_info(&drawing.sheets()[index], index == drawing.active_sheet())?,
        })?,
    };
    files
        .commands(owner)
        .map_err(api_error)?
        .run(label, |snapshot, _| {
            snapshot.drawing = Some(drawing);
            Ok::<_, Error>(())
        })
        .map_err(api_error)?;
    Ok(reply)
}

fn update_sheets(
    drawing: &mut DrawingDocument,
    method: &str,
    request: &[u8],
) -> Result<(&'static str, Option<usize>), Error> {
    match method {
        wire::METHOD_DRAWING_ADD_SHEET => {
            add_sheet(drawing, crate::decode(method, request)?)?;
            Ok(("Add Sheet", Some(drawing.active_sheet())))
        }
        wire::METHOD_DRAWING_REMOVE_SHEET => {
            let args: RemoveSheetArgs = crate::decode(method, request)?;
            drawing.remove_sheet(&args.name).map_err(api_error)?;
            Ok(("Delete Sheet", None))
        }
        wire::METHOD_DRAWING_SET_ACTIVE_SHEET => {
            let args: SetActiveSheetArgs = crate::decode(method, request)?;
            let index = drawing.sheet_index(&args.name).map_err(api_error)?;
            drawing.set_active_sheet(index).map_err(api_error)?;
            Ok(("Activate Sheet", Some(index)))
        }
        wire::METHOD_DRAWING_DEFINE_SHEET_FORMAT => {
            reusable_format::define(drawing, crate::decode(method, request)?)?;
            Ok(("Define Sheet Format", None))
        }
        wire::METHOD_DRAWING_ADD_SHEET_USING_FORMAT => {
            let args: AddSheetUsingFormatArgs = crate::decode(method, request)?;
            let name = sheet_name(drawing, args.name);
            drawing
                .add_sheet_using_format(&name, &args.format)
                .map_err(api_error)?;
            Ok(("Add Sheet", Some(drawing.active_sheet())))
        }
        _ => update_sheet_format(drawing, method, request),
    }
}

fn update_sheet_format(
    drawing: &mut DrawingDocument,
    method: &str,
    request: &[u8],
) -> Result<(&'static str, Option<usize>), Error> {
    match method {
        wire::METHOD_DRAWING_SET_TITLE_BLOCK => {
            let args: SetTitleBlockArgs = crate::decode(method, request)?;
            let index = drawing
                .named_or_active_sheet(&args.sheet)
                .map_err(api_error)?;
            drawing
                .set_title_block(index, &args.location)
                .map_err(api_error)?;
            Ok(("Set Title Block", Some(index)))
        }
        wire::METHOD_DRAWING_SET_SHEET_REVISION => {
            let args: SetSheetRevisionArgs = crate::decode(method, request)?;
            let index = drawing
                .named_or_active_sheet(&args.sheet)
                .map_err(api_error)?;
            drawing
                .set_sheet_revision(index, &args.revision)
                .map_err(api_error)?;
            Ok(("Set Sheet Revision", Some(index)))
        }
        wire::METHOD_DRAWING_ADD_DEFAULT_BORDER => {
            let args: AddDefaultBorderArgs = crate::decode(method, request)?;
            Ok(("Add Border", Some(add_border(drawing, args)?)))
        }
        _ => Err(Error(format!(
            "method {method:?}: expected a sheet format edit"
        ))),
    }
}

fn add_border(drawing: &mut DrawingDocument, args: AddDefaultBorderArgs) -> Result<usize, Error> {
    let index = drawing
        .named_or_active_sheet(&args.sheet)
        .map_err(api_error)?;
    drawing
        .set_zoned_border(
            index,
            [args.h_zones, args.v_zones],
            [&args.h_label_mode, &args.v_label_mode],
        )
        .map_err(api_error)?;
    Ok(index)
}

fn add_sheet(drawing: &mut DrawingDocument, args: AddSheetArgs) -> Result<(), Error> {
    let name = sheet_name(drawing, args.name);
    let sheet = DrawingSheetRecipe::with_paper(
        &name,
        &args.size,
        &args.orientation,
        [args.width_mm, args.height_mm],
    )
    .map_err(api_error)?;
    drawing.add_sheet(sheet).map_err(api_error)
}

fn sheet_name(drawing: &DrawingDocument, requested: String) -> String {
    if !requested.is_empty() {
        return requested;
    }
    let mut index = drawing.sheets().len() + 1;
    loop {
        let name = format!("Sheet:{index}");
        if drawing.sheet_index(&name).is_err() {
            return name;
        }
        index += 1;
    }
}

fn list_sheets(drawing: &DrawingDocument) -> Result<Vec<u8>, Error> {
    let sheets = drawing
        .sheets()
        .iter()
        .enumerate()
        .map(|(index, sheet)| sheet_info(sheet, index == drawing.active_sheet()))
        .collect::<Result<Vec<_>, _>>()?;
    crate::encode(&ListSheetsResult {
        sheets: Some(sheets),
        model_reference: drawing.model_reference().to_string(),
    })
}

fn sheet_info(sheet: &DrawingSheetRecipe, active: bool) -> Result<SheetInfo, Error> {
    let [size, orientation, corner] = sheet.paper_spelling().map_err(api_error)?;
    let [width_mm, height_mm] = sheet.dimensions_mm().map_err(api_error)?;
    Ok(SheetInfo {
        name: sheet.name.clone(),
        size: size.into(),
        orientation: orientation.into(),
        width_mm,
        height_mm,
        active,
        has_border: sheet.border,
        has_title_block: !sheet.title_block.is_empty(),
        revision: sheet.revision.clone(),
        title_block_location: corner.into(),
        border_h_zones: sheet.border_h_zones,
        border_v_zones: sheet.border_v_zones,
    })
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
