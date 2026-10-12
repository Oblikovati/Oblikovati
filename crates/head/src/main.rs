// SPDX-License-Identifier: GPL-2.0-only
use std::{io, process::ExitCode};
fn main() -> ExitCode {
    let arguments: Vec<_> = std::env::args_os().skip(1).collect();
    match oblikovati_head::run(&arguments, &mut io::stdout().lock()) {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("oblikovati-head: {error}");
            ExitCode::FAILURE
        }
    }
}
