package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
		envSetScript.WriteString(fmt.Sprintf("unset FIRSTUPDOTENV_FILE\n"))
	}

	return envSetScript.String(), nil
}

func loadEnvFile(directory string) (string, error) {
	filename := filepath.Join(directory, nameDotEnv)
	if _, err := os.Stat(filename); err != nil {
		return "", nil
	}

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
// It ignores empty lines and lines starting with # and lines without an equals sign.
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
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return env, scanner.Err()
}
