package edid

import (
	"bytes"
	"errors"
	"testing"
)

// Decode is reached from PUT /api/vm/edid with operator supplied bytes, and
// what it accepts is handed to a flash tool whose own validator is three
// checks. The two properties worth holding are that it never panics or reads
// out of bounds, and that anything it accepts re-encodes to the same bytes,
// since Apply stages Normalize's output while the store archives what Encode
// produced.
func FuzzDecode(f *testing.F) {
	f.Add(fixture(f))
	for _, data := range corpus(f) {
		f.Add(data)
	}
	for _, profile := range Profiles() {
		f.Add(profile.Data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		decodeProperties(t, data)

		// Both checksums have to be right before Decode looks at anything
		// structural, so without this the fuzzer spends its whole budget on
		// the checksum reject and never reaches parseCTA or parseTiming.
		if len(data) == Size {
			decodeProperties(t, repaired(data))
		}
	})
}

func decodeProperties(t *testing.T, data []byte) {
	t.Helper()

	parsed, err := Decode(data)
	if err != nil {
		var rejected *RejectError
		if !errors.As(err, &rejected) {
			t.Fatalf("%d bytes: error %v is not a reject", len(data), err)
		}
		if parsed != nil {
			t.Fatalf("%d bytes: rejected as %q but returned %+v", len(data), rejected.Kind, parsed)
		}
		return
	}

	out, err := parsed.Encode()
	if err != nil {
		t.Fatalf("accepted then failed to encode: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("accepted % X but re-encoded to % X", data, out)
	}

	again, err := Decode(out)
	if err != nil {
		t.Fatalf("re-encoded bytes were rejected: %v", err)
	}
	second, err := again.Encode()
	if err != nil {
		t.Fatalf("second encode: %v", err)
	}
	if !bytes.Equal(second, out) {
		t.Fatal("decode and encode are not idempotent")
	}
}

// The fuzzer reaches these eventually; running them on every go test keeps the
// arithmetic in parseTiming and parseCTA honest without one.
func TestDecodeSurvivesEverySingleByteEdit(t *testing.T) {
	data := fixture(t)

	for offset := 0; offset < Size; offset++ {
		for _, value := range []byte{0x00, 0x01, 0x0F, 0x7F, 0x80, 0xF0, 0xFF, data[offset] ^ 0xFF} {
			if value == data[offset] {
				continue
			}
			edit := repaired(mutate(data, map[int]byte{offset: value}))

			parsed, err := Decode(edit)
			if err != nil {
				var rejected *RejectError
				if !errors.As(err, &rejected) {
					t.Fatalf("byte %d = 0x%02X: error %v is not a reject", offset, value, err)
				}
				continue
			}

			out, err := parsed.Encode()
			if err != nil {
				t.Fatalf("byte %d = 0x%02X: accepted then failed to encode: %v", offset, value, err)
			}
			if !bytes.Equal(out, edit) {
				for i := range edit {
					if out[i] != edit[i] {
						t.Fatalf("byte %d = 0x%02X: re-encoded byte %d as 0x%02X, want 0x%02X", offset, value, i, out[i], edit[i])
					}
				}
			}
		}
	}
}

// Every edit that is not itself a checksum byte has to be caught, because the
// flash tool checks nothing else about the bytes it programs.
func TestDecodeCatchesEverySingleByteCorruption(t *testing.T) {
	data := fixture(t)

	for offset := 0; offset < Size; offset++ {
		if offset == BlockSize-1 || offset == Size-1 {
			continue
		}
		edit := mutate(data, map[int]byte{offset: data[offset] ^ 0x01})

		_, err := Decode(edit)
		var rejected *RejectError
		if !errors.As(err, &rejected) {
			t.Fatalf("flipping bit 0 of byte %d was not rejected: %v", offset, err)
		}
		if rejected.Kind != RejectChecksum && rejected.Kind != RejectHeader {
			t.Fatalf("flipping bit 0 of byte %d was rejected as %q, want a checksum or header reject", offset, rejected.Kind)
		}
	}
}

func TestEncodeRecomputesBothChecksums(t *testing.T) {
	blobs := map[string][]byte{"testdata/E21_NanoKVM.bin": fixture(t)}
	for name, data := range corpus(t) {
		blobs[name] = data
	}
	for _, profile := range Profiles() {
		blobs[profile.Source] = profile.Data
	}

	for name, raw := range blobs {
		blob, err := Normalize(raw)
		if err != nil {
			t.Fatalf("%s: normalize: %v", name, err)
		}
		parsed, err := Decode(blob)
		if err != nil {
			continue
		}

		parsed.Serial++
		parsed.Week++
		out, err := parsed.Encode()
		if err != nil {
			t.Fatalf("%s: encode: %v", name, err)
		}
		if sum(out[:BlockSize]) != 0 {
			t.Errorf("%s: base block sums to %d after an edit", name, sum(out[:BlockSize]))
		}
		if sum(out[BlockSize:]) != 0 {
			t.Errorf("%s: extension block sums to %d after an edit", name, sum(out[BlockSize:]))
		}
		if bytes.Equal(out, blob) {
			t.Errorf("%s: editing the serial and week did not change the bytes", name)
		}
		if _, err := Decode(out); err != nil {
			t.Errorf("%s: re-encoded blob was rejected: %v", name, err)
		}
	}
}
