// SPDX-License-Identifier: GPL-2.0-only
use super::*;
struct NativeEventFixture {
    window: NativeWindow,
}
impl NativeEventFixture {
    fn new() -> Self {
        Self {
            window: NativeWindow::create(
                NativeX11::open().unwrap(),
                "Oblikovati event fixture",
                [64, 48],
            )
            .unwrap(),
        }
    }
    fn event(&self, kind: i32) -> xlib::XEvent {
        let mut event = xlib::XEvent { pad: [0; 24] };
        event.any.type_ = kind;
        event.any.window = self.window.window.native_handle();
        event
    }
    fn close(&self) -> xlib::XEvent {
        let mut event = self.event(xlib::ClientMessage);
        event.client_message.format = 32;
        event.client_message.message_type = unsafe {
            (self.window.x11.api.XInternAtom)(self.window.pointer(), c"WM_PROTOCOLS".as_ptr(), 0)
        };
        let atom = self
            .window
            .display
            .get_atom(AspectXAtom::DeleteWindow)
            .unwrap();
        unsafe {
            event.client_message.data.set_long(0, atom as i64);
        }
        event
    }
}
#[test]
fn native_event_translation_checks_target_and_window_protocol_shape() {
    let fixture = NativeEventFixture::new();
    for (kind, expected) in [
        (xlib::Expose, ViewportEvent::Expose),
        (xlib::FocusIn, ViewportEvent::Focus(true)),
        (xlib::FocusOut, ViewportEvent::Focus(false)),
        (xlib::MapNotify, ViewportEvent::Mapped(true)),
        (xlib::UnmapNotify, ViewportEvent::Mapped(false)),
    ] {
        assert_eq!(
            fixture.window.translate(fixture.event(kind)),
            Some(expected)
        );
    }
    verify_configure_event(&fixture);
    verify_target_and_close(&fixture);
}
fn verify_configure_event(fixture: &NativeEventFixture) {
    let resize = fixture
        .window
        .translate(fixture.event(xlib::ConfigureNotify));
    let (width, height) = fixture.window.window.size();
    assert_eq!(
        resize,
        Some(ViewportEvent::Resized([width as u32, height as u32]))
    );
}
fn verify_target_and_close(fixture: &NativeEventFixture) {
    assert_eq!(
        fixture.window.translate(fixture.close()),
        Some(ViewportEvent::CloseRequested)
    );
    assert!(
        fixture
            .window
            .translate(fixture.event(xlib::KeyPress))
            .is_none()
    );
    let mut other = fixture.event(xlib::Expose);
    other.any.window = 0;
    assert!(fixture.window.translate(other).is_none());
}
#[test]
fn unrelated_client_messages_are_not_close_requests() {
    let fixture = NativeEventFixture::new();
    let mut event = fixture.close();
    event.client_message.format = 8;
    assert!(fixture.window.translate(event).is_none());
    event = fixture.close();
    event.client_message.message_type = 0;
    assert!(fixture.window.translate(event).is_none());
    event = fixture.close();
    unsafe {
        event.client_message.data.set_long(0, 0);
    }
    assert!(fixture.window.translate(event).is_none());
}
#[test]
fn native_title_validation_and_rgb_masks_have_named_bounds() {
    assert!(validate_title("").unwrap_err().contains("nonempty"));
    assert!(validate_title("wrong\0title").is_err());
    assert!(validate_title("Oblikovati").is_ok());
    assert_eq!(color_channel(0x80_ff_00, 0xff_0000), 128);
    assert_eq!(color_channel(0x80_ff_00, 0x00_ff00), 255);
    assert_eq!(color_channel(0x80_ff_00, 0x00_00ff), 0);
    assert_eq!(color_channel(0, 0), 0);
}
#[test]
fn unsupported_native_color_layout_is_declined_instead_of_converted_to_black() {
    assert!(validate_masks([0xff_0000, 0xff00, 0xff]).is_ok());
    for masks in [[0, 0, 0], [0b101, 0b1000, 0b10000], [0xff, 0xff, 0xff00]] {
        assert!(validate_masks(masks).unwrap_err().contains("expected"));
    }
    assert_eq!(color_channel(u64::MAX, u64::MAX), 255);
}
#[test]
fn native_capture_buffer_rejects_invalid_sizes_before_allocating() {
    assert_eq!(rgba_buffer(4, 3).unwrap().capacity(), 48);
    assert!(rgba_buffer(0, 3).is_err());
    assert!(rgba_buffer(-1, 3).is_err());
    assert!(rgba_buffer(i32::MAX, i32::MAX).is_err());
}
#[test]
fn native_wait_preserves_pending_events_and_consumes_worker_wakes() {
    let fixture = NativeEventFixture::new();
    let wake = crate::ViewportWake::new().unwrap();
    let ready = fixture
        .window
        .wait_events(&wake, Some(Duration::ZERO))
        .unwrap();
    assert!(ready.native_events);
    assert!(!fixture.window.poll(64).is_empty());
    let worker = wake.clone();
    std::thread::spawn(move || worker.wake().unwrap())
        .join()
        .unwrap();
    let ready = fixture
        .window
        .wait_events(&wake, Some(Duration::ZERO))
        .unwrap();
    assert!(ready.worker_wake);
}

#[test]
fn concurrent_native_runtimes_share_one_initialized_xlib_library() {
    let initial = NativeX11::open().unwrap().api.clone();
    let threads: Vec<_> = (0..8)
        .map(|_| std::thread::spawn(|| NativeX11::open().unwrap().api.clone()))
        .collect();
    for thread in threads {
        assert!(Arc::ptr_eq(&initial, &thread.join().unwrap()));
    }
    assert!(
        require_threaded_xlib(0)
            .unwrap_err()
            .contains("XInitThreads returned 0")
    );
    require_threaded_xlib(1).unwrap();
}
