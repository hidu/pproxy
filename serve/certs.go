package serve

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
)

func NewCaCert(ca_cert []byte, ca_key []byte) (tls.Certificate, error) {
	ca, err := tls.X509KeyPair(ca_cert, ca_key)
	if err != nil {
		log.Println("NewCaCert error:", err)
		return ca, err
	}
	if ca.Leaf, err = x509.ParseCertificate(ca.Certificate[0]); err != nil {
		log.Println("NewCaCert error:", err)
		return ca, err
	}
	log.Println("NewCaCert Ok")
	return ca, nil
}

func getSslCert(ca_cert_path string, ca_key_path string) (ca tls.Certificate, err error) {
	if ca_cert_path == "" {
		ca_cert := Assest.GetContent("/res/private/cert.pem")
		ca_key := Assest.GetContent("/res/private/key.pem")
		return NewCaCert([]byte(ca_cert), []byte(ca_key))
	}
	cert, err := ioutil.ReadFile(ca_cert_path)
	if err != nil {
		return ca, err
	}
	key, err := ioutil.ReadFile(ca_key_path)
	if err != nil {
		return ca, err
	}
	return NewCaCert(cert, key)
}
