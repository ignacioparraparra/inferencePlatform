package main

import (
	"inference-platform/LoadBalancer"
	"inference-platform/Logger"
	"inference-platform/PodControlPlane"
	"log"
	"os"
	"sync"
)

var logger *log.Logger

func main() {
	var logFile *os.File

	logger, logFile, _ = Logger.CreateLogger("main.log")

	defer logFile.Close()

	var wg sync.WaitGroup

	wg.Add(2)

	var pc = PodControlPlane.NewPodController()
	var lb = LoadBalancer.Registry{}

	go func() {
		logger.Println("Starting docker client")
		defer wg.Done()
		go pc.StartDockerClient()

		<-pc.AddrsSet
		logger.Println("CLOSED CHANNEL")
		pc.UpdateRegistry(&lb)
	}()

	// go func() {
	// 	logger.Println("Starting HTTP server")
	// 	defer wg.Done()
	// 	Router.StartHTTPServer()
	// }()

	wg.Wait()
}
