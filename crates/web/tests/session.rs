// SPDX-License-Identifier: GPL-2.0-only
//! End-to-end native WebGPU tests for `oblikovati-web`'s `WebSession`.
#![cfg(any(target_os = "linux", target_os = "windows"))]

use oblikovati_viewport::{
    ViewportOptions, ViewportPresentation, WebKeyEvent, WebModifiers, WebPointerEvent,
    WebPointerKind, WebResizeEvent, WebWheelEvent,
};
use oblikovati_web::{WebCommand, WebSession, WebTheme, WebViewOrientation};

#[test]
fn web_session_evaluates_edits_undoes_exports_and_renders_on_webgpu() {
    let mut session = WebSession::new_offscreen(
        ViewportOptions {
            validation: true,
            physical_device: None,
        },
        [256, 192],
    )
    .unwrap();

    let sample_id = session.load_sample_part().unwrap();
    let initial = session.snapshot();
    assert_eq!(initial.active_document, Some(sample_id));
    assert_eq!(initial.visible_bodies, 1);
    assert!(initial.triangle_count >= 12);
    assert_eq!(initial.parameters.len(), 3);
    assert_eq!(initial.features.len(), 1);
    assert!(!initial.can_undo);
    assert!(session.render_frame().unwrap());

    // Edit parameter -> rebuild solid -> undo -> redo
    let edited = session
        .execute_command(WebCommand::SetParameter {
            name: "depth".into(),
            expression: "5 cm".into(),
        })
        .unwrap();
    assert!(edited.can_undo);
    assert_eq!(
        edited
            .parameters
            .iter()
            .find(|p| p.name == "depth")
            .unwrap()
            .expression,
        "5 cm"
    );

    let undone = session.execute_command(WebCommand::Undo).unwrap();
    assert!(undone.can_redo);
    assert_eq!(
        undone
            .parameters
            .iter()
            .find(|p| p.name == "depth")
            .unwrap()
            .expression,
        "2.5 cm"
    );

    let redone = session.execute_command(WebCommand::Redo).unwrap();
    assert!(redone.can_undo);
    assert_eq!(
        redone
            .parameters
            .iter()
            .find(|p| p.name == "depth")
            .unwrap()
            .expression,
        "5 cm"
    );

    // Export .opd bytes and re-open into a second document
    let (filename, bytes) = session.export_active_document_bytes().unwrap();
    assert!(filename.ends_with(".opd"));
    assert!(!bytes.is_empty());
    let reopened = session.open_document_bytes("reopened.opd", &bytes).unwrap();
    assert_ne!(reopened, sample_id);
    assert_eq!(session.snapshot().visible_bodies, 1);

    // Camera, theme, projection, normal debug, JSON command dispatch, and input handlers
    let json_state = session
        .execute_command_json(r#"{"type":"setTheme","theme":"light"}"#)
        .unwrap();
    assert!(json_state.contains("\"theme\":\"light\""));
    session
        .execute_command(WebCommand::SetProjection { perspective: true })
        .unwrap();
    session
        .execute_command(WebCommand::SetNormalDebug { enabled: true })
        .unwrap();
    session
        .execute_command(WebCommand::Orient {
            view: WebViewOrientation::Front,
        })
        .unwrap();
    session.execute_command(WebCommand::Fit).unwrap();
    session.execute_command(WebCommand::Home).unwrap();
    session
        .execute_command(WebCommand::Orbit {
            yaw: 0.2,
            pitch: 0.1,
        })
        .unwrap();
    session
        .execute_command(WebCommand::Pan { dx: 10, dy: -5 })
        .unwrap();
    session
        .execute_command(WebCommand::Zoom {
            factor: 1.1,
            anchor: Some([128, 96]),
        })
        .unwrap();
    session
        .execute_command(WebCommand::SetTheme {
            theme: WebTheme::Dark,
        })
        .unwrap();
    session
        .execute_command(WebCommand::SetNormalDebug { enabled: false })
        .unwrap();

    assert!(
        !session
            .handle_pointer(WebPointerEvent {
                kind: WebPointerKind::Down,
                css_x: 40.0,
                css_y: 40.0,
                button: 0,
                modifiers: WebModifiers::default(),
            })
            .unwrap()
    );
    assert!(
        session
            .handle_pointer(WebPointerEvent {
                kind: WebPointerKind::Move,
                css_x: 60.0,
                css_y: 50.0,
                button: 0,
                modifiers: WebModifiers::default(),
            })
            .unwrap()
    );
    assert!(
        session
            .handle_wheel(WebWheelEvent {
                css_x: 128.0,
                css_y: 96.0,
                delta_y: -120.0,
                delta_mode: 0,
            })
            .unwrap()
    );
    assert!(
        session
            .handle_key(&WebKeyEvent {
                code: "Home".into(),
                key: "Home".into(),
                pressed: true,
                modifiers: WebModifiers::default(),
            })
            .unwrap()
    );
    let resized = session
        .resize(WebResizeEvent {
            css_width: 320.0,
            css_height: 240.0,
            device_pixel_ratio: 1.0,
        })
        .unwrap();
    assert_eq!(resized, [320, 240]);

    // Document creation, activation, close, and snapshot JSON
    let blank = session.new_part(Some("BlankPart".into())).unwrap();
    session
        .execute_command(WebCommand::ActivateDocument { id: sample_id })
        .unwrap();
    session
        .execute_command(WebCommand::CloseDocument { id: blank })
        .unwrap();
    assert!(session.snapshot_json().unwrap().contains("SamplePart"));

    // Verify rendered pixels and zero WebGPU validation errors
    assert!(session.render_frame().unwrap());
    let cap = futures_block_on(session.capture_async()).unwrap();
    assert_eq!(cap.extent, [320, 240]);
    let presented = futures_block_on(session.present_async()).unwrap();
    let ViewportPresentation::Completed(frame) = presented else {
        panic!("expected completed WebGPU frame");
    };
    let non_bg = frame
        .rgba
        .as_chunks::<4>()
        .0
        .iter()
        .filter(|px| px[0] > 60 && px[1] > 60 && px[2] > 60)
        .count();
    assert!(
        non_bg > 200,
        "expected rendered parametric solid pixels, got {non_bg}"
    );
    assert!(session.validation_errors().unwrap().is_empty());
    session.close();
}

fn futures_block_on<F: std::future::Future>(future: F) -> F::Output {
    use std::{
        pin::pin,
        task::{Context, Poll, Waker},
    };
    let mut future = pin!(future);
    let waker = Waker::noop();
    let mut context = Context::from_waker(waker);
    loop {
        if let Poll::Ready(value) = future.as_mut().poll(&mut context) {
            return value;
        }
        std::thread::yield_now();
    }
}
