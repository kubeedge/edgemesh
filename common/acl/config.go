package acl

type TunnelACLConfig struct {
	// TLSCAFile set ca file path
	TLSCAFile string `json:"tlsCaFile,omitempty"`
	// TLSCertFile indicates the file containing x509 Certificate for HTTPS
	TLSCertFile string `json:"tlsCertFile,omitempty"`
	// TLSPrivateKeyFile indicates the file containing x509 private key matching tlsCertFile
	TLSPrivateKeyFile string `json:"tlsPrivateKeyFile,omitempty"`
	// Token indicates the token of joining the cluster for the edge
	Token string `json:"token"`
	// HTTPServer indicates the server for edge to apply for the certificate.
	HTTPServer string `json:"httpServer,omitempty"`
}
