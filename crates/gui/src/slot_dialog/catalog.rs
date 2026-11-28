// SPDX-License-Identifier: GPL-2.0-only
//! Fastener clearance lookup and live validation for the Part Modify Slot Properties panel.
use crate::{GuiHoleSeatKind, GuiHoleTermination};
use oblikovati_model::feature::hole::{
    HoleFastenerFit, HoleFastenerSpec, HoleOriginPlane, HoleTerminationSurface,
};

pub(super) const CLEARANCE_STANDARDS: &[&str] =
    &["ANSI Unified Screw Threads", "ISO Metric profile"];

pub(super) const CLEARANCE_FASTENER_TYPES: &[&str] = &[
    "Hex Head Bolt",
    "Socket Head Cap Screw",
    "Flat Head Machine Screw",
];

pub(super) const CLEARANCE_SIZES: &[&str] = &[
    "#0", "#2", "#4", "#6", "#8", "#10", "1/4", "5/16", "3/8", "1/2",
];

pub(super) fn default_fastener() -> HoleFastenerSpec {
    HoleFastenerSpec {
        standard: CLEARANCE_STANDARDS[0].into(),
        fastener_type: CLEARANCE_FASTENER_TYPES[0].into(),
        size: "#4".into(),
        fit: HoleFastenerFit::Normal,
    }
}

/// Deterministic clearance slot width and seat dimensions (mm strings) for fastener size + fit.
pub(super) fn clearance_dimensions(
    size: &str,
    fit: HoleFastenerFit,
) -> (&'static str, &'static str, &'static str) {
    match (size, fit) {
        ("#0", HoleFastenerFit::Close) => ("1.60 mm", "3.00 mm", "1.60 mm"),
        ("#0", HoleFastenerFit::Normal) => ("1.70 mm", "3.00 mm", "1.60 mm"),
        ("#0", HoleFastenerFit::Loose) => ("1.85 mm", "3.20 mm", "1.75 mm"),
        ("#2", HoleFastenerFit::Close) => ("2.26 mm", "4.40 mm", "2.20 mm"),
        ("#2", HoleFastenerFit::Normal) => ("2.40 mm", "4.40 mm", "2.20 mm"),
        ("#2", HoleFastenerFit::Loose) => ("2.56 mm", "4.60 mm", "2.35 mm"),
        ("#4", HoleFastenerFit::Close) => ("3.05 mm", "5.84 mm", "2.85 mm"),
        ("#4", HoleFastenerFit::Normal) => ("3.26 mm", "5.84 mm", "2.85 mm"),
        ("#4", HoleFastenerFit::Loose) => ("3.45 mm", "6.10 mm", "3.18 mm"),
        ("#6", HoleFastenerFit::Close) => ("3.66 mm", "6.80 mm", "3.50 mm"),
        ("#6", HoleFastenerFit::Normal) => ("3.80 mm", "6.80 mm", "3.50 mm"),
        ("#6", HoleFastenerFit::Loose) => ("4.04 mm", "7.10 mm", "3.80 mm"),
        ("#8", HoleFastenerFit::Close) => ("4.39 mm", "7.90 mm", "4.20 mm"),
        ("#8", HoleFastenerFit::Normal) => ("4.50 mm", "7.90 mm", "4.20 mm"),
        ("#8", HoleFastenerFit::Loose) => ("4.75 mm", "8.30 mm", "4.50 mm"),
        _ => match fit {
            HoleFastenerFit::Close => ("4.80 mm", "8.80 mm", "4.60 mm"),
            HoleFastenerFit::Normal => ("5.10 mm", "9.00 mm", "4.80 mm"),
            HoleFastenerFit::Loose => ("5.40 mm", "9.40 mm", "5.10 mm"),
        },
    }
}

pub(super) fn fit_label(fit: HoleFastenerFit) -> &'static str {
    match fit {
        HoleFastenerFit::Close => "Close",
        HoleFastenerFit::Normal => "Normal",
        HoleFastenerFit::Loose => "Loose",
    }
}

pub(super) fn termination_label(termination: GuiHoleTermination) -> &'static str {
    match termination {
        GuiHoleTermination::Distance => "Distance",
        GuiHoleTermination::ThroughAll => "Through All",
        GuiHoleTermination::ToSurface => "To Surface",
    }
}

pub(super) fn surface_label(surface: &HoleTerminationSurface) -> &'static str {
    match surface {
        HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xy) => "1 Surface (XY Plane)",
        HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xz) => "1 Surface (XZ Plane)",
        HoleTerminationSurface::OriginPlane(HoleOriginPlane::Yz) => "1 Surface (YZ Plane)",
        HoleTerminationSurface::Face(_) => "1 Surface",
    }
}

pub(super) fn parse_browser_plane(reference: &str) -> Option<HoleTerminationSurface> {
    let lower = reference.trim().to_ascii_lowercase();
    if lower.contains("xy") {
        Some(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xy))
    } else if lower.contains("xz") {
        Some(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Xz))
    } else if lower.contains("yz") {
        Some(HoleTerminationSurface::OriginPlane(HoleOriginPlane::Yz))
    } else {
        None
    }
}

pub(super) fn parse_length_mm(text: &str) -> Option<f64> {
    let trimmed = text.trim().replace(',', ".");
    let (num_str, factor) = if let Some(s) = trimmed.strip_suffix("mm") {
        (s.trim(), 1.0)
    } else if let Some(s) = trimmed.strip_suffix("cm") {
        (s.trim(), 10.0)
    } else if let Some(s) = trimmed.strip_suffix("in") {
        (s.trim(), 25.4)
    } else {
        (trimmed.as_str(), 1.0)
    };
    let value: f64 = num_str.parse().ok()?;
    (value.is_finite() && value > 0.0).then_some(value * factor)
}

pub(super) fn parse_rotation_deg(text: &str) -> Option<f64> {
    let trimmed = text
        .trim()
        .replace(',', ".")
        .replace('\u{00b0}', "")
        .replace("deg", "");
    let value: f64 = trimmed.trim().parse().ok()?;
    value.is_finite().then_some(value)
}

pub(super) fn parse_angle_deg(text: &str) -> Option<f64> {
    let value = parse_rotation_deg(text)?;
    (value > 0.0 && value < 180.0).then_some(value)
}

/// Return a validation message when current Slot dialog values violate CAD constraints.
pub(super) fn validate_fields(dialog: &super::SlotDialog) -> Option<&'static str> {
    let Some(length_mm) = parse_length_mm(&dialog.length) else {
        return Some("Slot length must be a positive length");
    };
    let Some(width_mm) = parse_length_mm(&dialog.width) else {
        return Some("Slot width must be a positive length");
    };
    if length_mm + 1e-6 < width_mm {
        return Some("Slot length must be greater than or equal to width");
    }
    if parse_rotation_deg(&dialog.rotation).is_none() {
        return Some("Slot rotation must be a finite angle");
    }
    if dialog.termination == GuiHoleTermination::Distance
        && parse_length_mm(&dialog.depth).is_none()
    {
        return Some("Slot depth must be a positive length");
    }
    match dialog.seat {
        GuiHoleSeatKind::None => {}
        GuiHoleSeatKind::Counterbore | GuiHoleSeatKind::Spotface => {
            let Some(sw) = parse_length_mm(&dialog.seat_width) else {
                return Some("Seat width must be a positive length");
            };
            if sw <= width_mm {
                return Some("Seat width must be greater than slot width");
            }
            if parse_length_mm(&dialog.seat_depth).is_none() {
                return Some("Seat depth must be a positive length");
            }
        }
        GuiHoleSeatKind::Countersink => {
            let Some(sw) = parse_length_mm(&dialog.seat_width) else {
                return Some("Countersink width must be a positive length");
            };
            if sw <= width_mm {
                return Some("Countersink width must be greater than slot width");
            }
            if parse_angle_deg(&dialog.seat_angle).is_none() {
                return Some("Countersink angle must be between 0° and 180°");
            }
        }
    }
    if dialog.end_mill_pointed && parse_angle_deg(&dialog.end_mill_angle).is_none() {
        return Some("Point angle must be between 0° and 180°");
    }
    None
}
