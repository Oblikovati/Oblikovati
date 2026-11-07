// SPDX-License-Identifier: GPL-2.0-only
use super::*;

fn frame() -> Vec<u8> {
    // Left half red, right half blue, in a 4 by 2 image.
    let mut rgba = Vec::new();
    for _ in 0..2 {
        rgba.extend_from_slice(&[
            200, 0, 0, 255, 200, 0, 0, 255, 0, 0, 200, 255, 0, 0, 200, 255,
        ]);
    }
    rgba
}
#[test]
fn a_picture_comes_only_from_the_given_area() {
    let image = cropped_picture([4, 2], &frame(), [2., 0.], [4., 2.]).unwrap();
    assert!(image.rgb.iter().all(|cell| *cell == [0, 0, 200]));
    assert_eq!(image.rgb.len(), (CELLS[0] * CELLS[1]) as usize);
}
#[test]
fn an_empty_area_or_a_short_frame_gives_no_picture() {
    assert!(cropped_picture([4, 2], &frame(), [3., 0.], [3., 2.]).is_none());
    assert!(cropped_picture([4, 2], &frame()[..8], [0., 0.], [4., 2.]).is_none());
}
#[test]
fn each_window_keeps_its_own_last_picture() {
    let mut previews = TabPreviews::default();
    previews.note_frame([4, 2], &frame());
    assert!(previews.images.is_empty(), "nothing is in front yet");
    previews.showing = Some((PreviewKey::Window(7), [[0., 0.], [2., 2.]]));
    previews.note_frame([4, 2], &frame());
    previews.showing = Some((PreviewKey::Home, [[2., 0.], [4., 2.]]));
    previews.note_frame([4, 2], &frame());
    assert_eq!(previews.images[&PreviewKey::Window(7)].rgb[0], [200, 0, 0]);
    assert_eq!(previews.images[&PreviewKey::Home].rgb[0], [0, 0, 200]);
}
#[test]
fn an_unchanged_picture_is_told_once() {
    let mut previews = TabPreviews {
        showing: Some((PreviewKey::Window(1), [[0., 0.], [2., 2.]])),
        showing_document: Some(1),
        ..Default::default()
    };
    let rgba = [9_u8; 16];
    previews.note_frame([2, 2], &rgba);
    assert_eq!(previews.take_changed_picture().map(|(id, _)| id), Some(1));
    assert_eq!(previews.take_changed_picture(), None);
    previews.note_frame([2, 2], &rgba);
    assert_eq!(previews.take_changed_picture(), None);
    previews.note_frame([2, 2], &[200_u8; 16]);
    assert!(previews.take_changed_picture().is_some());
}
