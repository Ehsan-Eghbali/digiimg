package main

import (
	"fmt"
	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
	"image"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExtractTextWithEnglishAndNumbers performs OCR on the image and extracts specific text
func ExtractTextWithEnglishAndNumbers(imagePath string) (string, error) {
	img := gocv.IMRead(imagePath, gocv.IMReadColor)
	if img.Empty() {
		return "", fmt.Errorf("unable to read image: %s", imagePath)
	}
	defer img.Close()

	// Convert to grayscale
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Perform OCR
	client := gosseract.NewClient()
	defer client.Close()
	client.SetImage(imagePath)
	text, err := client.Text()
	if err != nil {
		return "", err
	}

	// Extract specific data
	words := strings.Fields(text)
	if len(words) > 3 {
		return words[len(words)-1], nil
	}
	return "", nil
}

// CompareImages computes SSIM between two images
func CompareImages(image1Path, image2Path string) (float64, error) {
	img1, err := ValidateImage(image1Path)
	if err != nil {
		return 0, err
	}
	defer img1.Close()

	img2, err := ValidateImage(image2Path)
	if err != nil {
		return 0, err
	}
	defer img2.Close()

	// Resize images to the same size
	size := image.Point{X: 300, Y: 300}
	img1Resized := gocv.NewMat()
	defer img1Resized.Close()
	img2Resized := gocv.NewMat()
	defer img2Resized.Close()

	gocv.Resize(img1, &img1Resized, size, 0, 0, gocv.InterpolationLinear)
	gocv.Resize(img2, &img2Resized, size, 0, 0, gocv.InterpolationLinear)

	// Compute SSIM
	ssimValue, err := gocv.SSIM(img1Resized.ToImage(), img2Resized.ToImage())
	if err != nil {
		return 0, err
	}
	return ssimValue, nil
}

// AnalyzeDirectory monitors a directory for new images and processes them
func AnalyzeDirectory(directory, referenceImagePath string) {
	processedFiles := make(map[string]bool)

	// Check if reference image exists
	_, err := ValidateImage(referenceImagePath)
	if err != nil {
		log.Printf("Reference image not found: %s", err)
		return
	}

	for {
		files, err := os.ReadDir(directory)
		if err != nil {
			log.Printf("Error reading directory: %s", err)
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			filename := file.Name()
			filePath := filepath.Join(directory, filename)

			// Ensure we check the correct key (filename) in processedFiles
			if processedFiles[filename] || !strings.HasSuffix(strings.ToLower(filename), ".jpg") {
				continue
			}

			// Mark the file as processed
			processedFiles[filename] = true

			go func(path string) {
				defer func() { processedFiles[filename] = true }() // Mark processed after handling
				similarity, err := CompareImages(path, referenceImagePath)
				if err != nil {
					log.Printf("Error comparing images: %s", err)
					return
				}

				if similarity > 0.8 {
					extractedText, err := ExtractTextWithEnglishAndNumbers(path)
					if err != nil {
						log.Printf("Error extracting text: %s", err)
						return
					}

					lines := strings.Split(extractedText, "\n")
					if len(lines) > 1 && len(lines[1]) == 12 {
						fmt.Println(lines[1])
					}
				}
			}(filePath)
		}

		time.Sleep(1 * time.Second)
	}
}

// ValidateImage ensures the image can be read and returns it
func ValidateImage(imagePath string) (gocv.Mat, error) {
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return gocv.NewMat(), fmt.Errorf("image not found: %s", imagePath)
	}

	img := gocv.IMRead(imagePath, gocv.IMReadGrayScale)
	if img.Empty() {
		return gocv.NewMat(), fmt.Errorf("unable to read image: %s", imagePath)
	}

	return img, nil
}

func main() {
	directoryToMonitor := "./img" // Replace with the directory path
	referenceImage := "img.jpg"

	AnalyzeDirectory(directoryToMonitor, referenceImage)
}
