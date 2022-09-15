package gateway

import (
	"crypto/tls"
	"fmt"

	apiv1alpha3 "istio.io/api/networking/v1alpha3"
	v1 "k8s.io/api/core/v1"
)

var (
	tlsVersionMaps = map[apiv1alpha3.ServerTLSSettings_TLSProtocol]uint16{
		apiv1alpha3.ServerTLSSettings_TLSV1_0: tls.VersionTLS10,
		apiv1alpha3.ServerTLSSettings_TLSV1_1: tls.VersionTLS11,
		apiv1alpha3.ServerTLSSettings_TLSV1_2: tls.VersionTLS12,
		apiv1alpha3.ServerTLSSettings_TLSV1_3: tls.VersionTLS13,
	}

	tlsCipherSuitesMaps = map[string]uint16{
		// TLS 1.0 - 1.2 cipher suites.
		"TLS_RSA_WITH_RC4_128_SHA":                      tls.TLS_RSA_WITH_RC4_128_SHA,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":                 tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA":                  tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                  tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA256":               tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":              tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA":                tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,

		// TLS 1.3 cipher suites.
		"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,

		// Legacy names for the corresponding cipher suites with the correct _SHA256
		// suffix, retained for backward compatibility.
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	}
)

func transformTLSVersion(istioTLSVersion apiv1alpha3.ServerTLSSettings_TLSProtocol) uint16 {
	tlsVersion, ok := tlsVersionMaps[istioTLSVersion]
	if !ok {
		return tls.VersionTLS10
	}
	return tlsVersion
}

func transformTLSCipherSuites(istioCipherSuites []string) (cipherSuites []uint16) {
	for _, c := range istioCipherSuites {
		if cs, ok := tlsCipherSuitesMaps[c]; ok {
			cipherSuites = append(cipherSuites, cs)
		}
	}
	return
}

func getTLSCertAndKey(s v1.Secret) ([]byte, []byte, []byte, error) {
	if s.Type != "kubernetes.io/tls" {
		return nil, nil, nil, fmt.Errorf("secret %s not tls secret", s.Name)
	}
	if s.Data == nil {
		return nil, nil, nil, fmt.Errorf("secret %s data is empty", s.Name)
	}
	tlsCrt, ok := s.Data["tls.crt"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("tls cert not found in secret %s data", s.Name)
	}
	tlsKey, ok := s.Data["tls.key"]
	if !ok {
		return nil, nil, nil, fmt.Errorf("tls key not found in secret %s data", s.Name)
	}
	rootCA, ok := s.Data["ca.crt"]
	if !ok {
		return tlsCrt, tlsKey, nil, nil
	}
	return tlsCrt, tlsKey, rootCA, nil
}
