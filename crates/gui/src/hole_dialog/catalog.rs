// SPDX-License-Identifier: GPL-2.0-only
//! Fastener clearance and thread lookup tables plus client-side Hole validation.
use super::types::{GuiHoleSeatKind, GuiHoleTermination};
use oblikovati_model::feature::hole::{
    HoleFastenerFit, HoleFastenerSpec, HoleKind, HoleOriginPlane, HoleTerminationSurface,
    HoleThreadSpec,
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

pub(super) const TAPPED_THREAD_TYPES: &[&str] = &[
    "ISO Metric profile",
    "ANSI Unified Screw Threads",
    "ANSI Metric M Profile",
];

pub(super) const TAPPED_SIZES: &[(&str, &str, &str)] = &[
    ("M3", "M3x0.5", "2.5 mm"),
    ("M4", "M4x0.7", "3.3 mm"),
    ("M5", "M5x0.8", "4.2 mm"),
    ("M6", "M6x1", "5 mm"),
    ("M8", "M8x1.25", "6.8 mm"),
    ("M10", "M10x1.5", "8.5 mm"),
];

pub(super) const TAPER_THREAD_TYPES: &[&str] = &["NPT", "ANSI Metric Pipe Thread"];

pub(super) const TAPER_SIZES: &[(&str, &str, &str)] = &[
    ("1/16", "1/16\u{2013}27 NPT", "6.2 mm"),
    ("1/8", "1/8\u{2013}27 NPT", "8.5 mm"),
    ("1/4", "1/4\u{2013}18 NPT", "11.1 mm"),
    ("3/8", "3/8\u{2013}18 NPT", "14.5 mm"),
    ("1/2", "1/2\u{2013}14 NPT", "18 mm"),
];

pub(super) fn default_fastener() -> HoleFastenerSpec {
    HoleFastenerSpec {
        standard: CLEARANCE_STANDARDS[0].into(),
        fastener_type: CLEARANCE_FASTENER_TYPES[0].into(),
        size: "#6".into(),
        fit: HoleFastenerFit::Normal,
    }
}

pub(super) fn default_tapped_thread() -> HoleThreadSpec {
    HoleThreadSpec {
        profile: TAPPED_THREAD_TYPES[0].into(),
        designation: "M4x0.7".into(),
        full_depth: false,
        left_hand: false,
    }
}

pub(super) fn default_taper_thread() -> HoleThreadSpec {
    HoleThreadSpec {
        profile: TAPER_THREAD_TYPES[0].into(),
        designation: "1/16\u{2013}27 NPT".into(),
        full_depth: false,
        left_hand: false,
    }
}

/// Deterministic clearance bore and seat dimensions (mm strings) for fastener size + fit.
pub(super) fn clearance_dimensions(
    size: &str,
    fit: HoleFastenerFit,
) -> (&'static str, &'static str, &'static str) {
    let base = match size {
        "#0" => (1.70, 3.00, 1.60),
        "#2" => (2.40, 4.40, 2.20),
        "#4" => (3.26, 5.84, 2.85),
        "#6" => (3.80, 6.80, 3.50),
        "#8" => (4.50, 7.90, 4.20),
        "#10" => (5.10, 9.00, 4.80),
        "1/4" => (6.76, 11.10, 6.40),
        "5/16" => (8.33, 13.50, 7.90),
        "3/8" => (10.00, 15.90, 9.50),
        "1/2" => (13.50, 20.60, 12.70),
        _ => (4.00, 7.00, 3.50),
    };
    match (size, fit) {
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
            HoleFastenerFit::Close if base.0 < 3.0 => ("2.20 mm", "4.40 mm", "2.20 mm"),
            HoleFastenerFit::Loose if base.0 < 3.0 => ("2.60 mm", "4.80 mm", "2.40 mm"),
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

/// Parse a simple numeric length text (mm, cm, in) for live UI validation & diagram preview.
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

pub(super) fn parse_angle_deg(text: &str) -> Option<f64> {
    let trimmed = text
        .trim()
        .replace(',', ".")
        .replace('\u{00b0}', "")
        .replace("deg", "");
    let value: f64 = trimmed.trim().parse().ok()?;
    (value.is_finite() && value > 0.0 && value < 180.0).then_some(value)
}

/// Return a validation message when current Hole dialog values violate CAD constraints.
pub(super) fn validate_fields(dialog: &super::HoleDialog) -> Option<&'static str> {
    let Some(bore_mm) = parse_length_mm(&dialog.diameter) else {
        return Some("Hole diameter must be a positive length");
    };
    if dialog.termination == GuiHoleTermination::Distance
        && parse_length_mm(&dialog.depth).is_none()
    {
        return Some("Hole depth must be a positive length");
    }
    if dialog.kind == HoleKind::TaperTapped && dialog.seat != GuiHoleSeatKind::None {
        return Some("Taper tapped holes do not support a seat");
    }
    match dialog.seat {
        GuiHoleSeatKind::None => {}
        GuiHoleSeatKind::Counterbore | GuiHoleSeatKind::Spotface => {
            let Some(sd) = parse_length_mm(&dialog.seat_diameter) else {
                return Some("Seat diameter must be a positive length");
            };
            if sd <= bore_mm {
                return Some("Seat diameter must be greater than hole diameter");
            }
            if parse_length_mm(&dialog.seat_depth).is_none() {
                return Some("Seat depth must be a positive length");
            }
        }
        GuiHoleSeatKind::Countersink => {
            let Some(sd) = parse_length_mm(&dialog.seat_diameter) else {
                return Some("Countersink diameter must be a positive length");
            };
            if sd <= bore_mm {
                return Some("Countersink diameter must be greater than hole diameter");
            }
            if parse_angle_deg(&dialog.seat_angle).is_none() {
                return Some("Countersink angle must be between 0° and 180°");
            }
        }
    }
    if dialog.termination == GuiHoleTermination::Distance
        && dialog.drill_point_angled
        && parse_angle_deg(&dialog.drill_point_angle).is_none()
    {
        return Some("Drill point angle must be between 0° and 180°");
    }
    None
}
