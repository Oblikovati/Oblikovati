// SPDX-License-Identifier: GPL-2.0-only
//! Standalone worker for development/containment tests; packaged CLI uses its private entry.
fn main() -> std::process::ExitCode {
    #[cfg(target_os = "linux")]
    let _ = rustix::process::setrlimit(
        rustix::process::Resource::Core,
        rustix::process::Rlimit {
            current: Some(0),
            maximum: Some(0),
        },
    );
    if std::env::args().nth(1).as_deref() == Some("--abort-fixture") {
        std::process::abort();
    }
    match oblikovati_script::worker_stdio() {
        Ok(()) => std::process::ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("{error}");
            std::process::ExitCode::FAILURE
        }
    }
}
