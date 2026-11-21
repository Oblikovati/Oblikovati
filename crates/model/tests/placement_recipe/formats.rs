// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn creation_formats_preserve_intrinsic_construction_and_only_lines_become_axes() {
    let mut rectangle = SketchPlacementRecipe::center_rectangle([0.; 2], [4., 3.]).unwrap();
    let intrinsic: Vec<_> = rectangle
        .members
        .iter()
        .map(|member| member.construction)
        .collect();
    rectangle.apply_creation_formats(false, true).unwrap();
    assert!(rectangle.members.iter().all(|member| member.centerline));
    assert_eq!(
        rectangle
            .members
            .iter()
            .map(|member| member.construction)
            .collect::<Vec<_>>(),
        intrinsic
    );
    rectangle.apply_creation_formats(true, false).unwrap();
    assert!(rectangle.members.iter().all(|member| member.construction));
}
#[test]
fn only_line_members_receive_an_armed_axis_role() {
    let mut circle = SketchPlacementRecipe::circle([0.; 2], 2.).unwrap();
    circle.apply_creation_formats(true, true).unwrap();
    assert!(circle.members[0].construction && !circle.members[0].centerline);
}
#[test]
fn unsupported_construction_point_admission_is_atomic() {
    let mut point = SketchPlacementRecipe::point([0.; 2]).unwrap();
    let original = point.clone();
    assert!(
        point
            .apply_creation_formats(true, false)
            .unwrap_err()
            .to_string()
            .contains("construction points are unsupported")
    );
    assert_eq!(point, original);
    point.apply_creation_formats(false, true).unwrap();
    assert_eq!(point, original);
}
