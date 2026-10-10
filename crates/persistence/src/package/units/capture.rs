// SPDX-License-Identifier: GPL-2.0-only

use super::*;

pub(in crate::package) fn capture_units(
    units: &UnitsOfMeasure,
) -> Result<Value, PackageCodecError> {
    let mut entries: BTreeMap<String, String> = CATEGORIES
        .iter()
        .map(|dimension| {
            (
                category_name(*dimension).into(),
                units.preferred_name(*dimension).into(),
            )
        })
        .collect();
    let display = units.display_options();
    entries.insert(
        "lengthPrecision".into(),
        display.length_precision.to_string(),
    );
    entries.insert("anglePrecision".into(), display.angle_precision.to_string());
    entries.insert(
        "lengthFormat".into(),
        format_name(display.length_format)?.into(),
    );
    entries.insert(
        "angleFormat".into(),
        match display.angle_format {
            AngleFormat::Decimal => "decimal",
            AngleFormat::DegreesMinutesSeconds => "dms",
        }
        .into(),
    );
    if units.working_scale() != 1.0 {
        entries.insert("workingScale".into(), units.working_scale().to_string());
    }
    serde_yaml_ng::to_value(entries).map_err(codec_error)
}

fn format_name(format: ParameterDisplayFormat) -> Result<&'static str, PackageCodecError> {
    match format {
        DISPLAY_FORMAT_DECIMAL => Ok("decimal"),
        DISPLAY_FORMAT_FRACTIONAL => Ok("fractional"),
        DISPLAY_FORMAT_ARCHITECTURAL => Ok("architectural"),
        _ => Err(invalid_unit(
            "lengthFormat",
            &format!("{format:?}"),
            "a supported length format",
        )),
    }
}
