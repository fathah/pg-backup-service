package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Config struct {
	PGHost       string
	PGPort       string
	PGUser       string
	PGPassword   string
	ZDriveKey    string
	ZDriveSecret string
	BackupPrefix string
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func loadConfig() Config {
	return Config{
		PGHost:       getEnv("PG_HOST", "localhost"),
		PGPort:       getEnv("PG_PORT", "5432"),
		PGUser:       getEnv("PG_USER", "postgres"),
		PGPassword:   os.Getenv("PG_PASSWORD"),
		ZDriveKey:    os.Getenv("ZDRIVE_KEY"),
		ZDriveSecret: os.Getenv("ZDRIVE_SECRET"),
		BackupPrefix: getEnv("BACKUP_PREFIX", "pg_backup"),
	}
}

func main() {
	config := loadConfig()

	if config.PGPassword == "" || config.ZDriveKey == "" || config.ZDriveSecret == "" {
		log.Fatal("PG_PASSWORD, ZDRIVE_KEY, and ZDRIVE_SECRET must be set")
	}

	timestamp := time.Now().Format("20060102_150405")

	// 1. Get list of databases (optional, can fallback to dumpall if discovery fails)
	databases, err := getDatabaseList(config)
	if err != nil {
		log.Printf("Warning: Database discovery failed: %v. Falling back to single pg_dumpall.", err)
		performSingleFullBackup(config, timestamp)
		return
	}

	log.Printf("Found %d databases: %v", len(databases), databases)

	// 2. Perform Global Backup (Roles/Users)
	globalFilename := fmt.Sprintf("%s_%s_globals.sql", config.BackupPrefix, timestamp)
	err = performBackupAndUpload(config, globalFilename, true, "")
	if err != nil {
		log.Printf("Error: Failed to backup global data: %v", err)
	}

	// 3. Perform Per-Database Backups
	for _, db := range databases {
		dbFilename := fmt.Sprintf("%s_%s_db_%s.sql", config.BackupPrefix, timestamp, db)
		err = performBackupAndUpload(config, dbFilename, false, db)
		if err != nil {
			log.Printf("Error: Failed to backup database %s: %v", db, err)
			continue
		}
	}

	log.Printf("Backup process completed.")
}

func performSingleFullBackup(config Config, timestamp string) {
	filename := fmt.Sprintf("%s_%s_full.sql", config.BackupPrefix, timestamp)
	err := performBackupAndUpload(config, filename, false, "") // empty db means dumpall
	if err != nil {
		log.Fatalf("Critical: Failed to perform fallback backup: %v", err)
	}
}

func performBackupAndUpload(config Config, filename string, isGlobals bool, dbName string) error {
	tmpPath := filepath.Join(os.TempDir(), filename)
	defer os.Remove(tmpPath)

	log.Printf("Processing %s...", filename)

	var err error
	if isGlobals {
		err = runPgDumpGlobals(config, tmpPath)
	} else if dbName == "" {
		err = runPgDumpAll(config, tmpPath)
	} else {
		err = runPgDump(config, dbName, tmpPath)
	}

	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	signedURL, err := getSignedURL(config, filename)
	if err != nil {
		return fmt.Errorf("signed URL failed: %w", err)
	}

	err = uploadToZDrive(signedURL, tmpPath, filename)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("Successfully uploaded: %s", filename)
	return nil
}

func getDatabaseList(config Config) ([]string, error) {
	// Query to list all databases except templates and system dbs
	query := "SELECT datname FROM pg_database WHERE datistemplate = false AND datname NOT IN ('postgres', 'information_schema', 'pg_catalog');"
	cmd := exec.Command("psql",
		"-h", config.PGHost,
		"-p", config.PGPort,
		"-U", config.PGUser,
		"-d", "postgres",
		"-t", "-A", "-c", query,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", config.PGPassword))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%v: %s", err, string(output))
	}

	lines := bytes.Split(bytes.TrimSpace(output), []byte("\n"))
	var dbs []string
	for _, line := range lines {
		if len(line) > 0 {
			dbs = append(dbs, string(line))
		}
	}

	return dbs, nil
}

func runPgDumpGlobals(config Config, outputPath string) error {
	cmd := exec.Command("pg_dumpall",
		"-h", config.PGHost,
		"-p", config.PGPort,
		"-U", config.PGUser,
		"--globals-only",
		"-f", outputPath,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", config.PGPassword))
	return runCmd(cmd)
}

func runPgDump(config Config, dbName, outputPath string) error {
	cmd := exec.Command("pg_dump",
		"-h", config.PGHost,
		"-p", config.PGPort,
		"-U", config.PGUser,
		"-d", dbName,
		"-f", outputPath,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", config.PGPassword))
	return runCmd(cmd)
}

func runPgDumpAll(config Config, outputPath string) error {
	cmd := exec.Command("pg_dumpall",
		"-h", config.PGHost,
		"-p", config.PGPort,
		"-U", config.PGUser,
		"-f", outputPath,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", config.PGPassword))
	return runCmd(cmd)
}

func runCmd(cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}

type SignedURLResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

func getSignedURL(config Config, filename string) (string, error) {
	url := fmt.Sprintf("https://ziqx.cc/api/drive/sign-url?filename=%s", filename)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("x-drive-key", config.ZDriveKey)
	req.Header.Set("x-drive-secret", config.ZDriveSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("signed URL request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var res SignedURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if !res.Success {
		return "", fmt.Errorf("signed URL request returned success=false: %s", res.Message)
	}

	return res.URL, nil
}

func uploadToZDrive(uploadURL, filePath, filename string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return err
	}
	writer.Close()

	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 1 * time.Hour} // Backups can be large
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
