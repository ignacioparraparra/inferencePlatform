package Logger

import (
	"log"
	"os"
)

func CreateLogger(filename string) (*log.Logger, *os.File, error) {
	os.MkdirAll("logs", 0755)

	logFile, err := os.OpenFile("logs/"+filename, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	logger := log.New(
		logFile,
		"",
		log.LstdFlags|log.Lshortfile,
	)

	log.Printf("%s logger initialized\n", filename)

	return logger, logFile, nil
}
