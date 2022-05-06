package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
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

func run(args []string) (string, error) {
	cmd := exec.Command("aws", args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func readCache(cacheFile string) (string, error) {
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

func writeCache(cacheFile string, token string) error {
	return os.WriteFile(cacheFile, []byte(token), 0600)
}

func cacheFile(args, environ []string) string {
	var hashKeys []string
	for _, env := range environ {
		if strings.HasPrefix(env, "AWS_") {
			hashKeys = append(hashKeys, env)
		}
	}
	sort.Strings(hashKeys)
	hashKeys = append(hashKeys, args...)
	hash := sha256.Sum256([]byte(strings.Join(hashKeys, "\x00")))
	return fmt.Sprintf("%s/.aws/eks-iam-cache-%x.json", os.Getenv("HOME"), hash)
}

func main() {
	args := os.Args[1:]
	cache := cacheFile(args, os.Environ())
	cached, err := readCache(cache)
	if err == nil {
		fmt.Print(cached)
		return
	}
	fmt.Fprintln(os.Stderr, err)

	token, err := run(args)
	if err != nil {
		if err != nil {
			_, err := run([]string{"sso", "login"})
			if err != nil {
				panic(err)
			}
		}
		token, err = run(args)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println(token)
	if err := writeCache(cache, token); err != nil {
		panic(err)
	}
}
