// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use martensite_tkbrep::{brep::BRepBuilder, topods::Face};

#[test]
fn validity_checker_failure_retains_the_probe_input() {
    let extents = [2.0, 3.0, 4.0];
    let message = require_valid_probe(&Shape::default(), extents).unwrap_err();
    assert!(message.contains(&format!("{extents:?}")));
    assert!(message.contains("expected geometric validity"));
}

#[test]
fn invalid_geometry_is_refused_before_reading_properties() {
    let mut face = Face::default();
    BRepBuilder::default().make_face(&mut face);
    let extents = [2.0, 3.0, 4.0];
    let message = require_valid_probe(&face.into(), extents).unwrap_err();
    assert!(message.contains(&format!("{extents:?}")));
    assert!(message.contains("expected a geometrically valid B-rep"));
}
