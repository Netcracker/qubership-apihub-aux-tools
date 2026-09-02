package main

import (
	"apihub-bulk-src-fix/internal/client"
	"apihub-bulk-src-fix/internal/file"
	"apihub-bulk-src-fix/internal/manifest"
	"apihub-bulk-src-fix/internal/tasks"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const workersCount = 10

var download = flag.NewFlagSet("download", flag.ExitOnError)
var upload = flag.NewFlagSet("upload", flag.ExitOnError)

var downloadIDsFilename = download.String("ids", "", "ids file")
var downloadURL = download.String("url", "", "url")
var downloadPAT = download.String("pat", "", "personal access token")
var downloadOutput = download.String("output", "./output", "output folder")

var uploadURL = upload.String("url", "", "url")
var uploadPAT = upload.String("pat", "", "personal access token")
var uploadOutput = upload.String("output", "./output", "output folder")

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: apihub-bulk-src-fix <command> [options]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "download":
		download.Parse(os.Args[2:])
		downloadCommand()

	case "upload":
		upload.Parse(os.Args[2:])
		uploadCommand()

	default:
		fmt.Printf("unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func downloadCommand() {
	c := client.NewClient(*downloadURL, *downloadPAT)

	sourceFileIds, err := file.ReadTasksFromFile(*downloadIDsFilename)
	if err != nil {
		log.Fatal(err)
	}

	downloadedDir := filepath.Join(*downloadOutput, "downloaded")
	extractedDir := filepath.Join(*downloadOutput, "extracted")
	manifestPath := filepath.Join(*downloadOutput, "manifest.json")

	if err := os.MkdirAll(*downloadOutput, 0755); err != nil {
		log.Fatal(err)
	}

	var mu sync.Mutex
	var downloadedTasks []tasks.Task

	tasks.RunWorkers(workersCount, sourceFileIds, func(task tasks.Task) {
		filename := folderName(task) + ".zip"

		// Download ZIP
		body, err := c.GetVersionSource(task.PackageId, task.Version)
		if err != nil {
			log.Printf(
				"ERROR downloading %s/%s: %v",
				task.PackageId,
				task.Version,
				err,
			)
			return
		}

		err = file.SaveZipIntoFolder(downloadedDir, filename, body)
		if err != nil {
			log.Printf(
				"ERROR saving %s/%s: %v",
				task.PackageId,
				task.Version,
				err,
			)
			return
		}

		// Extract ZIP
		zipPath := filepath.Join(downloadedDir, filename)
		extractPath := filepath.Join(extractedDir, folderName(task))

		err = file.Unzip(zipPath, extractPath)
		if err != nil {
			log.Printf(
				"ERROR extracting %s/%s: %v",
				task.PackageId,
				task.Version,
				err,
			)
			return
		}

		mu.Lock()
		downloadedTasks = append(downloadedTasks, task)
		mu.Unlock()

		log.Printf("Downloaded: %s/%s", task.PackageId, task.Version)
	})

	err = manifest.Save(manifestPath, downloadedTasks)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"Manifest saved: %s (%d of %d revisions)",
		manifestPath,
		len(downloadedTasks),
		len(sourceFileIds),
	)
}

func uploadCommand() {
	c := client.NewClient(*uploadURL, *uploadPAT)

	manifestPath := filepath.Join(*uploadOutput, "manifest.json")
	extractedDir := filepath.Join(*uploadOutput, "extracted")
	fixedDir := filepath.Join(*uploadOutput, "fixed")

	sourceFileIds, err := manifest.Read(manifestPath)
	if err != nil {
		log.Fatal(err)
	}

	tasks.RunWorkers(workersCount, sourceFileIds, func(task tasks.Task) {
		sourcePath := filepath.Join(extractedDir, folderName(task))
		zipPath := filepath.Join(fixedDir, folderName(task)+".zip")

		// Zip fixed sources
		err := file.Zip(sourcePath, zipPath)
		if err != nil {
			log.Printf(
				"ERROR zipping %s/%s: %v",
				task.PackageId,
				task.Version,
				err,
			)
			return
		}

		// Read ZIP
		zipArchive, err := file.UploadZipFile(zipPath)
		if err != nil {
			log.Printf(
				"ERROR reading %s/%s: %v",
				task.PackageId,
				task.Version,
				err,
			)
			return
		}

		// Replace sources
		err = c.UpdateVersionSource(
			task.PackageId,
			task.Version,
			zipArchive,
		)
		if err != nil {
			log.Printf(
				"ERROR uploading %s/%s: %v",
				task.PackageId,
				task.Version,
				err,
			)
			return
		}

		log.Printf("Uploaded: %s/%s", task.PackageId, task.Version)
	})
}

func folderName(task tasks.Task) string {
	return fmt.Sprintf("%s-%s", task.PackageId, task.Version)
}
