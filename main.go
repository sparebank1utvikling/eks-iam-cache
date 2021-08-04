package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// {"kind": "ExecCredential", "apiVersion": "client.authentication.k8s.io/v1alpha1", "spec": {}, "status": {"expirationTimestamp": "2021-08-04T07:57:20Z", "token": "k..."}}

type ExecCredential struct {
	Kind       string `json:"kind"`
	ApiVersion string `json:"apiVersion"`
	Status     struct {
		ExpirationTimestamp string `json:"expirationTimestamp"`
	} `json:"status"`
}

func run() (string, error) {
	cmd := exec.Command("aws", os.Args[1:]...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

var cacheFile = os.Getenv("HOME") + "/.aws/eks-iam-cache.json"

func readCache() (string, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return "", err
	}
	var creds ExecCredential
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", err
	}
	expires, err := time.Parse(time.RFC3339, creds.Status.ExpirationTimestamp)
	if err != nil {
		return "", err
	}
	if expires.Before(time.Now().Add(30 * time.Second)) {
		return "", errors.New("expiring/expired: " + creds.Status.ExpirationTimestamp)
	}
	return string(data), nil
}

func writeCache(token string) error {
	return os.WriteFile(cacheFile, []byte(token), 0600)
}

func main() {
	cached, err := readCache()
	if err == nil {
		fmt.Print(cached)
		return
	}
	fmt.Fprintln(os.Stderr, err)

	token, err := run()
	if err != nil {
		panic(err)
	}

	fmt.Println(token)
	if err := writeCache(token); err != nil {
		panic(err)
	}
}
