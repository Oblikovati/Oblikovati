// SPDX-License-Identifier: GPL-2.0-only
use serde::{Deserialize, Serialize};
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ScriptLimits {
    pub wall_millis: u64,
    pub memory_bytes: u64,
    pub instructions: u64,
    pub stack_frames: usize,
    pub output_bytes: usize,
    pub host_calls: u64,
    pub source_bytes: usize,
}
impl Default for ScriptLimits {
    fn default() -> Self {
        Self {
            wall_millis: 30_000,
            memory_bytes: 256 << 20,
            instructions: 20_000_000,
            stack_frames: 256,
            output_bytes: 1 << 20,
            host_calls: 100_000,
            source_bytes: 1 << 20,
        }
    }
}
impl ScriptLimits {
    /// Refuse unbounded production runs; example: `limits.validate(source)?;`.
    pub fn validate(&self, source: &str) -> Result<(), String> {
        self.require_bounds()?;
        if source.len() > self.source_bytes {
            return Err(format!(
                "script source {} bytes: expected at most {}",
                source.len(),
                self.source_bytes
            ));
        }
        Ok(())
    }
    fn require_bounds(&self) -> Result<(), String> {
        let bounds = [
            self.wall_millis,
            self.memory_bytes,
            self.instructions,
            self.stack_frames as u64,
            self.output_bytes as u64,
            self.host_calls,
            self.source_bytes as u64,
        ];
        if bounds.contains(&0) {
            return Err(format!(
                "script limits {self:?}: expected positive finite bounds"
            ));
        }
        Ok(())
    }
}
#[derive(Clone, Debug, Serialize, Deserialize)]
pub struct ScriptProgram {
    pub source: String,
    pub methods: Vec<String>,
    pub limits: ScriptLimits,
    pub parse_only: bool,
}
#[derive(Clone, Debug, Default, PartialEq, Eq, Serialize, Deserialize)]
pub struct ScriptReport {
    pub instructions: u64,
    pub output_bytes: usize,
    pub host_calls: u64,
}
