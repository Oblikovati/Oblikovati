// SPDX-License-Identifier: GPL-2.0-only
//! Standard thread tables, face-based default selection, and client-side validation (#3918).
use oblikovati_model::feature::thread::{ThreadHandedness, ThreadSpecification};

pub(super) const THREAD_TYPES: &[&str] = &[
    "ISO Taper External",
    "JIS Pipe Threads",
    "ANSI Metric M Profile",
];

pub(super) fn sizes_for_type(thread_type: &str) -> &'static [&'static str] {
    match thread_type {
        "ISO Taper External" => &["R 1/4", "R 3/8", "R 1/2", "R 6"],
        "JIS Pipe Threads" => &["13.157", "16.662", "20.955"],
        "ANSI Metric M Profile" => &["5", "6", "8", "9", "10", "12"],
        _ => &["9"],
    }
}

pub(super) fn designations_for(thread_type: &str, size: &str) -> &'static [&'static str] {
    match (thread_type, size) {
        ("ISO Taper External", "R 1/4") => &["R 1/4"],
        ("ISO Taper External", "R 3/8") => &["R 3/8"],
        ("ISO Taper External", "R 1/2") => &["R 1/2"],
        ("ISO Taper External", "R 6") => &["R 6"],
        ("JIS Pipe Threads", "13.157") => &["G 1/4"],
        ("JIS Pipe Threads", "16.662") => &["G 3/8"],
        ("JIS Pipe Threads", "20.955") => &["G 1/2"],
        ("ANSI Metric M Profile", "5") => &["M5x0.8", "M5x0.5"],
        ("ANSI Metric M Profile", "6") => &["M6x1", "M6x0.75", "M6x0.5"],
        ("ANSI Metric M Profile", "8") => &["M8x1.25", "M8x1", "M8x0.75"],
        ("ANSI Metric M Profile", "9") => &["M9x1.25", "M9x1", "M9x0.75", "M9x0.5"],
        ("ANSI Metric M Profile", "10") => &["M10x1.5", "M10x1.25", "M10x1", "M10x0.75"],
        ("ANSI Metric M Profile", "12") => &["M12x1.75", "M12x1.5", "M12x1.25", "M12x1"],
        _ => &["M9x1.25"],
    }
}

pub(super) fn classes_for(thread_type: &str, internal: bool) -> &'static [&'static str] {
    match thread_type {
        "ISO Taper External" => &["6g", "A", "B"],
        "JIS Pipe Threads" => &["A", "B"],
        "ANSI Metric M Profile" if internal => &["6H", "5H", "7H"],
        "ANSI Metric M Profile" => &["6g", "4g6g"],
        _ => &["6H"],
    }
}

/// Apply defaults when the user changes `Thread Type`.
pub(super) fn on_thread_type_changed(spec: &mut ThreadSpecification, internal: bool) {
    let sizes = sizes_for_type(&spec.thread_type);
    spec.size = match spec.thread_type.as_str() {
        "ISO Taper External" => "R 6".into(),
        "JIS Pipe Threads" => "13.157".into(),
        "ANSI Metric M Profile" => "9".into(),
        _ => sizes.first().copied().unwrap_or("9").into(),
    };
    let designations = designations_for(&spec.thread_type, &spec.size);
    spec.designation = designations.first().copied().unwrap_or("M9x1.25").into();
    let classes = classes_for(&spec.thread_type, internal);
    spec.class = classes.first().copied().unwrap_or("A").into();
}

/// Update `Designation` and keep `Class` when still valid after a `Size` change.
pub(super) fn on_size_changed(spec: &mut ThreadSpecification, internal: bool) {
    let designations = designations_for(&spec.thread_type, &spec.size);
    spec.designation = designations.first().copied().unwrap_or("M9x1.25").into();
    let classes = classes_for(&spec.thread_type, internal);
    if !classes.iter().any(|candidate| *candidate == spec.class) {
        spec.class = classes.first().copied().unwrap_or("A").into();
    }
}

/// Choose the default specification and initial depth string for a picked face.
pub(super) fn default_for_face(
    diameter_mm: f64,
    full_depth_mm: f64,
    conical: bool,
    internal: bool,
) -> (ThreadSpecification, String) {
    if conical && !internal {
        let spec = ThreadSpecification {
            thread_type: "ISO Taper External".into(),
            size: "R 6".into(),
            designation: "R 6".into(),
            class: "6g".into(),
            handedness: ThreadHandedness::RightHand,
        };
        return (spec, "10 mm".into());
    }
    let size = nearest_metric_size(diameter_mm);
    let designation = designations_for("ANSI Metric M Profile", size)
        .first()
        .copied()
        .unwrap_or("M9x1.25");
    let class = classes_for("ANSI Metric M Profile", internal)
        .first()
        .copied()
        .unwrap_or("6H");
    let depth = if full_depth_mm >= 10.0 {
        "10 mm".into()
    } else {
        format_mm_3dp(full_depth_mm)
    };
    (
        ThreadSpecification {
            thread_type: "ANSI Metric M Profile".into(),
            size: size.into(),
            designation: designation.into(),
            class: class.into(),
            handedness: ThreadHandedness::RightHand,
        },
        depth,
    )
}

fn nearest_metric_size(diameter_mm: f64) -> &'static str {
    let candidates: [(f64, &str); 6] = [
        (5.0, "5"),
        (6.0, "6"),
        (8.0, "8"),
        (9.0, "9"),
        (10.0, "10"),
        (12.0, "12"),
    ];
    candidates
        .into_iter()
        .min_by(|(a, _), (b, _)| (a - diameter_mm).abs().total_cmp(&(b - diameter_mm).abs()))
        .map_or("9", |(_, label)| label)
}

pub(super) fn format_mm_3dp(mm: f64) -> String {
    format!("{mm:.3} mm")
}

pub(super) fn parse_length_mm(raw: &str) -> Option<f64> {
    let cleaned = raw.trim().replace(',', ".");
    if cleaned.is_empty() {
        return None;
    }
    let lower = cleaned.to_ascii_lowercase();
    let (num_str, factor) = if let Some(prefix) = lower.strip_suffix("mm") {
        (prefix.trim(), 1.0)
    } else if let Some(prefix) = lower.strip_suffix("cm") {
        (prefix.trim(), 10.0)
    } else if let Some(prefix) = lower.strip_suffix("in") {
        (prefix.trim(), 25.4)
    } else if let Some(prefix) = lower.strip_suffix('m') {
        (prefix.trim(), 1000.0)
    } else {
        (lower.as_str(), 1.0)
    };
    let val: f64 = num_str.parse().ok()?;
    val.is_finite().then_some(val * factor)
}

/// Validate Depth and Offset fields client-side so zero/negative depths immediately disable OK.
pub(super) fn validate_fields(full_depth: bool, depth: &str, offset: &str) -> Option<&'static str> {
    if full_depth {
        return None;
    }
    match parse_length_mm(depth) {
        Some(value) if value > 0.0 => {}
        _ => return Some("Depth must be greater than zero"),
    }
    let trimmed_offset = offset.trim();
    if !trimmed_offset.is_empty() {
        match parse_length_mm(trimmed_offset) {
            Some(value) if value >= 0.0 => {}
            _ => return Some("Offset must be zero or positive"),
        }
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn jis_pipe_size_change_updates_designation_and_preserves_class() {
        let mut spec = ThreadSpecification {
            thread_type: "JIS Pipe Threads".into(),
            size: String::new(),
            designation: String::new(),
            class: String::new(),
            handedness: ThreadHandedness::RightHand,
        };
        on_thread_type_changed(&mut spec, false);
        assert_eq!(spec.size, "13.157");
        assert_eq!(spec.designation, "G 1/4");
        assert_eq!(spec.class, "A");

        spec.size = "16.662".into();
        on_size_changed(&mut spec, false);
        assert_eq!(spec.designation, "G 3/8");
        assert_eq!(spec.class, "A");
        assert_eq!(classes_for(&spec.thread_type, false), &["A", "B"]);
    }

    #[test]
    fn metric_size_9_offers_all_four_pitches_and_validates_zero_depth() {
        assert_eq!(
            designations_for("ANSI Metric M Profile", "9"),
            &["M9x1.25", "M9x1", "M9x0.75", "M9x0.5"]
        );
        assert!(validate_fields(false, "0", "0 mm").is_some());
        assert!(validate_fields(false, "0 mm", "0 mm").is_some());
        assert!(validate_fields(false, "20 mm", "99 mm").is_none());
        assert!(validate_fields(true, "0", "0 mm").is_none());
    }
}
