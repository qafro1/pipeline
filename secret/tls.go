package secret

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"time"
)

// CertificateChain represents a full certificate chain with a root CA, a server and client certificate
// All values are in PEM format
type CertificateChain struct {
	CAKey      string `mapstructure:"caKey"`
	CACert     string `mapstructure:"caCert"`
	ServerKey  string `mapstructure:"serverKey"`
	ServerCert string `mapstructure:"serverCert"`
	ClientKey  string `mapstructure:"clientKey"`
	ClientCert string `mapstructure:"clientCert"`
}

// GenerateTLS generates ca, server, and client TLS certificates.
// hosts: Comma-separated hostnames and IPs to generate a certificate for
// validity: Duration that certificate is valid for, in Go Duration format
func GenerateTLS(hosts string, validity string) (*CertificateChain, error) {
	notBefore := time.Now()
	validityDuration, err := time.ParseDuration(validity)
	if err != nil {
		return nil, err
	}
	notAfter := notBefore.Add(validityDuration)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	caKeyBytes, err := keyToBytes(caKey)
	if err != nil {
		return nil, err
	}

	caCertTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Banzai Cloud"},
			CommonName:   "Root CA",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA: true,
	}

	caCert, err := x509.CreateCertificate(rand.Reader, &caCertTemplate, &caCertTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	caCertBytes, err := certToBytes(caCert)
	if err != nil {
		return nil, err
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serverKeyBytes, err := keyToBytes(serverKey)
	if err != nil {
		return nil, err
	}

	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	serverCertTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Banzai Cloud"},
			CommonName:   "Banzai Genereted Server Cert",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA: false,
	}
	for _, h := range strings.Split(hosts, ",") {
		if ip := net.ParseIP(h); ip != nil {
			serverCertTemplate.IPAddresses = append(serverCertTemplate.IPAddresses, ip)
		} else {
			serverCertTemplate.DNSNames = append(serverCertTemplate.DNSNames, h)
		}
	}

	serverCert, err := x509.CreateCertificate(rand.Reader, &serverCertTemplate, &caCertTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	serverCertBytes, err := certToBytes(serverCert)
	if err != nil {
		return nil, err
	}

	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	clientKeyBytes, err := keyToBytes(clientKey)
	if err != nil {
		return nil, err
	}

	clientCertTemplate := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(4),
		Subject: pkix.Name{
			Organization: []string{"Banzai Cloud"},
			CommonName:   "Banzai Genereted Client Cert",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA: false,
	}

	clientCert, err := x509.CreateCertificate(rand.Reader, &clientCertTemplate, &caCertTemplate, &clientKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	clientCertBytes, err := certToBytes(clientCert)
	if err != nil {
		return nil, err
	}

	cc := CertificateChain{
		CAKey:      string(caKeyBytes),
		CACert:     string(caCertBytes),
		ServerKey:  string(serverKeyBytes),
		ServerCert: string(serverCertBytes),
		ClientKey:  string(clientKeyBytes),
		ClientCert: string(clientCertBytes),
	}

	return &cc, nil
}

func keyToBytes(key *rsa.PrivateKey) ([]byte, error) {
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	buffer := bytes.NewBuffer(nil)
	if err := pem.Encode(buffer, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func certToBytes(certBytes []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(nil)
	if err := pem.Encode(buffer, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
