package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "time"

    "github.com/gorilla/websocket"
)

const (
    websocketURI   = "ws://parser:6003/ws"
    reconnectDelay = 5 * time.Second
)

func connectToServer(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            log.Println("Attempting to connect to the server...")
            conn, _, err := websocket.DefaultDialer.Dial(websocketURI, nil)
            if err != nil {
                log.Printf("Error connecting to the server: %v", err)
                time.Sleep(reconnectDelay)
                continue
            }
            log.Println("Connected to the server")

            for {
                _, message, err := conn.ReadMessage()
                if err != nil {
                    log.Printf("Connection closed, attempting to reconnect: %v", err)
                    break
                }

                var data map[string]interface{}
                if err := json.Unmarshal(message, &data); err != nil {
                    log.Printf("Error parsing message: %v", err)
                    continue
                }

                log.Printf("Received data for %d matches", len(data))
                // Process the received data here
                // For example, print the data:
                log.Println(data)
            }

            conn.Close()
            log.Printf("Reconnecting in %v seconds...", reconnectDelay)
            time.Sleep(reconnectDelay)
        }
    }
}

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle OS signals to gracefully shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt)
    go func() {
        <-sigChan
        cancel()
    }()

    connectToServer(ctx)
}