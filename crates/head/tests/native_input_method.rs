// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::HeadWindow;
use std::process::{Command, Output};
const MISSING_TEXT_PROVIDER: &str = "@im=oblikovati_missing_provider_fixture";
struct MissingTextProviderProcess {
    variable: &'static str,
    value: &'static str,
}
impl MissingTextProviderProcess {
    fn run(&self) -> Output {
        Command::new(std::env::current_exe().unwrap())
            .args(["--exact", "native::gui_loop::provider_tests::missing_text_provider_keeps_native_window_and_gui_available", "--nocapture"])
            .env("OBLIKOVATI_TEXT_PROVIDER_PROBE", self.value)
            .env(self.variable, self.value)
            .output()
            .unwrap()
    }
}
#[test]
fn missing_text_provider_keeps_native_window_and_gui_available() {
    if std::env::var_os("OBLIKOVATI_TEXT_PROVIDER_PROBE").is_some() {
        verify_missing_provider_window();
        return;
    }
    for (variable, value) in [
        ("XMODIFIERS", MISSING_TEXT_PROVIDER),
        ("LC_ALL", "oblikovati_missing_locale_fixture"),
    ] {
        let receipt = MissingTextProviderProcess { variable, value }.run();
        require_native_probe_success(receipt, variable, value);
    }
}
fn verify_missing_provider_window() {
    let mut window = NativeHeadWindow::open().unwrap();
    window.gui_frame(None, &Default::default()).unwrap();
    window.present().unwrap();
    assert!(window.view.capture().is_ok());
    let offending_value = std::env::var("OBLIKOVATI_TEXT_PROVIDER_PROBE").unwrap();
    let expected = (!cfg!(target_os = "windows")).then_some(offending_value.as_str());
    verify_text_provider_state(&window, expected);
    show_command_feedback(&mut window);
    verify_text_provider_state(&window, expected);
    window.finish().unwrap();
}

fn verify_text_provider_state(window: &NativeHeadWindow, expected: Option<&str>) {
    let issue = window.view.text_input_issue().unwrap();
    let notice = window.gui.as_ref().unwrap().notice.as_deref().unwrap_or("");
    if let Some(offending_value) = expected {
        assert!(issue.unwrap().contains(offending_value));
        assert!(notice.contains("Text input unavailable:"));
        assert!(notice.contains(offending_value));
    } else {
        assert_eq!(issue, None);
        assert!(!notice.contains("Text input unavailable:"));
    }
}

fn show_command_feedback(window: &mut NativeHeadWindow) {
    let command = crate::DesktopCommandNotice::NoActiveDocument;
    let message = command.message();
    window.show_command_notice(command).unwrap();
    window.gui_frame(None, &Default::default()).unwrap();
    let notice = window.gui.as_ref().unwrap().notice.as_deref().unwrap();
    assert!(notice.contains(&message));
}

fn require_native_probe_success(receipt: Output, variable: &str, value: &str) {
    assert!(
        receipt.status.success(),
        "input configuration {variable}={value}, child {:?}: expected healthy native GUI: {}{}",
        receipt.status,
        String::from_utf8_lossy(&receipt.stdout),
        String::from_utf8_lossy(&receipt.stderr)
    );
}
