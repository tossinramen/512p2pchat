package main

import (
	"encoding/base64"
	"os"
)

func encodeImageToBase64(filePath string) (string, error) {
	file, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(file), nil
}

func decodeBase64ToImage(encoded string, outputPath string) error {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}