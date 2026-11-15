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

fn document(id: u64, visible: bool) -> oblikovati_api::wire::DocumentInfo {
    oblikovati_api::wire::DocumentInfo {
        id,
        visible,
        ..Default::default()
    }
}
fn window(id: u64, document: u64) -> crate::GuiDocumentWindow {
    crate::GuiDocumentWindow {
        id,
        document,
        number: 1,
    }
}
fn capture(previews: &mut TabPreviews, document: u64, window: u64) {
    previews.showing = Some((PreviewKey::Window(window), [[0., 0.], [2., 2.]]));
    previews.showing_document = Some(document);
    previews.note_frame([2, 2], &[9; 16]);
}

#[test]
fn closed_document_loses_picture_and_pending_notification() {
    let mut previews = TabPreviews::default();
    let mut state = crate::GuiCommandState {
        documents: vec![document(1, true), document(2, true)],
        ..Default::default()
    };
    previews.synchronize(&state);
    previews.showing = Some((PreviewKey::Home, [[0., 0.], [2., 2.]]));
    previews.note_frame([2, 2], &[8; 16]);
    capture(&mut previews, 2, 2);
    capture(&mut previews, 1, 1);
    state.documents.remove(0);
    previews.synchronize(&state);
    assert!(!previews.images.contains_key(&PreviewKey::Window(1)));
    assert!(previews.images.contains_key(&PreviewKey::Window(2)));
    assert!(previews.images.contains_key(&PreviewKey::Home));
    assert!(previews.take_changed_picture().is_none());
    assert!(previews.showing.is_none());
}

#[test]
fn hidden_document_remains_open_and_keeps_its_picture() {
    let mut previews = TabPreviews::default();
    let state = crate::GuiCommandState {
        documents: vec![document(1, false)],
        ..Default::default()
    };
    previews.synchronize(&state);
    capture(&mut previews, 1, 1);
    previews.synchronize(&state);
    assert!(previews.images.contains_key(&PreviewKey::Window(1)));
    assert_eq!(previews.take_changed_picture().map(|(id, _)| id), Some(1));
}

#[test]
fn closing_one_of_three_windows_only_discards_that_window() {
    let mut previews = TabPreviews::default();
    let mut state = crate::GuiCommandState {
        documents: vec![document(1, true)],
        document_windows: vec![window(1, 1), window(90, 1), window(91, 1)],
        ..Default::default()
    };
    previews.synchronize(&state);
    capture(&mut previews, 1, 1);
    capture(&mut previews, 1, 90);
    capture(&mut previews, 1, 91);
    state.document_windows.retain(|item| item.id != 91);
    previews.synchronize(&state);
    assert!(!previews.images.contains_key(&PreviewKey::Window(91)));
    assert!(previews.images.contains_key(&PreviewKey::Window(90)));
    assert!(previews.images.contains_key(&PreviewKey::Window(1)));
    assert!(previews.take_changed_picture().is_none());
}

#[test]
fn collapsing_windows_discards_ambiguous_original_identity() {
    let mut previews = TabPreviews::default();
    let mut state = crate::GuiCommandState {
        documents: vec![document(1, true)],
        document_windows: vec![window(1, 1), window(90, 1)],
        ..Default::default()
    };
    previews.synchronize(&state);
    capture(&mut previews, 1, 1);
    state.document_windows.clear();
    state.document_window_identity_revisions.insert(1, 1);
    previews.synchronize(&state);
    assert!(!previews.images.contains_key(&PreviewKey::Window(1)));
    assert!(previews.take_changed_picture().is_none());
    capture(&mut previews, 1, 1);
    assert!(previews.take_changed_picture().is_some());
}

#[test]
fn stable_survivor_keeps_its_picture_when_extra_window_closes() {
    let mut previews = TabPreviews::default();
    let mut state = crate::GuiCommandState {
        documents: vec![document(1, true)],
        document_windows: vec![window(1, 1), window(90, 1)],
        ..Default::default()
    };
    previews.synchronize(&state);
    capture(&mut previews, 1, 1);
    state.document_windows.clear();
    previews.synchronize(&state);
    assert!(previews.images.contains_key(&PreviewKey::Window(1)));
    assert!(previews.take_changed_picture().is_some());
}

#[test]
fn identity_revision_detects_open_and_close_between_frames() {
    let mut previews = TabPreviews::default();
    let mut state = crate::GuiCommandState {
        documents: vec![document(1, true)],
        ..Default::default()
    };
    previews.synchronize(&state);
    capture(&mut previews, 1, 1);
    state.document_window_identity_revisions.insert(1, 7);
    previews.synchronize(&state);
    assert!(!previews.images.contains_key(&PreviewKey::Window(1)));
    assert!(previews.take_changed_picture().is_none());
    capture(&mut previews, 1, 1);
    assert!(previews.take_changed_picture().is_some());
}

#[test]
fn pending_picture_is_delivered_before_its_document_or_window_closes() {
    for close in [
        crate::GuiShellAction::CloseDocument(oblikovati_api::wire::CloseDocumentArgs {
            id: 1,
            ..Default::default()
        }),
        crate::GuiShellAction::CloseDocumentWindow(90),
    ] {
        let mut previews = TabPreviews::default();
        capture(&mut previews, 1, 90);
        let mut actions = vec![close.clone()];
        previews.enqueue_picture(&mut actions);
        assert!(matches!(
            actions[0],
            crate::GuiShellAction::KeepDocumentPicture {
                document: 1,
                picture: Some(_)
            }
        ));
        assert_eq!(actions[1], close);
    }
}

#[test]
fn reopened_document_identity_sends_a_new_picture() {
    let mut previews = TabPreviews::default();
    let mut state = crate::GuiCommandState {
        documents: vec![document(1, true)],
        ..Default::default()
    };
    previews.synchronize(&state);
    capture(&mut previews, 1, 1);
    assert!(previews.take_changed_picture().is_some());
    state.documents.clear();
    previews.synchronize(&state);
    state.documents.push(document(1, true));
    previews.synchronize(&state);
    capture(&mut previews, 1, 1);
    assert!(previews.take_changed_picture().is_some());
}

#[test]
fn closing_another_window_does_not_discard_the_open_view_picture() {
    let mut previews = TabPreviews::default();
    capture(&mut previews, 1, 90);
    let close = crate::GuiShellAction::CloseDocumentWindow(91);
    let mut actions = vec![close.clone()];
    previews.enqueue_picture(&mut actions);
    assert_eq!(actions[0], close);
    assert!(matches!(
        actions[1],
        crate::GuiShellAction::KeepDocumentPicture {
            document: 1,
            picture: Some(_)
        }
    ));
}
