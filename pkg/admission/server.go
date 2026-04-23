package admission

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"

	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	"github.com/noobaa/noobaa-operator/v5/pkg/options"
	"github.com/noobaa/noobaa-operator/v5/pkg/util"
	ocstlsv1 "github.com/red-hat-storage/ocs-tls-profiles/api/v1"
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
// from disk and reading the NooBaa CR's spec.security.tlsConfig, then
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
	if err := applySecurityTLSConfig(cfg, log); err != nil {
		log.Errorf("Invalid spec.security.tlsConfig, TLS config not reloaded: %v", err)
		return err
	}
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

// applySecurityTLSConfig fetches the NooBaa CR and applies spec.security.tlsConfig
// to the given tls.Config when it is set.
func applySecurityTLSConfig(tlsConfig *tls.Config, log *logrus.Entry) error {
	noobaa := &nbv1.NooBaa{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "noobaa",
			Namespace: options.Namespace,
		},
	}
	if !util.KubeCheckQuiet(noobaa) {
		log.Info("NooBaa CR not found, using default TLS config for admission server")
		return nil
	}

	spec := noobaa.Spec.Security.TLSConfig
	if spec == nil {
		log.Info("spec.security.tlsConfig not set, using default TLS config for admission server")
		return nil
	}

	if spec.Version != "" {
		switch spec.Version {
		case ocstlsv1.VersionTLS1_2:
			tlsConfig.MinVersion = tls.VersionTLS12
			log.Info("Admission server TLS min version set to TLSv1.2")
		case ocstlsv1.VersionTLS1_3:
			tlsConfig.MinVersion = tls.VersionTLS13
			log.Info("Admission server TLS min version set to TLSv1.3")
		}
	}

	if len(spec.Ciphers) > 0 || len(spec.Groups) > 0 {
		ciphers, curves, err := util.GoCiphersAndCurvesFromTLSConfig(spec)
		if err != nil {
			return fmt.Errorf("spec.security.tlsConfig: %w", err)
		}
		tlsConfig.CipherSuites = ciphers
		tlsConfig.CurvePreferences = curves
	}
	return nil
}
