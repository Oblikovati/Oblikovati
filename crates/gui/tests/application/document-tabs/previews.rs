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
fn narrow_viewport_wraps_a_long_label_and_scales_the_picture() {
    let mut context = crate::test_context::create();
    context.set_ini_filename(None);
    context.io_mut().display_size = [150., 400.];
    context.fonts().build_rgba32_texture();
    let ui = context.frame();
    let tab = HoveredTab {
        key: PreviewKey::Home,
        bounds: [[120., 10.], [170., 30.]],
        labels: vec!["Long document name with spaces and a saved location".into()],
    };
    let (placement, picture) = preview_dimensions(ui, &tab).expect("labels and picture fit");
    assert!(placement.position[0] >= 0. && placement.position[0] + placement.size[0] <= 150.);
    assert!(placement.position[1] >= tab.bounds[1][1]);
    assert!(placement.position[1] + placement.size[1] <= 400.);
    assert!(picture[0] > 0. && picture[0] < CELLS[0] as f32 * CELL_SIZE);
    assert!(wrapped_labels_height(ui, &tab.labels, 130., 0.) > ui.calc_text_size("A")[1]);
}

#[test]
fn impossible_viewport_omits_the_preview_instead_of_covering_the_tab() {
    let mut context = crate::test_context::create();
    context.set_ini_filename(None);
    context.io_mut().display_size = [50., 50.];
    context.fonts().build_rgba32_texture();
    let ui = context.frame();
    let tab = HoveredTab {
        key: PreviewKey::Home,
        bounds: [[0., 20.], [40., 30.]],
        labels: vec!["a long label that cannot fit".into()],
    };
    assert!(preview_dimensions(ui, &tab).is_none());
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
    assert!(!previews.needs_frame_pixels());
    previews.note_frame([4, 2], &frame());
    assert!(previews.images.is_empty(), "nothing is in front yet");
    previews.showing = Some((PreviewKey::Window(7), [[0., 0.], [2., 2.]]));
    assert!(previews.needs_frame_pixels());
    previews.note_frame([4, 2], &frame());
    previews.showing = Some((PreviewKey::Home, [[2., 0.], [4., 2.]]));
    previews.note_frame([4, 2], &frame());
    assert_eq!(previews.images[&PreviewKey::Window(7)].rgb[0], [200, 0, 0]);
    assert_eq!(previews.images[&PreviewKey::Home].rgb[0], [0, 0, 200]);
}

#[test]
fn preview_readback_is_reused_only_for_the_same_composition_and_crop() {
    let mut previews = TabPreviews {
        showing: Some((PreviewKey::Home, [[0., 0.], [2., 2.]])),
        ..Default::default()
    };
    let generations = Some([1, 2, 3, 4]);
    assert!(previews.needs_frame_pixels_at(generations));
    previews.note_frame_at([4, 2], &frame()[..8], generations);
    assert!(previews.needs_frame_pixels_at(generations));
    previews.note_frame_at([4, 2], &frame(), generations);
    assert!(!previews.needs_frame_pixels_at(generations));
    assert!(previews.needs_frame_pixels_at(None));
    assert!(previews.needs_frame_pixels_at(Some([1, 3, 3, 4])));
    previews.showing = Some((PreviewKey::Home, [[2., 0.], [4., 2.]]));
    assert!(previews.needs_frame_pixels_at(generations));
    previews.note_frame_at([4, 2], &frame(), generations);
    assert_eq!(previews.images[&PreviewKey::Home].rgb[0], [0, 0, 200]);
    previews.showing = Some((PreviewKey::Window(7), [[2., 0.], [4., 2.]]));
    assert!(previews.needs_frame_pixels_at(generations));
}

#[test]
fn visible_preview_is_not_captured_inside_its_own_picture() {
    let mut previews = TabPreviews {
        showing: Some((PreviewKey::Window(7), [[0., 0.], [2., 2.]])),
        showing_document: Some(7),
        ..Default::default()
    };
    previews.note_frame([2, 2], &[9; 16]);
    let original = previews.images[&PreviewKey::Window(7)].clone();
    previews.preview_visible = true;
    assert!(!previews.needs_frame_pixels());
    previews.note_frame([2, 2], &[200; 16]);
    assert_eq!(previews.images[&PreviewKey::Window(7)], original);
    previews.synchronize(&crate::GuiCommandState {
        documents: vec![document(7, true)],
        ..Default::default()
    });
    previews.note_frame([2, 2], &[200; 16]);
    assert_ne!(previews.images[&PreviewKey::Window(7)], original);
}

#[test]
fn changed_front_gets_a_clean_frame_before_the_preview() {
    let mut previews = TabPreviews {
        showing: Some((PreviewKey::Window(7), [[0., 0.], [2., 2.]])),
        showing_document: Some(7),
        ..Default::default()
    };
    assert!(previews.needs_clean_front_capture(Some(PreviewKey::Home)));
    previews.showing = Some((PreviewKey::Home, [[0., 0.], [2., 2.]]));
    previews.showing_document = None;
    previews.note_frame([2, 2], &[9; 16]);
    assert!(previews.images.contains_key(&PreviewKey::Home));
    assert!(!previews.needs_clean_front_capture(Some(PreviewKey::Home)));
    previews.preview_visible = true;
    previews.note_frame([2, 2], &[200; 16]);
    assert_eq!(previews.images[&PreviewKey::Home].rgb[0], [9, 9, 9]);
}

#[test]
fn front_key_uses_home_or_the_active_document_window() {
    let mut state = crate::GuiCommandState {
        document: Some(7),
        ..Default::default()
    };
    assert_eq!(front_key(&state, true), Some(PreviewKey::Home));
    assert_eq!(front_key(&state, false), Some(PreviewKey::Window(7)));
    state.active_window = Some(90);
    assert_eq!(front_key(&state, false), Some(PreviewKey::Window(90)));
    state.document = None;
    assert_eq!(front_key(&state, false), None);
}

#[test]
fn deferred_preview_requests_another_frame() {
    let mut context = crate::test_context::create();
    context.set_ini_filename(None);
    context.io_mut().display_size = [800., 600.];
    context.fonts().build_rgba32_texture();
    let ui = context.frame();
    let tab = HoveredTab {
        key: PreviewKey::Home,
        bounds: [[50., 300.], [150., 325.]],
        labels: vec!["Home".into()],
    };
    let mut previews = TabPreviews {
        showing: Some((PreviewKey::Window(7), [[0., 0.], [2., 2.]])),
        ..Default::default()
    };
    previews.panel(ui, Some(&tab), Some(PreviewKey::Home));
    assert!(previews.needs_frame());
    assert!(!previews.preview_visible);
    previews.synchronize(&crate::GuiCommandState::default());
    previews.showing = Some((PreviewKey::Home, [[0., 0.], [2., 2.]]));
    previews.panel(ui, Some(&tab), Some(PreviewKey::Home));
    assert!(previews.needs_frame());
    assert!(!previews.preview_visible);
    previews.note_frame([2, 2], &[9; 16]);
    previews.synchronize(&crate::GuiCommandState::default());
    previews.panel(ui, Some(&tab), Some(PreviewKey::Home));
    assert!(!previews.needs_frame());
    assert!(previews.preview_visible);
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
    assert_eq!(previews.captured_front, None);
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
fn closing_original_window_discards_only_its_picture() {
    let mut previews = TabPreviews::default();
    let mut state = crate::GuiCommandState {
        documents: vec![document(1, true)],
        document_windows: vec![window(1, 1), window(90, 1)],
        ..Default::default()
    };
    previews.synchronize(&state);
    capture(&mut previews, 1, 90);
    capture(&mut previews, 1, 1);
    state.document_windows.retain(|item| item.id == 90);
    previews.synchronize(&state);
    assert!(!previews.images.contains_key(&PreviewKey::Window(1)));
    assert!(previews.images.contains_key(&PreviewKey::Window(90)));
    assert_eq!(previews.captured_front, None);
    assert!(previews.take_changed_picture().is_none());
    capture(&mut previews, 1, 90);
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
fn stable_survivor_detects_open_and_close_between_frames() {
    let mut previews = TabPreviews::default();
    let mut state = crate::GuiCommandState {
        documents: vec![document(1, true)],
        ..Default::default()
    };
    previews.synchronize(&state);
    capture(&mut previews, 1, 1);
    state.document_windows.push(window(90, 1));
    previews.synchronize(&state);
    assert!(!previews.images.contains_key(&PreviewKey::Window(1)));
    assert!(previews.take_changed_picture().is_none());
    capture(&mut previews, 1, 90);
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
