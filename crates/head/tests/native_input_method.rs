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
    let notice = window.gui.as_ref().unwrap().notice.as_deref().unwrap();
    assert!(notice.contains("Text input unavailable:"));
    let offending_value = std::env::var("OBLIKOVATI_TEXT_PROVIDER_PROBE").unwrap();
    assert!(notice.contains(&offending_value));
    verify_provider_issue_survives_command_feedback(&mut window, &offending_value);
    window.finish().unwrap();
}
fn verify_provider_issue_survives_command_feedback(
    window: &mut NativeHeadWindow,
    offending_value: &str,
) {
    let command = crate::DesktopCommandNotice::NoActiveDocument;
    let message = command.message();
    window.show_command_notice(command).unwrap();
    window.gui_frame(None, &Default::default()).unwrap();
    let notice = window.gui.as_ref().unwrap().notice.as_deref().unwrap();
    assert!(notice.contains(&message));
    assert!(notice.contains(offending_value));
    assert!(
        window
            .view
            .text_input_issue()
            .unwrap()
            .unwrap()
            .contains(offending_value)
    );
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
