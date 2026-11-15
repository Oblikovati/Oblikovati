// SPDX-License-Identifier: GPL-2.0-only
use crate::GuiFrameEngine;
use std::sync::mpsc;
use std::time::Duration;

#[test]
fn separate_test_threads_can_create_gui_owners_without_shared_context_failures() {
    let (ready, started) = mpsc::channel();
    let (release, wait) = mpsc::channel();
    let first = std::thread::spawn(move || {
        let engine = GuiFrameEngine::new().unwrap();
        ready.send(()).unwrap();
        wait.recv().unwrap();
        drop(engine);
    });
    started.recv().unwrap();
    let (done, result) = mpsc::channel();
    let second = std::thread::spawn(move || {
        done.send(GuiFrameEngine::new().is_ok()).unwrap();
    });
    assert!(result.recv_timeout(Duration::from_millis(100)).is_err());
    release.send(()).unwrap();
    first.join().unwrap();
    assert!(result.recv().unwrap());
    second.join().unwrap();
}

#[test]
fn nested_owner_refusal_keeps_the_original_context_and_releases_the_lease() {
    let context = crate::test_context::create();
    assert!(matches!(GuiFrameEngine::new(), Err(error) if error.contains("GUI context occupied")));
    drop(context);
    assert!(GuiFrameEngine::new().is_ok());
}

#[test]
fn a_panicking_owner_releases_the_test_context_lease() {
    let result = std::thread::spawn(|| {
        let _context = crate::test_context::create();
        panic!("exercise test context cleanup");
    })
    .join();
    assert!(result.is_err());
    let context = crate::test_context::create();
    drop(context);
}
