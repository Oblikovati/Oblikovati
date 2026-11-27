// SPDX-License-Identifier: GPL-2.0-only

use super::PackageCodecError;
use oblikovati_model::{quantity::Quantity, units::parse_category};

pub(super) fn parse_saved_constant(source: &str) -> Result<Option<Quantity>, PackageCodecError> {
    let mut tokens = source.split_whitespace();
    let (Some(number), Some(category), None) = (tokens.next(), tokens.next(), tokens.next()) else {
        return Ok(None);
    };
    // Go constantExpr writes category names, not named units. Older Rust diagnostics used capitals.
    let Some(dimension) = parse_category(&category.to_ascii_lowercase()) else {
        return Ok(None);
    };
    let value = number.parse::<f64>().ok().filter(|value| value.is_finite());
    if !dimension.is_arithmetic() || value.is_none() {
        return Err(PackageCodecError(format!(
            "measured expression {source:?}: expected a finite number and numeric quantity category"
        )));
    }
    Ok(Some(Quantity::new(
        value.expect("validated finite number"),
        dimension,
    )))
}
