package admission

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"syscall"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/options"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	port       = "8080"
	tlscert    = "/etc/certs/tls.cert"
	tlskey     = "/etc/certs/tls.key"
	tlscertolm = "/tmp/k8s-webhook-server/serving-certs/tls.crt"
	tlskeyolm  = "/tmp/k8s-webhook-server/serving-certs/tls.key"
)

var currentTLSConfig atomic.Pointer[tls.Config]

// ReloadTLSConfig rebuilds the TLS configuration by loading certificates
// from disk and reading the NooBaa CR's APIServerSecurity settings, then
// swaps it atomically. New TLS connections will use the updated config.
func ReloadTLSConfig() error {
	log := logrus.WithField("admission server", options.Namespace)

	var certPath, keyPath string
	if _, ok := os.LookupEnv("NOOBAA_CLI_DEPLOYMENT"); !ok {
		certPath, keyPath = tlscertolm, tlskeyolm
	} else {
		certPath, keyPath = tlscert, tlskey
	}

	certs, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		log.Errorf("Failed to reload TLS key pair: %v", err)
		return err
	}

	cfg := &tls.Config{Certificates: []tls.Certificate{certs}}
	applyAPIServerTLS(cfg, log)
	currentTLSConfig.Store(cfg)
	log.Info("Admission server TLS configuration reloaded")
	return nil
}

// RunAdmissionServer starts the admission HTTPS server.
func RunAdmissionServer() {
	log := logrus.WithField("admission server", options.Namespace)

	if err := ReloadTLSConfig(); err != nil {
		log.Errorf("Failed to load initial TLS config, admission server not started: %v", err)
		return
	}

	server := &http.Server{
		Addr: fmt.Sprintf(":%v", port),
		TLSConfig: &tls.Config{
			GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
				return currentTLSConfig.Load(), nil
			},
		},
	}

	sh := ServerHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", sh.serve)
	server.Handler = mux

	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			log.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	log.Infof("Admission server start running and listening on port: %s", port)

	util.OnSignal(func() {
		log.Info("Got shutdown signal, shutting down webhook server gracefully...")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Errorf("Failed to shut down the admission server: %v", err)
		}
	}, syscall.SIGINT, syscall.SIGTERM)
}

// applyAPIServerTLS fetches the NooBaa CR and applies APIServerSecurity TLS
// properties to the given tls.Config when they are set.
func applyAPIServerTLS(tlsConfig *tls.Config, log *logrus.Entry) {
	noobaa := &nbv1.NooBaa{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "noobaa",
			Namespace: options.Namespace,
		},
	}
	if !util.KubeCheckQuiet(noobaa) {
		log.Info("NooBaa CR not found, using default TLS config for admission server")
		return
	}

	spec := noobaa.Spec.Security.APIServerSecurity

	if spec.TLSMinVersion != nil {
		switch *spec.TLSMinVersion {
		case nbv1.VersionTLS12:
			tlsConfig.MinVersion = tls.VersionTLS12
			log.Info("Admission server TLS min version set to TLSv1.2")
		case nbv1.VersionTLS13:
			tlsConfig.MinVersion = tls.VersionTLS13
			log.Info("Admission server TLS min version set to TLSv1.3")
		}
	}

	if len(spec.TLSCiphers) > 0 {
		tlsConfig.CipherSuites = mapCipherSuites(spec.TLSCiphers, log)
	}

	if len(spec.TLSGroups) > 0 {
		tlsConfig.CurvePreferences = mapGroupPreferences(spec.TLSGroups, log)
	}
}

// mapCipherSuites converts cipher suite names to uint16 IDs for tls.Config.CipherSuites.
// Only Go/IANA TLS 1.2 names from tls.CipherSuites() are accepted — insecure suites
// are rejected and OpenSSL-format names (e.g. ECDHE-RSA-AES128-GCM-SHA256) are not
// supported. TLS 1.3 cipher suites are always enabled in Go and cannot be configured
// via tls.Config.CipherSuites; any TLS 1.3 names are logged at Info level and skipped.
func mapCipherSuites(names []string, log *logrus.Entry) []uint16 {
	lookup := make(map[string]uint16)
	for _, cs := range tls.CipherSuites() {
		lookup[cs.Name] = cs.ID
	}
	var ids []uint16
	var applied []string
	for _, name := range names {
		if id, ok := lookup[name]; ok {
			ids = append(ids, id)
			applied = append(applied, name)
		} else if strings.HasPrefix(name, "TLS_AES_") || strings.HasPrefix(name, "TLS_CHACHA20_") {
			log.Infof("mapCipherSuites: Skipping TLS 1.3 cipher suite %q — TLS 1.3 suites are always enabled and cannot be configured in Go", name)
		} else {
			log.Warnf("mapCipherSuites: Ignoring unrecognized cipher suite %q (only Go/IANA TLS 1.2 names from tls.CipherSuites() are accepted)", name)
		}
	}
	if len(applied) > 0 {
		log.Infof("mapCipherSuites: TLS config supported cipher suites %s", strings.Join(applied, ":"))
	}
	return ids
}

// tlsGroupToID maps the NooBaa TLSGroup constants to Go tls.CurveID values.
// TODO: When ODF is updated to include the new TLSGroups, we should switch to using the
// ODF supported tls groups
var tlsGroupToID = map[nbv1.TLSGroup]tls.CurveID{
	nbv1.TLSGroupX25519:             tls.X25519,
	nbv1.TLSGroupSecp256r1:          tls.CurveP256,
	nbv1.TLSGroupSecp384r1:          tls.CurveP384,
	nbv1.TLSGroupSecp521r1:          tls.CurveP521,
	nbv1.TLSGroupX25519MLKEM768:     tls.X25519MLKEM768,
	nbv1.TLSGroupSecP256r1MLKEM768:  4587, // tls.SecP256r1MLKEM768 in Go 1.26+
	nbv1.TLSGroupSecP384r1MLKEM1024: 4589, // tls.SecP384r1MLKEM1024 in Go 1.26+
}

// mapGroupPreferences converts a list of NooBaa TLSGroup values to the corresponding
// tls.CurveID slice for use in tls.Config.CurvePreferences.
func mapGroupPreferences(groups []nbv1.TLSGroup, log *logrus.Entry) []tls.CurveID {
	var ids []tls.CurveID
	var applied []string
	for _, g := range groups {
		if id, ok := tlsGroupToID[g]; ok {
			ids = append(ids, id)
			applied = append(applied, string(g))
		} else {
			log.Warnf("Ignoring unsupported TLS group %q", g)
		}
	}
	if len(applied) > 0 {
		log.Infof("Admission server TLS group preferences set to %s", strings.Join(applied, ":"))
	}
	return ids
}
