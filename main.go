package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/karrick/golf"
	log "github.com/sirupsen/logrus"
)

// example secret ID for CCP: appId/safe/folder/object/property
type ccpConfig struct {
	url      string
	query    string
	property string
}

func loadConfig() (string, string, string, error) {
	var err error
	url, exists := os.LookupEnv("CYBERARK_CCP_URL")

	if exists == false {
		err = errors.New("Environment variable 'CYBERARK_CCP_URL' is not present and is mandatory")
	}

	clientCert := os.Getenv("CYBERARK_CCP_CLIENT_CERT")
	clientKey := os.Getenv("CYBERARK_CCP_CLIENT_KEY")

	log.Debugf("CCP URL: %s", url)

	return url, clientCert, clientKey, err
}

func parseSecretId(secretId string) (string, string, error) {
	vars := strings.SplitN(secretId, "/", 2)
	if len(vars) != 2 {
		return secretId, "", errors.New(fmt.Sprintf("Failed to parse secret id '%s'. The secret id should look like: AppID=app-name&Query=Safe=safeName;UserName=appUsername/UserName", secretId))
	}
	urlQuery, property := vars[0], vars[1]
	return urlQuery, property, nil
}

func constructSecretUrl(url string, urlQuery string) string {
	url = fmt.Sprintf("%s/AIMWebService/api/Accounts?%s", url, urlQuery)
	// Currently only replace space in url for URL encdoing, looking for a better method
	url = strings.Replace(url, " ", "%20", -1)
	return url
}

func sendHttpRequest(url string, useClientCert bool, cert tls.Certificate) ([]byte, error) {
	// Setup HTTPS client
	tlsConfig := &tls.Config{}

	// Use correct TLS version
	tlsConfig.Renegotiation = tls.RenegotiateOnceAsClient

	// Set client cert if using it
	if useClientCert {
		tlsConfig.Certificates = []tls.Certificate{cert}
		tlsConfig.BuildNameToCertificate()
	}

	// Insecure Skip Verify
	ignore := strings.ToLower(os.Getenv("CYBERARK_CCP_IGNORE_CERT"))
	if ignore == "yes" || ignore == "true" {
		tlsConfig.InsecureSkipVerify = true
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	// Send the http request
	client := &http.Client{Transport: tr}
	log.Debugf("Url: %s", url)
	resp, err := client.Get(url)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return streamToByte(resp.Body), fmt.Errorf("Invalid response from CCP. Status Code: %s", resp.Status)
	}
	body := streamToByte(resp.Body)
	return body, err
}

func streamToByte(stream io.Reader) []byte {
	buf := new(bytes.Buffer)
	buf.ReadFrom(stream)
	return buf.Bytes()
}

func parseSecretProperty(body []byte, propertyKey string) (string, error) {
	jsonMap := make(map[string]interface{})
	err := json.Unmarshal(body, &jsonMap)

	if err != nil {
		return "", err
	}

	if val, ok := jsonMap[propertyKey]; ok {
		return fmt.Sprintf("%s", val), err
	}

	return "", fmt.Errorf("Failed to parse secret property '%s' from the response", propertyKey)
}

func RetrieveSecret(variableName string) {
	// Load environment variables and needed config
	url, clientCert, clientKey, err := loadConfig()
	if err != nil {
		log.Errorf("Failed loading CCP environment variables: %s\n", err)
		os.Exit(1)
	}

	// If using client certificate make sure to load
	cert := tls.Certificate{}
	useClientCert := false
	if clientCert != "" && clientKey != "" {
		cert, err = tls.LoadX509KeyPair(clientCert, clientKey)
		if err != nil {
			log.Fatal(err)
		}
		useClientCert = true
	}

	urlQuery, property, err := parseSecretId(variableName)
	if err != nil {
		log.Errorf("Failed to parse secret id: %s", err)
		os.Exit(1)
	}
	url = constructSecretUrl(url, urlQuery)

	body, err := sendHttpRequest(url, useClientCert, cert)
	if err != nil {
		log.Errorf("Failed to send http request to CCP. %s\n", err)
		os.Exit(1)
	}

	value, err := parseSecretProperty(body, property)
	if err != nil {
		log.Errorf("Failed to parse property from the response. %s\n", err)
	}

	os.Stdout.Write([]byte(value))
}

func main() {
	var help = golf.BoolP('h', "help", false, "show help")
	var verbose = golf.BoolP('v', "verbose", false, "be verbose")

	golf.Parse()
	args := golf.Args()

	if *help {
		golf.Usage()
		os.Exit(0)
	}
	if len(args) == 0 {
		golf.Usage()
		os.Exit(1)
	}

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true, DisableLevelTruncation: true})
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}

	RetrieveSecret(args[0])
}
