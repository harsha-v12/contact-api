package handlers

import (
	"contact-management/config"
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)
var upgrader =websocket.Upgrader{
	CheckOrigin: func (r *http.Request) bool {
		return true // allow the frontend to connect 
	},
}
func ImportProgressWS(c echo.Context)error{
	importID:=c.Param("id")

	// 1. Upgrade the standard HTTP request to a WebSocket connection
	ws,err:=upgrader.Upgrade(c.Response(),c.Request(),nil)
	if err!=nil{
		return err 
	}
	defer ws.Close()

	ctx:=context.Background()

	// 2. subcribe to the exact redis channel created one 
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
		if msg.Payload==`{"status":"completed"}`{
			break 
		}
	}
	return  nil
}