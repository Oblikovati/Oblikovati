// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::cell::RefCell;
use std::collections::VecDeque;
struct FakeTextReply {
    bytes: Vec<u8>,
    count: i32,
    status: i32,
}
struct FakeCommitLookup {
    replies: RefCell<VecDeque<FakeTextReply>>,
    capacities: RefCell<Vec<usize>>,
}
impl FakeCommitLookup {
    fn new(replies: Vec<FakeTextReply>) -> Self {
        Self {
            replies: RefCell::new(replies.into()),
            capacities: RefCell::new(Vec::new()),
        }
    }
}
impl CommittedTextLookup for FakeCommitLookup {
    fn lookup(&self, _: &mut xlib::XKeyEvent, buffer: &mut [u8]) -> Utf8LookupReply {
        self.capacities.borrow_mut().push(buffer.len());
        let reply = self
            .replies
            .borrow_mut()
            .pop_front()
            .expect("expected one named fake lookup reply");
        let copied = reply.bytes.len().min(buffer.len());
        buffer[..copied].copy_from_slice(&reply.bytes[..copied]);
        Utf8LookupReply {
            count: reply.count,
            status: reply.status,
        }
    }
}
fn committed_reply(text: &str) -> FakeTextReply {
    FakeTextReply {
        bytes: text.as_bytes().to_vec(),
        count: text.len() as i32,
        status: xlib::XLookupBoth,
    }
}
fn key_event() -> xlib::XKeyEvent {
    unsafe { std::mem::zeroed() }
}
#[test]
fn committed_utf8_preserves_composed_and_supplementary_characters() {
    let lookup = FakeCommitLookup::new(vec![committed_reply("é界🙂e\u{301}")]);
    assert_eq!(
        lookup_committed_text(&lookup, &mut key_event()).unwrap(),
        "é界🙂e\u{301}"
    );
    assert_eq!(*lookup.capacities.borrow(), [64]);
}
#[test]
fn overflow_uses_the_reported_capacity_and_never_truncates_a_commit() {
    let text = "🙂".repeat(40);
    let overflow = FakeTextReply {
        bytes: vec![],
        count: text.len() as i32,
        status: xlib::XBufferOverflow,
    };
    let lookup = FakeCommitLookup::new(vec![overflow, committed_reply(&text)]);
    assert_eq!(
        lookup_committed_text(&lookup, &mut key_event()).unwrap(),
        text
    );
    assert_eq!(*lookup.capacities.borrow(), [64, 160]);
}
#[test]
fn malformed_lookup_status_lengths_and_utf8_are_named_failures() {
    const MALFORMED_LOOKUPS: &[(i32, i32, &[u8])] = &[
        (-1, xlib::XBufferOverflow, &[]),
        (65, xlib::XLookupChars, &[]),
        (1, xlib::XLookupChars, &[255]),
        (0, 999, &[]),
    ];
    for &(count, status, bytes) in MALFORMED_LOOKUPS {
        let reply = FakeTextReply {
            bytes: bytes.to_vec(),
            count,
            status,
        };
        let lookup = FakeCommitLookup::new(vec![reply]);
        assert!(
            lookup_committed_text(&lookup, &mut key_event())
                .unwrap_err()
                .contains("expected")
        );
    }
}
#[test]
fn keysym_only_and_control_characters_are_not_committed_text() {
    let lookup = FakeCommitLookup::new(vec![FakeTextReply {
        bytes: vec![],
        count: 0,
        status: xlib::XLookupKeySym,
    }]);
    assert_eq!(
        lookup_committed_text(&lookup, &mut key_event()).unwrap(),
        ""
    );
    let lookup = FakeCommitLookup::new(vec![committed_reply("\r\t\u{0}é")]);
    assert_eq!(
        lookup_committed_text(&lookup, &mut key_event()).unwrap(),
        "é"
    );
}
#[test]
fn root_style_admission_rejects_empty_and_foreign_style_sets() {
    let mut offered = [0x408 as std::ffi::c_ulong];
    let styles = NativeInputStyles {
        count_styles: 1,
        supported_styles: offered.as_mut_ptr(),
    };
    assert!(unsafe { supported_style(&styles, 0x408) }.is_ok());
    assert!(
        unsafe { supported_style(&styles, 0x402) }
            .unwrap_err()
            .contains("1026")
    );
    let styles = NativeInputStyles {
        count_styles: 0,
        supported_styles: std::ptr::null_mut(),
    };
    assert!(unsafe { supported_style(&styles, 0x408) }.is_err());
}
#[test]
fn provider_retirement_is_idempotent_and_invalidates_its_owned_liveness_flag() {
    let alive = AtomicBool::new(true);
    let owner = (&alive as *const AtomicBool).cast_mut().cast();
    unsafe { input_method_retired(std::ptr::null_mut(), owner, std::ptr::null_mut()) };
    assert!(!alive.load(Ordering::Acquire));
    unsafe { input_method_retired(std::ptr::null_mut(), owner, std::ptr::null_mut()) };
    unsafe {
        input_method_retired(
            std::ptr::null_mut(),
            std::ptr::null_mut(),
            std::ptr::null_mut(),
        )
    };
    assert!(!alive.load(Ordering::Acquire));
}
#[test]
fn retired_or_missing_context_is_refused_before_focus_and_commit_calls() {
    let window =
        NativeWindow::create(NativeX11::open().unwrap(), "XIM admission", [64, 48]).unwrap();
    let mut input = NativeInputMethod::open(
        window.x11.clone(),
        window.display.clone(),
        window.window.native_handle(),
    )
    .unwrap();
    input.alive.store(false, Ordering::Release);
    let focus_error = input.focus(true).unwrap_err();
    let commit_error = input.committed_text(&mut key_event()).unwrap_err();
    input.alive.store(true, Ordering::Release);
    assert!(focus_error.contains("alive=false"));
    assert!(commit_error.contains("alive=false"));
    let context = input.context.take();
    let missing = input.require_live_context().unwrap_err();
    input.context = context;
    assert!(missing.contains("expected a live owned"));
}
#[test]
fn a_second_overflow_is_refused_instead_of_retrying_or_truncating() {
    let reply = || FakeTextReply {
        bytes: vec![],
        count: 128,
        status: xlib::XBufferOverflow,
    };
    let lookup = FakeCommitLookup::new(vec![reply(), reply()]);
    assert!(
        lookup_committed_text(&lookup, &mut key_event())
            .unwrap_err()
            .contains("expected a completed")
    );
    assert_eq!(*lookup.capacities.borrow(), [64, 128]);
}
#[test]
fn no_character_status_requires_zero_bytes() {
    for status in [xlib::XLookupNone, xlib::XLookupKeySym] {
        let lookup = FakeCommitLookup::new(vec![FakeTextReply {
            bytes: vec![],
            count: -1,
            status,
        }]);
        assert!(
            lookup_committed_text(&lookup, &mut key_event())
                .unwrap_err()
                .contains("expected zero")
        );
    }
}
#[test]
fn commit_lookup_admission_requires_a_press_from_the_owned_display() {
    let mut event = key_event();
    let owner = std::ptr::NonNull::<xlib::Display>::dangling().as_ptr();
    assert!(require_owned_commit(&event, owner).is_err());
    event.type_ = xlib::KeyPress;
    assert!(require_owned_commit(&event, owner).is_err());
    event.display = owner;
    assert!(require_owned_commit(&event, owner).is_ok());
    event.type_ = xlib::KeyRelease;
    assert!(require_owned_commit(&event, owner).is_err());
}
