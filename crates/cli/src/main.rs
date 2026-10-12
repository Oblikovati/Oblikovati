// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::identity::SystemGuidSource;
use oblikovati_persistence::file_access::NativeDocumentFiles;
use std::{io, process::ExitCode};

fn main() -> ExitCode {
    let arguments: Vec<_> = std::env::args_os().skip(1).collect();
    if arguments == [std::ffi::OsString::from(oblikovati_script::WORKER_ARGUMENT)] {
        return worker_exit();
    }
    match oblikovati_cli::run_with_files(
        &arguments,
        NativeDocumentFiles,
        &mut SystemGuidSource,
        &mut io::stdout().lock(),
    ) {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("{error}");
            ExitCode::FAILURE
        }
    }
}

fn worker_exit() -> ExitCode {
    match oblikovati_script::worker_stdio() {
        Ok(()) => ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("{error}");
            ExitCode::FAILURE
        }
    }
}
