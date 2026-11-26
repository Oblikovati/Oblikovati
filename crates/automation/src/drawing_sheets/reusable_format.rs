// SPDX-License-Identifier: GPL-2.0-only
use super::api_error;
use oblikovati_api::{Error, wire::DefineSheetFormatArgs};
use oblikovati_model::drawing::{DrawingDocument, DrawingSheetFormat, DrawingSheetRecipe};

pub(super) fn define(
    drawing: &mut DrawingDocument,
    args: DefineSheetFormatArgs,
) -> Result<(), Error> {
    let paper = DrawingSheetRecipe::with_paper(
        &args.name,
        &args.size,
        &args.orientation,
        [args.width_mm, args.height_mm],
    )
    .map_err(api_error)?;
    let format = DrawingSheetFormat::new(
        paper,
        [args.h_zones, args.v_zones],
        [&args.h_label_mode, &args.v_label_mode],
        &args.title_block_location,
    )
    .map_err(api_error)?;
    drawing.define_sheet_format(format).map_err(api_error)
}
