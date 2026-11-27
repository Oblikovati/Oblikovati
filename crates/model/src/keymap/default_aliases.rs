// SPDX-License-Identifier: GPL-2.0-only
//! Go reference aliases for commands the desktop sketch router supports.

const ALIASES: &[(&str, &[&str])] = &[
    ("sketch-line", &["LINE"]),
    ("sketch-circle", &["CIRCLE"]),
    ("sketch-three-point-arc", &["ARC"]),
    ("sketch-rectangle", &["RECTANG", "RECTANGLE", "REC"]),
    ("sketch-point", &["POINT", "PO"]),
    ("sketch-ellipse", &["ELLIPSE", "EL"]),
    ("constraint-coincident", &["GCCOINCIDENT"]),
    ("constraint-collinear", &["GCCOLLINEAR"]),
    ("constraint-concentric", &["GCCONCENTRIC"]),
    ("constraint-equal", &["GCEQUAL"]),
    ("constraint-fix", &["GCFIX"]),
    ("constraint-horizontal", &["GCHORIZONTAL"]),
    ("constraint-vertical", &["GCVERTICAL"]),
    ("constraint-parallel", &["GCPARALLEL"]),
    ("constraint-perpendicular", &["GCPERPENDICULAR"]),
    ("constraint-tangent", &["GCTANGENT"]),
    ("dimension", &["DIMENSION", "DIM"]),
    ("edit.undo", &["UNDO"]),
    ("edit.redo", &["REDO", "MREDO"]),
    ("file.save", &["SAVE", "QSAVE"]),
    ("view.home", &["HOME", "HOMEVIEW"]),
];

pub(super) fn resolve(alias: &str) -> Option<&'static str> {
    ALIASES.iter().find_map(|(id, names)| {
        names
            .iter()
            .any(|name| name.eq_ignore_ascii_case(alias))
            .then_some(*id)
    })
}
