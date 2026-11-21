// SPDX-License-Identifier: GPL-2.0-only
use std::{error::Error, path::PathBuf};
fn main() -> Result<(), Box<dyn Error>> {
    let root = std::env::var_os("DEP_IMGUI_THIRD_PARTY")
        .ok_or("DEP_IMGUI_THIRD_PARTY=None: expected the pinned imgui-sys header directory")?;
    let headers = PathBuf::from(root).join("imgui");
    if !headers.join("imgui.h").is_file() {
        return Err(
            format!("ImGui headers {headers:?}: expected a directory containing imgui.h").into(),
        );
    }
    cc::Build::new()
        .cpp(true)
        .include(headers)
        .define("IMGUI_USE_WCHAR32", None)
        .flag_if_supported("-std=c++11")
        .file("src/widget_abi.cpp")
        .try_compile("oblikovati_imgui_widgets")?;
    println!("cargo:rerun-if-changed=src/widget_abi.cpp");
    Ok(())
}
