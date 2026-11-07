# SPDX-License-Identifier: GPL-2.0-only
function Get-BootstrapManager {
    [OutputType([string])]
    param([BuildBootstrapHost] $HostAccess)
    foreach ($manager in @('winget', 'choco', 'scoop')) {
        if ($HostAccess.Has($manager)) { return $manager }
    }
    return 'direct'
}
function Get-LockedDownload {
    [OutputType([string[]])]
    param([string] $Root, [string] $Key)
    $rows = @(Get-Content -LiteralPath (Join-Path $Root 'tools/bootstrap/downloads.lock') | Where-Object { $_.StartsWith("$Key ", [StringComparison]::Ordinal) })
    if ($rows.Count -ne 1) { throw "download key $Key`: expected exactly one locked artifact" }
    $fields = $rows[0].Split(' ')
    if ($fields.Count -ne 3 -or $fields[1] -cnotmatch '^[a-f0-9]{64}$' -or -not $fields[2].StartsWith('https://')) {
        throw "download entry $($rows[0]): expected key, pinned SHA-256 and HTTPS URL"
    }
    return @($fields[1], $fields[2])
}
function Save-LockedDownload {
    param([string] $Root, [string] $Key, [string] $Output, [BuildBootstrapHost] $HostAccess)
    $locked = @(Get-LockedDownload $Root $Key)
    $HostAccess.Download($locked[1], "$Output.part")
    $actual = (Get-FileHash -LiteralPath "$Output.part" -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -cne $locked[0]) { throw "download $Key SHA-256 $actual`: expected $($locked[0])" }
    Move-Item -LiteralPath "$Output.part" -Destination $Output -Force
}
function Install-ManagedDependency {
    param([string] $Manager, [string] $Winget, [string] $Chocolatey, [string] $Scoop, [BuildBootstrapHost] $HostAccess)
    switch ($Manager) {
        'winget' { Invoke-RequiredCommand $HostAccess 'winget' @('install', '--id', $Winget, '--exact', '--source', 'winget', '--silent', '--accept-package-agreements', '--accept-source-agreements') }
        'choco' { Invoke-RequiredCommand $HostAccess 'choco' @('install', $Chocolatey, '-y', '--no-progress') }
        'scoop' { Invoke-RequiredCommand $HostAccess 'scoop' @('install', $Scoop) }
        default { throw "package manager $Manager`: expected winget, choco or scoop" }
    }
    $HostAccess.RefreshPath()
}
function Install-BootstrapGit {
    param([string] $Root, [string] $Temporary, [string] $Architecture, [BuildBootstrapHost] $HostAccess)
    if ($HostAccess.Has('git')) { return }
    $manager = Get-BootstrapManager $HostAccess
    if ($manager -ne 'direct') { Install-ManagedDependency $manager 'Git.Git' 'git' 'git' $HostAccess; return }
    $installer = Join-Path $Temporary 'git-install.exe'
    Save-LockedDownload $Root "git-$Architecture" $installer $HostAccess
    Invoke-RequiredCommand $HostAccess $installer @('/VERYSILENT', '/NORESTART', '/NOCANCEL', '/SP-', '/CURRENTUSER')
    $HostAccess.RefreshPath()
}
function Install-BootstrapMsvc {
    param([string] $Temporary, [BuildBootstrapHost] $HostAccess)
    $workload = '--add Microsoft.VisualStudio.Workload.VCTools --add Microsoft.VisualStudio.Component.VC.Tools.ARM64 --includeRecommended --wait --passive --norestart'
    switch (Get-BootstrapManager $HostAccess) {
        'winget' { Invoke-RequiredCommand $HostAccess 'winget' @('install', '--id', 'Microsoft.VisualStudio.2022.BuildTools', '--exact', '--source', 'winget', '--accept-package-agreements', '--accept-source-agreements', '--override', $workload) }
        'choco' { Invoke-RequiredCommand $HostAccess 'choco' @('install', 'visualstudio2022buildtools', '-y', '--no-progress', '--package-parameters', $workload) }
        default { Install-MicrosoftBuildTools $Temporary $workload $HostAccess }
    }
    $HostAccess.RefreshPath()
}
function Install-MicrosoftBuildTools {
    param([string] $Temporary, [string] $Workload, [BuildBootstrapHost] $HostAccess)
    $installer = Join-Path $Temporary 'vs-buildtools.exe'
    $HostAccess.Download('https://aka.ms/vs/17/release/vs_BuildTools.exe', $installer)
    $publisher = $HostAccess.MicrosoftSigner($installer)
    if ($publisher -notmatch '(?:^|,\s*)O=Microsoft Corporation(?:,|$)') {
        throw "installer signer $publisher`: expected Microsoft Corporation"
    }
    Invoke-RequiredCommand $HostAccess $installer $Workload.Split(' ')
}
function Set-BootstrapMsvc {
    param([string] $Temporary, [string] $Architecture, [bool] $Check, [BuildBootstrapHost] $HostAccess)
    $installation = $HostAccess.MsvcInstallation()
    if ($installation -eq '' -and -not $HostAccess.Has('cl')) {
        if ($Check) { throw 'MSVC is missing: run .\build.ps1 to install the C++ workload and Windows SDK' }
        Install-BootstrapMsvc $Temporary $HostAccess
        $installation = $HostAccess.MsvcInstallation()
    }
    if ($installation -ne '') { $HostAccess.EnterMsvc($installation, $Architecture) }
    if (-not $HostAccess.Has('cl')) { throw 'MSVC cl.exe is missing after setup: expected an x64/ARM64 C++ compiler and Windows SDK' }
    $probe = Join-Path $Temporary 'compiler-probe.cpp'
    Set-Content -LiteralPath $probe -Value 'int main() { return 0; }' -Encoding ASCII
    Invoke-RequiredCommand $HostAccess 'cl' @('/nologo', '/EHsc', $probe, "/Fe$(Join-Path $Temporary 'compiler-probe.exe')", "/Fo$(Join-Path $Temporary 'compiler-probe.obj')")
}
function Get-RustToolchainPin {
    [OutputType([string])]
    param([string] $Root)
    $contents = Get-Content -LiteralPath (Join-Path $Root 'rust-toolchain.toml') -Raw
    $match = [regex]::Match($contents, '(?m)^channel = "([0-9]+\.[0-9]+\.[0-9]+)"\r?$')
    if (-not $match.Success) { throw 'Rust channel: expected a pinned semantic version in rust-toolchain.toml' }
    return $match.Groups[1].Value
}
function Get-RustToolchainComponents {
    [OutputType([string[]])]
    param([string] $Root)
    $contents = Get-Content -LiteralPath (Join-Path $Root 'rust-toolchain.toml') -Raw
    $match = [regex]::Match($contents, '(?m)^components = \[([^\]]+)\]\r?$')
    if (-not $match.Success) { throw 'Rust components: expected a nonempty pinned component list' }
    return @($match.Groups[1].Value.Split(',') | ForEach-Object { $_.Trim(' ', '"') })
}
function Test-BootstrapRust {
    [OutputType([bool])]
    param([string] $Root, [BuildBootstrapHost] $HostAccess)
    if (-not $HostAccess.Has('rustup')) { return $false }
    $pin = Get-RustToolchainPin $Root
    try { $null = $HostAccess.Capture('rustup', @('run', $pin, 'rustc', '--version')) } catch { return $false }
    $installed = $HostAccess.Capture('rustup', @('component', 'list', '--installed', '--toolchain', $pin))
    foreach ($component in (Get-RustToolchainComponents $Root)) {
        $prefix = $component -replace '-preview$', ''
        if ($installed -notmatch "(?m)^$([regex]::Escape($prefix))(-|$)") { return $false }
    }
    return $true
}
function Install-BootstrapRust {
    param([string] $Root, [string] $Temporary, [string] $Architecture, [BuildBootstrapHost] $HostAccess)
    if (Test-BootstrapRust $Root $HostAccess) { return }
    if (-not $HostAccess.Has('rustup')) {
        $target = 'x86_64-pc-windows-msvc'
        if ($Architecture -eq 'arm64') { $target = 'aarch64-pc-windows-msvc' }
        $installer = Join-Path $Temporary 'rustup-init.exe'
        Save-LockedDownload $Root "rustup-$target" $installer $HostAccess
        Invoke-RequiredCommand $HostAccess $installer @('-y', '--no-modify-path', '--default-toolchain', 'none')
        $HostAccess.RefreshPath()
    }
    $arguments = @('toolchain', 'install', (Get-RustToolchainPin $Root), '--profile', 'minimal')
    foreach ($component in (Get-RustToolchainComponents $Root)) { $arguments += @('--component', $component) }
    Invoke-RequiredCommand $HostAccess 'rustup' $arguments
    if (-not (Test-BootstrapRust $Root $HostAccess)) { throw 'Rust installation did not provide the pinned toolchain and every component' }
}
