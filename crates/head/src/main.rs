// SPDX-License-Identifier: GPL-2.0-only
use std::{io, process::ExitCode};
fn main() -> ExitCode {
    let arguments: Vec<_> = std::env::args_os().skip(1).collect();
    let outcome = if arguments == [std::ffi::OsString::from(oblikovati_script::WORKER_ARGUMENT)] {
        oblikovati_script::worker_stdio()
    } else {
        oblikovati_head::run(&arguments, &mut io::stdout().lock())
    };
    match outcome {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("oblikovati-head: {error}");
            ExitCode::FAILURE
        }
    }
}
