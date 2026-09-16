package exit

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
)

// Config is /etc/kvm/exit/<n>.json, mode 0600, written atomically. Everything
// derivable from the slot id (ports, tun, table, chains, paths) is derived by
// Slot and never stored.
type Config struct {
	Slot           string         `json:"slot"`
	Enabled        bool           `json:"enabled"`
	Pending        bool           `json:"pending"`
	Mode           proto.ExitMode `json:"mode"`
	Token          string         `json:"token"`
	TokenCreatedAt time.Time      `json:"tokenCreatedAt"`
	NIC            string         `json:"nic"`
	DNS            []string       `json:"dns"`
	MTU            int            `json:"mtu"`
	AllowPrivate   bool           `json:"allowPrivate"`
	PinPeer        bool           `json:"pinPeer"`
}

// State is /etc/kvm/exit/<n>.state.json: the little that has to survive a
// restart so the panel can say when an exit was last seen and from where.
type State struct {
	LastConnectedAt *time.Time      `json:"lastConnectedAt"`
	PreviousPeer    *proto.ExitPeer `json:"previousPeer"`
	PeerChangedAt   *time.Time      `json:"peerChangedAt"`
}

const (
	// NICGadget is the one NIC role that exists: the USB gadget NIC (D9).
	NICGadget = "gadget"
	// DefaultMTU is D7: IP in SOCKS in WS in TLS black-holes anything larger.
	DefaultMTU = 1280
	// TokenLength and tokenAlphabet are D10: 31 symbols, no 0 o 1 l i.
	TokenLength   = 8
	tokenAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"
)

// DefaultDNS are the pinned public resolvers the forwarder uses (D3).
var DefaultDNS = []string{"1.1.1.1", "8.8.8.8"}

var configMu sync.Mutex

// NewToken draws TokenLength symbols from tokenAlphabet with rejection
// sampling, so every symbol is uniform.
func NewToken() (string, error) {
	const alphabetLen = len(tokenAlphabet)
	limit := 256 - 256%alphabetLen
	out := make([]byte, 0, TokenLength)
	buf := make([]byte, 1)
	for len(out) < TokenLength {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("token: %w", err)
		}
		if int(buf[0]) >= limit {
			continue
		}
		out = append(out, tokenAlphabet[int(buf[0])%alphabetLen])
	}
	return string(out), nil
}

// ValidToken reports whether s is shaped like a token this package issued.
func ValidToken(s string) bool {
	if len(s) != TokenLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune(tokenAlphabet, rune(s[i])) {
			return false
		}
	}
	return true
}

// DefaultConfig is a disabled Mode A slot with a fresh token.
func DefaultConfig(slot Slot) (Config, error) {
	token, err := NewToken()
	if err != nil {
		return Config{}, err
	}
	return Config{
		Slot:           slot.ID,
		Mode:           proto.ExitModeNative,
		Token:          token,
		TokenCreatedAt: time.Now().UTC().Truncate(time.Second),
		NIC:            NICGadget,
		DNS:            append([]string(nil), DefaultDNS...),
		MTU:            DefaultMTU,
	}, nil
}

// Normalize fills defaults a hand-edited or older file may lack.
func (c *Config) Normalize() {
	if c.Mode != proto.ExitModeWstunnel {
		c.Mode = proto.ExitModeNative
	}
	if c.NIC == "" {
		c.NIC = NICGadget
	}
	reachable := make([]string, 0, len(c.DNS))
	for _, up := range c.Upstreams() {
		if resolverReachable(up, c.Policy()) {
			reachable = append(reachable, up.String())
		}
	}
	if len(reachable) == 0 {
		reachable = append([]string(nil), DefaultDNS...)
	}
	c.DNS = reachable
	if c.MTU < 576 || c.MTU > 1400 {
		c.MTU = DefaultMTU
	}
}

// Policy is the D21 policy for this slot.
func (c Config) Policy() Policy { return Policy{AllowPrivate: c.AllowPrivate} }

// Upstreams are the parsed resolvers, invalid entries dropped.
func (c Config) Upstreams() []netip.Addr {
	out := make([]netip.Addr, 0, len(c.DNS))
	for _, s := range c.DNS {
		if a, err := netip.ParseAddr(strings.TrimSpace(s)); err == nil && a.IsValid() && !a.IsUnspecified() {
			out = append(out, a)
		}
	}
	return out
}

// resolverReachable reports whether up can be used as a DNS upstream for a
// slot with this policy: the forwarder dials it through the front door, which
// denies anything the policy denies and answers atyp 4 for IPv6 (D22), so a
// resolver the slot can never reach is not a resolver (MGR-8).
func resolverReachable(up netip.Addr, policy Policy) bool {
	return up.Is4() && policy.Allow(up)
}

// ValidateResolvers rejects a DNS list with an entry the slot can never
// reach, naming the offender so the operator sees why (MGR-8). Parse-invalid
// entries are ignored here; Normalize drops them.
func ValidateResolvers(cfg Config) error {
	policy := cfg.Policy()
	for _, s := range cfg.DNS {
		up, err := netip.ParseAddr(strings.TrimSpace(s))
		if err != nil || !up.IsValid() || up.IsUnspecified() {
			continue
		}
		if up.Is6() {
			return fmt.Errorf("dns resolver %s: IPv6 resolvers are not supported yet", up)
		}
		if !policy.Allow(up) {
			return fmt.Errorf("dns resolver %s is unreachable under this slot's policy", up)
		}
	}
	return nil
}

// LoadConfig reads the slot's config. ok is false when the file is absent.
func LoadConfig(slot Slot) (cfg Config, ok bool, err error) {
	configMu.Lock()
	defer configMu.Unlock()
	return loadConfig(slot)
}

func loadConfig(slot Slot) (Config, bool, error) {
	data, err := os.ReadFile(slot.ConfigPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, false, nil
		}
		return Config{}, false, fmt.Errorf("read %s: %w", slot.ConfigPath(), err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, false, fmt.Errorf("decode %s: %w", slot.ConfigPath(), err)
	}
	cfg.Slot = slot.ID
	cfg.Normalize()
	if !ValidToken(cfg.Token) {
		// A token this package never issued (hand-edited, restored from
		// elsewhere, or missing from the JSON) is the one field the gate
		// cannot be defensive about on its own: an empty token would compare
		// equal to an absent header. Repair it here, force the slot off so the
		// operator re-enables with the new token in hand, and write the repair
		// back so every boot sees the same token.
		token, err := NewToken()
		if err != nil {
			return Config{}, false, fmt.Errorf("repair %s: %w", slot.ConfigPath(), err)
		}
		cfg.Token = token
		cfg.TokenCreatedAt = time.Now().UTC().Truncate(time.Second)
		cfg.Enabled, cfg.Pending = false, false
		if err := saveConfig(cfg); err != nil {
			return Config{}, false, fmt.Errorf("repair %s: %w", slot.ConfigPath(), err)
		}
	}
	return cfg, true, nil
}

// SaveConfig writes the slot's config atomically, mode 0600.
func SaveConfig(cfg Config) error {
	configMu.Lock()
	defer configMu.Unlock()
	return saveConfig(cfg)
}

func saveConfig(cfg Config) error {
	slot, err := ParseSlot(cfg.Slot)
	if err != nil {
		return err
	}
	if !ValidToken(cfg.Token) {
		return errors.New("config: token is not one this package issued")
	}
	cfg.Normalize()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(ConfigDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", ConfigDir, err)
	}
	return utils.WriteFileAtomic(slot.ConfigPath(), append(data, '\n'), 0o600)
}

// LoadAll returns every slot config in ConfigDir, ordered by slot id.
func LoadAll() ([]Config, error) {
	configMu.Lock()
	defer configMu.Unlock()

	entries, err := os.ReadDir(ConfigDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", ConfigDir, err)
	}
	var out []Config
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".state.json") {
			continue
		}
		slot, err := ParseSlot(strings.TrimSuffix(name, ".json"))
		if err != nil {
			continue
		}
		cfg, ok, err := loadConfig(slot)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, cfg)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out, nil
}

// AnyEnabled reports the first slot whose config says enabled: true, which is
// what the bridge consults before taking the gadget NIC (D17). It reads the
// files rather than the manager so it answers before the manager exists.
func AnyEnabled() (slot string, ok bool) {
	cfgs, err := LoadAll()
	if err != nil {
		return "", false
	}
	for _, cfg := range cfgs {
		if cfg.Enabled {
			return cfg.Slot, true
		}
	}
	return "", false
}

// LoadState reads the slot's state file; an absent file is the zero State.
func LoadState(slot Slot) (State, error) {
	data, err := os.ReadFile(slot.StatePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("read %s: %w", slot.StatePath(), err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, fmt.Errorf("decode %s: %w", slot.StatePath(), err)
	}
	return st, nil
}

// SaveState writes the slot's state file atomically.
func SaveState(slot Slot, st State) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	return utils.WriteFileAtomic(slot.StatePath(), append(data, '\n'), 0o600)
}

// WriteGadgetRoute writes the D4 marker naming the slot that owns the gadget
// NIC's gateway; RemoveGadgetRoute removes it only when that slot owns it.
func WriteGadgetRoute(slot Slot) error {
	return utils.WriteFileAtomic(GadgetRoutePath(), []byte(slot.ID+"\n"), 0o644)
}

// GadgetRouteOwner returns the slot id named by the marker, ok false when the
// marker is absent or empty.
func GadgetRouteOwner() (string, bool) {
	data, err := os.ReadFile(GadgetRoutePath())
	if err != nil {
		return "", false
	}
	owner := strings.TrimSpace(string(data))
	return owner, owner != ""
}

func RemoveGadgetRoute(slot Slot) error {
	data, err := os.ReadFile(GadgetRoutePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if owner := strings.TrimSpace(string(data)); owner != "" && owner != slot.ID {
		return nil
	}
	if err := os.Remove(GadgetRoutePath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return utils.SyncDir(filepath.Dir(GadgetRoutePath()))
}
