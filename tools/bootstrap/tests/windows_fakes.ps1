# SPDX-License-Identifier: GPL-2.0-only
#requires -Version 5.1
class FakeWindowsBuildHost : BuildBootstrapHost {
    [string] $Root
    [string] $GitExecutable
    [string] $Manager = ''
    [string] $Cpu = 'x64'
    [bool] $GitInstalled = $true
    [bool] $RustInstalled = $true
    [bool] $MsvcInstalled = $true
    [bool] $CacheInstalled = $true
    [bool] $ChangeSource = $false
    [bool] $BadSigner = $false
    [bool] $CorruptDownload = $false
    [int] $BuildExit = 0
    [int] $GuiExit = 0
    [string[]] $GuiArguments = @()
    [string] $MsvcArchitecture = ''
    [Collections.Generic.List[string]] $Calls = [Collections.Generic.List[string]]::new()
    FakeWindowsBuildHost([string] $Root, [string] $GitExecutable) { $this.Root = $Root; $this.GitExecutable = $GitExecutable }
    [bool] IsWindows() { return $true }
    [string] Architecture() { return $this.Cpu }
    [bool] Has([string] $Name) {
        switch ($Name) {
            'git' { return $this.GitInstalled }
            'rustup' { return $this.RustInstalled }
            'cargo' { return $this.RustInstalled }
            'cl' { return $this.MsvcInstalled }
            'kache' { return $this.CacheInstalled }
            'curl.exe' { return $true }
            'tar.exe' { return $true }
            default { return $Name -eq $this.Manager }
        }
        return $false
    }
    [int] Run([string] $Executable, [string[]] $Arguments) {
        $this.Calls.Add($Executable + ' ' + [string]::Join('|', $Arguments))
        $name = [IO.Path]::GetFileName($Executable)
        if ($name -eq 'cargo' -and $Arguments[1] -eq 'build') { return $this.BuildFixtureGui() }
        if ($name -eq 'oblikovati.exe') { $this.GuiArguments = $Arguments; return $this.GuiExit }
        if ($name -eq 'git-install.exe' -or $Arguments -contains 'Git.Git' -or $Arguments -contains 'git') { $this.GitInstalled = $true }
        if ($name -eq 'rustup-init.exe' -or $name -eq 'rustup') { $this.RustInstalled = $true }
        if ($name -eq 'vs-buildtools.exe' -or $Arguments -contains 'Microsoft.VisualStudio.2022.BuildTools' -or $Arguments -contains 'visualstudio2022buildtools') { $this.MsvcInstalled = $true }
        if ($name -eq 'cargo' -and $Arguments[1] -eq 'cache-tools') { $this.CacheInstalled = $true }
        return 0
    }
    [int] BuildFixtureGui() {
        if ($this.BuildExit -ne 0) { return $this.BuildExit }
        $directory = Join-Path $env:CARGO_TARGET_DIR 'debug'
        $null = New-Item -ItemType Directory -Path $directory -Force
        Set-Content -LiteralPath (Join-Path $directory 'oblikovati.exe') -Value 'fixture GUI' -Encoding ASCII
        if ($this.ChangeSource) { Set-Content -LiteralPath (Join-Path $this.Root 'changed.rs') -Value 'changed source' -Encoding ASCII }
        return 0
    }
    [string] Capture([string] $Executable, [string[]] $Arguments) {
        if ($Executable -eq 'git') { return ([BuildBootstrapHost]::new()).Capture($this.GitExecutable, $Arguments) }
        if ($Executable -eq 'kache') { return 'kache 2.0.0' }
        if ($Executable -eq 'rustup' -and $this.RustInstalled) {
            if ($Arguments[0] -eq 'component') { return "rustfmt-x86_64-pc-windows-msvc`nclippy-x86_64-pc-windows-msvc`nllvm-tools-x86_64-pc-windows-msvc" }
            return 'rustc 1.99.0'
        }
        throw "unexpected capture $Executable $Arguments"
    }
    [void] Download([string] $Url, [string] $Output) {
        $this.Calls.Add('download ' + $Url)
        $payload = 'fixture installer'
        if ($this.CorruptDownload) { $payload = 'corrupted download' }
        [IO.File]::WriteAllText($Output, $payload, [Text.UTF8Encoding]::new($false))
    }
    [string] MicrosoftSigner([string] $Path) {
        if ($this.BadSigner) { return 'CN=Untrusted, O=Other Corporation, C=US' }
        return 'CN=Microsoft Corporation, O=Microsoft Corporation, C=US'
    }
    [void] RefreshPath() { $this.Calls.Add('refresh-path') }
    [string] MsvcInstallation() { if ($this.MsvcInstalled) { return 'fixture MSVC' }; return '' }
    [void] EnterMsvc([string] $Installation, [string] $Architecture) { $this.MsvcArchitecture = $Architecture }
}
class FakeWindowsCheckout {
    [string] $Root
    [string] $Directory
    [string] $PreviousLocation
    [FakeWindowsBuildHost] $HostAccess
    [hashtable] $Environment = @{}
    FakeWindowsCheckout([string] $Repository) {
        $this.Directory = Join-Path ([IO.Path]::GetTempPath()) "oblikovati-ps-test-$([guid]::NewGuid().ToString('N'))"
        $this.Root = Join-Path $this.Directory 'checkout with spaces'
        $null = New-Item -ItemType Directory -Path (Join-Path $this.Root 'tools/bootstrap') -Force
        Copy-Item -Path (Join-Path $Repository 'tools/bootstrap/*') -Destination (Join-Path $this.Root 'tools/bootstrap') -Recurse
        $null = New-Item -ItemType Directory -Path (Join-Path $this.Root 'tools/build/src') -Force
        Copy-Item -LiteralPath (Join-Path $Repository 'tools/build/src/kache.rs') -Destination (Join-Path $this.Root 'tools/build/src/kache.rs')
        Copy-Item -LiteralPath (Join-Path $Repository 'rust-toolchain.toml') -Destination $this.Root
        [IO.File]::WriteAllText((Join-Path $this.Root '.gitignore'), "target/`n.tools/`n")
        $git = (Get-Command git -CommandType Application | Select-Object -First 1).Source
        $this.HostAccess = [FakeWindowsBuildHost]::new($this.Root, $git)
        $this.PreviousLocation = (Get-Location).Path
        Set-Location -LiteralPath $this.Root
        $this.PrepareEnvironment()
        $this.PrepareDownloadFixture()
        $this.PrepareGit($git)
    }
    [void] PrepareEnvironment() {
        foreach ($name in @('PATH', 'USERPROFILE', 'CARGO_HOME', 'CARGO_TARGET_DIR', 'CARGO_BUILD_TARGET', 'BUILD_CACHE', 'RUSTC_WRAPPER', 'RUSTUP_TOOLCHAIN', 'KACHE_CONFIG', 'KACHE_CACHE_DIR', 'GIT_CONFIG_COUNT', 'GIT_CONFIG_KEY_0', 'GIT_CONFIG_VALUE_0')) {
            $this.Environment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
        }
        $env:USERPROFILE = $this.Directory
        $env:CARGO_HOME = Join-Path $this.Directory 'cargo home'
        foreach ($name in @('CARGO_TARGET_DIR', 'CARGO_BUILD_TARGET', 'BUILD_CACHE', 'RUSTC_WRAPPER', 'KACHE_CACHE_DIR', 'GIT_CONFIG_COUNT')) { [Environment]::SetEnvironmentVariable($name, $null, 'Process') }
    }
    [void] PrepareDownloadFixture() {
        $file = Join-Path $this.Root 'tools/bootstrap/downloads.lock'
        $digest = Get-TextSha256 'fixture installer'
        $rows = Get-Content -LiteralPath $file
        $rows = @($rows | ForEach-Object { if ($_ -match '^([^# ]+) [a-f0-9]{64} (https://.*)$') { "$($Matches[1]) $digest $($Matches[2])" } else { $_ } })
        [IO.File]::WriteAllLines($file, $rows)
    }
    [void] PrepareGit([string] $Git) {
        foreach ($arguments in @(@('init', '-q'), @('add', '.'), @('-c', 'user.name=Bootstrap test', '-c', 'user.email=bootstrap@example.invalid', 'commit', '-qm', 'Fixture'))) {
            if (([BuildBootstrapHost]::new()).Run($Git, $arguments) -ne 0) { throw 'fixture git initialization failed' }
        }
    }
    [int] Invoke([string] $Mode, [string[]] $Arguments) {
        Set-Location -LiteralPath $this.Directory
        return Invoke-OblikovatiBootstrap $this.Root $Mode $Arguments $this.HostAccess
    }
    [void] Dispose() {
        Set-Location -LiteralPath $this.PreviousLocation
        foreach ($name in $this.Environment.Keys) { [Environment]::SetEnvironmentVariable($name, $this.Environment[$name], 'Process') }
        Remove-Item -LiteralPath $this.Directory -Recurse -Force
    }
}
