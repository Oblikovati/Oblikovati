// SPDX-License-Identifier: GPL-2.0-only
use std::process::Command;
/// Run a singleton callback case in its own process; example: `case("OBK_TEST_CASE", test)`.
pub fn case(variable: &str, test: &str) -> bool {
    if std::env::var(variable).as_deref() == Ok(test) {
        return false;
    }
    let output = Command::new(std::env::current_exe().unwrap())
        .args(["--exact", test, "--nocapture"])
        .env(variable, test)
        .output()
        .unwrap();
    assert!(
        output.status.success(),
        "isolated case {test}: {} {}",
        String::from_utf8_lossy(&output.stdout),
        String::from_utf8_lossy(&output.stderr)
    );
    true
}
