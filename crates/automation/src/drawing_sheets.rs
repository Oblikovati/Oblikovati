// SPDX-License-Identifier: GPL-2.0-only
//! Sheet collection operations over the drawing's authored recipe.
use oblikovati_api::{
    Error,
    wire::{
        self, AddSheetArgs, ListSheetsResult, RemoveSheetArgs, SetActiveSheetArgs, SheetInfo,
        SheetResult,
    },
};
use oblikovati_model::drawing::{DrawingDocument, DrawingSheetRecipe};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(super) const SHEET_METHODS: [&str; 4] = [
    wire::METHOD_DRAWING_LIST_SHEETS,
    wire::METHOD_DRAWING_ADD_SHEET,
    wire::METHOD_DRAWING_REMOVE_SHEET,
    wire::METHOD_DRAWING_SET_ACTIVE_SHEET,
];

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let (owner, drawing) = active_drawing(files)?;
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
    let label = update_sheets(&mut drawing, method, request)?;
    let reply = if method == wire::METHOD_DRAWING_REMOVE_SHEET {
        list_sheets(&drawing)?
    } else {
        let sheet = &drawing.sheets()[drawing.active_sheet()];
        crate::encode(&SheetResult {
            sheet: sheet_info(sheet, true)?,
        })?
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
) -> Result<&'static str, Error> {
    match method {
        wire::METHOD_DRAWING_ADD_SHEET => {
            add_sheet(drawing, crate::decode(method, request)?)?;
            Ok("Add Sheet")
        }
        wire::METHOD_DRAWING_REMOVE_SHEET => {
            let args: RemoveSheetArgs = crate::decode(method, request)?;
            drawing.remove_sheet(&args.name).map_err(api_error)?;
            Ok("Delete Sheet")
        }
        wire::METHOD_DRAWING_SET_ACTIVE_SHEET => {
            let args: SetActiveSheetArgs = crate::decode(method, request)?;
            let index = drawing.sheet_index(&args.name).map_err(api_error)?;
            drawing.set_active_sheet(index).map_err(api_error)?;
            Ok("Activate Sheet")
        }
        _ => Err(Error(format!("method {method:?}: expected a sheet edit"))),
    }
}

fn add_sheet(drawing: &mut DrawingDocument, args: AddSheetArgs) -> Result<(), Error> {
    let name = if args.name.is_empty() {
        next_sheet_name(drawing)
    } else {
        args.name
    };
    let sheet = DrawingSheetRecipe::with_paper(
        &name,
        &args.size,
        &args.orientation,
        [args.width_mm, args.height_mm],
    )
    .map_err(api_error)?;
    drawing.add_sheet(sheet).map_err(api_error)
}

fn next_sheet_name(drawing: &DrawingDocument) -> String {
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
        ..Default::default()
    })
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
