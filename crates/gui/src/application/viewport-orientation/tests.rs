// SPDX-License-Identifier: GPL-2.0-only
use super::*;
const FRONT_ON: GuiViewAxes = GuiViewAxes {
    right: [1., 0., 0.],
    up: [0., 1., 0.],
    toward: [0., 0., 1.],
};
fn isometric() -> GuiViewAxes {
    let third = 1. / 3f32.sqrt();
    let (a, b) = (1. / 2f32.sqrt(), 1. / 6f32.sqrt());
    GuiViewAxes {
        right: [a, 0., -a],
        up: [-b, 2. * b, -b],
        toward: [third; 3],
    }
}
#[test]
fn an_iso_view_shows_three_faces_and_a_press_picks_the_one_under_it() {
    let layout = GuiOrientationLayout::for_side(120.);
    let faces = shown_faces(&isometric(), &layout, [0., 0.]);
    let mut names: Vec<_> = faces.iter().map(|shown| shown.face.label()).collect();
    names.sort_unstable();
    assert_eq!(names, ["Front", "Right", "Top"]);
    let above = [layout.center[0], layout.center[1] - layout.half * 0.9];
    let picked = cube_hit(above, &isometric(), &layout, &faces, [0., 0.]).map(|hit| hit.command);
    assert_eq!(picked, Some(GuiOrientCommand::Face(GuiCubeFace::Top)));
}
#[test]
fn face_labels_follow_the_projected_horizontal_and_vertical_edges() {
    let layout = GuiOrientationLayout::for_side(120.);
    let front = shown_faces(&FRONT_ON, &layout, [0., 0.]).remove(0);
    assert_eq!(front.label_basis(layout.half), [[1., 0.], [0., 1.]]);
    let faces = shown_faces(&isometric(), &layout, [0., 0.]);
    for side in faces.iter().filter(|shown| shown.face != GuiCubeFace::Top) {
        let [right, down] = side.label_basis(layout.half);
        assert!(right[0] > 0.7 && right[1].abs() > 0.4);
        assert!(down[0].abs() < 0.00001 && down[1] > 0.8);
    }
    let top = faces
        .iter()
        .find(|shown| shown.face == GuiCubeFace::Top)
        .unwrap();
    let [right, down] = top.label_basis(layout.half);
    assert!(right[0] > 0.7 && right[1] > 0.4);
    assert!(down[0] < -0.7 && down[1] > 0.4);
}
#[test]
fn a_face_on_rim_picks_corners_and_the_face_beyond_an_edge() {
    let layout = GuiOrientationLayout::for_side(120.);
    let at = |across: f32, down: f32| {
        [
            layout.center[0] + across * layout.half,
            layout.center[1] + down * layout.half,
        ]
    };
    let faces = shown_faces(&FRONT_ON, &layout, [0., 0.]);
    let pick = |point| cube_hit(point, &FRONT_ON, &layout, &faces, [0., 0.]).map(|hit| hit.command);
    assert_eq!(
        pick(at(0., 0.)),
        Some(GuiOrientCommand::Face(GuiCubeFace::Front))
    );
    assert_eq!(
        pick(at(0., -0.9)),
        Some(GuiOrientCommand::Face(GuiCubeFace::Top))
    );
    assert_eq!(
        pick(at(0.9, -0.9)),
        Some(GuiOrientCommand::Corner {
            right: true,
            up: true
        })
    );
    assert_eq!(pick(at(1.5, 0.)), None);
}
#[test]
fn the_arrows_answer_only_face_on() {
    let layout = GuiOrientationLayout::for_side(120.);
    assert_eq!(
        arrow_at(layout.step_right, &FRONT_ON, &layout),
        Some(GuiOrientCommand::Step(GuiViewStep::Right))
    );
    assert_eq!(arrow_at(layout.step_right, &isometric(), &layout), None);
}

#[test]
fn oblique_face_strips_route_to_model_space_edges_and_corners() {
    let axes = isometric();
    let layout = GuiOrientationLayout::for_side(120.);
    assert_eq!(layout.half, 32.);
    let faces = shown_faces(&axes, &layout, [0., 0.]);
    let face = faces
        .iter()
        .find(|face| face.face == GuiCubeFace::Front)
        .unwrap();
    let [a, b, _, d] = face.corners;
    let sample =
        |u: f32, v: f32| std::array::from_fn(|i| a[i] + u * (b[i] - a[i]) + v * (d[i] - a[i]));
    for (point, command) in [
        (sample(0.5, 0.5), GuiOrientCommand::Face(GuiCubeFace::Front)),
        (sample(0.97, 0.5), GuiOrientCommand::Bevel([1, 0, 1])),
        (sample(0.97, 0.97), GuiOrientCommand::Bevel([1, 1, 1])),
    ] {
        let hit = cube_hit(point, &axes, &layout, &faces, [0., 0.]).unwrap();
        assert_eq!(hit.command, command);
        assert!(hit.regions.iter().any(|region| cube::inside(point, region)));
    }
}

#[test]
fn every_side_of_a_bevel_highlights_the_same_region_with_only_an_outer_border() {
    let axes = isometric();
    let layout = GuiOrientationLayout::for_side(120.);
    let origin = [1370., 130.];
    let faces = shown_faces(&axes, &layout, origin);
    for (samples, pieces, vertices) in [
        (&[[0., 1., 0.94], [0., 0.94, 1.]][..], 2, 7),
        (
            &[[0.94, 1., 0.94], [0.94, 0.94, 1.], [1., 0.94, 0.94]][..],
            3,
            7,
        ),
    ] {
        let hits: Vec<_> = samples
            .iter()
            .map(|&sample| {
                let point = cube::project(sample, &axes, &layout, origin);
                cube_hit(point, &axes, &layout, &faces, origin).unwrap()
            })
            .collect();
        let hit = &hits[0];
        assert_eq!(hit.regions.len(), pieces);
        for other in &hits[1..] {
            assert_eq!(other.command, hit.command);
            assert_eq!(other.regions, hit.regions);
        }
        let outline = hit.outline();
        assert_eq!(outline.len(), vertices);
        assert_eq!(outline.first(), outline.last());
    }
}
