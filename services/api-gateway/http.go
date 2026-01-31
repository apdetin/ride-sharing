package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/api-gateway/grpcclients"
	"ride-sharing/shared/contracts"
)

func handleTripStart(w http.ResponseWriter, r *http.Request) {

	var reqBody startTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// validation
	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	// why we need to create a new client for each connection?
	// because if a service is down, we don't want to block the entire application
	tripService, err := grpcclients.NewTripServiceClient()
	if err != nil {
		log.Println("error creating trip service client", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tripService.Close()

	trip, err := tripService.Client.CreateTrip(r.Context(), reqBody.toProto())
	if err != nil {
		log.Println("error creating trip", err)
		http.Error(w, "failed to create trip: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: trip}

	writeJSON(w, http.StatusOK, response)

}

func handleTripPreview(w http.ResponseWriter, r *http.Request) {

	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	// validation
	if reqBody.UserID == "" {
		http.Error(w, "user ID is required", http.StatusBadRequest)
		return
	}

	// why we need to create a new client for each connection?
	// because if a service is down, we don't want to block the entire application
	tripService, err := grpcclients.NewTripServiceClient()
	if err != nil {
		log.Println("error creating trip service client", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tripService.Close()

	tripPreview, err := tripService.Client.PreviewTrip(context.Background(), reqBody.toProto())
	if err != nil {
		log.Println("error previewing trip", err)
		http.Error(w, "failed to preview trip: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := contracts.APIResponse{Data: tripPreview}

	writeJSON(w, http.StatusOK, response)

}
