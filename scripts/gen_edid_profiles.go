package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"NanoKVM-Server/service/edid"
	"NanoKVM-Server/utils"
)

const (
	corpusHost    = "raw.githubusercontent.com"
	corpusRepo    = "/linuxhw/EDID/"
	corpusCommit  = "9c0c1bffc9c0f1cb2044115149a5ecb1652803f8"
	factorySource = "tools/nanokvm_update_edid/E21_NanoKVM.bin"
	extensionByte = 126
	maxHorizontal = 1920
	maxVertical   = 1080
	maxRefresh    = 60
)

var candidates = []string{
	"Digital/Dell/DELD0D8/117711A54D0B",
	"Digital/Dell/DELF068/50158153B538",
	"Digital/Dell/DELA0D8/4D9FF7AF0CB6",
	"Digital/HP/HPN3660/60113C7F4B42",
	"Digital/HP/HPN3620/5F1E95B1E975",
	"Digital/Lenovo/LEN60C8/AC0EC9010CB7",
	"Digital/Acer/ACR032D/4B642A105641",
	"Digital/BenQ/BNQ78A4/05C5A4D29E84",
	"Digital/ViewSonic/VSC732E/09B254CD7B43",
	"Digital/Philips/PHL08DD/3DC3CDCF6BC0",
	"Digital/NEC/NEC2C86/5DB82313D6A5",
	"Digital/NEC/NEC2EA1/8D46B59A747A",
	"Digital/Eizo/ENC2393/186AD365FDFB",
	"Digital/AOC/AOC2401/29D18A596E7F",
	"Digital/ASUS/AUS2402/649407BC0DB1",
	"Digital/ASUS/AUS2426/5649967DE6D8",
	"Digital/AOC/AOC2001/7F3588AF02A7",
	"Digital/Dell/DELF00D/490B7756F848",
	"Digital/Dell/DELA019/2A7916E93C6B",
	"Digital/Dell/DEL4025/C73F88AA2DAF",
	"Digital/Eizo/ENC1837/48327990BEF5",
	"Digital/AOC/AOC1670/1820E6BC1AAF",
	"Digital/Acer/ACRAD05/E6FF478FDFFF",
	"Digital/Goldstar/GSM0001/3C8D8B9A1831",
	"Digital/Samsung/SAM0000/2D56A8334AA9",
}

var dumpLine = regexp.MustCompile(`(?m)^(?:[0-9a-f]{32}|(?:[0-9a-f]{2} ){15}[0-9a-f]{2})$`)

func main() {
	out := flag.String("out", "service/edid/profiles_gen.go", "generated table, relative to the server module")
	factory := flag.String("factory", "../"+factorySource, "factory EDID blob")
	flag.Parse()

	if err := generate(*out, *factory); err != nil {
		log.Fatalf("gen_edid_profiles: %v", err)
	}
}

func generate(out, factory string) error {
	blob, err := os.ReadFile(factory)
	if err != nil {
		return fmt.Errorf("read %s: %w", factory, err)
	}

	factoryProfile, err := build(blob, factorySource)
	if err != nil {
		return fmt.Errorf("factory blob %s: %w", factory, err)
	}

	shipped := []edid.Profile{factoryProfile}
	client := &http.Client{Timeout: 30 * time.Second}
	for _, path := range candidates {
		blob, err := fetch(client, path)
		if err != nil {
			return fmt.Errorf("candidate %s: %w", path, err)
		}
		profile, err := build(blob, path+"@"+corpusCommit)
		if err != nil {
			log.Printf("drop %s: %v", path, err)
			continue
		}
		shipped = append(shipped, profile)
	}

	log.Printf("shipping %d of %d candidates", len(shipped), len(candidates)+1)
	return emit(out, shipped)
}

func build(blob []byte, source string) (edid.Profile, error) {
	data, err := edid.Normalize(blob)
	if err != nil {
		return edid.Profile{}, fmt.Errorf("normalize: %w", err)
	}
	if got, want := data[edid.BlockSize-1], checksum(data[:edid.BlockSize]); got != want {
		return edid.Profile{}, fmt.Errorf("base checksum 0x%02X, recomputed 0x%02X", got, want)
	}
	if got, want := data[edid.Size-1], checksum(data[edid.BlockSize:]); got != want {
		return edid.Profile{}, fmt.Errorf("extension checksum 0x%02X, recomputed 0x%02X", got, want)
	}

	parsed, err := edid.Decode(data)
	if err != nil {
		return edid.Profile{}, fmt.Errorf("validate: %w", err)
	}
	model := parsed.Name()
	if model == "" {
		return edid.Profile{}, errors.New("no monitor name descriptor")
	}
	timing := parsed.PreferredTiming()
	if timing == nil {
		return edid.Profile{}, errors.New("no preferred timing")
	}
	if timing.HActive > maxHorizontal || timing.VActive > maxVertical || math.Round(timing.RefreshHz) > maxRefresh {
		return edid.Profile{}, fmt.Errorf("preferred mode %s is beyond the capture path", timing.Mode())
	}

	return edid.Profile{
		SHA256:        sha256.Sum256(data),
		Manufacturer:  parsed.Manufacturer,
		Model:         model,
		PreferredMode: timing.Mode(),
		Source:        source,
		Data:          data,
	}, nil
}

func fetch(client *http.Client, path string) ([]byte, error) {
	source := url.URL{Scheme: "https", Host: corpusHost, Path: corpusRepo + corpusCommit + "/" + path}

	response, err := client.Get(source.String())
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", source.String(), err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: %s", source.String(), response.Status)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", source.String(), err)
	}
	return decodeDump(body)
}

func decodeDump(body []byte) ([]byte, error) {
	lines := dumpLine.FindAllString(string(body), -1)
	if len(lines) == 0 {
		return nil, errors.New("no hex dump")
	}

	data, err := hex.DecodeString(strings.ReplaceAll(strings.Join(lines, ""), " ", ""))
	if err != nil {
		return nil, fmt.Errorf("decode hex dump: %w", err)
	}
	if len(data) < edid.BlockSize {
		return nil, fmt.Errorf("hex dump holds %d bytes", len(data))
	}

	blocks := edid.BlockSize * (1 + int(data[extensionByte]))
	if len(data) < blocks {
		return nil, fmt.Errorf("declares %d extension blocks, hex dump holds %d bytes", data[extensionByte], len(data))
	}
	return data[:blocks], nil
}

func checksum(block []byte) byte {
	var sum byte
	for _, value := range block[:len(block)-1] {
		sum += value
	}
	return -sum
}

func emit(path string, profiles []edid.Profile) error {
	var out strings.Builder
	out.WriteString("// Code generated by scripts/gen_edid_profiles.go. DO NOT EDIT.\n\npackage edid\n\nvar profiles = []Profile{\n")
	for _, profile := range profiles {
		fmt.Fprintf(&out, "{\nSHA256: [32]byte{\n%s},\nManufacturer: %q,\nModel: %q,\nPreferredMode: %q,\nSource: %q,\nData: []byte{\n%s},\n},\n",
			literal(profile.SHA256[:]), profile.Manufacturer, profile.Model, profile.PreferredMode, profile.Source, literal(profile.Data))
	}
	out.WriteString("}\n")

	source, err := format.Source([]byte(out.String()))
	if err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}
	return utils.WriteFileAtomic(path, source, 0o644)
}

func literal(data []byte) string {
	var out strings.Builder
	for index, value := range data {
		fmt.Fprintf(&out, "0x%02x,", value)
		if (index+1)%16 == 0 {
			out.WriteString("\n")
		} else {
			out.WriteString(" ")
		}
	}
	return out.String()
}
