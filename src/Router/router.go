package Router

import (
	"fmt"
	"inference-platform/Logger"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var logger *log.Logger

func sendPromptFromServer(client *http.Client, prompt string) *http.Response {
	body := fmt.Sprintf(`{
    "model": "Qwen/Qwen2.5-0.5B-Instruct",
    "messages": [
      {
        "role": "user",
        "content": "%s"
      }
    ],
    "max_tokens": 100
  	}`, prompt)

	var reader io.Reader = strings.NewReader(body)

	resp, err := client.Post("http://localhost:8000/v1/chat/completions", "application/json", reader)

	if err != nil {
		logger.Fatalf("Failed to send POST %v", err)
	}

	return resp
}

func StartHTTPServer() error {
	var logFile *os.File
	var err error

	logger, logFile, err = Logger.CreateLogger("router.log")

	if err != nil {
		return err
	}

	defer logFile.Close()

	server := &http.Server{
		Addr:         ":8383",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	client := &http.Client{}

	ServeInference := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		prompt, err := io.ReadAll(r.Body)

		if err != nil {
			logger.Printf("[ERROR]: Failed to read user prompt: %v\n", err)
			return
		}

		logger.Println("Read user prompt")

		resp := sendPromptFromServer(client, string(prompt))

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			logger.Printf("[ERROR]: Failed to read model response: %v\n", err)
			return
		}

		logger.Println("Returning model response")
		logger.Println(string(body))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)

		_, err = w.Write(body)

		if err != nil {
			logger.Printf("[ERROR] Writing response: %v", err)
		}
	}

	http.HandleFunc("/request", ServeInference)

	logger.Println("Starting HTTP Server")
	logger.Println("Listening on port 8383")
	logger.Fatal(server.ListenAndServe())

	return nil
}
