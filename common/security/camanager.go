// copy and update from https://github.com/kubeedge/kubeedge/blob/c77318a1c3a97d63bcc3d5fd6cf4607d5df939ff/edge/pkg/edgehub/certificate/certmanager.go

package security

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	nethttp "net/http"
	"strings"

	"github.com/libp2p/go-libp2p-core/crypto"
	"k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"

	"github.com/kubeedge/edgemesh/common/util"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/certutil"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub/common/http"
)

const (
	retry = 3
)

type caManager struct {
	CR *x509.CertificateRequest

	caFile   string
	certFile string
	keyFile  string

	token string

	caURL   string
	certURL string
}

func init() {
	constructMap[TypeWithCA] = newCAManager
}

// newCAManager creates a caManager for edge security management according to EdgeHub config
func newCAManager(tunnel Security) Manager {
	certReq := &x509.CertificateRequest{
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Organization: []string{"kubeEdge"},
			Locality:     []string{"Hangzhou"},
			Province:     []string{"Zhejiang"},
			CommonName:   "kubeedge.io",
		},
	}
	return &caManager{
		token:    tunnel.Token,
		CR:       certReq,
		caFile:   tunnel.TLSCAFile,
		certFile: tunnel.TLSCertFile,
		keyFile:  tunnel.TLSPrivateKeyFile,
		caURL:    tunnel.HTTPServer + constants.DefaultCAURL,
		certURL:  tunnel.HTTPServer + constants.DefaultCertURL,
	}
}

// Start starts the caManager
func (m *caManager) Start() {
	_, err := m.getCurrentCert()
	if err != nil {
		klog.Infof("security cert, key file are no exist, start generate and fetch")
		err = m.applyCerts()
		if err != nil {
			klog.Fatalf("Error: %v", err)
		}
	} else {
		klog.Infof("security cert, key file already exist, skip generate")
	}
}

func (m *caManager) Name() string {
	return "caManager"
}

// getCurrentCert returns current edge certificate
func (m *caManager) getCurrentCert() (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(m.certFile, m.keyFile)
	if err != nil {
		return nil, err
	}
	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("cert.Certificate size is zero")
	}
	certs, err := x509.ParseCertificates(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate data: %w", err)
	}
	cert.Leaf = certs[0]
	return &cert, nil
}

// applyCerts realizes the certificate application by token
func (m *caManager) applyCerts() error {
	// validate the CA certificate by hashcode
	// token's format: <sha256.Sum256(cacert)>.<jwt token>, and jwt token has three parts split by '.'
	tokenParts := strings.Split(m.token, ".")
	if len(tokenParts) != 4 {
		return fmt.Errorf("token credentials are in the wrong format, token: %s", m.token)
	}

	cacert, err := GetCACert(m.caURL)
	if err != nil {
		return fmt.Errorf("failed to get CA certificate: %w", err)
	}

	hash, newHash, ok := ValidateCACerts(cacert, tokenParts[0])
	if !ok {
		return fmt.Errorf("failed to validate CA certificate. tokenCAhash: %s, CAhash: %s", hash, newHash)
	}

	// save the ca.crt to file
	ca, err := x509.ParseCertificate(cacert)
	if err != nil {
		return fmt.Errorf("failed to parse the CA certificate, error: %w", err)
	}

	if err = certutil.WriteCert(m.caFile, ca); err != nil {
		return fmt.Errorf("failed to save the CA certificate to file: %s, error: %v", m.caFile, err)
	}

	// get the edge.crt
	caPem := pem.EncodeToMemory(&pem.Block{Bytes: cacert, Type: cert.CertificateBlockType})
	pk, edgeCert, err := m.GetNodeCert(m.certURL, caPem, tls.Certificate{}, strings.Join(tokenParts[1:], "."))
	if err != nil {
		return fmt.Errorf("failed to get edge certificate from the cloudcore: %w", err)
	}

	// save the edge.crt to the file
	crt, _ := x509.ParseCertificate(edgeCert)
	if err = certutil.WriteKeyAndCert(m.keyFile, m.certFile, pk, crt); err != nil {
		return fmt.Errorf("failed to save the edge key and certificate to file: %s, error: %v", m.certFile, err)
	}

	return nil
}

// generateKey realizes the security application by token
func (m *caManager) generateKey() error {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}
	// save the private key to the file
	if err = certutil.WriteKey(m.keyFile, pk); err != nil {
		return fmt.Errorf("failed to save the private key %s, error: %v", m.keyFile, err)
	}
	return nil
}

// getCA returns the CA in pem format.
func (m *caManager) getCA() ([]byte, error) {
	return ioutil.ReadFile(m.caFile)
}

// GetNodeCert applies for the certificate from cloudcore
func (m *caManager) GetNodeCert(url string, capem []byte, cert tls.Certificate, token string) (*ecdsa.PrivateKey, []byte, error) {
	pk, csr, err := m.getCSR()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get CSR: %v", err)
	}

	client, err := http.NewHTTPClientWithCA(capem, cert)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create http client with CA:%w", err)
	}

	nodeName := util.FetchNodeName()

	usages := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
	req, err := buildRequest(nethttp.MethodGet, url, bytes.NewReader(csr), token, nodeName, usages)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate http request:%w", err)
	}

	var res *nethttp.Response
	for i := 0; i < retry; i++ {
		res, err = http.SendRequest(req, client)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode != 200 {
		// we just fetch 50 chars in content, because the content may have huge size
		contentMsg := string(content)
		if len(content) > 50 {
			contentMsg = string(content)[:50]
		}
		return nil, nil, fmt.Errorf("status code %d, content: %s", res.StatusCode, contentMsg)
	}

	return pk, content, nil
}

func (m *caManager) getCSR() (*ecdsa.PrivateKey, []byte, error) {
	pk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, m.CR, pk)
	if err != nil {
		return nil, nil, err
	}

	return pk, csr, nil
}

func (m *caManager) GetPrivateKey() (crypto.PrivKey, error) {
	certBytes, err := ioutil.ReadFile(m.keyFile)
	if err != nil {
		return nil, fmt.Errorf("read file %s err: %v", m.keyFile, err)
	}

	block, _ := pem.Decode(certBytes)

	privatekey, err := crypto.UnmarshalECDSAPrivateKey(block.Bytes)
	if err != nil {
		return privatekey, fmt.Errorf("failed to unmarshal private key: %w", err)
	}
	return privatekey, nil
}

// ValidateCACerts validates the CA certificate by hash code
func ValidateCACerts(cacerts []byte, hash string) (string, string, bool) {
	if len(cacerts) == 0 && hash == "" {
		return "", "", true
	}

	newHash := hashCA(cacerts)
	return hash, newHash, hash == newHash
}

func hashCA(cacerts []byte) string {
	digest := sha256.Sum256(cacerts)
	return hex.EncodeToString(digest[:])
}

// GetCACert gets the cloudcore CA certificate
func GetCACert(url string) ([]byte, error) {
	client := http.NewHTTPClient()
	req, err := http.BuildRequest("GET", url, nil, "", "")
	if err != nil {
		return nil, err
	}
	res, err := http.SendRequest(req, client)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	caCert, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return caCert, nil
}

// BuildRequest Creates a HTTP request.
func buildRequest(method string, urlStr string, body io.Reader, token string, nodeName string,
	usages []x509.ExtKeyUsage) (*nethttp.Request, error) {
	req, err := nethttp.NewRequest(method, urlStr, body)
	if err != nil {
		return nil, err
	}
	if token != "" {
		bearerToken := "Bearer " + token
		req.Header.Add("Authorization", bearerToken)
	}
	if nodeName != "" {
		req.Header.Add("NodeName", nodeName)
	}
	if usages != nil {
		usagesStr, err := json.Marshal(&usages)
		if err != nil {
			return nil, err
		}
		req.Header.Add("ExtKeyUsages", string(usagesStr))
	}
	return req, nil
}
