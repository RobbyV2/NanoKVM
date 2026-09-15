#!/usr/bin/perl
# nexit/1 perl client
#
# NanoKVM exit client: holds one WebSocket open to the NanoKVM and originates
# every TCP connection and UDP datagram the consumer behind it asks for.
# Served fully templated from GET /exit/<slot>/client.pl; takes no arguments.
# Perl 5.18+ core modules only; IO::Socket::SSL is loaded only for wss.
#
# Protocol: docs/superpowers/specs/2026-09-15-exit-tunnel-design.md, section
# "The nexit/1 protocol (Mode A)". Frames are type:u8 stream:u32be payload,
# exactly one per WebSocket binary message, at most 16 KiB of payload.
# HELLO's hostname and os strings carry a one-byte length prefix.
use strict;
use warnings;
use IO::Handle;
use IO::Select;
use IO::Socket;
use Socket qw(AF_INET AF_INET6 PF_INET PF_INET6 SOCK_STREAM SOCK_DGRAM SOL_SOCKET SO_ERROR
    IPPROTO_TCP TCP_NODELAY INADDR_ANY inet_aton inet_ntoa inet_pton inet_ntop
    pack_sockaddr_in unpack_sockaddr_in pack_sockaddr_in6 unpack_sockaddr_in6 sockaddr_family);
use Errno qw(EINPROGRESS EWOULDBLOCK EAGAIN EINTR ECONNREFUSED EHOSTUNREACH ENETUNREACH
    ETIMEDOUT EADDRNOTAVAIL EAFNOSUPPORT);
use Digest::SHA qw(sha1);
use MIME::Base64 qw(encode_base64);
use Sys::Hostname qw(hostname);
use Time::HiRes qw(time sleep);

my $SCHEME        = "__SCHEME__";
my $HOST          = "__HOST__";
my $SLOT          = "__SLOT__";
my $TOKEN         = "__TOKEN__";
my $FINGERPRINT   = "__FINGERPRINT__";
my $ALLOW_PRIVATE = "__ALLOW_PRIVATE__" eq "1";

use constant {
    T_HELLO => 0x01, T_WELCOME => 0x02, T_OPEN => 0x10, T_OPENED => 0x11, T_OPEN_FAIL => 0x12,
    T_DATA => 0x20, T_EOF => 0x21, T_RST => 0x22, T_WINDOW => 0x30,
    R_GENERAL => 1, R_NOT_ALLOWED => 2, R_NET_UNREACH => 3, R_HOST_UNREACH => 4,
    R_REFUSED => 5, R_TIMEOUT => 6, R_CMD_UNSUPP => 7, R_ATYP_UNSUPP => 8,
    MAX_PAYLOAD => 16384, WS_READ_LIMIT => 16384 + 64, CONNECT_BUDGET => 6,
    PING_EVERY => 30, DEAD_AFTER => 75, UDP_QUEUE => 32,
};
my $WS_GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";

my %ERRNO_REASON = (
    ECONNREFUSED() => R_REFUSED, EHOSTUNREACH() => R_HOST_UNREACH, ENETUNREACH() => R_NET_UNREACH,
    ETIMEDOUT() => R_TIMEOUT, EADDRNOTAVAIL() => R_ATYP_UNSUPP, EAFNOSUPPORT() => R_ATYP_UNSUPP,
);

sub log_msg {
    my @t = localtime;
    printf STDERR "%02d:%02d:%02d nexit: %s\n", $t[2], $t[1], $t[0], $_[0];
}

# --- destination policy (D21, mirrors server/service/exit/policy.go) ------------

sub prefix { my ($s, $bits) = @_; my ($a, $b) = split m{/}, $s; return [inet_pton($a =~ /:/ ? AF_INET6 : AF_INET, $a), $b]; }
my @ALWAYS4  = map { prefix($_) } qw(0.0.0.0/8 127.0.0.0/8 169.254.0.0/16 198.18.0.0/15 224.0.0.0/3);
my @PRIVATE4 = map { prefix($_) } qw(10.0.0.0/8 100.64.0.0/10 172.16.0.0/12 192.168.0.0/16);
my @ALWAYS6  = map { prefix($_) } qw(::/128 ::1/128 fe80::/10 ff00::/8);
my @PRIVATE6 = map { prefix($_) } qw(fc00::/7);

sub in_prefix {
    my ($addr, $p) = @_;
    my ($net, $bits) = @$p;
    my $full = int($bits / 8);
    return 0 if substr($addr, 0, $full) ne substr($net, 0, $full);
    my $rest = $bits % 8;
    return 1 unless $rest;
    my $mask = (0xff << (8 - $rest)) & 0xff;
    return ((ord(substr($addr, $full, 1)) & $mask) == (ord(substr($net, $full, 1)) & $mask)) ? 1 : 0;
}

sub policy_allows {
    my ($fam, $addr) = @_;    # packed address
    if ($fam == AF_INET6 && substr($addr, 0, 12) eq ("\0" x 10) . "\xff\xff") {
        ($fam, $addr) = (AF_INET, substr($addr, 12));    # IPv4-mapped is judged as IPv4
    }
    my ($always, $private) = $fam == AF_INET ? (\@ALWAYS4, \@PRIVATE4) : (\@ALWAYS6, \@PRIVATE6);
    for (@$always) { return 0 if in_prefix($addr, $_) }
    return 1 if $ALLOW_PRIVATE;
    for (@$private) { return 0 if in_prefix($addr, $_) }
    return 1;
}

# --- helpers --------------------------------------------------------------------

sub parse_host {
    my ($hp, $def) = @_;
    if ($hp =~ /^\[([^\]]+)\](?::(\d+))?$/) { return ($1, defined $2 ? $2 : $def) }
    if ($hp =~ /^([^:]+):(\d+)$/) { return ($1, $2) }
    return ($hp, $def);
}

# returns (fam, packed addr, port, new offset) or () on error
sub parse_addr {
    my ($p, $off) = @_;
    return () if length($p) < $off + 1;
    my $atyp = ord(substr($p, $off, 1));
    my ($fam, $len) = $atyp == 1 ? (AF_INET, 4) : $atyp == 4 ? (AF_INET6, 16) : return ();
    return () if length($p) < $off + 1 + $len + 2;
    return ($fam, substr($p, $off + 1, $len), unpack("n", substr($p, $off + 1 + $len, 2)), $off + 3 + $len);
}

sub pack_addr {
    my ($fam, $addr, $port) = @_;
    return ($fam == AF_INET ? "\x01" : "\x04") . $addr . pack("n", $port);
}

sub sockname_addr {
    my $s = shift;
    my $sa = getsockname($s) or return (AF_INET, "\0\0\0\0", 0);
    if (sockaddr_family($sa) == AF_INET6) {
        my ($port, $addr) = unpack_sockaddr_in6($sa);
        return (AF_INET, substr($addr, 12), $port) if substr($addr, 0, 12) eq ("\0" x 10) . "\xff\xff";
        return (AF_INET6, $addr, $port);
    }
    my ($port, $addr) = unpack_sockaddr_in($sa);
    return (AF_INET, $addr, $port);
}

sub has_default_route {
    socket(my $u, PF_INET, SOCK_DGRAM, 0) or return 0;
    my $ok = connect($u, pack_sockaddr_in(53, inet_aton("1.1.1.1"))) ? 1 : 0;
    close $u;
    return $ok;
}

sub os_name { my %m = (linux => "linux", darwin => "darwin", MSWin32 => "windows", cygwin => "windows"); return $m{$^O} || $^O }

sub ws_mask {
    my ($payload, $key) = @_;
    my $n = length $payload;
    return "" unless $n;
    return $payload ^ substr($key x (int($n / 4) + 1), 0, $n);
}

sub ws_frame {
    my ($op, $payload) = @_;
    my $n = length $payload;
    my $head = chr(0x80 | $op);
    if    ($n < 126)   { $head .= chr(0x80 | $n) }
    elsif ($n < 65536) { $head .= chr(0x80 | 126) . pack("n", $n) }
    else               { $head .= chr(0x80 | 127) . pack("N2", 0, $n) }
    my $key = pack("C4", map { int rand 256 } 1 .. 4);
    return $head . $key . ws_mask($payload, $key);
}

sub nexit { my ($t, $sid, $p) = @_; return pack("CN", $t, $sid) . (defined $p ? $p : "") }

sub errno_reason { my $e = shift; return exists $ERRNO_REASON{$e} ? $ERRNO_REASON{$e} : R_GENERAL }

sub would_block { return $!{EAGAIN} || $!{EWOULDBLOCK} || $!{EINTR} }

# --- session ---------------------------------------------------------------------
# One package-scoped session at a time; %S is reset for every connection.

my %S;
my $STOP = 0;

sub end_session { die "SESSION_END: $_[0]\n" }

sub send_ws {
    my ($op, $payload, $sid) = @_;
    push @{ $S{out} }, [ $sid || 0, ws_frame($op, $payload) ];
}

sub send_frame { my ($t, $sid, $p) = @_; send_ws(0x2, nexit($t, $sid, $p)) }

sub send_udp_data {
    my ($st, $payload) = @_;
    if ($st->{udp_queued} >= UDP_QUEUE) {
        my $out = $S{out};
        for my $i (($S{out_off} ? 1 : 0) .. $#$out) {
            if ($out->[$i][0] == $st->{sid}) { splice @$out, $i, 1; $st->{udp_queued}--; last }
        }
    }
    $st->{udp_queued}++;
    send_ws(0x2, nexit(T_DATA, $st->{sid}, $payload), $st->{sid});
}

sub flush_ws {
    my $ws = $S{ws};
    while (@{ $S{out} }) {
        my ($sid, $msg) = @{ $S{out}[0] };
        my $n = syswrite($ws, $msg, length($msg) - $S{out_off}, $S{out_off});
        if (!defined $n) {
            return if would_block();
            if ($S{tls} && $IO::Socket::SSL::SSL_ERROR) {
                $S{ws_want_write_for_read} = 0;
                return;
            }
            end_session("websocket write: $!");
        }
        $S{out_off} += $n;
        if ($S{out_off} >= length $msg) {
            shift @{ $S{out} };
            $S{out_off} = 0;
            $S{streams}{$sid}{udp_queued}-- if $sid && $S{streams}{$sid};
        }
    }
}

sub read_ws {
    my $ws = $S{ws};
    while (1) {
        my $n = sysread($ws, my $chunk, 65536);
        if (!defined $n) {
            last if would_block();
            end_session("websocket read: $!");
        }
        end_session("websocket closed by peer") if $n == 0;
        $S{inbuf} .= $chunk;
        last if $n < 65536 && !($S{tls} && $ws->pending);
    }
    parse_ws();
}

sub parse_ws {
    while (1) {
        my $buf = \$S{inbuf};
        return if length($$buf) < 2;
        my ($b0, $b1) = unpack("CC", $$buf);
        my ($fin, $op) = ($b0 & 0x80, $b0 & 0x0f);
        end_session("websocket: reserved bits set") if $b0 & 0x70;
        end_session("websocket: masked frame from server") if $b1 & 0x80;
        my ($ln, $off) = ($b1 & 0x7f, 2);
        if ($ln == 126) {
            return if length($$buf) < 4;
            ($ln, $off) = (unpack("n", substr($$buf, 2, 2)), 4);
        } elsif ($ln == 127) {
            return if length($$buf) < 10;
            my ($hi, $lo) = unpack("NN", substr($$buf, 2, 8));
            end_session("websocket: absurd length") if $hi;
            ($ln, $off) = ($lo, 10);
        }
        end_session("websocket: message over read limit ($ln)") if $ln + length($S{frag}) > WS_READ_LIMIT;
        return if length($$buf) < $off + $ln;
        my $payload = substr($$buf, $off, $ln);
        substr($$buf, 0, $off + $ln) = "";
        $S{last_rx} = time;
        if ($op == 0x8) {
            my $code = length($payload) >= 2 ? unpack("n", $payload) : 1005;
            end_session(sprintf("websocket closed by kvm (%d %s)", $code, substr($payload, 2)));
        }
        if ($op == 0x9) { send_ws(0xA, $payload); next }
        next if $op == 0xA;
        end_session("websocket: text message is a protocol error") if $op == 0x1;
        if ($op == 0x0) {
            end_session("websocket: continuation without start") unless defined $S{frag_op};
            $S{frag} .= $payload;
            next unless $fin;
            ($op, $payload) = ($S{frag_op}, $S{frag});
            ($S{frag_op}, $S{frag}) = (undef, "");
        } elsif ($op == 0x2) {
            end_session("websocket: interleaved data frames") if defined $S{frag_op};
            unless ($fin) { ($S{frag_op}, $S{frag}) = ($op, $payload); next }
        } else {
            end_session("websocket: unknown opcode $op");
        }
        on_message($payload);
    }
}

sub on_message {
    my $msg = shift;
    end_session("short frame") if length($msg) < 5;
    my ($type, $sid) = unpack("CN", $msg);
    my $payload = substr($msg, 5);
    unless ($S{welcomed}) {
        end_session(sprintf("expected WELCOME, got 0x%02x", $type)) if $type != T_WELCOME || length($payload) < 11;
        my ($ver, $sw, $cw, $ms) = unpack("CNNn", $payload);
        end_session("unsupported version $ver") if $ver != 1;
        @S{qw(stream_window conn_window max_streams conn_credit welcomed)} = ($sw, $cw, $ms, $cw, 1);
        log_msg("connected: streamWindow=$sw connWindow=$cw maxStreams=$ms");
        return;
    }
    if ($type == T_OPEN) { on_open($sid, $payload); return }
    if ($type == T_WINDOW) {
        end_session("short WINDOW") if length($payload) < 4;
        my $inc = unpack("N", $payload);
        if ($sid == 0) { $S{conn_credit} += $inc }
        elsif (my $st = $S{streams}{$sid}) { $st->{tx_credit} += $inc }
        return;
    }
    my $st = $S{streams}{$sid} or return;    # released or unknown: ignored
    if ($type == T_DATA) {
        end_session("DATA on stream $sid before OPENED") if $st->{state} ne "open";
        if    ($st->{proto} == 2) { udp_out($st, $payload) }
        elsif ($st->{peer_eof})   { rst($st, R_GENERAL) }
        else                      { $st->{wbuf} .= $payload; drain($st) }
    } elsif ($type == T_EOF) {
        if ($st->{proto} == 2 || $st->{state} ne "open" || $st->{peer_eof}) { rst($st, R_GENERAL) }
        else { $st->{peer_eof} = 1; drain($st) }
    } elsif ($type == T_RST) {
        release($st);
    } else {
        end_session(sprintf("unexpected frame 0x%02x on stream %d", $type, $sid));
    }
}

sub on_open {
    my ($sid, $payload) = @_;
    end_session("OPEN on live or invalid stream id $sid") if $S{streams}{$sid} || $sid == 0 || $sid % 2 == 0;
    end_session("short OPEN") unless length $payload;
    my $proto = ord substr($payload, 0, 1);
    my $st = { sid => $sid, proto => $proto, state => "connecting", tx_credit => $S{stream_window}, rx_unacked => 0,
        wbuf => "", peer_eof => 0, shut => 0, read_eof => 0, udp_queued => 0, deadline => 0 };
    $S{streams}{$sid} = $st;
    if ($proto == 1) {
        my ($fam, $addr, $port) = parse_addr($payload, 1);
        return open_fail($st, R_ATYP_UNSUPP) unless defined $fam;
        return open_fail($st, R_NOT_ALLOWED) unless policy_allows($fam, $addr);
        my $s = IO::Socket->new;
        unless ($s->socket($fam == AF_INET ? PF_INET : PF_INET6, SOCK_STREAM, 0)) {
            return open_fail($st, $!{EAFNOSUPPORT} ? R_ATYP_UNSUPP : R_GENERAL);
        }
        $s->blocking(0);
        setsockopt($s, IPPROTO_TCP, TCP_NODELAY, 1);
        my $sa = $fam == AF_INET ? pack_sockaddr_in($port, $addr) : pack_sockaddr_in6($port, $addr);
        $st->{sock} = $s;
        $st->{deadline} = time + CONNECT_BUDGET;
        if (connect($s, $sa)) { connected($st) }
        elsif (!($!{EINPROGRESS} || $!{EWOULDBLOCK} || $!{EAGAIN})) {
            my $e = 0 + $!;
            open_fail($st, errno_reason($e));
        }
    } elsif ($proto == 2) {
        my $s = IO::Socket->new;
        unless ($s->socket(PF_INET, SOCK_DGRAM, 0) && bind($s, pack_sockaddr_in(0, INADDR_ANY))) {
            return open_fail($st, R_GENERAL);
        }
        $s->blocking(0);
        $st->{sock}  = $s;
        $st->{state} = "open";
        send_frame(T_OPENED, $sid);
    } else {
        open_fail($st, R_CMD_UNSUPP);
    }
}

sub open_fail { my ($st, $r) = @_; send_frame(T_OPEN_FAIL, $st->{sid}, chr $r); release($st) }

sub connected {
    my $st  = shift;
    my $err = $st->{sock}->sockopt(SO_ERROR);
    return open_fail($st, errno_reason($err)) if $err;
    $st->{state} = "open";
    my ($fam, $addr, $port) = sockname_addr($st->{sock});
    send_frame(T_OPENED, $st->{sid}, pack_addr($fam, $addr, $port));
}

sub rst { my ($st, $r) = @_; send_frame(T_RST, $st->{sid}, chr $r); release($st) }

sub release {
    my $st = shift;
    delete $S{streams}{ $st->{sid} };
    close $st->{sock} if $st->{sock};
    if ($st->{proto} == 1 && length $st->{wbuf}) {
        consumed($st, length $st->{wbuf}, 0);    # never reached the socket: return the connection window
        $st->{wbuf} = "";
    }
    if ($st->{udp_queued}) {
        my @keep = $S{out_off} ? (shift @{ $S{out} }) : ();
        push @keep, grep { $_->[0] != $st->{sid} } @{ $S{out} };
        $S{out} = \@keep;
        $st->{udp_queued} = 0;
    }
}

sub consumed {
    my ($st, $n, $stream_too) = @_;
    $S{conn_unacked} += $n;
    if ($S{conn_unacked} >= int($S{conn_window} / 2)) {
        send_frame(T_WINDOW, 0, pack("N", $S{conn_unacked}));
        $S{conn_unacked} = 0;
    }
    if ($stream_too) {
        $st->{rx_unacked} += $n;
        if ($st->{rx_unacked} >= int($S{stream_window} / 2)) {
            send_frame(T_WINDOW, $st->{sid}, pack("N", $st->{rx_unacked}));
            $st->{rx_unacked} = 0;
        }
    }
}

sub drain {
    my $st = shift;
    while (length $st->{wbuf}) {
        my $n = syswrite($st->{sock}, $st->{wbuf}, 65536);
        if (!defined $n) {
            last if would_block();
            return rst($st, errno_reason(0 + $!));
        }
        substr($st->{wbuf}, 0, $n) = "";
        consumed($st, $n, 1);
    }
    if (!length($st->{wbuf}) && $st->{peer_eof} && !$st->{shut}) {
        $st->{shut} = 1;
        shutdown($st->{sock}, 1);
        release($st) if $st->{read_eof};
    }
}

sub read_stream {
    my $st     = shift;
    my $budget = MAX_PAYLOAD;
    $budget = $st->{tx_credit} if $st->{tx_credit} < $budget;
    $budget = $S{conn_credit}  if $S{conn_credit} < $budget;
    return if $budget <= 0;
    my $n = sysread($st->{sock}, my $data, $budget);
    if (!defined $n) {
        return if would_block();
        return rst($st, errno_reason(0 + $!));
    }
    if ($n == 0) {
        $st->{read_eof} = 1;
        send_frame(T_EOF, $st->{sid});
        release($st) if $st->{shut};
        return;
    }
    $st->{tx_credit} -= $n;
    $S{conn_credit}  -= $n;
    send_frame(T_DATA, $st->{sid}, $data);
}

sub udp_out {
    my ($st, $payload) = @_;
    my ($fam, $addr, $port, $off) = parse_addr($payload, 0);
    return unless defined $fam && $fam == AF_INET && policy_allows($fam, $addr);
    send($st->{sock}, substr($payload, $off), 0, pack_sockaddr_in($port, $addr));
}

sub udp_in {
    my $st = shift;
    my $sa = recv($st->{sock}, my $data, 65535, 0);
    unless (defined $sa) {
        return if would_block();
        return rst($st, errno_reason(0 + $!));
    }
    my ($port, $addr) = unpack_sockaddr_in($sa);
    my $head = pack_addr(AF_INET, $addr, $port);
    return if length($head) + length($data) > MAX_PAYLOAD;
    send_udp_data($st, $head . $data);
}

sub hello_payload {
    my $host = substr(hostname() || "unknown", 0, 255);
    my $os   = substr(os_name(), 0, 255);
    return pack("CCC", 1, has_default_route() ? 1 : 0, length $host) . $host . chr(length $os) . $os;
}

sub run_session {
    send_frame(T_HELLO, 0, hello_payload());
    parse_ws();
    while (1) {
        end_session("client exiting") if $STOP;
        my $now = time;
        end_session(sprintf("no frame from kvm for %ds", DEAD_AFTER)) if $now - $S{last_rx} > DEAD_AFTER;
        if ($now - $S{last_ping} >= PING_EVERY) { $S{last_ping} = $now; send_ws(0x9, "nexit") }
        my $timeout = 1.0;
        for my $t (PING_EVERY - ($now - $S{last_ping}), DEAD_AFTER - ($now - $S{last_rx})) { $timeout = $t if $t < $timeout }
        for my $st (values %{ $S{streams} }) {
            next unless $st->{state} eq "connecting" && $st->{proto} == 1;
            if ($st->{deadline} <= $now) { open_fail($st, R_TIMEOUT) }
            elsif ($st->{deadline} - $now < $timeout) { $timeout = $st->{deadline} - $now }
        }
        flush_ws() if @{ $S{out} };
        my $rs = IO::Select->new($S{ws});
        my $ws = IO::Select->new;
        $ws->add($S{ws}) if @{ $S{out} };
        my %by_fd;
        for my $st (values %{ $S{streams} }) {
            my $s = $st->{sock} or next;
            $by_fd{ fileno $s } = $st;
            if ($st->{state} eq "connecting") { $ws->add($s); next }
            if ($st->{proto} == 2) { $rs->add($s); next }
            $rs->add($s) if !$st->{read_eof} && $st->{tx_credit} > 0 && $S{conn_credit} > 0;
            $ws->add($s) if length $st->{wbuf};
        }
        $timeout = 0.01 if $timeout < 0.01;
        my ($r, $w) = IO::Select->select($rs, $ws, undef, $timeout);
        next unless $r || $w;    # timeout or EINTR
        for my $h (@$w) {
            if ($h == $S{ws}) { flush_ws(); next }
            my $st = $by_fd{ fileno $h } or next;
            next unless $S{streams}{ $st->{sid} };
            if ($st->{state} eq "connecting") { connected($st) }
            else                               { drain($st) }
        }
        for my $h (@$r) {
            if ($h == $S{ws}) { read_ws(); next }
            my $st = $by_fd{ fileno $h } or next;
            next unless $S{streams}{ $st->{sid} };
            if    ($st->{proto} == 2)             { udp_in($st) }
            elsif ($st->{state} eq "open")        { read_stream($st) }
        }
    }
}

sub close_session {
    my $reason = shift;
    release($_) for values %{ $S{streams} };
    if ($S{ws}) {
        $S{ws}->blocking(1);
        eval { local $SIG{ALRM} = sub { die "t\n" }; alarm 2; syswrite($S{ws}, ws_frame(0x8, pack("n", 1000) . substr($reason, 0, 100))); alarm 0 };
        alarm 0;
        close $S{ws};
    }
}

# --- connect + upgrade -------------------------------------------------------------------

sub connect_ws {
    my %map = (http => "ws", https => "wss");
    my $scheme = $map{$SCHEME} || $SCHEME;
    die "bad scheme '$SCHEME'\n" unless $scheme eq "ws" || $scheme eq "wss";
    my $tls = $scheme eq "wss";
    (my $fp = lc $FINGERPRINT) =~ s/[^0-9a-f]//g;
    die "wss without a certificate fingerprint: refusing to connect unverified\n" if $tls && !$fp;
    my ($host, $port) = parse_host($HOST, $tls ? 443 : 80);
    my $sock;
    if ($tls) {
        eval { require IO::Socket::SSL; 1 } or die "IO::Socket::SSL is required for wss: $@";
        my $sni = ($host =~ /^[\d.]+$/ || $host =~ /:/) ? "" : $host;
        # The NanoKVM's certificate is normally self-signed: trust is the pin, not a CA.
        $sock = IO::Socket::SSL->new(PeerHost => $host, PeerPort => $port, Timeout => 10,
            SSL_verify_mode => IO::Socket::SSL::SSL_VERIFY_NONE(), SSL_hostname => $sni)
            or die "tls connect to $host:$port: " . ($IO::Socket::SSL::SSL_ERROR || $!) . "\n";
        my $got = $sock->get_fingerprint("sha256") || "";
        $got =~ s/^sha256\$//;
        $got = lc $got;
        die "certificate fingerprint mismatch: expected $fp got $got; refusing\n" if $got ne $fp;
    } else {
        my $class = eval { require IO::Socket::IP; 1 } ? "IO::Socket::IP" : "IO::Socket::INET";
        $sock = $class->new(PeerHost => $host, PeerPort => $port, Proto => "tcp", Timeout => 10)
            or die "connect to $host:$port: $@\n";
    }
    setsockopt($sock, IPPROTO_TCP, TCP_NODELAY, 1);
    my $key = encode_base64(pack("C16", map { int rand 256 } 1 .. 16), "");
    my $req = "GET /exit/$SLOT/native HTTP/1.1\r\nHost: $HOST\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
        . "Sec-WebSocket-Key: $key\r\nSec-WebSocket-Version: 13\r\nAuthorization: Bearer $TOKEN\r\n"
        . "User-Agent: nexit-perl/1\r\n\r\n";
    for (my $off = 0; $off < length $req;) {
        my $n = syswrite($sock, $req, length($req) - $off, $off) or die "handshake write: $!\n";
        $off += $n;
    }
    my $buf      = "";
    my $deadline = time + 10;
    while ($buf !~ /\r\n\r\n/) {
        die "handshake timeout\n" if time > $deadline;
        my $sel = IO::Select->new($sock);
        next unless $sel->can_read(1) || ($tls && $sock->pending);
        my $n = sysread($sock, my $chunk, 4096);
        die "connection closed during handshake\n" if defined $n && $n == 0;
        next if !defined $n && would_block();
        die "handshake read: $!\n" unless defined $n;
        $buf .= $chunk;
        die "oversized handshake response\n" if length($buf) > 65536;
    }
    my ($head, $rest) = split /\r\n\r\n/, $buf, 2;
    my @lines = split /\r\n/, $head;
    my ($status) = $lines[0] =~ /^HTTP\/\d\.\d (\d{3})/;
    if (!defined $status || $status ne "101") {
        die "rejected (404): wrong token or slot, or this source is rate limited\n" if defined $status && $status eq "404";
        die "upgrade refused: $lines[0]\n";
    }
    my %h;
    for (@lines[1 .. $#lines]) { $h{lc $1} = $2 if /^([^:]+):\s*(.*?)\s*$/ }
    my $expect = encode_base64(sha1($key . $WS_GUID), "");
    die "bad Sec-WebSocket-Accept\n" if ($h{"sec-websocket-accept"} || "") ne $expect;
    die "no Upgrade: websocket in response\n" if lc($h{upgrade} || "") ne "websocket";
    $sock->blocking(0);
    return ($sock, $tls, $rest);
}

sub main {
    if ($HOST =~ /^__/ || $TOKEN =~ /^__/) {
        log_msg("this script must be fetched from the NanoKVM, which fills in its parameters");
        return 2;
    }
    $SIG{INT} = $SIG{TERM} = sub { $STOP = 1 };
    $SIG{PIPE} = "IGNORE";
    my %map = (http => "ws", https => "wss");
    log_msg(sprintf("exit client for %s://%s/exit/%s (allowPrivate=%s, pin=%s)", $map{$SCHEME} || $SCHEME, $HOST, $SLOT,
        $ALLOW_PRIVATE ? "true" : "false", $FINGERPRINT ? substr($FINGERPRINT, 0, 16) . "..." : "none"));
    my $backoff = 1;
    until ($STOP) {
        %S = (ws => undef, tls => 0, inbuf => "", out => [], out_off => 0, frag_op => undef, frag => "", streams => {},
            stream_window => 0, conn_window => 0, max_streams => 0, conn_credit => 0, conn_unacked => 0, welcomed => 0,
            last_rx => time, last_ping => time);
        my ($sock, $tls, $rest) = eval { connect_ws() };
        if (!$sock) {
            my $e = $@; chomp $e;
            log_msg("connect failed: $e");
        } else {
            @S{qw(ws tls inbuf last_rx last_ping)} = ($sock, $tls, $rest || "", time, time);
            eval { run_session(); 1 } or do {
                my $e = $@; chomp $e;
                $e =~ s/^SESSION_END: //;
                $backoff = 1 if $S{welcomed};
                log_msg("session ended: $e");
                close_session($e);
            };
        }
        last if $STOP;
        log_msg("reconnecting in ${backoff}s");
        sleep $backoff;
        $backoff = $backoff * 2 > 30 ? 30 : $backoff * 2;
    }
    log_msg("stopping");
    return 0;
}

exit main();
