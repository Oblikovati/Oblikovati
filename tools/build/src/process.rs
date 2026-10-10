// SPDX-License-Identifier: GPL-2.0-only

use std::{path::Path, process::Command};

pub(crate) type BuildResult<T> = Result<T, String>;

pub(crate) trait Runner {
    fn output(&self, root: &Path, program: &str, arguments: &[&str]) -> BuildResult<String>;
    fn output_env(
        &self,
        root: &Path,
        program: &str,
        arguments: &[&str],
        environment: &[(&str, &str)],
    ) -> BuildResult<String>;
}

pub(crate) struct SystemRunner;

impl Runner for SystemRunner {
    fn output(&self, root: &Path, program: &str, arguments: &[&str]) -> BuildResult<String> {
        self.output_env(root, program, arguments, &[])
    }

    fn output_env(
        &self,
        root: &Path,
        program: &str,
        arguments: &[&str],
        environment: &[(&str, &str)],
    ) -> BuildResult<String> {
        let output = Command::new(program)
            .args(arguments)
            .envs(environment.iter().copied())
            .current_dir(root)
            .output()
            .map_err(|error| format!("required command {program:?}: {error}"))?;
        if !output.status.success() {
            return Err(format!(
                "{program} {arguments:?} failed ({}):\n{}\n{}",
                output.status,
                String::from_utf8_lossy(&output.stdout),
                String::from_utf8_lossy(&output.stderr)
            ));
        }
        String::from_utf8(output.stdout)
            .map_err(|error| format!("{program}: expected UTF-8 output: {error}"))
    }
}

pub(crate) fn execute(runner: &dyn Runner, root: &Path, command: &[&str]) -> BuildResult<()> {
    println!("+ {}", command.join(" "));
    print!("{}", runner.output(root, command[0], &command[1..])?);
    Ok(())
}

#[cfg(test)]
#[path = "../tests/process.rs"]
pub(crate) mod tests;
