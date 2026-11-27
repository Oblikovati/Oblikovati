// SPDX-License-Identifier: GPL-2.0-only
//! Public drafting-style operations use the same preset as the native canvas.
use oblikovati_api::{
    Error,
    types::DIMENSION_INCH,
    wire::{
        self, DimensionStyleInfo, LineStyleInfo, ListStandardsResult, SetStandardArgs,
        StandardStyleInfo, StandardStyleResult, TextStyleInfo,
    },
};
use oblikovati_model::drawing::{DRAWING_STANDARD_PRESETS, DrawingStandardPreset};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(super) const STYLE_METHODS: [&str; 3] = [
    wire::METHOD_DRAWING_STYLES_LIST_STANDARDS,
    wire::METHOD_DRAWING_STYLES_GET_ACTIVE_STYLE,
    wire::METHOD_DRAWING_STYLES_SET_STANDARD,
];

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let (owner, drawing) = crate::drawing_sheets::active_drawing(files)?;
    if method == wire::METHOD_DRAWING_STYLES_SET_STANDARD {
        return set_standard(files, owner, crate::decode(method, request)?);
    }
    if !request.is_empty() {
        crate::decode::<oblikovati_api::json::EmptyObject>(method, request)?;
    }
    let style = drawing.standard_style();
    if method == wire::METHOD_DRAWING_STYLES_LIST_STANDARDS {
        return crate::encode(&ListStandardsResult {
            standards: Some(
                DRAWING_STANDARD_PRESETS
                    .iter()
                    .map(|preset| preset.wire_name.into())
                    .collect(),
            ),
            active: style.wire_name.into(),
        });
    }
    style_reply(style)
}

fn set_standard<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    owner: u64,
    request: SetStandardArgs,
) -> Result<Vec<u8>, Error> {
    if files
        .deferred_drawing_views(owner)
        .map_err(api_error)?
        .is_some()
    {
        return Err(Error(format!(
            "drawing {owner}: reopen with updates enabled before editing its standard"
        )));
    }
    files
        .commands(owner)
        .map_err(api_error)?
        .run_projected(
            "Set Drawing Standard",
            |snapshot, _| {
                let drawing = snapshot
                    .drawing
                    .as_mut()
                    .ok_or_else(|| Error(format!("document {owner}: expected drawing content")))?;
                drawing
                    .set_drafting_standard(&request.standard)
                    .map_err(api_error)?;
                Ok(drawing.standard_style())
            },
            |_, preset| style_reply(preset),
        )
        .map_err(api_error)
}

fn style_reply(preset: &DrawingStandardPreset) -> Result<Vec<u8>, Error> {
    let display_name = preset.wire_name.to_ascii_uppercase();
    crate::encode(&StandardStyleResult {
        style: StandardStyleInfo {
            standard: preset.wire_name.into(),
            dimension: dimension_style(preset, &display_name),
            text: TextStyleInfo {
                name: format!("Note Text ({display_name})"),
                font_name: "Arial".into(),
                height_mm: preset.text_height_mm,
            },
            line: LineStyleInfo {
                name: format!("Visible ({display_name})"),
                weight_mm: preset.visible_line_mm,
            },
        },
    })
}

fn dimension_style(preset: &DrawingStandardPreset, name: &str) -> DimensionStyleInfo {
    let unit = if preset.dimension_unit == DIMENSION_INCH {
        "in"
    } else {
        "mm"
    };
    DimensionStyleInfo {
        name: format!("Default ({name})"),
        text_height_mm: preset.text_height_mm,
        arrow_size_mm: preset.arrow_size_mm,
        decimal_places: preset.decimal_places,
        unit: unit.into(),
        line_weight_mm: preset.dimension_line_mm,
    }
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
