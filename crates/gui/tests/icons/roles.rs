// SPDX-License-Identifier: GPL-2.0-only
use super::*;
const QUADRANTS: &[u8] = br##"<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" stroke="none" fill="none"><rect x="0" y="0" width="5" height="5" fill="#00ff00"/><rect x="5" y="0" width="5" height="5" fill="#0000ff"/><rect x="0" y="5" width="5" height="5" fill="#ff0000"/><rect x="5" y="5" width="5" height="5" fill="#000000"/></svg>"##;
fn quadrant(role: usize) -> usize {
    let x = if role.is_multiple_of(2) { 4 } else { 12 };
    let y = if role < 2 { 4 } else { 12 };
    y * 16 + x
}
#[test]
fn sentinel_passes_share_one_crop_and_preserve_their_quadrants() {
    let masks = rasterize_roles(QUADRANTS, 16).unwrap();
    assert_eq!(masks.edge, 16);
    for role in 0..4 {
        for layer in 0..4 {
            let coverage = masks.cover[layer][quadrant(role)];
            assert!(
                if role == layer {
                    coverage > 200
                } else {
                    coverage < 30
                },
                "role {role}, layer {layer}: unexpected coverage {coverage}"
            );
        }
    }
}
#[test]
fn composition_replaces_sentinel_paints_with_theme_colors() {
    let colors = [
        [0., 0., 0., 1.],
        [0., 0., 1., 1.],
        [0., 1., 0., 1.],
        [1., 0., 0., 1.],
    ];
    let image = rasterize_roles(QUADRANTS, 16)
        .unwrap()
        .compose(colors)
        .unwrap();
    for (role, expected) in [
        [0, 0, 0, 255],
        [0, 0, 255, 255],
        [0, 255, 0, 255],
        [255, 0, 0, 255],
    ]
    .into_iter()
    .enumerate()
    {
        assert_eq!(image.rgba[quadrant(role)], expected);
    }
}
#[test]
fn source_over_outputs_straight_color_and_handles_fully_transparent_pixels() {
    let masks = IconRoleMasks {
        edge: 1,
        cover: [vec![255], vec![0], vec![255], vec![0]],
    };
    let colors = [[0., 0., 1., 1.], [0.; 4], [1., 0., 0., 0.5], [0.; 4]];
    assert_eq!(masks.compose(colors).unwrap().rgba[0], [128, 0, 128, 255]);
    assert_eq!(masks.compose([[0.; 4]; 4]).unwrap().rgba[0], [0; 4]);
    assert!(masks.compose([[f32::NAN; 4]; 4]).is_err());
}
#[test]
fn unsupported_content_is_declined_before_a_role_image_is_published() {
    for source in [
        b"<svg><image href='file:///etc/passwd'/></svg>".as_slice(),
        b"<svg><text>A</text></svg>",
        b"<!DOCTYPE svg><svg/>",
        b"<svg><path fill='orange'/></svg>",
        b"<svg>",
    ] {
        assert!(
            rasterize_roles(source, 16).is_err(),
            "expected decline for {source:?}"
        );
    }
    assert!(
        rasterize_roles(b"<svg xmlns='http://www.w3.org/2000/svg'/>", 16)
            .unwrap_err()
            .contains("nonempty alpha")
    );
    for edge in [0, 513] {
        assert!(rasterize_roles(QUADRANTS, edge).is_err());
    }
}
#[test]
fn inherited_named_and_short_hex_paints_keep_the_same_roles() {
    let source=br##"<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10" fill="RED"><g><rect x="0" y="0" width="10" height="10"/></g></svg>"##;
    let mask = rasterize_roles(source, 16).unwrap();
    assert_eq!(mask.cover[2][8 * 16 + 8], 255);
    assert_eq!(mask.cover[3][8 * 16 + 8], 0);
}

#[test]
fn all_shipped_command_icons_rasterize_at_go_small_and_large_sizes() {
    for expected in ["revolve", "hole", "fillet", "chamfer"] {
        assert!(
            catalog::SOURCES.iter().any(|(name, _)| *name == expected),
            "missing {expected} icon"
        );
    }
    for (name, source) in catalog::SOURCES {
        for edge in [18, 40] {
            let masks = rasterize_roles(source, edge)
                .unwrap_or_else(|cause| panic!("{name} at {edge}px: {cause}"));
            assert!(
                masks.cover[0].iter().any(|alpha| *alpha > 16),
                "{name}: expected background coverage"
            );
            assert!(
                masks.cover[3].iter().any(|alpha| *alpha > 16),
                "{name}: expected primary coverage"
            );
            assert!(
                masks
                    .cover
                    .iter()
                    .all(|mask| mask.len() == (edge * edge) as usize)
            );
        }
    }
}
#[test]
fn foreign_namespaces_and_inherited_non_role_paints_are_named_declines() {
    for source in [
        b"<svg xmlns='urn:foreign'><rect width='1' height='1'/></svg>".as_slice(),
        b"<svg fill='orange'><g><rect width='1' height='1'/></g></svg>",
    ] {
        assert!(rasterize_roles(source, 18).is_err());
    }
}
