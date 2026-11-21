// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn scenario_images_are_isolated_and_never_overwritten() {
    let root = tempfile::tempdir().unwrap();
    let capture = oblikovati_viewport::ViewportCapture {
        view_id: 1,
        scene_generation: 1,
        camera_generation: 1,
        surface_generation: 1,
        overlay_generation: 1,
        extent: [1, 1],
        rgba: vec![0, 0, 0, 255],
    };
    let first = capture_folder(root.path(), Some("first"));
    let second = capture_folder(root.path(), Some("second"));
    std::fs::create_dir_all(&first).unwrap();
    std::fs::create_dir_all(&second).unwrap();
    write_png(&first.join("view.png"), &capture).unwrap();
    write_png(&second.join("view.png"), &capture).unwrap();
    assert!(write_png(&first.join("view.png"), &capture).is_err());
    assert_eq!(capture_folder(root.path(), None), root.path());
}
