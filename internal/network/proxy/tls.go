package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CA manages the local certificate authority used for MITM inspection
type CA struct {
	cert     *x509.Certificate
	key      *ecdsa.PrivateKey
	tlsCert  tls.Certificate
	mu       sync.RWMutex
	cache    map[string]*tls.Certificate // hostname → leaf cert
	certDir  string
}

// newCA loads or generates the local CA from certDir
func newCA(certDir string) (*CA, error) {
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, err
	}
	caCertPath := filepath.Join(certDir, "ca.crt")
	caKeyPath := filepath.Join(certDir, "ca.key")

	ca := &CA{certDir: certDir, cache: make(map[string]*tls.Certificate)}

	// Load existing CA if present
	if _, err := os.Stat(caCertPath); err == nil {
		tlsCert, err := tls.LoadX509KeyPair(caCertPath, caKeyPath)
		if err != nil {
			return nil, err
		}
		ca.tlsCert = tlsCert
		ca.cert, err = x509.ParseCertificate(tlsCert.Certificate[0])
		if err != nil {
			return nil, err
		}
		ca.key = tlsCert.PrivateKey.(*ecdsa.PrivateKey)
		return ca, nil
	}

	// Generate new CA
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"O.W.A.S.A.K.A. Local CA"},
			CommonName:   "OWASAKA SIEM Root CA",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	// Persist
	if err := writePEM(caCertPath, "CERTIFICATE", certDER); err != nil {
		return nil, err
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	if err := writePEM(caKeyPath, "EC PRIVATE KEY", keyDER); err != nil {
		return nil, err
	}

	ca.cert, _ = x509.ParseCertificate(certDER)
	ca.key = key
	ca.tlsCert = tls.Certificate{Certificate: [][]byte{certDER}, PrivateKey: key, Leaf: ca.cert}
	return ca, nil
}

// certFor returns a cached or freshly-generated leaf cert for the given hostname
func (ca *CA) certFor(hostname string) (*tls.Certificate, error) {
	ca.mu.RLock()
	if c, ok := ca.cache[hostname]; ok {
		ca.mu.RUnlock()
		return c, nil
	}
	ca.mu.RUnlock()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: hostname},
		DNSNames:     []string{hostname},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		return nil, err
	}
	leaf, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}
	c := &tls.Certificate{Certificate: [][]byte{certDER}, PrivateKey: key, Leaf: leaf}

	ca.mu.Lock()
	ca.cache[hostname] = c
	ca.mu.Unlock()
	return c, nil
}

func writePEM(path, kind string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return pem.Encode(f, &pem.Block{Type: kind, Bytes: data})
}
