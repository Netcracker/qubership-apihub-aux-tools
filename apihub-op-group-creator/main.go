package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	apiType          = "rest"
	listPath         = "/api/v2/packages/%s/versions/%s/%s/operations"
	createPath       = "/api/v3/packages/%s/versions/%s/%s/groups"
	updatePath       = "/api/v3/packages/%s/versions/%s/%s/groups/%s"
	deletePath       = "/api/v2/packages/%s/versions/%s/%s/groups/%s"
	exportPath       = "/api/v1/export"
	exportStatusPath = "/api/v1/export/%s/status"
	pageSize         = 100
	personalToken    = "X-Personal-Access-Token"
)

type Operation struct {
	OperationID string         `json:"operationId"`
	CustomTags  map[string]any `json:"customTags,omitempty"`
	PackageRef  string         `json:"packageRef"`
}

type OperationRef struct {
	OperationID string `json:"operationId"`
}

type ListResponse struct {
	Operations []Operation `json:"operations"`
}

type ExportRequest struct {
	ExportedEntity               string `json:"exportedEntity"`
	PackageID                    string `json:"packageId"`
	Version                      string `json:"version"`
	GroupName                    string `json:"groupName"`
	OperationsSpecTransformation string `json:"operationsSpecTransformation"`
	Format                       string `json:"format"`
	RemoveOasExtensions          bool   `json:"removeOasExtensions"`
}

type ExportStatusResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func main() {
	apihubUrl := flag.String("apihubURL", "", "Base URL of the Apihub instance")
	packageID := flag.String("packageId", "", "Package unique identifier (full alias)")
	version := flag.String("version", "", "Package version")
	groupName := flag.String("group", "", "Operation group name")
	apiKey := flag.String("token", "", "Personal API key")
	customTagKey := flag.String("x-key", "", "Custom tag key")
	customTagValue := flag.String("x-value", "", "Custom tag value")
	force := flag.Bool("force", false, "Recreate group if exists")
	outputFormat := flag.String("outputFormat", "yaml", "Export output format. Json or Yaml.")

	flag.Parse()

	if *apihubUrl == "" || *packageID == "" || *version == "" || *groupName == "" || *apiKey == "" ||
		*customTagKey == "" || *customTagValue == "" {
		fmt.Println("Missing required parameters")
		flag.Usage()
		os.Exit(1)
	}
	if *outputFormat != "yaml" && *outputFormat != "json" {
		fmt.Println("Invalid output format")
		flag.Usage()
		os.Exit(1)
	}

	// List all operations
	operations, err := listOperations(*apihubUrl, *packageID, *version, *apiKey)
	if err != nil {
		fmt.Printf("Error listing operations: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Operations count: %d\n", len(operations))

	// Filter operations by custom tag
	filteredOps := filterOperations(operations, *customTagKey, *customTagValue)
	fmt.Printf("Found %d operations matching conditions\n", len(filteredOps))

	if len(filteredOps) == 0 {
		fmt.Println("No operations matching criteria found, exiting")
		return
	}

	// Re-create group if required
	if *force {
		exists, err := groupExists(*apihubUrl, *packageID, *version, *groupName, *apiKey)
		if err != nil {
			fmt.Printf("Error checking group existence: %v\n", err)
			os.Exit(1)
		} else if exists {
			if err := deleteGroup(*apihubUrl, *packageID, *version, *groupName, *apiKey); err != nil {
				fmt.Printf("Error deleting group: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Existing group deleted")
		}
	}

	// Create new group
	if err := createGroup(*apihubUrl, *packageID, *version, *groupName, *apiKey); err != nil {
		fmt.Printf("Error creating group: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Group created successfully")

	// Update group with operations
	if err := updateGroupOperations(*apihubUrl, *packageID, *version, *groupName, filteredOps, *apiKey); err != nil {
		fmt.Printf("Error updating group: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Group updated with %d operations\n", len(filteredOps))

	// Start export
	exportId, err := startExport(*apihubUrl, *packageID, *version, *groupName, *apiKey, *outputFormat)
	if err != nil {
		fmt.Printf("Error starting export: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Export started, id:", exportId)

	// Wait for export and save result
	filePath := fmt.Sprintf("%s.%s", *groupName, *outputFormat)
	if err := waitAndSaveExport(*apihubUrl, exportId, *apiKey, filePath); err != nil {
		fmt.Printf("Error during export: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Export result saved to " + filePath)
}

func listOperations(apihubUrl, packageID, version, apiKey string) ([]Operation, error) {
	var allOps []Operation
	page := 0

	for {
		path := fmt.Sprintf(listPath, packageID, version, apiType)
		reqURL := fmt.Sprintf("%s%s?skipRefs=true&limit=%d&page=%d", apihubUrl, path, pageSize, page)

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set(personalToken, apiKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}

		var listResp ListResponse
		if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
			return nil, err
		}

		allOps = append(allOps, listResp.Operations...)
		if len(listResp.Operations) < pageSize {
			break
		}
		page++
	}
	return allOps, nil
}

func filterOperations(ops []Operation, customTagKey, customTagValue string) []Operation {
	var filtered []Operation
	for _, op := range ops {
		val, exists := op.CustomTags[customTagKey]
		if !exists {
			continue
		}

		var found bool
		switch v := val.(type) {
		case string:
			found = (v == customTagValue)
		case []string:
			for _, s := range v {
				if s == customTagValue {
					found = true
					break
				}
			}
		case []interface{}:
			for _, elem := range v {
				if s, ok := elem.(string); ok && s == customTagValue {
					found = true
					break
				}
			}
		}

		if found {
			filtered = append(filtered, op)
		}
	}
	return filtered
}

func groupExists(apihubUrl, packageID, version, groupName, apiKey string) (bool, error) {
	path := fmt.Sprintf("/api/v2/packages/%s/versions/%s/%s/groups/%s", packageID, version, apiType, url.PathEscape(groupName))
	reqURL := fmt.Sprintf("%s%s", apihubUrl, path)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set(personalToken, apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}

func deleteGroup(apihubUrl, packageID, version, groupName, apiKey string) error {
	path := fmt.Sprintf(deletePath, packageID, version, apiType, url.PathEscape(groupName))
	reqURL := fmt.Sprintf("%s%s", apihubUrl, path)

	req, err := http.NewRequest("DELETE", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set(personalToken, apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func createGroup(apihubUrl, packageID, version, groupName, apiKey string) error {
	path := fmt.Sprintf(createPath, packageID, version, apiType)
	reqURL := fmt.Sprintf("%s%s", apihubUrl, path)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("groupName", groupName); err != nil {
		return err
	}
	writer.Close()

	req, err := http.NewRequest("POST", reqURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set(personalToken, apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func updateGroupOperations(apihubUrl, packageID, version, groupName string, operations []Operation, apiKey string) error {
	path := fmt.Sprintf(updatePath, packageID, version, apiType, url.PathEscape(groupName))
	reqURL := fmt.Sprintf("%s%s", apihubUrl, path)

	// Prepare operations payload
	operationRefs := make([]OperationRef, len(operations))
	for i, op := range operations {
		operationRefs[i] = OperationRef{OperationID: op.OperationID}
	}

	operationsJSON, err := json.Marshal(operationRefs)
	if err != nil {
		return err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormField("operations")
	if err != nil {
		return err
	}
	part.Write(operationsJSON)
	writer.Close()

	req, err := http.NewRequest("PATCH", reqURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set(personalToken, apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

func startExport(apihubUrl, packageID, version, groupName, apiKey, outputFormat string) (string, error) {
	reqURL := fmt.Sprintf("%s%s", apihubUrl, exportPath)

	exportReq := ExportRequest{
		ExportedEntity:               "restOperationsGroup",
		PackageID:                    packageID,
		Version:                      version,
		GroupName:                    groupName,
		OperationsSpecTransformation: "reducedSourceSpecifications",
		Format:                       outputFormat,
		RemoveOasExtensions:          true,
	}

	body, err := json.Marshal(exportReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(personalToken, apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ExportID string `json:"exportId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ExportID, nil
}

func waitAndSaveExport(apihubUrl, exportId, apiKey, filePath string) error {
	const maxAttempts = 30
	const sleepDuration = 5 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		status, fileData, err := getExportStatus(apihubUrl, exportId, apiKey)
		if err != nil {
			return err
		}

		switch status {
		case "completed":
			if fileData != nil {
				return os.WriteFile(filePath, fileData, 0644)
			} else {
				return fmt.Errorf("export data is empty")
			}
		case "error":
			return fmt.Errorf("export failed")
		case "none":
			// just wait
		}

		time.Sleep(sleepDuration)
	}

	return fmt.Errorf("export timed out after %d attempts", maxAttempts)
}

func getExportStatus(apihubUrl, exportId, apiKey string) (string, []byte, error) {
	path := fmt.Sprintf(exportStatusPath, exportId)
	reqURL := fmt.Sprintf("%s%s", apihubUrl, path)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set(personalToken, apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", nil, fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, string(body))
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var statusResp ExportStatusResponse
		if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
			return "", nil, err
		}
		if statusResp.Status == "error" {
			return statusResp.Status, nil, fmt.Errorf("response message: %s", statusResp.Message)
		}

		return statusResp.Status, nil, nil
	}

	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}
	return "completed", fileData, nil
}
