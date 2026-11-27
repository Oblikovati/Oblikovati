# SPDX-License-Identifier: GPL-2.0-only
param([Parameter(Mandatory)][string] $Directory)

$ErrorActionPreference = 'Stop'
. "$PSScriptRoot/windows_process.ps1"

if (-not $env:GITHUB_ENV -or -not $env:GITHUB_PATH) {
    throw 'Expected GitHub Actions environment and path files for the Windows Vulkan test driver'
}

# Chrome for Testing publishes a Windows x64 SwiftShader ICD and Vulkan loader together.
$url = 'https://storage.googleapis.com/chrome-for-testing-public/158.0.8127.88/win64/chrome-win64.zip'
$expectedHash = 'fda73ca9ffc7f02c5c85165cb472152540a76a7cecd8ec11314ea781e4454109'
$hostAccess = [BuildBootstrapHost]::new()
[void] [IO.Directory]::CreateDirectory($Directory)
$archive = Join-Path $Directory 'chrome-win64.zip'
try {
    $hostAccess.Download($url, $archive)
    $actualHash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash
    if ($actualHash -ne $expectedHash) {
        throw "Chrome for Testing archive SHA-256 $actualHash does not match the pinned SwiftShader package"
    }

    $zip = [IO.Compression.ZipFile]::OpenRead($archive)
    try {
        foreach ($name in 'vk_swiftshader.dll', 'vk_swiftshader_icd.json', 'vulkan-1.dll') {
            $entry = $zip.GetEntry("chrome-win64/$name")
            if ($null -eq $entry) { throw "Expected $name in the pinned Chrome for Testing archive" }
            [IO.Compression.ZipFileExtensions]::ExtractToFile($entry, (Join-Path $Directory $name), $true)
        }
    } finally { $zip.Dispose() }
} finally { Remove-Item -LiteralPath $archive -Force -ErrorAction SilentlyContinue }

$manifest = Join-Path $Directory 'vk_swiftshader_icd.json'
# New loaders prefer VK_DRIVER_FILES. Earlier loaders recognize VK_ICD_FILENAMES.
@("VK_DRIVER_FILES=$manifest", "VK_ICD_FILENAMES=$manifest") |
    Add-Content -LiteralPath $env:GITHUB_ENV -Encoding utf8
Add-Content -LiteralPath $env:GITHUB_PATH -Value $Directory -Encoding utf8
