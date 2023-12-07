package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
)

type Item struct {
	ID   string `json:"ID"`
	Name string `json:"Name"`
}

type CreateEvent struct {
	ItemID string `json:"itemID"`
	// Otros campos relevantes para la creación
}

type UpdateEvent struct {
	ItemID string `json:"itemID"`
	// Otros campos relevantes para la actualización
}

type DeleteEvent struct {
	ItemID string `json:"itemID"`
	// Otros campos relevantes para la eliminación
}

type contextKey string

func (c contextKey) String() string {
    return string(c)
}

var (
    ConnectionKey = contextKey("natsConnection")
	items []Item
)

func main() {
    log.Println("Starting the application...")
    var err error
    nc, err := nats.Connect(nats.DefaultURL)
    if err != nil {
        log.Fatal(err)
    }
	
    log.Println("Connected to NATS server.")
    defer nc.Close()
	
    // Subscribe event handlers
    log.Println("Subscribing to events...")
	if err := subscribeToCreateEvent(nc); err != nil {
		log.Printf("Error subscribing to create event: %v", err)
	}
	
	if err := subscribeToUpdateEvent(nc); err != nil {
		log.Printf("Error subscribing to update event: %v", err)
	}
	
	if err := subscribeToDeleteEvent(nc); err != nil {
		log.Printf("Error subscribing to delete event: %v", err)
	}

    router := mux.NewRouter()
    router.HandleFunc("/items", GetItems).Methods("GET")
    router.HandleFunc("/items/{id}", GetItem).Methods("GET")
	router.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(context.Background(), ConnectionKey, nc)
		CreateItem(ctx, w, r)
	}).Methods("POST")
    router.HandleFunc("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(context.Background(), ConnectionKey, nc)
		UpdateItem(ctx, w, r)
	}).Methods("PUT")
    router.HandleFunc("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(context.Background(), ConnectionKey, nc)
		DeleteItem(ctx, w, r)
	}).Methods("DELETE")

    log.Println("Starting server on port 8080...")
    log.Fatal(http.ListenAndServe(":8080", router))
}

func GetItems(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(items)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for _, item := range items {
		if item.ID == params["id"] {
			json.NewEncoder(w).Encode(item)
			return
		}
	}
	json.NewEncoder(w).Encode(&Item{})
}

func CreateItem(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var item Item
	_ = json.NewDecoder(r.Body).Decode(&item)
	items = append(items, item)
	json.NewEncoder(w).Encode(item)

	// Publicar evento de creación
	event := CreateEvent{ /* Datos relevantes */ 
		ItemID: item.ID,
	}
	publishEvent(ctx, "create", event)
}

func UpdateItem(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, item := range items {
		if item.ID == params["id"] {
			items = append(items[:index], items[index+1:]...)
			var newItem Item
			_ = json.NewDecoder(r.Body).Decode(&newItem)
			newItem.ID = params["id"]
			items = append(items, newItem)
			json.NewEncoder(w).Encode(newItem)

			// Publicar evento de actualización
			event := UpdateEvent{ /* Datos relevantes */ 
				ItemID: newItem.ID,
			}
			publishEvent(ctx, "update", event)
			return
		}
	}
}

func DeleteItem(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	for index, item := range items {
		if item.ID == params["id"] {
			items = append(items[:index], items[index+1:]...)
			break
		}
	}
	json.NewEncoder(w).Encode(items)

	// Publicar evento de eliminación
	event := DeleteEvent{ /* Datos relevantes */ 
		ItemID: params["id"],
	}
	publishEvent(ctx, "delete", event)
}

func publishEvent(ctx context.Context, eventType string, event interface{}) error {
	nc, ok := ctx.Value(ConnectionKey).(*nats.Conn)
	if !ok {
		return errors.New("could not get NATS connection from context")
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshalling event: %v", err)
		return err
	}

	err = nc.Publish(eventType, eventBytes)
	if err != nil {
		log.Printf("Error publishing event: %v", err)
		return err
	}

	return nil
}

func subscribeToCreateEvent(nc *nats.Conn) error {
	_, err := nc.Subscribe("create", func(msg *nats.Msg) {
		var event CreateEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Printf("Error unmarshalling create event: %v", err)
			return
		}

		// Logic to handle the creation
		log.Println("Handling Create Event for ItemID:", event.ItemID)
	})
	if err != nil {
		log.Printf("Error subscribing to create event: %v", err)
		return err
	}

	return nil
}

func subscribeToUpdateEvent(nc *nats.Conn) error {
	_, err := nc.Subscribe("update", func(msg *nats.Msg) {
		var event UpdateEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Printf("Error unmarshalling update event: %v", err)
			return
		}

		// Logic to handle the update
		log.Println("Handling Update Event for ItemID:", event.ItemID)
	})
	if err != nil {
		log.Printf("Error subscribing to update event: %v", err)
		return err
	}

	return nil
}

func subscribeToDeleteEvent(nc *nats.Conn) error {
	_, err := nc.Subscribe("delete", func(msg *nats.Msg) {
		var event DeleteEvent
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			log.Printf("Error unmarshalling delete event: %v", err)
			return
		}

		// Logic to handle the deletion
		log.Println("Handling Delete Event for ItemID:", event.ItemID)
	})
	if err != nil {
		log.Printf("Error subscribing to delete event: %v", err)
		return err
	}

	return nil
}