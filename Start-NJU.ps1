[CmdletBinding()]
param(
    [ValidateSet('LDAP', 'Browser')][string]$LoginMode = 'LDAP',
    [string]$Username,
    [ValidateRange(1, 65535)][int]$SocksPort = 11080,
    [ValidateRange(1, 65535)][int]$HttpPort = 11081,
    [ValidateRange(1, 65535)][int]$SshPort = 2222,
    [string]$Target = '',
    [string]$CheckTarget = '',
    [string]$LoginDomain = 'auto',
    [string]$BindInterface = '',
    [switch]$RememberSession,
    [switch]$AuthInfo
)

$ErrorActionPreference = 'Stop'
$binary = Join-Path $PSScriptRoot 'dist\nju-connect.exe'
if (!(Test-Path -LiteralPath $binary)) {
    throw 'Build dist\nju-connect.exe first: .\Build-NJU.ps1'
}

$clientArgs = @(
    '--profile=nju', '--config', (Join-Path $PSScriptRoot 'nju.toml'),
    '--login-domain', $LoginDomain,
    '--socks-bind', "127.0.0.1:$SocksPort",
    '--http-bind', "127.0.0.1:$HttpPort",
    '--clash-rules-file', (Join-Path $PSScriptRoot 'clash-nju.generated.yaml'),
    "--check-target=$CheckTarget",
    '--tcp-tunnel-mode=true', '--tun-mode=false', '--add-route=false',
    '--dns-hijack=false', '--fake-ip=false'
)
if ($Target) {
    $clientArgs += @('--tcp-port-forwarding', "127.0.0.1:$SshPort-$Target")
} else {
    $clientArgs += '--tcp-port-forwarding='
}
if ($BindInterface) { $clientArgs += @('--bind-interface', $BindInterface) }
if ($AuthInfo) {
    & $binary @clientArgs --auth-info
    if ($LASTEXITCODE -ne 0) { throw "Authentication discovery failed ($LASTEXITCODE)" }
    return
}
$listenPorts = @($SocksPort, $HttpPort)
if ($Target) { $listenPorts += $SshPort }
if (@($listenPorts | Select-Object -Unique).Count -ne $listenPorts.Count) {
    throw 'Enabled local listener ports must be distinct.'
}
# Read-only: never stop a listener or reconfigure another application.
if (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue) {
    $busy = @(Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
        Where-Object { $_.LocalPort -in $listenPorts })
    if ($busy.Count) {
        $busyPorts = ($busy.LocalPort | Sort-Object -Unique) -join ', '
        throw "Port(s) $busyPorts already occupied. Choose different script port parameters."
    }
}

if ($RememberSession) {
    $clientArgs += @('--client-data-file', (Join-Path $PSScriptRoot 'nju-client-data.json'))
} else {
    $clientArgs += '--client-data-file='
}

$previousPassword = [Environment]::GetEnvironmentVariable('ZJU_CONNECT_PASSWORD', 'Process')
$exitCode = 0
try {
    if ($LoginMode -eq 'LDAP') {
        if (!$Username) { $Username = Read-Host 'NJU account' }
        if (!$Username) { throw 'Account must not be empty.' }
        $securePassword = Read-Host 'NJU password (hidden)' -AsSecureString
        $credential = New-Object System.Net.NetworkCredential('', $securePassword)
        [Environment]::SetEnvironmentVariable('ZJU_CONNECT_PASSWORD', $credential.Password, 'Process')
        $clientArgs += @('--auth-type=auth/psw', '--username', $Username)
    } else {
        $clientArgs += '--auth-type=auth/httpsOauth2'
    }
    Write-Host "Campus proxy: SOCKS 127.0.0.1:$SocksPort | HTTP 127.0.0.1:$HttpPort"
    if ($Target) { Write-Host "Optional SSH forwarding: 127.0.0.1:$SshPort -> $Target" }
    Write-Host 'After login, merge clash-nju.generated.yaml into your existing Clash rules.'
    Write-Host 'System proxy, existing TUN and system routes will not be changed. Ctrl+C stops this client.'
    & $binary @clientArgs
    $exitCode = $LASTEXITCODE
} finally {
    [Environment]::SetEnvironmentVariable('ZJU_CONNECT_PASSWORD', $previousPassword, 'Process')
    $credential = $null
    $securePassword = $null
}
if ($exitCode -ne 0) { throw "NJU client exited with code $exitCode" }
