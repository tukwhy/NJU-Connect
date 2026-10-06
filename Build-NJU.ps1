[CmdletBinding()]
param(
    [string]$GoPath = '',
    [switch]$Test,
    [string]$OutputPath = 'dist/nju-connect.exe',
    [string]$Version = 'v0.1.0-nju.1'
)

$ErrorActionPreference = 'Stop'
if (!$GoPath) {
    $localGo = Join-Path $PSScriptRoot 'work\toolchain\go\bin\go.exe'
    if (Test-Path -LiteralPath $localGo) {
        $GoPath = $localGo
    } else {
        $command = Get-Command go -ErrorAction Stop
        $GoPath = $command.Source
    }
}

$goVariables = @('GOPATH', 'GOCACHE', 'GOENV', 'GOTOOLCHAIN', 'CGO_ENABLED', 'GOOS', 'GOARCH')
$savedGoEnvironment = @{}
foreach ($name in $goVariables) {
    $savedGoEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, 'Process')
}
Push-Location $PSScriptRoot
try {
    # A nested module keeps the downloaded toolchain's own test sources out
    # of this project's ./... package pattern.
    New-Item -ItemType Directory -Force -Path (Join-Path $PSScriptRoot 'work') | Out-Null
    $workModule = Join-Path $PSScriptRoot 'work\go.mod'
    if (!(Test-Path -LiteralPath $workModule)) {
        Set-Content -LiteralPath $workModule -Value "module nju-connect.local/build-workspace`n`ngo 1.25.6`n" -Encoding Ascii
    }
    $env:GOPATH = Join-Path $PSScriptRoot 'work\gopath'
    $env:GOCACHE = Join-Path $PSScriptRoot 'work\gocache'
    $env:GOENV = 'off'
    $env:GOTOOLCHAIN = 'local'
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    if ($Test) {
        & $GoPath test ./...
        if ($LASTEXITCODE -ne 0) { throw 'Go tests failed.' }
    }
    if (![IO.Path]::IsPathRooted($OutputPath)) { $OutputPath = Join-Path $PSScriptRoot $OutputPath }
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $OutputPath) | Out-Null
    $commit = 'source-archive'
    if ((Test-Path -LiteralPath (Join-Path $PSScriptRoot '.git')) -and (Get-Command git -ErrorAction SilentlyContinue)) {
        $commit = (& git rev-parse --short HEAD)
        if ($LASTEXITCODE -ne 0) { throw 'Cannot determine source commit.' }
    }
    if ($Version -notmatch '^[A-Za-z0-9._-]+$') { throw 'Invalid build version.' }
    & $GoPath build -trimpath -ldflags "-s -w -X main.defaultProfile=nju -X main.zjuConnectVersion=$Version -X main.CommitID=$commit" -o $OutputPath .
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed.' }
    Get-FileHash -LiteralPath $OutputPath -Algorithm SHA256
} finally {
    Pop-Location
    foreach ($name in $goVariables) {
        [Environment]::SetEnvironmentVariable($name, $savedGoEnvironment[$name], 'Process')
    }
}
