// pkg/imaging/imaging.go
package imaging

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

// ConvertToNii converts an image to NII format using Python script
func ConvertToNii(imagePath string) (string, error) {
	// Generate output path
	outputPath := filepath.Join("/tmp", fmt.Sprintf("%s.nii", uuid.New().String()))

	// Get Python script path
	scriptPath := os.Getenv("PYTHON_CONVERTER_SCRIPT")
	if scriptPath == "" {
		scriptPath = "scripts/convert_to_nii.py"
	}

	// Get Python executable
	pythonExec := os.Getenv("PYTHON_EXECUTABLE")
	if pythonExec == "" {
		pythonExec = "python3"
	}

	// Execute Python script
	cmd := exec.Command(pythonExec, scriptPath, imagePath, outputPath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("conversion failed: %s - %s", err.Error(), stderr.String())
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("conversion output file not created")
	}

	return outputPath, nil
}

// ConvertNiiToImage converts NII back to image format
func ConvertNiiToImage(niiPath string, outputFormat string) (string, error) {
	if outputFormat == "" {
		outputFormat = "png"
	}

	outputPath := filepath.Join("/tmp", fmt.Sprintf("%s.%s", uuid.New().String(), outputFormat))

	scriptPath := os.Getenv("PYTHON_CONVERTER_SCRIPT")
	if scriptPath == "" {
		scriptPath = "scripts/convert_to_nii.py"
	}

	pythonExec := os.Getenv("PYTHON_EXECUTABLE")
	if pythonExec == "" {
		pythonExec = "python3"
	}

	cmd := exec.Command(pythonExec, scriptPath, niiPath, outputPath, "--reverse")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("conversion failed: %s - %s", err.Error(), stderr.String())
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("conversion output file not created")
	}

	return outputPath, nil
}

// CallModel sends NII file to your model and gets the result
func CallModel(inputNiiPath string) (string, error) {
	return inputNiiPath, nil // TODO: remove this line when implementing the function

	modelURL := os.Getenv("MODEL_URL")
	if modelURL == "" {
		return "", fmt.Errorf("MODEL_URL not set in environment")
	}

	// Open the input file
	file, err := os.Open(inputNiiPath)
	if err != nil {
		return "", fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(inputNiiPath))
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	writer.Close()

	// Send request to model
	req, err := http.NewRequest("POST", modelURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("model returned error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Save the output NII file
	outputPath := filepath.Join("/tmp", fmt.Sprintf("%s_output.nii", uuid.New().String()))

	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save output file: %w", err)
	}

	return outputPath, nil
}

// ValidateImageFile checks if file is a valid image
func ValidateImageFile(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read first 512 bytes to detect file type
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return err
	}

	contentType := http.DetectContentType(buffer)

	if contentType != "image/jpeg" && contentType != "image/png" {
		return fmt.Errorf("invalid file type: %s, only JPEG and PNG allowed", contentType)
	}

	return nil
}
