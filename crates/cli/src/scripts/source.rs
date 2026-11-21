// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::io::Read;
/// Script text access is injected separately from document package storage.
pub trait ScriptSources {
    /// Read bounded UTF-8 Lua source; example: `sources.read_source(path, limit)?;`.
    fn read_source(&mut self, path: &Path, maximum: usize) -> Result<String, String>;
}
pub struct NativeScriptSources;
impl ScriptSources for NativeScriptSources {
    fn read_source(&mut self, path: &Path, maximum: usize) -> Result<String, String> {
        let file = std::fs::File::open(path)
            .map_err(|error| format!("script run: read {path:?}: {error}"))?;
        let mut source = String::new();
        file.take(maximum.saturating_add(1) as u64)
            .read_to_string(&mut source)
            .map_err(|error| {
                format!("script source {path:?}: expected bounded UTF-8 text: {error}")
            })?;
        if source.len() > maximum {
            return Err(format!(
                "script source {path:?} is {} bytes: expected at most {maximum}",
                source.len()
            ));
        }
        Ok(source)
    }
}
