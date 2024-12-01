package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"syscall"

	"test_task_app/config"
	"test_task_app/service"

	"github.com/gorilla/websocket"
)

const (
	Live = "Live"
	// capChannelMatchData >= conf matches_per_batch
	capChannelMatchData = 100
)
// Global variables
var (
	g_chanMatchesData = make(chan map[string]interface{}, capChannelMatchData)
)

func main() {
	cfg := config.NewConfig()

	var requestSemaphore = make(chan struct{}, cfg.MatchesPerBatch)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	createDirForData(cfg.PathToData)

	for _, sportMode := range cfg.SportsToParse {
		go service.UpdateMatches(ctx, cfg, requestSemaphore, g_chanMatchesData, sportMode)
	}

	http.HandleFunc("/ws", websocketHandler)
	server := &http.Server{
		Addr: fmt.Sprintf("%s:%d", cfg.Websocket.Host, cfg.Websocket.Port),
	}

	go func() {
		log.Printf("WebSocket server started on %s:%d", cfg.Websocket.Host, cfg.Websocket.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down server...")
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited")
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Println("New client connected")
	broadcastData(conn)
	log.Println("Client disconnected")
}

func broadcastData(conn *websocket.Conn) {

	for {
		data, err := json.Marshal(<-g_chanMatchesData)

		if err != nil {
			log.Printf("Error marshalling data: %v", err)
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("WebSocket error: %v", err)
			return
		}

		time.Sleep(5 * time.Second)
	}
}

func createDirForData(dirName string) {
	_, err := os.Stat(dirName)
	if os.IsNotExist(err) {
		errDir := os.Mkdir(dirName, 0755)
		if errDir != nil {
			log.Fatal("Could not create directory")
		}
		log.Println("Directory created:", dirName)
	} else {
		log.Println("Directory already exists:", dirName)
	}
}