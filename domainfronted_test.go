package domainfronted

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/getlantern/keyman"
	"github.com/getlantern/proxytest"
	"github.com/getlantern/testify/assert"
)

const (
	expectedGoogleResponse = "Google is built by a large team of engineers, designers, researchers, robots, and others in many different sites across the globe. It is updated continuously, and built with more tools and technologies than we can shake a stick at. If you'd like to help us out, see google.com/careers.\n"
)

func TestUnit(t *testing.T) {
	server := &Server{
		Addr: "localhost:0",
		AllowNonGlobalDestinations: true,
		CertContext: &CertContext{
			PKFile:         "testpk.pem",
			ServerCertFile: "testcert.pem",
		},
	}
	l, err := server.Listen()
	if err != nil {
		t.Fatalf("Unable to listen: %s", err)
	}
	go func() {
		err = server.Serve(l)
		if err != nil {
			t.Fatalf("Unable to serve: %s", err)
		}
	}()

	addrParts := strings.Split(l.Addr().String(), ":")
	host := addrParts[0]
	port, err := strconv.Atoi(addrParts[1])
	if err != nil {
		t.Fatalf("Unable to parse port: %s", err)
	}
	client := NewClient(&ClientConfig{
		Host:               host,
		Port:               port,
		InsecureSkipVerify: true,
	})
	defer client.Close()

	proxytest.Go(t, client.Dial)
}

// TestIntegration tests against existing domain-fronted servers running on
// CloudFlare.
func TestIntegration(t *testing.T) {
	rootCAs, err := keyman.PoolContainingCerts("-----BEGIN CERTIFICATE-----\nMIIDdTCCAl2gAwIBAgILBAAAAAABFUtaw5QwDQYJKoZIhvcNAQEFBQAwVzELMAkG\nA1UEBhMCQkUxGTAXBgNVBAoTEEdsb2JhbFNpZ24gbnYtc2ExEDAOBgNVBAsTB1Jv\nb3QgQ0ExGzAZBgNVBAMTEkdsb2JhbFNpZ24gUm9vdCBDQTAeFw05ODA5MDExMjAw\nMDBaFw0yODAxMjgxMjAwMDBaMFcxCzAJBgNVBAYTAkJFMRkwFwYDVQQKExBHbG9i\nYWxTaWduIG52LXNhMRAwDgYDVQQLEwdSb290IENBMRswGQYDVQQDExJHbG9iYWxT\naWduIFJvb3QgQ0EwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDaDuaZ\njc6j40+Kfvvxi4Mla+pIH/EqsLmVEQS98GPR4mdmzxzdzxtIK+6NiY6arymAZavp\nxy0Sy6scTHAHoT0KMM0VjU/43dSMUBUc71DuxC73/OlS8pF94G3VNTCOXkNz8kHp\n1Wrjsok6Vjk4bwY8iGlbKk3Fp1S4bInMm/k8yuX9ifUSPJJ4ltbcdG6TRGHRjcdG\nsnUOhugZitVtbNV4FpWi6cgKOOvyJBNPc1STE4U6G7weNLWLBYy5d4ux2x8gkasJ\nU26Qzns3dLlwR5EiUWMWea6xrkEmCMgZK9FGqkjWZCrXgzT/LCrBbBlDSgeF59N8\n9iFo7+ryUp9/k5DPAgMBAAGjQjBAMA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8E\nBTADAQH/MB0GA1UdDgQWBBRge2YaRQ2XyolQL30EzTSo//z9SzANBgkqhkiG9w0B\nAQUFAAOCAQEA1nPnfE920I2/7LqivjTFKDK1fPxsnCwrvQmeU79rXqoRSLblCKOz\nyj1hTdNGCbM+w6DjY1Ub8rrvrTnhQ7k4o+YviiY776BQVvnGCv04zcQLcFGUl5gE\n38NflNUVyRRBnMRddWQVDf9VMOyGj/8N7yy5Y0b2qvzfvGn9LhJIZJrglfCm7ymP\nAbEVtQwdpf5pLGkkeB6zpxxxYu7KyJesF12KwvhHhm4qxFYxldBniYUr+WymXUad\nDKqC5JlR3XC321Y9YeRq4VzW9v493kHMB65jUr9TU/Qr6cf9tveCX4XSQRjbgbME\nHMUfpIBvFSDJ3gyICh3WZlXi/EjJKSZp4A==\n-----END CERTIFICATE-----\n")
	if err != nil {
		t.Fatalf("Unable to set up cert pool")
	}

	client := NewClient(&ClientConfig{
		Host: "roundrobin.getiantem.org",
		Port: 443,
		Masquerades: []*Masquerade{
			&Masquerade{
				Domain: "100partnerprogramme.de",
			},
			&Masquerade{
				Domain: "10minutemail.com",
			},
		},
		RootCAs: rootCAs,
	})
	defer client.Close()

	hc := &http.Client{
		Transport: &http.Transport{
			Dial: client.Dial,
		},
	}

	resp, err := hc.Get("https://www.google.com/humans.txt")
	if err != nil {
		t.Fatalf("Unable to fetch from Google: %s", err)
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Unable to read response from Google: %s", err)
	}
	assert.Equal(t, expectedGoogleResponse, string(b), "Didn't get expected response from Google")
}