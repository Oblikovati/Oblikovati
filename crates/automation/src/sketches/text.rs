// SPDX-License-Identifier: GPL-2.0-only
//! Route typed Sketch Text through the document-owned history used by the desktop.
use super::*;
use oblikovati_api::types::{SketchTextStyle, TextHorizontalAlign, TextVerticalAlign};
use oblikovati_model::{quantity::Dimension, sketch::SketchEntityId};

pub(super) fn add<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AddTextArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Add sketch text",
            |snapshot, ids| {
                let (anchor, style) = new_text(snapshot, request)?;
                snapshot
                    .create_sketch_text(sketch, anchor, style, ids)
                    .map_err(api_error)
            },
            |_, id| encode(&AddEntityIDResult { entity_id: id.0 }),
        )
        .map_err(api_error)
}

pub(super) fn edit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: EditTextArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let entity = SketchEntityId(request.entity_id);
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Edit sketch text",
            |snapshot, _| {
                let mut style = style_result(snapshot, sketch, entity)?.style;
                apply_fields(snapshot, &mut style, request)?;
                snapshot
                    .edit_sketch_text(sketch, entity, style)
                    .map_err(api_error)
            },
            |snapshot, ()| encode(&style_result(snapshot, sketch, entity)?),
        )
        .map_err(api_error)
}

pub(super) fn get<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: GetTextArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let owner = files.workspace().session(document).map_err(api_error)?;
    encode(&style_result(
        owner.snapshot(),
        sketch,
        SketchEntityId(request.entity_id),
    )?)
}

fn style_result(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    entity: SketchEntityId,
) -> Result<SketchTextResult, Error> {
    let placed = snapshot.sketches.get(sketch).map_err(api_error)?;
    let style = placed
        .geometry()
        .text(entity)
        .map_err(api_error)?
        .style
        .clone();
    Ok(SketchTextResult {
        entity_id: entity.0,
        style,
    })
}

fn new_text(
    snapshot: &DocumentSnapshot,
    request: AddTextArgs,
) -> Result<([f64; 2], SketchTextStyle), Error> {
    let anchor = coordinates::working_pair(
        request.anchor.as_deref().unwrap_or_default(),
        snapshot.parameters.units().working_scale(),
        "sketch text anchor",
    )?;
    let mut style = SketchTextStyle {
        content: request.text,
        family: request.font,
        h_align: TextHorizontalAlign(request.justify.into()),
        v_align: TextVerticalAlign(request.v_justify.into()),
        formatting_runs: request.formatting_runs,
        line_spacing: request.line_spacing,
        stacked_fractions: request.stacked_fractions,
        ..Default::default()
    };
    style.height = centimetres(snapshot, &request.height)?;
    apply_metrics(
        snapshot,
        &mut style,
        "",
        &request.font_size,
        &request.rotation,
    )?;
    Ok((anchor, style))
}

fn apply_fields(
    snapshot: &DocumentSnapshot,
    style: &mut SketchTextStyle,
    request: EditTextArgs,
) -> Result<(), Error> {
    if let Some(text) = request.text {
        style.content = *text;
        style.formatting_runs = None;
        style.stacked_fractions = None;
    }
    if let Some(runs) = request.formatting_runs {
        style.formatting_runs = *runs;
    }
    if let Some(fractions) = request.stacked_fractions {
        style.stacked_fractions = *fractions;
    }
    if let Some(spacing) = request.line_spacing {
        style.line_spacing = Some(spacing);
    }
    apply_font_and_alignment(style, request.font, request.justify, request.v_justify);
    apply_metrics(
        snapshot,
        style,
        &request.height,
        &request.font_size,
        &request.rotation,
    )
}

fn apply_font_and_alignment(
    style: &mut SketchTextStyle,
    font: String,
    horizontal: String,
    vertical: String,
) {
    if !font.is_empty() {
        style.family = font;
    }
    if !horizontal.is_empty() {
        style.h_align = TextHorizontalAlign(horizontal.into());
    }
    if !vertical.is_empty() {
        style.v_align = TextVerticalAlign(vertical.into());
    }
}

fn apply_metrics(
    snapshot: &DocumentSnapshot,
    style: &mut SketchTextStyle,
    height: &str,
    size: &str,
    angle: &str,
) -> Result<(), Error> {
    if !height.is_empty() {
        style.height = centimetres(snapshot, height)?;
    }
    if !size.is_empty() {
        style.font_size = centimetres(snapshot, size)?;
    }
    if !angle.is_empty() {
        style.rotation = snapshot
            .parameters
            .evaluate_scalar_expression(angle, Dimension::Angle)
            .map_err(api_error)?
            .value;
    }
    Ok(())
}

fn centimetres(snapshot: &DocumentSnapshot, input: &str) -> Result<f64, Error> {
    let length = snapshot
        .parameters
        .evaluate_scalar_expression(input, Dimension::Length)
        .map_err(api_error)?;
    Ok(length.value * snapshot.parameters.units().working_scale())
}
