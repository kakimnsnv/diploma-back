// pkg/imaging/imaging.go
package imaging

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"time"

	"resty.dev/v3"
)

type Imaging struct {
	resty          *resty.Client
	classifyClient *resty.Client
}

func NewImaging(modelURL string, classificationURL string) *Imaging {
	return &Imaging{
		resty: resty.New().
			SetBaseURL(modelURL).
			SetTimeout(0).
			SetRetryCount(3).
			SetRetryWaitTime(2 * time.Second).
			SetRetryMaxWaitTime(10 * time.Second),
		classifyClient: resty.New().
			SetBaseURL(classificationURL).
			SetTimeout(0).
			SetRetryCount(3).
			SetRetryWaitTime(2 * time.Second).
			SetRetryMaxWaitTime(10 * time.Second),
	}
}

type ModelResponse struct {
	Status          string `json:"status"`
	Message         string `json:"message"`
	Shape           []int  `json:"shape"`
	ClassesDetected []int  `json:"classes_detected"`
	OutputFormat    string `json:"output_format"`
	OutputPath      string `json:"output_path"`
	NiftiPath       string `json:"nifti_path"`
	ImageBase64     string `json:"image_base64"`
}

type ClassificationResponse struct {
	Status             string    `json:"status"`
	Message            string    `json:"message"`
	PredictedClass     int       `json:"predicted_class"`
	PredictedClassName string    `json:"predicted_class_name"`
	ClassProbabilities []float64 `json:"class_probabilities"`
	ClassNames         []string  `json:"class_names"`
	Confidence         float64   `json:"confidence"`
}

// CallClassifier sends an image to the classification model
func (i *Imaging) CallClassifier(imageBytes []byte, filename string) (*ClassificationResponse, error) {
	res := &ClassificationResponse{}

	_, err := i.classifyClient.R().EnableTrace().
		SetResult(res).
		SetMultipartField("file", filename, "image/png", bytes.NewReader(imageBytes)).
		Post("")
	if err != nil {
		return nil, fmt.Errorf("classification request failed: %w", err)
	}

	if res.Status != "success" {
		return nil, fmt.Errorf("classification error: %s", res.Message)
	}

	return res, nil
}

// CallModel sends NII file to  model and gets the result
func (i *Imaging) CallModel(file *multipart.FileHeader) (*ModelResponse, error) {
	fileStream, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer fileStream.Close()

	res := &ModelResponse{}

	i.resty.R().EnableTrace().
		SetResult(&res).
		SetMultipartField("file", file.Filename, file.Header.Get("Content-Type"), fileStream).
		Post("")

	if res.Status != "success" {
		return nil, fmt.Errorf("model returned error: %s", res.Message)
	}

	return res, nil
}
