// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_persistence::load::NativePackageReader;
use std::{io, process::ExitCode};

fn main() -> ExitCode {
    let arguments: Vec<_> = std::env::args_os().skip(1).collect();
    match oblikovati_cli::run(
        &arguments,
        &mut NativePackageReader,
        &mut io::stdout().lock(),
    ) {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("{error}");
            ExitCode::FAILURE
        }
    }
}
