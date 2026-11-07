# SPDX-License-Identifier: GPL-2.0-only
#requires -Version 5.1
$ErrorActionPreference = 'Stop'
$repository = (Resolve-Path (Join-Path $PSScriptRoot '../../..')).Path
. "$repository/tools/bootstrap/windows_process.ps1"
. "$repository/tools/bootstrap/windows_dependencies.ps1"
. "$repository/tools/bootstrap/windows_build.ps1"
. "$PSScriptRoot/windows_fakes.ps1"
function Assert-Equal {
    param([object] $Actual, [object] $Expected, [string] $Message)
    if ((ConvertTo-Json -InputObject $Actual -Compress) -cne (ConvertTo-Json -InputObject $Expected -Compress)) { throw "$Message`: actual $Actual; expected $Expected" }
}
function Assert-True {
    param([bool] $Condition, [string] $Message)
    if (-not $Condition) { throw $Message }
}
function Test-PrebuiltAndMissingGui {
    $fixture = [FakeWindowsCheckout]::new($repository)
    try {
        Assert-Equal ($fixture.Invoke('build', @())) 0 'build status'
        $fixture.HostAccess.Calls.Clear()
        $fixture.HostAccess.GuiExit = 7
        $arguments = @('document with spaces.obk', '', 'a"b', 'trailing\', '--frames', '3')
        Assert-Equal ($fixture.Invoke('run', $arguments)) 7 'GUI exit status'
        Assert-Equal $fixture.HostAccess.GuiArguments $arguments 'GUI arguments'
        Assert-True (-not ($fixture.HostAccess.Calls -match '^cargo ')) 'prebuilt launch must not compile'
    } finally { $fixture.Dispose() }
}
function Test-StaleSourceAndFailedBuild {
    $fixture = [FakeWindowsCheckout]::new($repository)
    try {
        Assert-Equal ($fixture.Invoke('run', @('--frames', '1'))) 0 'missing GUI builds then runs'
        Set-Content -LiteralPath (Join-Path $fixture.Root 'source.rs') -Value 'new source'
        $fixture.HostAccess.BuildExit = 19
        $fixture.HostAccess.Calls.Clear()
        Assert-Equal ($fixture.Invoke('run', @())) 19 'build failure status'
        Assert-True (-not ($fixture.HostAccess.Calls -match 'oblikovati.exe')) 'failed build must not run an old GUI'
    } finally { $fixture.Dispose() }
}
function Test-CustomTargetAndArm64 {
    $fixture = [FakeWindowsCheckout]::new($repository)
    try {
        $env:CARGO_TARGET_DIR = './custom target/../actual output'
        $fixture.HostAccess.Cpu = 'arm64'
        Assert-Equal ($fixture.Invoke('build', @())) 0 'custom target build'
        Assert-True (Test-Path -LiteralPath (Join-Path $fixture.Root 'actual output/debug/oblikovati.exe')) 'GUI output location'
        Assert-Equal $fixture.HostAccess.MsvcArchitecture 'arm64' 'native MSVC architecture'
        $fixture.HostAccess.Calls.Clear()
        Assert-Equal ($fixture.Invoke('run', @())) 0 'custom prebuilt run'
        Assert-True (-not ($fixture.HostAccess.Calls -match '^cargo ')) 'custom prebuilt launch must not compile'
    } finally { $fixture.Dispose() }
}
function Test-PackageManagersAndDirectInstall {
    foreach ($manager in @('winget', 'choco', 'scoop', 'direct')) {
        $fixture = [FakeWindowsCheckout]::new($repository)
        try {
            $fixture.HostAccess.Manager = $manager
            $fixture.HostAccess.GitInstalled = $false
            $fixture.HostAccess.RustInstalled = $false
            $fixture.HostAccess.MsvcInstalled = $false
            Assert-Equal ($fixture.Invoke('build', @())) 0 "$manager install/build"
            Assert-True $fixture.HostAccess.GitInstalled "$manager Git install"
            Assert-True $fixture.HostAccess.RustInstalled "$manager Rust install"
            Assert-True $fixture.HostAccess.MsvcInstalled "$manager MSVC install"
        } finally { $fixture.Dispose() }
    }
}
function Test-CorruptDownloadAndPublisherRefusal {
    $fixture = [FakeWindowsCheckout]::new($repository)
    try {
        $fixture.HostAccess.GitInstalled = $false
        $fixture.HostAccess.CorruptDownload = $true
        $failed = $false
        try { $null = $fixture.Invoke('build', @()) } catch { $failed = $_.Exception.Message.Contains('SHA-256') }
        Assert-True $failed 'a corrupt download must be refused'
        Assert-True (-not ($fixture.HostAccess.Calls -match 'git-install.exe')) 'a corrupt installer must never execute'
        $fixture.HostAccess.GitInstalled = $true
        $fixture.HostAccess.MsvcInstalled = $false
        $fixture.HostAccess.CorruptDownload = $false
        $fixture.HostAccess.BadSigner = $true
        $failed = $false
        try { $null = $fixture.Invoke('build', @()) } catch { $failed = $_.Exception.Message.Contains('Microsoft Corporation') }
        Assert-True $failed 'a non-Microsoft installer must be refused'
    } finally { $fixture.Dispose() }
}
function Test-CheckAndConcurrentSourceEdit {
    $fixture = [FakeWindowsCheckout]::new($repository)
    try {
        Assert-Equal ($fixture.Invoke('check', @())) 0 'check status'
        Assert-True (-not ($fixture.HostAccess.Calls -match '^cargo ')) 'check must not run Rust builds'
        $fixture.HostAccess.ChangeSource = $true
        $failed = $false
        try { $null = $fixture.Invoke('run', @()) } catch { $failed = $_.Exception.Message.Contains('sources changed') }
        Assert-True $failed 'source edits during build must not bless the binary'
        Assert-True (-not ($fixture.HostAccess.Calls -match 'oblikovati.exe')) 'changed sources must prevent a launch'
    } finally { $fixture.Dispose() }
}
function Test-NativeArgumentQuoting {
    Assert-Equal (ConvertTo-NativeArgument '') '""' 'empty argument quoting'
    Assert-Equal (ConvertTo-NativeArgument 'a"b') '"a\"b"' 'embedded quote escaping'
    Assert-Equal (ConvertTo-NativeArgument 'two words\') '"two words\\"' 'trailing slash escaping'
    Assert-Equal (ConvertTo-NativeArgument 'plain') 'plain' 'plain argument'
}
function Test-RustPathRefreshAndScoopScript {
    $fixture = [FakeWindowsCheckout]::new($repository)
    try {
        $real = [BuildBootstrapHost]::new()
        $real.RefreshPath()
        Assert-True ($env:PATH.Contains((Join-Path $env:CARGO_HOME 'bin'))) 'new Rust tool path must be visible without restarting the terminal'
        # Real script resolution is checked without installing or invoking Scoop itself.
        $script = Join-Path $fixture.Directory 'fake-scoop.ps1'
        Set-Content -LiteralPath $script -Value 'exit 23' -Encoding ASCII
        Assert-True ($real.Has($script)) 'PowerShell tool scripts must be discoverable'
        Assert-Equal ($real.Run($script, @())) 23 'PowerShell tool-script exit propagation'
    } finally { $fixture.Dispose() }
}
$tests = @('Test-PrebuiltAndMissingGui', 'Test-StaleSourceAndFailedBuild', 'Test-CustomTargetAndArm64', 'Test-PackageManagersAndDirectInstall', 'Test-CorruptDownloadAndPublisherRefusal', 'Test-CheckAndConcurrentSourceEdit', 'Test-NativeArgumentQuoting', 'Test-RustPathRefreshAndScoopScript')
foreach ($test in $tests) { & $test; Write-Host "PASS $test" }
Write-Host "$($tests.Count) Windows bootstrap test groups passed (mocked installers/builds; no native Windows runtime claim)."
