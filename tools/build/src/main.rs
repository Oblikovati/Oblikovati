// SPDX-License-Identifier: GPL-2.0-only

fn main() -> std::process::ExitCode {
    let arguments = std::env::args().skip(1).collect::<Vec<_>>();
    let result = std::env::current_dir()
        .map_err(|error| format!("current directory: {error}"))
        .and_then(|root| oblikovati_build::run(&arguments, &root));
    match result {
        Ok(()) => std::process::ExitCode::SUCCESS,
        Err(error) => {
            eprintln!("oblikovati-build: {error}");
            std::process::ExitCode::FAILURE
        }
    }
}
