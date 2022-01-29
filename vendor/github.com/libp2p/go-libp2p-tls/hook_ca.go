package libp2ptls

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"time"

	ic "github.com/libp2p/go-libp2p-core/crypto"
)

const CAID = "/tls-ca/1.0.0"

var (
	CAEnable bool
	CAConfig Config
)

type Config struct {
	CaFile   string
	CertFile string
	KeyFile  string
}

func EnableCAEncryption(caFile, certFile, keyFile string) {
	CAConfig = Config{
		CaFile:   caFile,
		CertFile: certFile,
		KeyFile:  keyFile,
	}
	CAEnable = true
}

func GenerateCertAndPool() (cert tls.Certificate, pool *x509.CertPool, err error) {
	cert, err = tls.LoadX509KeyPair(CAConfig.CertFile, CAConfig.KeyFile)
	if err != nil {
		return
	}

	certBytes, err := ioutil.ReadFile(CAConfig.CaFile)
	if err != nil {
		return
	}

	pool = x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(certBytes)
	if !ok {
		panic("failed to parse root certificate")
	}

	return
}

func PubKeyFromCAConfig(conf *tls.Config, chain []*x509.Certificate) (ic.PubKey, error) {
	if len(chain) < 1 {
		return nil, errors.New("less than one certificates in the chain")
	}

	// Code copy/pasted https://github.com/digitalbitbox/bitbox-wallet-app/blob/b04bd07852d5b37939da75b3555b5a1e34a976ee/backend/coins/btc/electrum/electrum.go#L76-L111
	opts := x509.VerifyOptions{
		Roots:         conf.ClientCAs,
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

	pubKey := &ECDSAPublicKey{
		pub: chain[0].PublicKey.(*ecdsa.PublicKey),
	}
	return pubKey, nil
}
