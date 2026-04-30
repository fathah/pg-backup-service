package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

type dumpMode int

const (
	dumpModeSingle dumpMode = iota
	dumpModeAll
	dumpModeGlobals
)

func main() {
	config := loadConfig()

	if config.PGPassword == "" || config.ZDriveKey == "" || config.ZDriveSecret == "" {
		log.Fatal("PG_PASSWORD, ZDRIVE_KEY, and ZDRIVE_SECRET must be set")
	}

	timestamp := time.Now().UTC().Format("20060102_150405")

	databases, err := getDatabaseList(config)
	if err != nil {
		log.Printf("Warning: database discovery failed: %v. Falling back to single pg_dumpall.", err)
		filename := fmt.Sprintf("%s_%s_full.sql.gz", config.BackupPrefix, timestamp)
		if err := performBackupAndUpload(config, filename, dumpModeAll, ""); err != nil {
			log.Fatalf("Critical: fallback backup failed: %v", err)
		}
		return
	}

	log.Printf("Found %d databases: %v", len(databases), databases)

	failures := 0

	globalFilename := fmt.Sprintf("%s_%s_globals.sql.gz", config.BackupPrefix, timestamp)
	if err := performBackupAndUpload(config, globalFilename, dumpModeGlobals, ""); err != nil {
		log.Printf("Error: failed to backup global data: %v", err)
		failures++
	}

	for _, db := range databases {
		dbFilename := fmt.Sprintf("%s_%s_db_%s.sql.gz", config.BackupPrefix, timestamp, safeFileSegment(db))
		if err := performBackupAndUpload(config, dbFilename, dumpModeSingle, db); err != nil {
			log.Printf("Error: failed to backup database %s: %v", db, err)
			failures++
			continue
		}
	}

	if failures > 0 {
		log.Fatalf("Backup completed with %d failure(s).", failures)
	}
	log.Printf("Backup process completed successfully.")
}

func performBackupAndUpload(config Config, filename string, mode dumpMode, dbName string) error {
	tmpPath := filepath.Join(os.TempDir(), filename)
	defer os.Remove(tmpPath)

	log.Printf("Processing %s...", filename)

	if err := dumpToFile(config, mode, dbName, tmpPath); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	signedURL, err := getSignedURL(config, filename)
	if err != nil {
		return fmt.Errorf("signed URL failed: %w", err)
	}

	if err := uploadWithRetry(signedURL, tmpPath, filename, 3); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("Successfully uploaded: %s", filename)
	return nil
}

func dumpToFile(config Config, mode dumpMode, dbName, outputPath string) error {
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	gzWriter := gzip.NewWriter(out)

	var cmd *exec.Cmd
	switch mode {
	case dumpModeGlobals:
		cmd = exec.Command("pg_dumpall",
			"-h", config.PGHost, "-p", config.PGPort, "-U", config.PGUser,
			"-w", "--globals-only",
		)
	case dumpModeAll:
		cmd = exec.Command("pg_dumpall",
			"-h", config.PGHost, "-p", config.PGPort, "-U", config.PGUser, "-w",
		)
	case dumpModeSingle:
		cmd = exec.Command("pg_dump",
			"-h", config.PGHost, "-p", config.PGPort, "-U", config.PGUser,
			"-w", "-d", dbName,
		)
	default:
		return fmt.Errorf("unknown dump mode: %d", mode)
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", config.PGPassword))
	cmd.Stdout = gzWriter
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		gzWriter.Close()
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	if err := gzWriter.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}
	return nil
}

func getDatabaseList(config Config) ([]string, error) {
	query := "SELECT datname FROM pg_database WHERE datistemplate = false AND datname <> 'postgres';"
	cmd := exec.Command("psql",
		"-h", config.PGHost,
		"-p", config.PGPort,
		"-U", config.PGUser,
		"-d", "postgres",
		"-w",
		"-t", "-A", "-c", query,
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", config.PGPassword))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, stderr.String())
	}

	lines := bytes.Split(bytes.TrimSpace(stdout.Bytes()), []byte("\n"))
	var dbs []string
	for _, line := range lines {
		if len(line) > 0 {
			dbs = append(dbs, string(line))
		}
	}
	return dbs, nil
}

var unsafePathChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func safeFileSegment(s string) string {
	return unsafePathChars.ReplaceAllString(s, "_")
}

type SignedURLResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	URL     string `json:"url"`
}

func getSignedURL(config Config, filename string) (string, error) {
	date := time.Now().UTC().Format("20060102")
	url := fmt.Sprintf("https://ziqx.cc/api/drive/sign-url?filename=%s&folder=%s", filename, date)
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

func uploadWithRetry(uploadURL, filePath, filename string, attempts int) error {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		err := uploadToZDrive(uploadURL, filePath, filename)
		if err == nil {
			return nil
		}
		lastErr = err
		if i < attempts {
			backoff := time.Duration(1<<uint(i-1)) * 5 * time.Second
			log.Printf("Upload attempt %d/%d failed: %v. Retrying in %s...", i, attempts, err, backoff)
			time.Sleep(backoff)
		}
	}
	return lastErr
}

func uploadToZDrive(uploadURL, filePath, filename string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	pr, pw := io.Pipe()
	mpWriter := multipart.NewWriter(pw)

	go func() {
		var copyErr error
		defer func() {
			if copyErr != nil {
				pw.CloseWithError(copyErr)
				return
			}
			if err := mpWriter.Close(); err != nil {
				pw.CloseWithError(err)
				return
			}
			pw.Close()
		}()

		part, err := mpWriter.CreateFormFile("file", filename)
		if err != nil {
			copyErr = err
			return
		}
		if _, err := io.Copy(part, file); err != nil {
			copyErr = err
			return
		}
	}()

	req, err := http.NewRequest("POST", uploadURL, pr)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mpWriter.FormDataContentType())

	client := &http.Client{Timeout: 1 * time.Hour}
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
