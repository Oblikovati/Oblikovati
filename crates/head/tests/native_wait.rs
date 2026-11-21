// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::run_window;

#[test]
fn native_window_extent_requires_two_positive_dimensions() {
    assert_eq!(validate_window_extent([800, 600]), Ok(()));
    assert_eq!(
        validate_window_extent([0, 600]),
        Err("native window extent [0, 600]: expected two positive pixel dimensions".into())
    );
}

#[test]
fn native_owner_wait_receives_model_free_worker_wakes_and_visibility_inputs() {
    let mut window = NativeHeadWindow::open().unwrap();
    window.events(64).unwrap();
    let wake = window.wake.clone();
    std::thread::spawn(move || wake.wake().unwrap())
        .join()
        .unwrap();
    assert!(window.wait(None).unwrap());
    verify_visibility_and_retirement(&mut window);
}
fn verify_visibility_and_retirement(window: &mut NativeHeadWindow) {
    window.view.set_mapped(false).unwrap();
    let hidden = window.events(64).unwrap();
    assert!(hidden.contains(&ViewportEvent::Mapped(false)));
    window.view.set_mapped(true).unwrap();
    let mapped = window.events(64).unwrap();
    assert!(mapped.contains(&ViewportEvent::Mapped(true)));
    assert!(
        run_window(window, HeadOptions { frames: Some(3) })
            .unwrap()
            .completed
            > 0
    );
    assert!(window.finish().is_ok());
}
