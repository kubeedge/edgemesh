package libp2ptls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/libp2p/go-libp2p-core/crypto/pb"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/minio/sha256-simd"
)

// Identity is used to generate secure config for connection
type Identity struct {
	config tls.Config
}

func NewIdentity(cert tls.Certificate, certPoll *x509.CertPool) (*Identity, error) {
	return &Identity{
		config: tls.Config{
			Certificates: []tls.Certificate{cert},
			// for server
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  certPoll,
			// for client
			// client need to skip hostname verify
			RootCAs:            certPoll,
			InsecureSkipVerify: true, // Not actually skipping, we check the cert in VerifyPeerCertificate
		},
	}, nil
}

func (i *Identity) ConfigForAny() (*tls.Config, <-chan libp2pcrypto.PubKey) {
	return i.ConfigForPeer("", "")
}

func (i *Identity) ConfigForPeer(remote peer.ID, addr string) (*tls.Config, <-chan libp2pcrypto.PubKey) {
	keyCh := make(chan libp2pcrypto.PubKey, 1)
	conf := i.config.Clone()

	// fetch the public key from the certs
	conf.VerifyPeerCertificate = func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		defer close(keyCh)

		chain := make([]*x509.Certificate, len(rawCerts))
		for i := 0; i < len(rawCerts); i++ {
			cert, err := x509.ParseCertificate(rawCerts[i])
			if err != nil {
				return err
			}
			chain[i] = cert
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
			return err
		}

		tmp := chain[0].PublicKey.(*ecdsa.PublicKey)
		pubKey := &ECDSAPublicKey{
			pub: tmp,
		}
		if remote != "" && !remote.MatchesPublicKey(pubKey) {
			peerID, err := peer.IDFromPublicKey(pubKey)
			if err != nil {
				peerID = peer.ID(fmt.Sprintf("(not determined: %s)", err.Error()))
			}
			return fmt.Errorf("peer IDs don't match: expected %s, got %s", remote, peerID)
		}
		keyCh <- pubKey
		return nil
	}
	return conf, keyCh
}

// RsaPublicKey is an rsa public key
type RsaPublicKey struct {
	k rsa.PublicKey
}

// Verify compares a signature against input data
func (pk *RsaPublicKey) Verify(data, sig []byte) (bool, error) {
	hashed := sha256.Sum256(data)
	err := rsa.VerifyPKCS1v15(&pk.k, crypto.SHA256, hashed[:], sig)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (pk *RsaPublicKey) Type() pb.KeyType {
	return pb.KeyType_RSA
}

// Bytes returns protobuf bytes of a public key
func (pk *RsaPublicKey) Bytes() ([]byte, error) {
	return libp2pcrypto.MarshalPublicKey(pk)
}

func (pk *RsaPublicKey) Raw() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(&pk.k)
}

// Equals checks whether this key is equal to another
func (pk *RsaPublicKey) Equals(k libp2pcrypto.Key) bool {
	// make sure this is an rsa public key
	other, ok := (k).(*RsaPublicKey)
	if !ok {
		return basicEquals(pk, k)
	}

	return pk.k.N.Cmp(other.k.N) == 0 && pk.k.E == other.k.E
}

func basicEquals(k1, k2 libp2pcrypto.Key) bool {
	if k1.Type() != k2.Type() {
		return false
	}

	a, err := k1.Raw()
	if err != nil {
		return false
	}
	b, err := k2.Raw()
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ECDSAPublicKey is an implementation of an ECDSA public key
type ECDSAPublicKey struct {
	pub *ecdsa.PublicKey
}

// Bytes returns the public key as protobuf bytes
func (ePub *ECDSAPublicKey) Bytes() ([]byte, error) {
	return libp2pcrypto.MarshalPublicKey(ePub)
}

// Type returns the key type
func (ePub *ECDSAPublicKey) Type() pb.KeyType {
	return pb.KeyType_ECDSA
}

// Raw returns x509 bytes from a public key
func (ePub *ECDSAPublicKey) Raw() ([]byte, error) {
	return x509.MarshalPKIXPublicKey(ePub.pub)
}

// Equals compares to public keys
func (ePub *ECDSAPublicKey) Equals(o libp2pcrypto.Key) bool {
	return basicEquals(ePub, o)
}

// Verify compares data to a signature
func (ePub *ECDSAPublicKey) Verify(data, sigBytes []byte) (bool, error) {
	return true, nil
}
