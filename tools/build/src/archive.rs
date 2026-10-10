// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    process::BuildResult,
    tree::{all_files, relative_name},
};
use flate2::{Compression, GzBuilder};
use std::{
    fs::{self, File},
    io::Write,
    path::Path,
};
use zip::{ZipWriter, write::SimpleFileOptions};

pub(crate) fn create(stage: &Path, output: &Path) -> BuildResult<()> {
    let files = all_files(stage)?;
    if files.is_empty() {
        return Err(format!(
            "archive source {}: expected nonempty package content",
            stage.display()
        ));
    }
    let parent = output
        .parent()
        .ok_or_else(|| format!("{}: expected an output parent", output.display()))?;
    fs::create_dir_all(parent).map_err(|error| error.to_string())?;
    let result = if output
        .extension()
        .is_some_and(|extension| extension == "zip")
    {
        zip(stage, output, &files)
    } else {
        tar_gz(stage, output, &files)
    };
    result.map_err(|error| format!("archive {}: {error}", output.display()))
}

fn mode(relative: &str) -> u32 {
    if relative.starts_with("bin/") || relative.starts_with("lib/") {
        0o755
    } else {
        0o644
    }
}

fn tar_gz(stage: &Path, output: &Path, files: &[std::path::PathBuf]) -> Result<(), std::io::Error> {
    let gzip = GzBuilder::new()
        .mtime(0)
        .write(File::create(output)?, Compression::default());
    let mut archive = tar::Builder::new(gzip);
    for file in files {
        let relative = relative_name(stage, file).map_err(std::io::Error::other)?;
        let bytes = fs::read(file)?;
        let mut header = tar::Header::new_gnu();
        header.set_size(bytes.len() as u64);
        header.set_mode(mode(&relative));
        header.set_mtime(0);
        header.set_cksum();
        archive.append_data(&mut header, relative, bytes.as_slice())?;
    }
    archive.into_inner()?.finish()?;
    Ok(())
}

fn zip(stage: &Path, output: &Path, files: &[std::path::PathBuf]) -> Result<(), std::io::Error> {
    let mut archive = ZipWriter::new(File::create(output)?);
    for file in files {
        let relative = relative_name(stage, file).map_err(std::io::Error::other)?;
        let options = SimpleFileOptions::default()
            .unix_permissions(mode(&relative))
            .compression_method(zip::CompressionMethod::Deflated)
            .last_modified_time(zip::DateTime::default());
        archive.start_file(relative, options)?;
        archive.write_all(&fs::read(file)?)?;
    }
    archive.finish()?;
    Ok(())
}

#[cfg(test)]
#[path = "../tests/archive.rs"]
mod tests;
