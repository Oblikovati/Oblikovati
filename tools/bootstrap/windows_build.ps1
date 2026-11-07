# SPDX-License-Identifier: GPL-2.0-only
function Get-TextSha256 {
    [OutputType([string])]
    param([AllowEmptyString()][string] $Text)
    $algorithm = [Security.Cryptography.SHA256]::Create()
    try { return [BitConverter]::ToString($algorithm.ComputeHash([Text.Encoding]::UTF8.GetBytes($Text))).Replace('-', '').ToLowerInvariant() }
    finally { $algorithm.Dispose() }
}
function Get-BootstrapSourceFingerprint {
    [OutputType([string])]
    param([string] $Root, [string] $Target, [BuildBootstrapHost] $HostAccess)
    $records = [Text.StringBuilder]::new()
    $null = $records.Append($HostAccess.Capture('git', @('rev-parse', 'HEAD'))).Append([char]0)
    $null = $records.Append($HostAccess.Capture('git', @('diff', '--binary', 'HEAD', '--', '.'))).Append([char]0)
    $untracked = $HostAccess.Capture('git', @('ls-files', '--others', '--exclude-standard', '-z'))
    foreach ($relative in $untracked.Split([char[]]@([char]0), [StringSplitOptions]::RemoveEmptyEntries)) {
        $path = [IO.Path]::GetFullPath((Join-Path $Root $relative))
        if ($path.StartsWith($Target + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { continue }
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { continue }
        $null = $records.Append($relative).Append([char]0).Append((Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash).Append([char]0)
    }
    return Get-TextSha256 $records.ToString()
}
function Test-BootstrapFresh {
    [OutputType([bool])]
    param([string] $Root, [string] $Target, [string] $Gui, [string] $Stamp, [BuildBootstrapHost] $HostAccess)
    if (-not (Test-Path -LiteralPath $Gui -PathType Leaf) -or -not (Test-Path -LiteralPath $Stamp -PathType Leaf) -or -not $HostAccess.Has('git')) { return $false }
    $lines = [IO.File]::ReadAllLines($Stamp)
    if ($lines.Length -ne 2) { return $false }
    $source = Get-BootstrapSourceFingerprint $Root $Target $HostAccess
    $binary = (Get-FileHash -LiteralPath $Gui -Algorithm SHA256).Hash.ToLowerInvariant()
    return $lines[0] -ceq $source -and $lines[1] -ceq $binary
}
function Test-BootstrapCache {
    [OutputType([bool])]
    param([string] $Root, [BuildBootstrapHost] $HostAccess)
    if ($env:BUILD_CACHE -eq 'off') { return $true }
    if (-not $HostAccess.Has('kache')) { return $false }
    $source = Get-Content -LiteralPath (Join-Path $Root 'tools/build/src/kache.rs') -Raw
    $match = [regex]::Match($source, 'const KACHE_VERSION: &str = "([^"]+)";')
    if (-not $match.Success) { throw 'Kache pin: expected KACHE_VERSION in the Rust build system' }
    return $HostAccess.Capture('kache', @('--version')).Trim() -ceq "kache $($match.Groups[1].Value)"
}
function Set-BootstrapCache {
    param([string] $Root, [BuildBootstrapHost] $HostAccess)
    if ($env:BUILD_CACHE -eq 'off') { $env:RUSTC_WRAPPER = ''; return }
    if (-not (Test-BootstrapCache $Root $HostAccess)) {
        $env:RUSTC_WRAPPER = ''
        Invoke-RequiredCommand $HostAccess 'cargo' @('xtask', 'cache-tools')
    }
    if (-not (Test-BootstrapCache $Root $HostAccess)) { throw 'cache-tools did not install the pinned kache executable' }
    $env:RUSTC_WRAPPER = 'kache'
    $env:KACHE_CONFIG = Join-Path $Root '.kache.toml'
    if ([string]::IsNullOrEmpty($env:KACHE_CACHE_DIR)) {
        $common = $HostAccess.Capture('git', @('rev-parse', '--path-format=absolute', '--git-common-dir')).Trim()
        $env:KACHE_CACHE_DIR = Join-Path $common 'oblikovati-kache'
    }
}
function Set-BootstrapPaths {
    [OutputType([string])]
    param([string] $Root)
    $cargoHome = $env:CARGO_HOME
    if ([string]::IsNullOrEmpty($cargoHome)) { $cargoHome = Join-Path $env:USERPROFILE '.cargo' }
    $env:PATH = "$(Join-Path $Root '.tools');$(Join-Path $cargoHome 'bin');$env:PATH"
    $target = $env:CARGO_TARGET_DIR
    if ([string]::IsNullOrEmpty($target)) { $target = Join-Path $Root 'target' }
    if (-not [IO.Path]::IsPathRooted($target)) { $target = Join-Path $Root $target }
    $target = [IO.Path]::GetFullPath($target)
    if ($target -eq $Root -or $target -eq [IO.Path]::GetPathRoot($target)) { throw "target directory $target`: expected a separate build output directory" }
    $env:CARGO_TARGET_DIR = $target
    return $target
}
function Set-BootstrapGitLongPaths {
    $count = 0
    if (-not [string]::IsNullOrEmpty($env:GIT_CONFIG_COUNT) -and -not [int]::TryParse($env:GIT_CONFIG_COUNT, [ref]$count)) { throw "GIT_CONFIG_COUNT=$env:GIT_CONFIG_COUNT`: expected a nonnegative integer" }
    if ($count -lt 0) { throw "GIT_CONFIG_COUNT=$count`: expected a nonnegative integer" }
    [Environment]::SetEnvironmentVariable("GIT_CONFIG_KEY_$count", 'core.longpaths', 'Process')
    [Environment]::SetEnvironmentVariable("GIT_CONFIG_VALUE_$count", 'true', 'Process')
    $env:GIT_CONFIG_COUNT = [string]($count + 1)
}
function Install-BootstrapDependencies {
    param([string] $Root, [string] $Temporary, [string] $Architecture, [bool] $Check, [BuildBootstrapHost] $HostAccess)
    if ($Check) {
        foreach ($command in @('git', 'curl.exe', 'tar.exe')) { if (-not $HostAccess.Has($command)) { throw "missing $command`: run build.ps1 to prepare build dependencies" } }
        if (-not (Test-BootstrapRust $Root $HostAccess)) { throw 'missing the pinned Rust toolchain/components: run build.ps1' }
        if (-not (Test-BootstrapCache $Root $HostAccess)) { throw 'missing the pinned kache version: run build.ps1' }
        Set-BootstrapMsvc $Temporary $Architecture $true $HostAccess
        return
    }
    Install-BootstrapGit $Root $Temporary $Architecture $HostAccess
    foreach ($command in @('git', 'curl.exe', 'tar.exe')) {
        if (-not $HostAccess.Has($command)) { throw "missing $command after setup: expected Git and the native curl/bsdtar tools shipped with supported Windows 10/11; repair the OS tools and rerun" }
    }
    Set-BootstrapMsvc $Temporary $Architecture $false $HostAccess
    Install-BootstrapRust $Root $Temporary $Architecture $HostAccess
    $env:RUSTUP_TOOLCHAIN = Get-RustToolchainPin $Root
    Set-BootstrapCache $Root $HostAccess
}
function Invoke-BootstrapBuild {
    [OutputType([int])]
    param([string] $Root, [string] $Target, [string] $Gui, [string] $Stamp, [BuildBootstrapHost] $HostAccess)
    $before = Get-BootstrapSourceFingerprint $Root $Target $HostAccess
    $null = New-Item -ItemType Directory -Path (Split-Path -Parent $Stamp) -Force
    if (Test-Path -LiteralPath $Stamp) { Remove-Item -LiteralPath $Stamp -Force }
    $status = $HostAccess.Run('cargo', @('xtask', 'build'))
    if ($status -ne 0) { return $status }
    if (-not (Test-Path -LiteralPath $Gui -PathType Leaf)) { throw "build output $Gui`: expected the oblikovati GUI executable" }
    $after = Get-BootstrapSourceFingerprint $Root $Target $HostAccess
    if ($before -cne $after) { throw 'sources changed during the build: rerun to create a current GUI' }
    $binary = (Get-FileHash -LiteralPath $Gui -Algorithm SHA256).Hash.ToLowerInvariant()
    [IO.File]::WriteAllLines("$Stamp.part", @($after, $binary), [Text.UTF8Encoding]::new($false))
    Move-Item -LiteralPath "$Stamp.part" -Destination $Stamp -Force
    Write-Host "Built $Gui"
    return 0
}
function Invoke-OblikovatiBootstrap {
    [OutputType([int])]
    param([string] $Root, [string] $Mode, [string[]] $GuiArguments, [BuildBootstrapHost] $HostAccess)
    if ($Mode -eq 'help') { Write-Host 'Usage: .\build.ps1 [build|run [GUI_ARGUMENTS...]|check|help]'; return 0 }
    if (-not $HostAccess.IsWindows()) { throw 'build.ps1 requires Windows; Linux/macOS use build.sh' }
    if ($Mode -ne 'run' -and $GuiArguments.Count -ne 0) { throw "arguments $GuiArguments`: expected GUI arguments only after run" }
    if ($env:CARGO_BUILD_TARGET) { throw "CARGO_BUILD_TARGET=$env:CARGO_BUILD_TARGET`: expected unset for a native desktop build" }
    if ($env:BUILD_CACHE -and $env:BUILD_CACHE -notin @('kache', 'off')) { throw "BUILD_CACHE=$env:BUILD_CACHE`: expected kache or off" }
    $architecture = $HostAccess.Architecture()
    $Root = [IO.Path]::GetFullPath($Root)
    Push-Location -LiteralPath $Root
    try { return Invoke-BootstrapInCheckout $Root $Mode $GuiArguments $architecture $HostAccess }
    finally { Pop-Location }
}
function Invoke-BootstrapInCheckout {
    [OutputType([int])]
    param([string] $Root, [string] $Mode, [string[]] $GuiArguments, [string] $Architecture, [BuildBootstrapHost] $HostAccess)
    $target = Set-BootstrapPaths $Root
    Set-BootstrapGitLongPaths
    $gui = Join-Path $target 'debug/oblikovati.exe'
    $stamp = Join-Path $Root ".tools/bootstrap-$(Get-TextSha256 $gui).sha256"
    if ($GuiArguments.Count -gt 0 -and $GuiArguments[0] -eq '--') { $GuiArguments = @($GuiArguments | Select-Object -Skip 1) }
    if ($Mode -eq 'run' -and (Test-BootstrapFresh $Root $target $gui $stamp $HostAccess)) { return $HostAccess.Run($gui, $GuiArguments) }
    $temporary = Join-Path ([IO.Path]::GetTempPath()) "oblikovati-build-$([guid]::NewGuid().ToString('N'))"
    $null = New-Item -ItemType Directory -Path $temporary
    try { return Invoke-PreparedBootstrap $Root $Mode $GuiArguments $Architecture $target $gui $stamp $temporary $HostAccess }
    finally { Remove-Item -LiteralPath $temporary -Recurse -Force }
}
function Invoke-PreparedBootstrap {
    [OutputType([int])]
    param([string] $Root, [string] $Mode, [string[]] $GuiArguments, [string] $Architecture, [string] $Target, [string] $Gui, [string] $Stamp, [string] $Temporary, [BuildBootstrapHost] $HostAccess)
    Install-BootstrapDependencies $Root $Temporary $Architecture ($Mode -eq 'check') $HostAccess
    if ($Mode -eq 'check') { Write-Host 'Build dependencies are installed.'; return 0 }
    $status = Invoke-BootstrapBuild $Root $Target $Gui $Stamp $HostAccess
    if ($status -ne 0 -or $Mode -ne 'run') { return $status }
    return $HostAccess.Run($Gui, $GuiArguments)
}
