package libp2ptls

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	ic "github.com/libp2p/go-libp2p-core/crypto"
)

var (
	caEnable bool
	caConfig struct {
		caFile    string
		certFile  string
		keyFile   string
		tlsConfig tls.Config
	}
)

func EnableCAEncryption(caFile, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	certBytes, err := ioutil.ReadFile(caFile)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(certBytes)
	if !ok {
		return fmt.Errorf("failed to parse root certificate")
	}
	caConfig.tlsConfig = tls.Config{
		MinVersion:               tls.VersionTLS13,
		PreferServerCipherSuites: preferServerCipherSuites(),
		// for server
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		Certificates: []tls.Certificate{cert},
		// for client
		// client need to skip hostname verify
		RootCAs:            pool,
		InsecureSkipVerify: true, // Not actually skipping, we check the cert in VerifyPeerCertificate
		VerifyPeerCertificate: func(_ [][]byte, _ [][]*x509.Certificate) error {
			panic("tls config not specialized for peer")
		},
		NextProtos:             []string{alpn},
		SessionTicketsDisabled: true,
	}
	caConfig.caFile = caFile
	caConfig.certFile = certFile
	caConfig.keyFile = keyFile
	caEnable = true
	return nil
}

func pubKeyFromCertChainAndTLSConfig(chain []*x509.Certificate) (ic.PubKey, error) {
	// Code copy/pasted https://github.com/digitalbitbox/bitbox-wallet-app/blob/b04bd07852d5b37939da75b3555b5a1e34a976ee/backend/coins/btc/electrum/electrum.go#L76-L111
	opts := x509.VerifyOptions{
		Roots:         caConfig.tlsConfig.ClientCAs,
		CurrentTime:   time.Now(),
		DNSName:       "", // <- skip hostname verification
		Intermediates: x509.NewCertPool(),
	}

	for i, cert := range chain {
		if i == 0 {
			continue
		}
		opts.Intermediates.AddCert(cert)
	}
	_, err := chain[0].Verify(opts)
	if err != nil {
		return nil, err
	}
	pKey, ok := chain[0].PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not a *ecdsa.PublicKey")
	}

	privKey := ecdsa.PrivateKey{PublicKey: *pKey}
	_, pubKey, err := ic.ECDSAKeyPairFromKey(&privKey)
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}
