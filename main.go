package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	name                     = "firstupdotenv"
	nameDotEnv               = "firstup.env"
	currentSetEnvVar         = "FIRSTUPDOTENV_CURRENT_SET_ENV"
	firstUpDotEnvFilenameVar = "FIRSTUPDOTENV_FILE"
	firstUpDotEnvFileHashVar = "FIRSTUPDOTENV_FILE_HASH"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix(name + ": ")

	env, err := createEnvSourceFromCurrentDir()
	if err != nil {
		log.Fatal(err)
	}
	if env != "" {
		fmt.Println(env)
	}
}

func createEnvSourceFromCurrentDir() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	var (
		envFromFile                   string
		firsUptDotEnvFilename         string
		contentHash                   string
		existingFirstUpDotEnvFileHash = os.Getenv(firstUpDotEnvFileHashVar)
	)

	for {
		if strings.Count(directory, string(os.PathSeparator)) < 2 {
			// Stop before the root directory.
			break
		}
		firsUptDotEnvFilename = filepath.Join(directory, nameDotEnv)
		contentHash, err = hashEnvFileContent(firsUptDotEnvFilename)
		if err == nil {
			if existingFirstUpDotEnvFileHash != "" && existingFirstUpDotEnvFileHash == contentHash {
				// No changes to the env file, skip reloading.
				return "", nil
			}
			envFromFile, err = loadEnvFile(firsUptDotEnvFilename)
			if err != nil {
				return "", err
			}
		}

		if envFromFile != "" {
			break
		}

		// Walk up one level.
		directory = filepath.Dir(directory)
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

	if envFromFile != "" {
		envSetScript.WriteString(envFromFile)
		envSetScript.WriteString(fmt.Sprintf("export %s=%s\n", firstUpDotEnvFilenameVar, firsUptDotEnvFilename))
		envSetScript.WriteString(fmt.Sprintf("export %s=%s\n", firstUpDotEnvFileHashVar, contentHash))
	} else {
		envSetScript.WriteString(fmt.Sprintf("unset %s\n", currentSetEnvVar))
		envSetScript.WriteString(fmt.Sprintf("unset %s\n", firstUpDotEnvFilenameVar))
		envSetScript.WriteString(fmt.Sprintf("unset %s\n", firstUpDotEnvFileHashVar))
	}

	return envSetScript.String(), nil
}

func hashEnvFileContent(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func loadEnvFile(filename string) (string, error) {
	envm, err := parseEnvFile(filename)
	if err != nil {
		return "", err
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
		envSetScript.WriteString(fmt.Sprintf("export %s=%s\n", k, v))
	}

	return envSetScript.String(), nil
}

// parseEnvFile loads environment variables from text file on the form key=value.
// It ignores empty lines and lines starting with #.
// Lines starting with op:// are treated as 1Password references and loaded via `op read`.
func parseEnvFile(filename string) (map[string]string, error) {
	fi, err := os.Stat(filename)
	if err != nil || fi.IsDir() {
		return nil, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "op://") {
			opEnv, err := readFromOnePassword(line)
			if err != nil {
				return nil, err
			}
			for k, v := range opEnv {
				env[k] = v
			}
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return env, scanner.Err()
}

// readFromOnePassword reads environment variables from a 1Password reference.
// The reference should be in the form op://vault/item/field.
// The field should contain line-separated KEY=value entries.
func readFromOnePassword(reference string) (map[string]string, error) {
	cmd := exec.Command("op", "read", reference)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("op read %s: %w: %s", reference, err, stderr.String())
	}

	env := make(map[string]string)
	scanner := bufio.NewScanner(&stdout)
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
	return env, scanner.Err()
}
