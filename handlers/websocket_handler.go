package handlers

import (
	"contact-management/config"
	"contact-management/services"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // allow the frontend to connect 
	},
}
func ImportProgressWS(c echo.Context)error{
	importID:=c.Param("id")
    println("WEBSOCKET HIT:", importID)

	// 1. Upgrade the standard HTTP request to a WebSocket connection
	ws,err:=upgrader.Upgrade(c.Response(),c.Request(),nil)
	if err!=nil{
		return err 
	}
	defer ws.Close()

	ctx:=context.Background()

	// NEW: Instantly fetch the current state from Redis and push it!
	// If the job finished before the user connected, this ensures they still see 100%!
	if initialJob, err := services.GetImportJob(ctx, importID); err == nil {
		initialJSON, _ := json.Marshal(initialJob)
		ws.WriteMessage(websocket.TextMessage, initialJSON)

		// If it's already completely done, we can just close the connection instantly
		if initialJob.Status == "completed" || initialJob.Status == "failed" {
			return nil
		}
	}

	// 2. subscribe to the exact redis channel created one 
	pubsub:=config.RedisClient.Subscribe(ctx,"import_channel:"+importID)
	defer pubsub.Close()

	ch:=pubsub.Channel()

	//keep the websocket open and wait for redis to braodcast!
	for msg:= range ch{
		// when redis shouts out the progress,instantly push it to the frontend
		err:=ws.WriteMessage(websocket.TextMessage,[]byte(msg.Payload))
		if err!=nil{
			break // if the user close the browser break the loop
		}

		// if the import is completed we can safely close the connection
		if strings.Contains(msg.Payload, `"status":"completed"`) || strings.Contains(msg.Payload, `"status":"failed"`) {
			break 
		}
	}
	return  nil
}