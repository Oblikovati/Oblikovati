// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use x11rb::{connection::Connection, protocol::xproto::*, rust_connection::RustConnection};
struct NativeDrawableFixture {
    window: NativeWindow,
    connection: RustConnection,
    _server: private_server::PrivateX11Server,
}
impl NativeDrawableFixture {
    fn new(extent: [u32; 2]) -> Self {
        let server = private_server::PrivateX11Server::open_compose_fixture().unwrap();
        let window = open_fixture_window(server.display(), extent);
        let (connection, _) = RustConnection::connect(Some(server.display())).unwrap();
        Self {
            window,
            connection,
            _server: server,
        }
    }
    fn paint(&self, extent: [u16; 2]) {
        let window = u32::try_from(self.window.window.native_handle()).unwrap();
        let graphics = self.connection.generate_id().unwrap();
        self.connection
            .create_gc(graphics, window, &CreateGCAux::new().foreground(0x1e5acd))
            .unwrap()
            .check()
            .unwrap();
        let rectangle = Rectangle {
            x: 0,
            y: 0,
            width: extent[0],
            height: extent[1],
        };
        self.connection
            .poly_fill_rectangle(window, graphics, &[rectangle])
            .unwrap()
            .check()
            .unwrap();
        self.connection.free_gc(graphics).unwrap().check().unwrap();
    }
}
fn open_fixture_window(name: &str, extent: [u32; 2]) -> NativeWindow {
    let x11 = NativeX11::open().unwrap();
    let display = Handle::new(
        AspectDisplayConnection::with_display_name(
            &rustycad_tkernel::tcollection::AsciiString::from(name),
        )
        .unwrap(),
    );
    NativeWindow::create_with_display(x11, display, "Oblikovati checked capture", extent).unwrap()
}
#[test]
fn clipped_native_capture_is_a_named_decline_and_the_window_remains_readable_after_resize() {
    let fixture = NativeDrawableFixture::new([640, 480]);
    let issue = fixture.window.pixels().unwrap_err();
    assert!(
        issue.contains("native capture") && issue.contains("Match"),
        "{issue}"
    );
    fixture.window.resize([32, 24]).unwrap();
    fixture.paint([32, 24]);
    let rgba = fixture.window.pixels().unwrap();
    assert_eq!(rgba.len(), 32 * 24 * 4);
    assert!(
        rgba.as_chunks::<4>()
            .0
            .iter()
            .all(|pixel| pixel == &[0x1e, 0x5a, 0xcd, 255])
    );
}
#[test]
fn retired_and_unrepresentable_resources_do_not_retire_the_checked_reader() {
    let fixture = NativeDrawableFixture::new([32, 24]);
    let reader =
        pixels::NativePixelReader::open_checked_capture(fixture._server.display()).unwrap();
    assert!(
        reader
            .read_drawable_rgba(u64::MAX)
            .unwrap_err()
            .contains("32-bit")
    );
    let retired = fixture.connection.generate_id().unwrap();
    create_retired_probe(&fixture.connection, retired);
    let issue = reader.read_drawable_rgba(u64::from(retired)).unwrap_err();
    assert!(
        issue.contains("native capture") && issue.contains("Window"),
        "{issue}"
    );
    fixture.paint([32, 24]);
    assert_eq!(
        reader
            .read_drawable_rgba(fixture.window.window.native_handle())
            .unwrap()
            .len(),
        32 * 24 * 4
    );
}
fn create_retired_probe(connection: &RustConnection, window: Window) {
    let root = connection.setup().roots[0].root;
    connection
        .create_window(
            0,
            window,
            root,
            0,
            0,
            1,
            1,
            0,
            WindowClass::INPUT_ONLY,
            0,
            &CreateWindowAux::new(),
        )
        .unwrap()
        .check()
        .unwrap();
    connection.destroy_window(window).unwrap().check().unwrap();
}
#[test]
fn an_unmapped_native_capture_is_declined_and_mapping_restores_the_same_reader() {
    let fixture = NativeDrawableFixture::new([32, 24]);
    fixture.window.pixels().unwrap();
    fixture.window.mapped(false).unwrap();
    assert!(
        fixture
            .window
            .pixels()
            .unwrap_err()
            .contains("mapped viewport")
    );
    fixture.window.mapped(true).unwrap();
    fixture.paint([32, 24]);
    assert_eq!(fixture.window.pixels().unwrap().len(), 32 * 24 * 4);
}
