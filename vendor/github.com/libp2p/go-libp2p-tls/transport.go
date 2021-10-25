package libp2ptls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"sync"

	libp2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/sec"
)

// ID is the protocol ID
const ID = "/tls-ca/1.0.0"

type Config struct {
	caFile   string
	certFile string
	keyFile  string
}

var config Config

type Transport struct {
	identity  *Identity
	localPeer peer.ID
	privKey   libp2pcrypto.PrivKey
}

func Init(caFile, certFile, keyFile string) {
	config = Config{
		caFile:   caFile,
		certFile: certFile,
		keyFile:  keyFile,
	}
}

func New(key libp2pcrypto.PrivKey) (*Transport, error) {
	id, err := peer.IDFromPrivateKey(key)
	if err != nil {
		return nil, err
	}
	t := &Transport{
		localPeer: id,
		privKey:   key,
	}

	var cert tls.Certificate
	cert, err = tls.LoadX509KeyPair(config.certFile, config.keyFile)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var clientCertPool *x509.CertPool
	caCertBytes, err := ioutil.ReadFile(config.caFile)
	if err != nil {
		panic("unable to read client.pem")
	}
	clientCertPool = x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(caCertBytes)
	if !ok {
		panic("failed to parse root certificate")
	}

	identity, err := NewIdentity(cert, clientCertPool)
	if err != nil {
		return nil, err
	}
	t.identity = identity
	return t, nil
}

// SecureInbound runs the TLS handshake as a server.
func (t *Transport) SecureInbound(ctx context.Context, insecure net.Conn) (sec.SecureConn, error) {
	config, keyCh := t.identity.ConfigForAny()
	cs, err := t.handshake(ctx, tls.Server(insecure, config), keyCh)
	if err != nil {
		insecure.Close()
		log.Printf("transport handshake error %s", err.Error())
	}
	return cs, err
}

// SecureOutbound runs the TLS handshake as a client
func (t *Transport) SecureOutbound(ctx context.Context, insecure net.Conn, p peer.ID) (sec.SecureConn, error) {
	addr := insecure.RemoteAddr().String()
	config, keyCh := t.identity.ConfigForPeer(p, addr)
	cs, err := t.handshake(ctx, tls.Client(insecure, config), keyCh)
	if err != nil {
		insecure.Close()
		log.Printf("transport handshake error %s", err.Error())
	}
	return cs, err
}

func (t *Transport) handshake(
	ctx context.Context,
	tlsConn *tls.Conn,
	keyCh <-chan libp2pcrypto.PubKey,
) (sec.SecureConn, error) {
	// There's no way to pass a context to tls.Conn.Handshake().
	// See https://github.com/golang/go/issues/18482.
	// Close the connection instead.
	select {
	case <-ctx.Done():
		tlsConn.Close()
	default:
	}

	done := make(chan struct{})
	var wg sync.WaitGroup

	// Ensure that we do not return before
	// either being done or having a context
	// cancellation.
	defer wg.Wait()
	defer close(done)

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-done:
		case <-ctx.Done():
			tlsConn.Close()
		}
	}()

	if err := tlsConn.Handshake(); err != nil {
		// if the context was canceled, return the context error
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}

	//Should be ready by this point, don't block.
	var remotePubKey libp2pcrypto.PubKey
	select {
	case remotePubKey = <-keyCh:
	default:
	}
	if remotePubKey == nil {
		return nil, errors.New("go-libp2p-tls BUG: expected remote pub key to be set")
	}

	conn, err := t.setupConn(tlsConn, remotePubKey)
	if err != nil {
		// if the context was canceled, return the context error
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	log.Printf("set up connection with remote %v", conn.RemoteAddr())
	return conn, nil
}

func (t *Transport) setupConn(tlsConn *tls.Conn, remotePubKey libp2pcrypto.PubKey) (sec.SecureConn, error) {
	remotePeerID, err := peer.IDFromPublicKey(remotePubKey)
	if err != nil {
		return nil, err
	}
	return &conn{
		Conn:         tlsConn,
		localPeer:    t.localPeer,
		privKey:      t.privKey,
		remotePeer:   remotePeerID,
		remotePubKey: remotePubKey,
	}, nil
}
