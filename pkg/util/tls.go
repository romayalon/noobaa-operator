package util

import (
	"crypto/tls"
	"fmt"
	"strings"

	ocstlsv1 "github.com/red-hat-storage/ocs-tls-profiles/api/v1"
)

// GoCiphersAndCurvesFromTLSConfig maps spec.security.tlsConfig using
// github.com/red-hat-storage/ocs-tls-profiles/api/v1.ValidateAndGetGoTLSConfig.
// It returns nil slices when cfg is nil or both cipher and group lists are empty.
func GoCiphersAndCurvesFromTLSConfig(cfg *ocstlsv1.TLSConfig) ([]uint16, []tls.CurveID, error) {
	if cfg == nil || (len(cfg.Ciphers) == 0 && len(cfg.Groups) == 0) {
		return nil, nil, nil
	}
	goCfg, err := ocstlsv1.ValidateAndGetGoTLSConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return goCfg.CipherSuites, goCfg.CurvePreferences, nil
}

// OpenSSLCipherAndGroupStringsFromTLSConfig returns colon-separated OpenSSL cipher and group
// strings for TLS_CIPHERS and TLS_GROUPS. It runs ValidateAndGetGoTLSConfig and OpenSSLConfigFrom once.
func OpenSSLCipherAndGroupStringsFromTLSConfig(cfg *ocstlsv1.TLSConfig) (ciphers, groups string, err error) {
	if cfg == nil {
		return "", "", nil
	}
	if len(cfg.Ciphers) == 0 && len(cfg.Groups) == 0 {
		return "", "", nil
	}
	goCfg, err := ocstlsv1.ValidateAndGetGoTLSConfig(cfg)
	if err != nil {
		return "", "", err
	}
	ssl := ocstlsv1.OpenSSLConfigFrom(goCfg)
	if ssl == nil {
		return "", "", fmt.Errorf("OpenSSLConfigFrom returned nil after successful TLS validation")
	}
	if len(ssl.Ciphers) > 0 {
		ciphers = strings.Join(ssl.Ciphers, ":")
	}
	if len(ssl.Groups) > 0 {
		groups = strings.Join(ssl.Groups, ":")
	}
	return ciphers, groups, nil
}
