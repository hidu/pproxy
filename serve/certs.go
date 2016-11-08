package serve

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"log"
)

func newCaCert(caCert []byte, caKey []byte) (tls.Certificate, error) {
	ca, err := tls.X509KeyPair(caCert, caKey)
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

// getSslCert get user's caCert or use the default buildin
func getSslCert(caCertPath string, caKeyPath string) (ca tls.Certificate, err error) {
	if caCertPath == "" {
		caCert := Assest.GetContent("/res/private/client_cert.pem")
		caKey := Assest.GetContent("/res/private/server_key.pem")
		return newCaCert([]byte(caCert), []byte(caKey))
	}
	cert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return ca, err
	}
	key, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		return ca, err
	}
	return newCaCert(cert, key)
}
