// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustycad_tkservice::aspect::AspectWindow;
struct FakeUnavailableTextProvider;
impl FakeUnavailableTextProvider {
    fn admission(&self) -> Result<NativeInputMethod, String> {
        Err("XIM provider fake-unavailable: expected an available configured provider".into())
    }
}
#[test]
fn declined_provider_keeps_its_named_issue_without_native_focus_or_lookup() {
    let channel = NativeTextChannel::admit(FakeUnavailableTextProvider.admission());
    let issue = channel.issue().unwrap();
    assert!(issue.contains("fake-unavailable"));
    channel.focus(true);
    let mut event = xlib::XKeyEvent {
        type_: xlib::KeyPress,
        ..unsafe { std::mem::zeroed() }
    };
    assert_eq!(channel.committed_text(&mut event), "");
    channel.focus(false);
    assert_eq!(channel.issue().as_deref(), Some(issue.as_str()));
}
struct NativeTextChannelFixture {
    channel: NativeTextChannel,
    window: crate::x11::NativeWindow,
}
impl NativeTextChannelFixture {
    fn new() -> Self {
        let window = crate::x11::NativeWindow::create(
            crate::x11::NativeX11::open().unwrap(),
            "Text channel health",
            [64, 48],
        )
        .unwrap();
        let method = NativeInputMethod::open(
            window.x11.clone(),
            window.display.clone(),
            window.window.native_handle(),
        )
        .unwrap();
        Self {
            channel: NativeTextChannel::admit(Ok(method)),
            window,
        }
    }
}
#[test]
fn invalid_commit_quarantines_only_text_and_preserves_the_first_named_issue() {
    let fixture = NativeTextChannelFixture::new();
    assert_eq!(fixture.channel.issue(), None);
    fixture.channel.focus(true);
    let mut event = xlib::XKeyEvent {
        type_: xlib::KeyRelease,
        display: fixture.window.pointer(),
        ..unsafe { std::mem::zeroed() }
    };
    assert_eq!(fixture.channel.committed_text(&mut event), "");
    let issue = fixture.channel.issue().unwrap();
    assert!(issue.contains("expected KeyPress"));
    event.type_ = xlib::KeyPress;
    assert_eq!(fixture.channel.committed_text(&mut event), "");
    fixture.channel.focus(true);
    assert_eq!(fixture.channel.issue().as_deref(), Some(issue.as_str()));
    assert!(fixture.window.poll(64).is_ok());
}
