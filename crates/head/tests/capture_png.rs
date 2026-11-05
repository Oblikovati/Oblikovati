// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn striped_capture() -> ViewportCapture {
    ViewportCapture {
        view_id: 1,
        scene_generation: 2,
        camera_generation: 3,
        surface_generation: 4,
        overlay_generation: 5,
        extent: [1, 2],
        rgba: vec![255, 0, 0, 128, 0, 255, 0, 255],
    }
}
fn decoded_rgba(bytes: &[u8]) -> Vec<u8> {
    let decoder = png::Decoder::new(std::io::Cursor::new(bytes));
    let mut reader = decoder.read_info().unwrap();
    assert_eq!(
        reader.info().srgb,
        Some(png::SrgbRenderingIntent::Perceptual)
    );
    let mut output = vec![0; reader.output_buffer_size()];
    let info = reader.next_frame(&mut output).unwrap();
    assert_eq!(
        (info.width, info.height, info.color_type, info.bit_depth),
        (1, 2, png::ColorType::Rgba, png::BitDepth::Eight)
    );
    output[..info.buffer_size()].to_vec()
}
#[test]
fn encoded_png_preserves_top_down_rgba_pixels_alpha_and_srgb_tag() {
    let capture = striped_capture();
    assert_eq!(
        decoded_rgba(&encode_window_png(&capture).unwrap()),
        capture.rgba
    );
}
#[test]
fn malformed_completed_frames_are_refused_before_an_existing_file_changes() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("unchanged.png");
    std::fs::write(&path, b"original").unwrap();
    for extent in [[0, 2], [2, 0], [2, 2], [u32::MAX, u32::MAX]] {
        let mut capture = striped_capture();
        capture.extent = extent;
        assert!(
            NativeWindowCaptureFiles
                .write_capture(&path, &capture)
                .unwrap_err()
                .contains("width * height * 4")
        );
        assert_eq!(std::fs::read(&path).unwrap(), b"original");
    }
}
#[test]
fn native_publication_atomically_replaces_the_destination_and_names_io_failures() {
    let directory = tempfile::tempdir().unwrap();
    let path = directory.path().join("frame.png");
    std::fs::write(&path, b"old").unwrap();
    NativeWindowCaptureFiles
        .write_capture(&path, &striped_capture())
        .unwrap();
    assert_eq!(
        decoded_rgba(&std::fs::read(&path).unwrap()),
        striped_capture().rgba
    );
    assert_eq!(std::fs::read_dir(directory.path()).unwrap().count(), 1);
    let missing = directory.path().join("missing/frame.png");
    let cause = NativeWindowCaptureFiles
        .write_capture(&missing, &striped_capture())
        .unwrap_err();
    assert!(cause.contains("missing") && cause.contains("atomic PNG publication"));
}
#[test]
fn encoding_failure_explains_the_expected_image_shape() {
    let message = png_encoding_error(png::EncodingError::IoError(std::io::Error::other(
        "refused",
    )));
    assert!(message.contains("refused") && message.contains("sRGB RGBA image"));
}
