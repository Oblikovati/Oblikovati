// SPDX-License-Identifier: GPL-2.0-only

//! ISO 286 fits within the application's supported size and grade range.
mod table;
use std::fmt;

/// Limit deviations from nominal, measured in millimetres.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct FitBand {
    pub upper_mm: f64,
    pub lower_mm: f64,
}

/// Refuse unsupported fits before changing a parameter.
#[derive(Clone, Debug, PartialEq)]
pub struct FitError {
    pub nominal_mm: f64,
    pub class: String,
    pub expected: &'static str,
}

impl fmt::Display for FitError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            formatter,
            "fit {:?} at {} mm: expected {}",
            self.class, self.nominal_mm, self.expected
        )
    }
}
impl std::error::Error for FitError {}

/// Resolve a supported fit; for example `fit_band(50.0, "H7")` gives +0.025/0 mm.
pub fn fit_band(nominal_mm: f64, class: &str) -> Result<FitBand, FitError> {
    let reject = |expected| FitError {
        nominal_mm,
        class: class.into(),
        expected,
    };
    let (letter, grade) = parse_class(class)
        .ok_or_else(|| reject("d/D, f/F, g/G, h/H, or js/JS followed by grade 5 through 11"))?;
    if !nominal_mm.is_finite() || nominal_mm <= 3.0 || nominal_mm > 500.0 {
        return Err(reject("a finite nominal size in (3, 500] mm"));
    }
    let step = table::STEPS
        .iter()
        .position(|step| nominal_mm <= step.upper_mm)
        .expect("bounded nominal size");
    let width = f64::from(table::STEPS[step].widths_um[grade - 5]);
    Ok(resolve_band(letter, width, &table::STEPS[step]))
}

fn parse_class(class: &str) -> Option<(&str, usize)> {
    let digit = class.find(|character: char| character.is_ascii_digit())?;
    let (letter, grade) = class.split_at(digit);
    if !matches!(
        letter,
        "d" | "D" | "f" | "F" | "g" | "G" | "h" | "H" | "js" | "JS"
    ) {
        return None;
    }
    let numeric: usize = grade.parse().ok()?;
    if !(5..=11).contains(&numeric) || grade != numeric.to_string() {
        return None;
    }
    Some((letter, numeric))
}

fn resolve_band(letter: &str, width: f64, step: &table::FitStep) -> FitBand {
    if matches!(letter, "js" | "JS") {
        // #3573: preserve half-micrometres; Go rounded these bands outward.
        return FitBand {
            upper_mm: width / 2000.0,
            lower_mm: -width / 2000.0,
        };
    }
    let upper = shaft_deviation(letter, step);
    let (upper, lower) = if letter.as_bytes()[0].is_ascii_uppercase() {
        (-upper + width, -upper)
    } else {
        (upper, upper - width)
    };
    FitBand {
        upper_mm: upper / 1000.0,
        lower_mm: lower / 1000.0,
    }
}

fn shaft_deviation(letter: &str, step: &table::FitStep) -> f64 {
    let micrometres = match letter {
        "d" | "D" => step.shaft_deviations_um[0],
        "f" | "F" => step.shaft_deviations_um[1],
        "g" | "G" => step.shaft_deviations_um[2],
        _ => 0,
    };
    f64::from(micrometres)
}
