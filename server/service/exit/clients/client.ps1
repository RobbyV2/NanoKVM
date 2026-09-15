# nexit/1 powershell client
#
# NanoKVM exit client for Windows PowerShell 5.1 on .NET Framework 4.5+ (Windows 8 /
# Server 2012 or newer: ClientWebSocket needs the OS WebSocket component). Holds one
# WebSocket open to the NanoKVM and originates every TCP connection and UDP datagram
# the consumer behind it asks for. Served fully templated from
# GET /exit/<slot>/client.ps1 and run with `irm ... | iex`; takes no arguments.
#
# Protocol: docs/superpowers/specs/2026-09-15-exit-tunnel-design.md, section
# "The nexit/1 protocol (Mode A)". Frames are type:u8 stream:u32be payload, exactly
# one per WebSocket binary message, at most 16 KiB of payload. HELLO's hostname and
# os strings carry a one-byte length prefix.
#
# Architecture (spec "Client architecture"): one outstanding ReceiveAsync for the
# WebSocket plus one ReadAsync/ReceiveAsync per stream, polled with Task.WaitAny
# every 250 ms; every SendAsync awaited synchronously; ReceiveAsync looped on
# EndOfMessage; a compiled (Add-Type) fingerprint-pinning callback installed on
# ServicePointManager, which is what ClientWebSocket uses on .NET Framework.
#
# Deviation from the Perl/Python clients, by necessity: ClientWebSocket hides
# ping/pong control frames, so this client cannot apply the "no frame for 75 s"
# rule literally. It sends keepalives every 30 s through Options.KeepAliveInterval,
# relies on the NanoKVM's own 20 s ping / 3 misses for liveness, and treats a
# faulted ReceiveAsync or a non-Open socket state as the dead-peer signal.

$NexitScheme = '__SCHEME__'
$NexitHost = '__HOST__'
$NexitSlot = '__SLOT__'
$NexitToken = '__TOKEN__'
$NexitFingerprint = '__FINGERPRINT__'
$NexitAllowPrivate = ('__ALLOW_PRIVATE__' -eq '1')

$T_HELLO = 0x01; $T_WELCOME = 0x02
$T_OPEN = 0x10; $T_OPENED = 0x11; $T_OPEN_FAIL = 0x12
$T_DATA = 0x20; $T_EOF = 0x21; $T_RST = 0x22; $T_WINDOW = 0x30
$R_GENERAL = 1; $R_NOT_ALLOWED = 2; $R_NET_UNREACH = 3; $R_HOST_UNREACH = 4
$R_REFUSED = 5; $R_TIMEOUT = 6; $R_CMD_UNSUPP = 7; $R_ATYP_UNSUPP = 8
$MaxPayload = 16384
$WsReadLimit = $MaxPayload + 64
$ConnectBudgetSeconds = 6
$CT = [System.Threading.CancellationToken]::None

function Write-NexitLog([string]$Message) {
    Write-Host ("{0} nexit: {1}" -f (Get-Date -Format 'HH:mm:ss'), $Message)
}

# --- destination policy (D21, mirrors server/service/exit/policy.go) ---------------

function New-Prefix([string]$Cidr) {
    $parts = $Cidr.Split('/')
    return @{ Net = [System.Net.IPAddress]::Parse($parts[0]).GetAddressBytes(); Bits = [int]$parts[1] }
}
$Always4 = @(@('0.0.0.0/8', '127.0.0.0/8', '169.254.0.0/16', '198.18.0.0/15', '224.0.0.0/3') | ForEach-Object { New-Prefix $_ })
$Private4 = @(@('10.0.0.0/8', '100.64.0.0/10', '172.16.0.0/12', '192.168.0.0/16') | ForEach-Object { New-Prefix $_ })
$Always6 = @(@('::/128', '::1/128', 'fe80::/10', 'ff00::/8') | ForEach-Object { New-Prefix $_ })
$Private6 = @(@('fc00::/7') | ForEach-Object { New-Prefix $_ })

function Test-InPrefix([byte[]]$Addr, $Prefix) {
    $full = [int][Math]::Floor($Prefix.Bits / 8)
    for ($i = 0; $i -lt $full; $i++) {
        if ($Addr[$i] -ne $Prefix.Net[$i]) { return $false }
    }
    $rest = $Prefix.Bits % 8
    if ($rest -eq 0) { return $true }
    $mask = (0xff -shl (8 - $rest)) -band 0xff
    return (([int]$Addr[$full] -band $mask) -eq ([int]$Prefix.Net[$full] -band $mask))
}

function Test-PolicyAllows([System.Net.IPAddress]$Ip) {
    if ($Ip.AddressFamily -eq [System.Net.Sockets.AddressFamily]::InterNetworkV6 -and $Ip.IsIPv4MappedToIPv6) {
        $Ip = $Ip.MapToIPv4()
    }
    $bytes = $Ip.GetAddressBytes()
    if ($bytes.Length -eq 4) { $always = $Always4; $private = $Private4 } else { $always = $Always6; $private = $Private6 }
    foreach ($p in $always) { if (Test-InPrefix $bytes $p) { return $false } }
    if ($NexitAllowPrivate) { return $true }
    foreach ($p in $private) { if (Test-InPrefix $bytes $p) { return $false } }
    return $true
}

# --- byte helpers -------------------------------------------------------------------

function Get-U32BE([byte[]]$B, [int]$O) {
    return [long]((([long]$B[$O]) -shl 24) -bor (([long]$B[$O + 1]) -shl 16) -bor (([long]$B[$O + 2]) -shl 8) -bor ([long]$B[$O + 3]))
}
function Get-U16BE([byte[]]$B, [int]$O) {
    return [int]((([int]$B[$O]) -shl 8) -bor ([int]$B[$O + 1]))
}
function Set-U32BE([byte[]]$B, [int]$O, [long]$V) {
    $B[$O] = [byte](($V -shr 24) -band 0xff)
    $B[$O + 1] = [byte](($V -shr 16) -band 0xff)
    $B[$O + 2] = [byte](($V -shr 8) -band 0xff)
    $B[$O + 3] = [byte]($V -band 0xff)
}
function Get-Slice([byte[]]$B, [int]$Off, [int]$Len) {
    $o = New-Object byte[] $Len
    if ($Len -gt 0) { [System.Buffer]::BlockCopy($B, $Off, $o, 0, $Len) }
    return , $o
}
function Join-Bytes([byte[]]$A, [byte[]]$B) {
    $o = New-Object byte[] ($A.Length + $B.Length)
    if ($A.Length -gt 0) { [System.Buffer]::BlockCopy($A, 0, $o, 0, $A.Length) }
    if ($B.Length -gt 0) { [System.Buffer]::BlockCopy($B, 0, $o, $A.Length, $B.Length) }
    return , $o
}

# atyp, addr, port at $Payload[$Off]; returns @{Ip; Port; Next} or $null
function Read-Addr([byte[]]$Payload, [int]$Off) {
    if ($Payload.Length -lt $Off + 1) { return $null }
    $atyp = $Payload[$Off]
    if ($atyp -eq 1) { $len = 4 } elseif ($atyp -eq 4) { $len = 16 } else { return $null }
    if ($Payload.Length -lt $Off + 1 + $len + 2) { return $null }
    $ip = New-Object System.Net.IPAddress -ArgumentList @(, [byte[]](Get-Slice $Payload ($Off + 1) $len))
    return @{ Ip = $ip; Port = (Get-U16BE $Payload ($Off + 1 + $len)); Next = ($Off + 3 + $len) }
}

function Get-AddrBytes([System.Net.IPAddress]$Ip, [int]$Port) {
    if ($Ip.AddressFamily -eq [System.Net.Sockets.AddressFamily]::InterNetworkV6 -and $Ip.IsIPv4MappedToIPv6) {
        $Ip = $Ip.MapToIPv4()
    }
    $ab = $Ip.GetAddressBytes()
    $o = New-Object byte[] (3 + $ab.Length)
    if ($ab.Length -eq 4) { $o[0] = 1 } else { $o[0] = 4 }
    [System.Buffer]::BlockCopy($ab, 0, $o, 1, $ab.Length)
    $o[1 + $ab.Length] = [byte](($Port -shr 8) -band 0xff)
    $o[2 + $ab.Length] = [byte]($Port -band 0xff)
    return , $o
}

function New-NexitFrame([byte]$Type, [long]$Sid, [byte[]]$Payload) {
    $n = 0
    if ($null -ne $Payload) { $n = $Payload.Length }
    $f = New-Object byte[] (5 + $n)
    $f[0] = $Type
    Set-U32BE $f 1 $Sid
    if ($n -gt 0) { [System.Buffer]::BlockCopy($Payload, 0, $f, 5, $n) }
    return , $f
}

function Get-SocketReason($Exception) {
    $e = $Exception
    if ($e -is [System.AggregateException]) { $e = $e.Flatten().InnerExceptions[0] }
    while ($null -ne $e -and -not ($e -is [System.Net.Sockets.SocketException]) -and $null -ne $e.InnerException) { $e = $e.InnerException }
    if (-not ($e -is [System.Net.Sockets.SocketException])) { return $R_GENERAL }
    switch ($e.SocketErrorCode) {
        'ConnectionRefused' { return $R_REFUSED }
        'HostUnreachable' { return $R_HOST_UNREACH }
        'NetworkUnreachable' { return $R_NET_UNREACH }
        'TimedOut' { return $R_TIMEOUT }
        'AddressFamilyNotSupported' { return $R_ATYP_UNSUPP }
        'AddressNotAvailable' { return $R_ATYP_UNSUPP }
        default { return $R_GENERAL }
    }
}

function Test-DefaultRoute {
    try {
        foreach ($nic in [System.Net.NetworkInformation.NetworkInterface]::GetAllNetworkInterfaces()) {
            if ($nic.OperationalStatus -ne [System.Net.NetworkInformation.OperationalStatus]::Up) { continue }
            foreach ($gw in $nic.GetIPProperties().GatewayAddresses) {
                if ($gw.Address.AddressFamily -eq [System.Net.Sockets.AddressFamily]::InterNetwork -and $gw.Address.ToString() -ne '0.0.0.0') { return $true }
            }
        }
        return $false
    } catch { return $true }
}

# --- certificate pinning (compiled: script-block delegates cannot run on the TLS thread) --

$NexitPinSource = @'
using System;
using System.Net;
using System.Net.Security;
using System.Security.Cryptography;
using System.Security.Cryptography.X509Certificates;

public static class NexitPin
{
    public static string Expected = "";
    public static string LastSeen = "";
    public static int Calls = 0;

    public static bool Validate(object sender, X509Certificate cert, X509Chain chain, SslPolicyErrors errors)
    {
        Calls++;
        if (cert == null) { LastSeen = ""; return false; }
        using (SHA256CryptoServiceProvider sha = new SHA256CryptoServiceProvider())
        {
            LastSeen = BitConverter.ToString(sha.ComputeHash(cert.GetRawCertData())).Replace("-", "").ToLowerInvariant();
        }
        return String.Equals(LastSeen, Expected, StringComparison.OrdinalIgnoreCase);
    }

    public static void Install(string expected)
    {
        Expected = expected;
        LastSeen = "";
        Calls = 0;
        ServicePointManager.ServerCertificateValidationCallback = new RemoteCertificateValidationCallback(Validate);
    }
}
'@

function Install-Pin([string]$Fingerprint) {
    if ($null -eq ('NexitPin' -as [type])) {
        Add-Type -TypeDefinition $NexitPinSource -ErrorAction Stop
    }
    [NexitPin]::Install($Fingerprint)
}

# --- session state --------------------------------------------------------------------

$script:Ws = $null
$script:RecvTask = $null
$script:RecvBuf = New-Object byte[] 32768
$script:RecvSeg = New-Object System.ArraySegment[byte] -ArgumentList @(, $script:RecvBuf)
$script:MsgStream = New-Object System.IO.MemoryStream
$script:Streams = @{}
$script:StreamWindow = 0
$script:ConnWindow = 0
$script:ConnCredit = 0
$script:ConnUnacked = 0
$script:Welcomed = $false

function Send-WsMessage([byte[]]$Bytes) {
    $seg = New-Object System.ArraySegment[byte] -ArgumentList @(, $Bytes)
    try {
        $script:Ws.SendAsync($seg, [System.Net.WebSockets.WebSocketMessageType]::Binary, $true, $CT).Wait()
    } catch {
        throw ("SESSION_END websocket send failed: " + $_.Exception.GetBaseException().Message)
    }
}

function Send-Frame([byte]$Type, [long]$Sid, [byte[]]$Payload) {
    Send-WsMessage (New-NexitFrame $Type $Sid $Payload)
}

function Send-Reason([byte]$Type, [long]$Sid, [int]$Reason) {
    Send-Frame $Type $Sid ([byte[]]@([byte]$Reason))
}

function Send-Window([long]$Sid, [long]$Increment) {
    $p = New-Object byte[] 4
    Set-U32BE $p 0 $Increment
    Send-Frame $T_WINDOW $Sid $p
}

function Release-Stream($St) {
    $script:Streams.Remove($St.Key)
    if ($null -ne $St.Client) { try { $St.Client.Close() } catch { } }
    if ($null -ne $St.Udp) { try { $St.Udp.Close() } catch { } }
    $St.Client = $null; $St.Udp = $null; $St.Stream = $null
    $St.ReadTask = $null; $St.ConnectTask = $null; $St.UdpTask = $null
    if ($St.Reserved -gt 0) {
        # a read that never completed had credit reserved; hand it back
        $script:ConnCredit += $St.Reserved
        $St.Reserved = 0
    }
}

function Fail-Open($St, [int]$Reason) {
    Send-Reason $T_OPEN_FAIL $St.Sid $Reason
    Release-Stream $St
}

function Reset-Stream($St, [int]$Reason) {
    Send-Reason $T_RST $St.Sid $Reason
    Release-Stream $St
}

function Add-Consumed($St, [long]$N) {
    $script:ConnUnacked += $N
    if ($script:ConnUnacked -ge [Math]::Floor($script:ConnWindow / 2)) {
        Send-Window 0 $script:ConnUnacked
        $script:ConnUnacked = 0
    }
    $St.RxUnacked += $N
    if ($St.RxUnacked -ge [Math]::Floor($script:StreamWindow / 2)) {
        Send-Window $St.Sid $St.RxUnacked
        $St.RxUnacked = 0
    }
}

function Start-StreamRead($St) {
    # Issue one ReadAsync when the stream is open, still readable and has credit.
    if ($St.Proto -ne 1 -or $St.State -ne 'open' -or $St.ReadEof -or $null -ne $St.ReadTask) { return }
    $budget = [Math]::Min([long]$MaxPayload, [Math]::Min([long]$St.TxCredit, [long]$script:ConnCredit))
    if ($budget -le 0) { return }
    $St.TxCredit -= $budget
    $script:ConnCredit -= $budget
    $St.Reserved = $budget
    try {
        $St.ReadTask = $St.Stream.ReadAsync($St.ReadBuf, 0, [int]$budget)
    } catch {
        Reset-Stream $St $R_GENERAL
    }
}

function Complete-StreamRead($St, $Task) {
    $St.ReadTask = $null
    if ($Task.IsFaulted -or $Task.IsCanceled) {
        Reset-Stream $St $R_GENERAL
        return
    }
    $n = [int]$Task.Result
    $unused = $St.Reserved - $n
    $St.Reserved = 0
    $St.TxCredit += $unused
    $script:ConnCredit += $unused
    if ($n -eq 0) {
        $St.ReadEof = $true
        Send-Frame $T_EOF $St.Sid $null
        if ($St.Shut) { Release-Stream $St }
        return
    }
    Send-Frame $T_DATA $St.Sid (Get-Slice $St.ReadBuf 0 $n)
}

function Complete-Connect($St, $Task) {
    $St.ConnectTask = $null
    if ($Task.IsFaulted -or $Task.IsCanceled) {
        $reason = $R_GENERAL
        if ($Task.IsFaulted) { $reason = Get-SocketReason $Task.Exception }
        Fail-Open $St $reason
        return
    }
    try {
        $St.Stream = $St.Client.GetStream()
        $St.Stream.WriteTimeout = 30000
        $ep = [System.Net.IPEndPoint]$St.Client.Client.LocalEndPoint
    } catch {
        Fail-Open $St $R_GENERAL
        return
    }
    $St.State = 'open'
    Send-Frame $T_OPENED $St.Sid (Get-AddrBytes $ep.Address $ep.Port)
}

function Start-UdpReceive($St) {
    if ($null -ne $St.UdpTask -or $null -eq $St.Udp) { return }
    try { $St.UdpTask = $St.Udp.ReceiveAsync() } catch { Reset-Stream $St $R_GENERAL }
}

function Complete-UdpReceive($St, $Task) {
    $St.UdpTask = $null
    if ($Task.IsFaulted -or $Task.IsCanceled) {
        # Windows reports ICMP port-unreachable for an earlier send as a failed receive; keep the stream.
        $St.UdpErrors++
        if ($St.UdpErrors -gt 16) { Reset-Stream $St $R_GENERAL }
        return
    }
    $r = $Task.Result
    $head = Get-AddrBytes $r.RemoteEndPoint.Address $r.RemoteEndPoint.Port
    if ($head.Length + $r.Buffer.Length -gt $MaxPayload) { return }
    Send-Frame $T_DATA $St.Sid (Join-Bytes $head $r.Buffer)
}

function Invoke-Open([long]$Sid, [byte[]]$Payload) {
    if ($script:Streams.ContainsKey("$Sid") -or $Sid -eq 0 -or ($Sid % 2) -eq 0) { throw "SESSION_END OPEN on live or invalid stream id $Sid" }
    if ($Payload.Length -lt 1) { throw 'SESSION_END short OPEN' }
    $proto = [int]$Payload[0]
    $st = @{
        Sid = $Sid; Key = "$Sid"; Proto = $proto; State = 'connecting'; Deadline = [DateTime]::UtcNow.AddSeconds($ConnectBudgetSeconds)
        Client = $null; Stream = $null; Udp = $null; ReadTask = $null; ConnectTask = $null; UdpTask = $null
        ReadBuf = $null; TxCredit = [long]$script:StreamWindow; RxUnacked = [long]0; Reserved = [long]0
        PeerEof = $false; Shut = $false; ReadEof = $false; UdpErrors = 0
    }
    $script:Streams[$st.Key] = $st
    if ($proto -eq 1) {
        $a = Read-Addr $Payload 1
        if ($null -eq $a) { Fail-Open $st $R_ATYP_UNSUPP; return }
        if (-not (Test-PolicyAllows $a.Ip)) { Fail-Open $st $R_NOT_ALLOWED; return }
        try {
            $st.Client = New-Object System.Net.Sockets.TcpClient -ArgumentList @($a.Ip.AddressFamily)
            $st.Client.NoDelay = $true
            $st.ReadBuf = New-Object byte[] $MaxPayload
            $st.ConnectTask = $st.Client.ConnectAsync($a.Ip, [int]$a.Port)
        } catch {
            Fail-Open $st (Get-SocketReason $_.Exception)
        }
    } elseif ($proto -eq 2) {
        try {
            $st.Udp = New-Object System.Net.Sockets.UdpClient -ArgumentList @(, (New-Object System.Net.IPEndPoint -ArgumentList @([System.Net.IPAddress]::Any, 0)))
        } catch {
            Fail-Open $st $R_GENERAL
            return
        }
        $st.State = 'open'
        Send-Frame $T_OPENED $Sid $null
        Start-UdpReceive $st
    } else {
        Fail-Open $st $R_CMD_UNSUPP
    }
}

function Invoke-UdpOut($St, [byte[]]$Payload) {
    $a = Read-Addr $Payload 0
    if ($null -eq $a) { return }
    if ($a.Ip.AddressFamily -ne [System.Net.Sockets.AddressFamily]::InterNetwork) { return }
    if (-not (Test-PolicyAllows $a.Ip)) { return }
    $data = Get-Slice $Payload $a.Next ($Payload.Length - $a.Next)
    try {
        $ep = New-Object System.Net.IPEndPoint -ArgumentList @($a.Ip, [int]$a.Port)
        [void]$St.Udp.Send($data, $data.Length, $ep)
    } catch { }
}

function Invoke-Message([byte[]]$Msg) {
    if ($Msg.Length -lt 5) { throw 'SESSION_END short frame' }
    $type = [int]$Msg[0]
    $sid = Get-U32BE $Msg 1
    $payload = Get-Slice $Msg 5 ($Msg.Length - 5)
    if (-not $script:Welcomed) {
        if ($type -ne $T_WELCOME -or $payload.Length -lt 11) { throw ("SESSION_END expected WELCOME, got 0x{0:x2}" -f $type) }
        if ($payload[0] -ne 1) { throw ("SESSION_END unsupported version " + $payload[0]) }
        $script:StreamWindow = Get-U32BE $payload 1
        $script:ConnWindow = Get-U32BE $payload 5
        $maxStreams = Get-U16BE $payload 9
        $script:ConnCredit = $script:ConnWindow
        $script:Welcomed = $true
        Write-NexitLog ("connected: streamWindow={0} connWindow={1} maxStreams={2}" -f $script:StreamWindow, $script:ConnWindow, $maxStreams)
        return
    }
    if ($type -eq $T_OPEN) { Invoke-Open $sid $payload; return }
    if ($type -eq $T_WINDOW) {
        if ($payload.Length -lt 4) { throw 'SESSION_END short WINDOW' }
        $inc = Get-U32BE $payload 0
        if ($sid -eq 0) { $script:ConnCredit += $inc }
        elseif ($script:Streams.ContainsKey("$sid")) { $script:Streams["$sid"].TxCredit += $inc }
        return
    }
    if (-not $script:Streams.ContainsKey("$sid")) { return }   # released or unknown: ignored by design
    $st = $script:Streams["$sid"]
    if ($type -eq $T_DATA) {
        if ($st.State -ne 'open') { throw "SESSION_END DATA on stream $sid before OPENED" }
        if ($st.Proto -eq 2) { Invoke-UdpOut $st $payload; return }
        if ($st.PeerEof) { Reset-Stream $st $R_GENERAL; return }
        try {
            $st.Stream.Write($payload, 0, $payload.Length)
        } catch {
            Reset-Stream $st $R_GENERAL
            return
        }
        Add-Consumed $st $payload.Length
    } elseif ($type -eq $T_EOF) {
        if ($st.Proto -eq 2 -or $st.State -ne 'open' -or $st.PeerEof) { Reset-Stream $st $R_GENERAL; return }
        $st.PeerEof = $true
        $st.Shut = $true
        try { $st.Client.Client.Shutdown([System.Net.Sockets.SocketShutdown]::Send) } catch { }
        if ($st.ReadEof) { Release-Stream $st }
    } elseif ($type -eq $T_RST) {
        Release-Stream $st
    } else {
        throw ("SESSION_END unexpected frame 0x{0:x2} on stream {1}" -f $type, $sid)
    }
}

function Start-Receive {
    $script:RecvTask = $script:Ws.ReceiveAsync($script:RecvSeg, $CT)
}

function Complete-Receive($Task) {
    $script:RecvTask = $null
    if ($Task.IsFaulted) { throw ("SESSION_END websocket receive failed: " + $Task.Exception.GetBaseException().Message) }
    if ($Task.IsCanceled) { throw 'SESSION_END websocket receive cancelled' }
    $r = $Task.Result
    if ($r.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
        throw ("SESSION_END websocket closed by kvm ({0} {1})" -f $script:Ws.CloseStatus, $script:Ws.CloseStatusDescription)
    }
    if ($r.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Text) { throw 'SESSION_END text message is a protocol error' }
    if ($r.Count -gt 0) { $script:MsgStream.Write($script:RecvBuf, 0, $r.Count) }
    if ($script:MsgStream.Length -gt $WsReadLimit) { throw 'SESSION_END message over read limit' }
    if ($r.EndOfMessage) {
        $msg = $script:MsgStream.ToArray()
        $script:MsgStream.SetLength(0)
        Invoke-Message $msg
    }
    Start-Receive
}

function Get-HelloPayload {
    $hostName = ''
    try { $hostName = [System.Net.Dns]::GetHostName() } catch { $hostName = $env:COMPUTERNAME }
    if ([string]::IsNullOrEmpty($hostName)) { $hostName = 'unknown' }
    $hb = [System.Text.Encoding]::UTF8.GetBytes($hostName)
    if ($hb.Length -gt 255) { $hb = Get-Slice $hb 0 255 }
    $ob = [System.Text.Encoding]::UTF8.GetBytes('windows')
    $flags = 0
    if (Test-DefaultRoute) { $flags = 1 }
    $p = New-Object byte[] (4 + $hb.Length + $ob.Length)
    $p[0] = 1; $p[1] = [byte]$flags; $p[2] = [byte]$hb.Length
    [System.Buffer]::BlockCopy($hb, 0, $p, 3, $hb.Length)
    $p[3 + $hb.Length] = [byte]$ob.Length
    [System.Buffer]::BlockCopy($ob, 0, $p, 4 + $hb.Length, $ob.Length)
    return , $p
}

function Connect-Kvm {
    $scheme = $NexitScheme
    if ($scheme -eq 'http') { $scheme = 'ws' } elseif ($scheme -eq 'https') { $scheme = 'wss' }
    if ($scheme -ne 'ws' -and $scheme -ne 'wss') { throw "bad scheme '$NexitScheme'" }
    $tls = ($scheme -eq 'wss')
    $fp = ($NexitFingerprint -replace '[^0-9a-fA-F]', '').ToLowerInvariant()
    if ($tls -and [string]::IsNullOrEmpty($fp)) { throw 'wss without a certificate fingerprint: refusing to connect unverified' }
    $uri = [System.Uri]("{0}://{1}/exit/{2}/native" -f $scheme, $NexitHost, $NexitSlot)
    if ($tls) {
        try { [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]'Tls12,Tls13' }
        catch { [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.SecurityProtocolType]::Tls12 }
        Install-Pin $fp
        # A TLS connection left open by the `irm` fetch was validated with the permissive
        # callback; make sure the WebSocket cannot reuse it.
        try {
            $httpUri = [System.Uri]("https://{0}/" -f $NexitHost)
            $sp = [System.Net.ServicePointManager]::FindServicePoint($httpUri)
            [void]$sp.CloseConnectionGroup('')
        } catch { }
    }
    $ws = New-Object System.Net.WebSockets.ClientWebSocket
    $ws.Options.SetRequestHeader('Authorization', "Bearer $NexitToken")
    $ws.Options.KeepAliveInterval = [TimeSpan]::FromSeconds(30)
    $t = $ws.ConnectAsync($uri, $CT)
    try {
        $t.Wait()
    } catch {
        $base = $_.Exception.GetBaseException().Message
        try { $ws.Dispose() } catch { }
        if ($tls -and [NexitPin]::Calls -gt 0 -and [NexitPin]::LastSeen -ne $fp) {
            throw ("certificate fingerprint mismatch: expected {0} got {1}; refusing" -f $fp, [NexitPin]::LastSeen)
        }
        if ($base -match '404') { throw 'rejected (404): wrong token or slot, or this source is rate limited' }
        throw "connect failed: $base"
    }
    if ($tls) {
        if ([NexitPin]::Calls -eq 0) {
            try { $ws.Abort(); $ws.Dispose() } catch { }
            throw 'the certificate was never presented for pinning (connection reused); refusing'
        }
        if ([NexitPin]::LastSeen -ne $fp) {
            try { $ws.Abort(); $ws.Dispose() } catch { }
            throw ("certificate fingerprint mismatch: expected {0} got {1}; refusing" -f $fp, [NexitPin]::LastSeen)
        }
    }
    return $ws
}

function Reset-Session {
    $script:RecvTask = $null
    $script:MsgStream.SetLength(0)
    $script:Streams = @{}
    $script:StreamWindow = 0
    $script:ConnWindow = 0
    $script:ConnCredit = 0
    $script:ConnUnacked = 0
    $script:Welcomed = $false
}

function Close-Session([string]$Reason) {
    foreach ($st in @($script:Streams.Values)) { Release-Stream $st }
    if ($null -ne $script:Ws) {
        try {
            if ($script:Ws.State -eq [System.Net.WebSockets.WebSocketState]::Open) {
                if ($Reason.Length -gt 100) { $Reason = $Reason.Substring(0, 100) }
                [void]$script:Ws.CloseOutputAsync([System.Net.WebSockets.WebSocketCloseStatus]::NormalClosure, $Reason, $CT).Wait(2000)
            }
        } catch { }
        try { $script:Ws.Abort() } catch { }
        try { $script:Ws.Dispose() } catch { }
        $script:Ws = $null
    }
}

function Run-Session {
    Send-Frame $T_HELLO 0 (Get-HelloPayload)
    Start-Receive
    while ($true) {
        if ($script:Ws.State -ne [System.Net.WebSockets.WebSocketState]::Open) {
            throw ("SESSION_END websocket state " + $script:Ws.State)
        }
        $now = [DateTime]::UtcNow
        foreach ($st in @($script:Streams.Values)) {
            if ($st.State -eq 'connecting' -and $st.Proto -eq 1 -and $st.Deadline -le $now) { Fail-Open $st $R_TIMEOUT }
        }
        foreach ($st in @($script:Streams.Values)) {
            if ($st.Proto -eq 1) { Start-StreamRead $st } else { Start-UdpReceive $st }
        }
        $tasks = New-Object 'System.Collections.Generic.List[System.Threading.Tasks.Task]'
        $owners = New-Object 'System.Collections.Generic.List[object]'
        $tasks.Add($script:RecvTask); $owners.Add($null)
        foreach ($st in $script:Streams.Values) {
            if ($null -ne $st.ConnectTask) { $tasks.Add($st.ConnectTask); $owners.Add($st) }
            if ($null -ne $st.ReadTask) { $tasks.Add($st.ReadTask); $owners.Add($st) }
            if ($null -ne $st.UdpTask) { $tasks.Add($st.UdpTask); $owners.Add($st) }
        }
        $idx = [System.Threading.Tasks.Task]::WaitAny($tasks.ToArray(), 250)
        if ($idx -lt 0) { continue }
        for ($i = 0; $i -lt $tasks.Count; $i++) {
            $t = $tasks[$i]
            if (-not $t.IsCompleted) { continue }
            $st = $owners[$i]
            if ($null -eq $st) {
                if ([object]::ReferenceEquals($t, $script:RecvTask)) { Complete-Receive $t }
                continue
            }
            if (-not $script:Streams.ContainsKey($st.Key)) { continue }
            if ([object]::ReferenceEquals($t, $st.ConnectTask)) { Complete-Connect $st $t }
            elseif ([object]::ReferenceEquals($t, $st.ReadTask)) { Complete-StreamRead $st $t }
            elseif ([object]::ReferenceEquals($t, $st.UdpTask)) { Complete-UdpReceive $st $t }
        }
    }
}

function Start-NexitClient {
    if ($NexitHost.StartsWith('__') -or $NexitToken.StartsWith('__')) {
        Write-NexitLog 'this script must be fetched from the NanoKVM, which fills in its parameters'
        return
    }
    $displayScheme = $NexitScheme
    if ($displayScheme -eq 'http') { $displayScheme = 'ws' } elseif ($displayScheme -eq 'https') { $displayScheme = 'wss' }
    $pin = 'none'
    if ($NexitFingerprint.Length -gt 0) { $pin = $NexitFingerprint.Substring(0, [Math]::Min(16, $NexitFingerprint.Length)) + '...' }
    Write-NexitLog ("exit client for {0}://{1}/exit/{2} (allowPrivate={3}, pin={4})" -f $displayScheme, $NexitHost, $NexitSlot, $NexitAllowPrivate, $pin)
    $savedCallback = [System.Net.ServicePointManager]::ServerCertificateValidationCallback
    $savedEap = $ErrorActionPreference
    $ErrorActionPreference = 'Stop'
    $backoff = 1
    try {
        while ($true) {
            Reset-Session
            try {
                $script:Ws = Connect-Kvm
                try {
                    Run-Session
                } catch {
                    $msg = $_.Exception.Message
                    if ($msg.StartsWith('SESSION_END ')) { $msg = $msg.Substring(12) }
                    if ($script:Welcomed) { $backoff = 1 }
                    Write-NexitLog ("session ended: " + $msg)
                    Close-Session $msg
                }
            } catch {
                Write-NexitLog ("connect failed: " + $_.Exception.Message)
                Close-Session 'connect failed'
            }
            Write-NexitLog ("reconnecting in {0}s" -f $backoff)
            Start-Sleep -Seconds $backoff
            $backoff = [Math]::Min($backoff * 2, 30)
        }
    } finally {
        Close-Session 'client exiting'
        [System.Net.ServicePointManager]::ServerCertificateValidationCallback = $savedCallback
        $ErrorActionPreference = $savedEap
        Write-NexitLog 'stopping'
    }
}

Start-NexitClient
