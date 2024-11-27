package main

import (
	"fmt"
	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
	"image"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SSIM calculates the Structural Similarity Index between two images
func SSIM(img1, img2 gocv.Mat) float64 {
	width := img1.Cols()
	height := img1.Rows()

	// Constants for SSIM calculation
	C1 := 6.5025
	C2 := 58.5225
	var sumL, sumC, sumS float64

	// Convert to grayscale images (to work with pixel values)
	img1Gray := gocv.NewMat()
	defer img1Gray.Close()
	img2Gray := gocv.NewMat()
	defer img2Gray.Close()

	gocv.CvtColor(img1, &img1Gray, gocv.ColorBGRToGray)
	gocv.CvtColor(img2, &img2Gray, gocv.ColorBGRToGray)

	// Calculate SSIM using pixel-wise comparison
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			I1 := float64(img1Gray.GetUCharAt(y, x))
			I2 := float64(img2Gray.GetUCharAt(y, x))

			// Luminance (mean of I1 and I2)
			luminance := I1 * I2
			// Contrast (standard deviation)
			contrast := math.Pow(I1, 2) + math.Pow(I2, 2)
			// Structure
			structure := I1 * I2

			// Sum values for SSIM calculation
			sumL += luminance
			sumC += contrast
			sumS += structure
		}
	}

	// SSIM formula
	L := (2*sumL + C1) / (sumC + C1)
	C := (2*sumC + C2) / (sumS + C2)

	// Return SSIM index
	ssimIndex := L * C
	return ssimIndex
}

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
	ssimValue := SSIM(img1Resized, img2Resized)
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
