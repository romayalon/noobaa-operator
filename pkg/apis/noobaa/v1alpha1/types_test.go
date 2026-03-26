package v1alpha1

import (
	"encoding/json"
	"reflect"
	"testing"
)

func tlsVersionPtr(v TLSProtocolVersion) *TLSProtocolVersion {
	return &v
}

func TestTLSSecuritySpec_JSONRoundTrip(t *testing.T) {
	spec := TLSSecuritySpec{
		TLSMinVersion:          tlsVersionPtr(TLSVersionTLS13),
		TLSCiphers:     []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384"},
		TLSGroups: []TLSGroup{TLSGroupX25519MLKEM768, TLSGroupX25519, TLSGroupSecp256r1},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded TLSSecuritySpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(spec, decoded) {
		t.Errorf("round-trip mismatch:\n  original: %+v\n  decoded:  %+v", spec, decoded)
	}
}

func TestTLSSecuritySpec_JSONFieldNames(t *testing.T) {
	spec := TLSSecuritySpec{
		TLSMinVersion:          tlsVersionPtr(TLSVersionTLS12),
		TLSCiphers:     []string{"cipher1"},
		TLSGroups: []TLSGroup{TLSGroupX25519},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map failed: %v", err)
	}

	for _, field := range []string{"tlsMinVersion", "tlsCiphers", "tlsGroups"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("expected JSON field %q not found in output: %s", field, string(data))
		}
	}

	for _, field := range []string{"tlsVersion", "tlsCipherSuites", "tlsCurvePreferences"} {
		if _, ok := raw[field]; ok {
			t.Errorf("old JSON field %q should not be present in output: %s", field, string(data))
		}
	}
}

func TestTLSSecuritySpec_EmptyJSON(t *testing.T) {
	spec := TLSSecuritySpec{}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	if string(data) != "{}" {
		t.Errorf("empty spec should marshal to {}, got %s", string(data))
	}
}

func TestTLSSecuritySpec_DeepCopy(t *testing.T) {
	spec := TLSSecuritySpec{
		TLSMinVersion:          tlsVersionPtr(TLSVersionTLS13),
		TLSCiphers:     []string{"TLS_AES_128_GCM_SHA256"},
		TLSGroups: []TLSGroup{TLSGroupX25519, TLSGroupSecp256r1},
	}

	copied := spec.DeepCopy()

	if !reflect.DeepEqual(&spec, copied) {
		t.Errorf("deep copy mismatch:\n  original: %+v\n  copy:     %+v", spec, *copied)
	}

	// Mutate the copy and verify original is unaffected
	*copied.TLSMinVersion = TLSVersionTLS12
	copied.TLSCiphers[0] = "MODIFIED"
	copied.TLSGroups = append(copied.TLSGroups, TLSGroupSecp384r1)

	if *spec.TLSMinVersion != TLSVersionTLS13 {
		t.Error("original TLSMinVersion was mutated by deep copy modification")
	}
	if spec.TLSCiphers[0] != "TLS_AES_128_GCM_SHA256" {
		t.Error("original TLSCiphers was mutated by deep copy modification")
	}
	if len(spec.TLSGroups) != 2 {
		t.Error("original TLSGroups was mutated by deep copy modification")
	}
}

func TestTLSSecuritySpec_DeepCopyNil(t *testing.T) {
	var spec *TLSSecuritySpec
	copied := spec.DeepCopy()
	if copied != nil {
		t.Error("deep copy of nil should return nil")
	}
}

func TestSecuritySpec_DeepCopyAPIServerSecurity(t *testing.T) {
	spec := SecuritySpec{
		APIServerSecurity: TLSSecuritySpec{
			TLSMinVersion:      tlsVersionPtr(TLSVersionTLS13),
			TLSCiphers: []string{"cipher-api"},
		},
	}

	copied := spec.DeepCopy()

	if !reflect.DeepEqual(&spec, copied) {
		t.Errorf("deep copy mismatch:\n  original: %+v\n  copy:     %+v", spec, *copied)
	}

	// Mutate API server in copy, verify original is unaffected
	*copied.APIServerSecurity.TLSMinVersion = TLSVersionTLS12
	if *spec.APIServerSecurity.TLSMinVersion != TLSVersionTLS13 {
		t.Error("original APIServerSecurity.TLSMinVersion was mutated")
	}

	copied.APIServerSecurity.TLSCiphers[0] = "MODIFIED"
	if spec.APIServerSecurity.TLSCiphers[0] != "cipher-api" {
		t.Error("original APIServerSecurity.TLSCiphers was mutated")
	}
}

func TestSecuritySpec_JSONRoundTrip(t *testing.T) {
	spec := SecuritySpec{
		APIServerSecurity: TLSSecuritySpec{
			TLSMinVersion:          tlsVersionPtr(TLSVersionTLS13),
			TLSCiphers:     []string{"TLS_AES_256_GCM_SHA384"},
			TLSGroups: []TLSGroup{TLSGroupX25519MLKEM768},
		},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map failed: %v", err)
	}

	if _, ok := raw["apiServerSecurity"]; !ok {
		t.Error("expected JSON field apiServerSecurity not found")
	}

	var decoded SecuritySpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if !reflect.DeepEqual(spec, decoded) {
		t.Errorf("round-trip mismatch:\n  original: %+v\n  decoded:  %+v", spec, decoded)
	}
}

func TestTLSProtocolVersion_Values(t *testing.T) {
	if TLSVersionTLS12 != "VersionTLS12" {
		t.Errorf("TLSVersionTLS12 = %q, want VersionTLS12", TLSVersionTLS12)
	}
	if TLSVersionTLS13 != "VersionTLS13" {
		t.Errorf("TLSVersionTLS13 = %q, want VersionTLS13", TLSVersionTLS13)
	}
}
