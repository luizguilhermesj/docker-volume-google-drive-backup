package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

const (
	backupDir      = "/backup"
	defaultCreds   = "/creds/credentials.json"
	tmpDir         = "/app/backup/tmp"
)

func compressFolder(src, dest string) error {
	log.Printf("[COMPRESS] Compressing %s to %s", src, dest)
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	defer gz.Close()
	tarWriter := tar.NewWriter(gz)
	defer tarWriter.Close()

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, path)
		if err != nil {
			return err
		}
		hdr.Name = filepath.Join(filepath.Base(src), relPath)
		if err := tarWriter.WriteHeader(hdr); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tarWriter, f); err != nil {
				return err
			}
		}
		return nil
	}
	return filepath.Walk(src, walkFn)
}

func uploadToDrive(srv *drive.Service, filePath, fileName, parentID string) (string, error) {
	log.Printf("[UPLOAD] Starting upload for %s", fileName)
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	file := &drive.File{Name: fileName}
	if parentID != "" {
		file.Parents = []string{parentID}
	} else {
		file.Parents = []string{"root"}
	}
	
	createCall := srv.Files.Create(file).
		SupportsAllDrives(true).
		Media(f).
		Fields("id")
	
	created, err := createCall.Do()
	if err != nil {
		return "", err
	}
	log.Printf("[UPLOAD] Finished upload for %s. File ID: %s", fileName, created.Id)
	return created.Id, nil
}

func cleanupOldBackups(srv *drive.Service, parentID string, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	log.Printf("[RETENTION] Checking for backups older than %d days (cutoff: %s)", retentionDays, cutoff.Format(time.RFC3339))
	debug := os.Getenv("DEBUG") != ""
	q := "name contains '.tar.gz' and trashed = false"
	if debug {
		log.Printf("[DEBUG] Retention query: %s", q)
	}
	filesList := srv.Files.List().Q(q).SupportsAllDrives(true).Fields("files(id, name, createdTime, parents)")
	if parentID != "" && parentID != "root" {
		filesList = filesList.DriveId(parentID).Corpora("drive").IncludeItemsFromAllDrives(true)
	}
	files, err := filesList.Do()
	if err != nil {
		return fmt.Errorf("listing files: %w", err)
	}
	if debug {
		log.Printf("[DEBUG] Found %d files for retention check", len(files.Files))
	}
	for _, f := range files.Files {
		if debug {
			log.Printf("[DEBUG] Checking file: %s (created %s) parents: %v", f.Name, f.CreatedTime, f.Parents)
		}
		created, err := time.Parse(time.RFC3339, f.CreatedTime)
		if err != nil {
			log.Printf("[RETENTION] Could not parse time for file %s: %v", f.Name, err)
			continue
		}
		if retentionDays == 0 || created.Before(cutoff) {
			log.Printf("[RETENTION] Deleting old backup: %s (created %s)", f.Name, f.CreatedTime)
			if err := srv.Files.Delete(f.Id).SupportsAllDrives(true).Do(); err != nil {
				log.Printf("[RETENTION] Error deleting %s: %v", f.Name, err)
			}
		}
	}
	return nil
}

func setTimezoneFromEnv() {
	tz := os.Getenv("TZ")
	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			log.Printf("[WARN] Could not load timezone %s: %v", tz, err)
			return
		}
		log.Printf("[INFO] Setting timezone to %s", tz)
		time.Local = loc
	}
}

func formatTimestampForFilename(t time.Time) string {
	// Use RFC3339 format as the base
	timestamp := t.Format(time.RFC3339)
	
	// Check if we should replace colons with dashes for filename compatibility
	if os.Getenv("FILENAME_SAFE_TIMESTAMP") == "true" {
		// Replace colons with dashes to make it safe for all filesystems
		timestamp = strings.ReplaceAll(timestamp, ":", "-")
	}
	
	return timestamp
}



func main() {
	setTimezoneFromEnv()
	log.Println("[INIT] Starting backup process")
	credsPath := os.Getenv("GOOGLE_CREDENTIALS")
	if credsPath == "" {
		credsPath = defaultCreds
	}
	parentID := os.Getenv("GDRIVE_FOLDER_ID")
	impersonateSubject := os.Getenv("GDRIVE_IMPERSONATE_SUBJECT")

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		log.Fatalf("[ERROR] Unable to create temp dir: %v", err)
	}

	folders, err := ioutil.ReadDir(backupDir)
	if err != nil {
		log.Fatalf("[ERROR] Failed to list backup dir: %v", err)
	}

	ctx := context.Background()
	b, err := ioutil.ReadFile(credsPath)
	if err != nil {
		log.Fatalf("[ERROR] Unable to read credentials: %v", err)
	}
	config, err := google.JWTConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("[ERROR] Unable to parse credentials: %v", err)
	}
	if impersonateSubject != "" {
		config.Subject = impersonateSubject
	}
	srv, err := drive.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx)))
	if err != nil {
		log.Fatalf("[ERROR] Unable to create Drive client: %v", err)
	}

	retentionDays := 30
	if v := os.Getenv("RETENTION_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			retentionDays = n
		}
	}

	if parentID == "" {
		parentID = "root"
	}
	if err := cleanupOldBackups(srv, parentID, retentionDays); err != nil {
		log.Printf("[RETENTION] Error during cleanup: %v", err)
	}

	for _, fi := range folders {
		if !fi.IsDir() {
			continue
		}
		folderName := fi.Name()
		folderPath := filepath.Join(backupDir, folderName)
		now := time.Now().In(time.Local)
		timestamp := formatTimestampForFilename(now)
		tarName := fmt.Sprintf("%s_%s.tar.gz", folderName, timestamp)
		tarPath := filepath.Join(tmpDir, tarName)
		if err := compressFolder(folderPath, tarPath); err != nil {
			log.Printf("[ERROR] Compressing %s: %v", folderPath, err)
			continue
		}
		if _, err := uploadToDrive(srv, tarPath, tarName, parentID); err != nil {
			log.Printf("[ERROR] Uploading %s: %v", tarPath, err)
			continue
		}
		if err := os.Remove(tarPath); err != nil {
			log.Printf("[ERROR] Deleting archive %s: %v", tarPath, err)
		} else {
			log.Printf("[CLEANUP] Deleted archive %s", tarPath)
		}
	}
	log.Println("[DONE] All folders processed.")
} 