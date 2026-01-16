package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/1password/onepassword-sdk-go"
)

const (
	name             = "firstupdotenv"
	nameDotEnv       = "firstup.env"
	currentSetEnvVar = "FIRSTUPDOTENV_CURRENT_SET_ENV"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix(name + ": ")

	env, err := createEnvSourceFromCurrentDir()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(env)
}

func createEnvSourceFromCurrentDir() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	var envSetScript strings.Builder

	// Always remove the old settings, even if we don't find a new one.
	old := os.Getenv(currentSetEnvVar)
	if old != "" {
		oldKeys := strings.Split(old, ",")
		for _, key := range oldKeys {
			envSetScript.WriteString(fmt.Sprintf("unset %s\n", key))
		}
	}

	var envFromFile string

	for {
		if strings.Count(directory, string(os.PathSeparator)) < 2 {
			// Stop before the root directory.
			break
		}

		envFromFile, err = loadEnvFile(directory)
		if err != nil {
			return "", err
		}

		if envFromFile != "" {
			break
		}

		// Walk up one directory.
		directory = filepath.Dir(directory)
	}

	if envFromFile != "" {
		envSetScript.WriteString(envFromFile)
		envSetScript.WriteString(fmt.Sprintf("export FIRSTUPDOTENV_FILE=%s\n", filepath.Join(directory, nameDotEnv)))
	} else {
		envSetScript.WriteString(fmt.Sprintf("unset %s\n", currentSetEnvVar))
		envSetScript.WriteString(fmt.Sprintln("unset FIRSTUPDOTENV_FILE"))
	}

	return envSetScript.String(), nil
}

func loadEnvFile(directory string) (string, error) {
	filename := filepath.Join(directory, nameDotEnv)
	if _, err := os.Stat(filename); err != nil {
		return "", nil
	}

	envm, opRefs, err := parseEnvFile(filename)
	if err != nil {
		return "", err
	}

	// Resolve 1Password references in bulk if any.
	if len(opRefs) > 0 {
		secrets, err := resolveOnePasswordSecrets(opRefs)
		if err != nil {
			return "", err
		}
		for _, secret := range secrets {
			for k, v := range parseSecretContent(secret) {
				envm[k] = v
			}
		}
	}

	if len(envm) == 0 {
		return "", nil
	}

	var envSetScript strings.Builder

	var keys []string
	for k, v := range envm {
		os.Setenv(k, v)
		keys = append(keys, k)
	}
	envSetScript.WriteString(fmt.Sprintf("export %s=%s\n", currentSetEnvVar, strings.Join(keys, ",")))

	for k, v := range envm {
		envSetScript.WriteString(fmt.Sprintf("export %s=%q\n", k, v))
	}

	return envSetScript.String(), nil
}

// parseEnvFile loads environment variables from text file on the form key=value.
// It ignores empty lines and lines starting with #.
// Lines starting with op:// are collected as 1Password references.
func parseEnvFile(filename string) (map[string]string, []string, error) {
	fi, err := os.Stat(filename)
	if err != nil || fi.IsDir() {
		return nil, nil, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	env := make(map[string]string)
	var opRefs []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "op://") {
			opRefs = append(opRefs, line)
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return env, opRefs, scanner.Err()
}

// resolveOnePasswordSecrets resolves multiple 1Password references in bulk using the SDK.
// It uses desktop app authentication (Touch ID, etc.) via the OP_ACCOUNT environment variable.
func resolveOnePasswordSecrets(refs []string) ([]string, error) {
	account := os.Getenv("OP_ACCOUNT")
	if account == "" {
		return nil, fmt.Errorf("OP_ACCOUNT environment variable must be set to your 1Password account name")
	}

	client, err := onepassword.NewClient(
		context.Background(),
		onepassword.WithDesktopAppIntegration(account),
		onepassword.WithIntegrationInfo(name, "v1.0.0"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create 1Password client: %w", err)
	}

	result, err := client.Secrets().ResolveAll(context.Background(), refs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve 1Password secrets: %w", err)
	}

	var secrets []string
	for ref, resp := range result.IndividualResponses {
		if resp.Error != nil {
			return nil, fmt.Errorf("failed to resolve secret %q: %v", ref, resp.Error.Type)
		}
		secrets = append(secrets, resp.Content.Secret)
	}

	return secrets, nil
}

// parseSecretContent parses KEY=value lines from a secret content string.
func parseSecretContent(content string) map[string]string {
	env := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return env
}
