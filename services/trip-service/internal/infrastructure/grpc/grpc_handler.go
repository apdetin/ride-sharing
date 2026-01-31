package grpc

import (
	"context"
	"ride-sharing/services/trip-service/internal/domain"
	pb "ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type gRPCHandler struct {
	pb.UnimplementedTripServiceServer
	svc domain.TripService
}

func NewGRPCHandler(server *grpc.Server, service domain.TripService) *gRPCHandler {
	handler := &gRPCHandler{
		svc: service,
	}
	pb.RegisterTripServiceServer(server, handler)
	return handler
}

func (h *gRPCHandler) PreviewTrip(ctx context.Context, req *pb.PreviewTripRequest) (*pb.PreviewTripResponse, error) {

	pickup := req.GetStartLocation()
	destination := req.GetEndLocation()

	route, err := h.svc.GetRoute(ctx, &types.Coordinate{
		Latitude:  pickup.Latitude,
		Longitude: pickup.Longitude,
	}, &types.Coordinate{
		Latitude:  destination.Latitude,
		Longitude: destination.Longitude,
	}, true)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get route: %v", err)
	}

	// 1. estimate the ride fares prices based on the route (ex: distance)
	estimatedFares := h.svc.EstimatePackagesPriceWithRoute(route)
	// 2. store the ride fares for the create trip (next lesson) to fetch and validate
	fares, err := h.svc.GenerateTripFares(ctx, estimatedFares, req.GetUserID(), route)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate trip fares: %v", err)
	}

	return &pb.PreviewTripResponse{
		Route:     route.ToProto(),
		RideFares: domain.ToRideFaresProto(fares),
	}, nil
}

func (h *gRPCHandler) CreateTrip(
	ctx context.Context,
	req *pb.CreateTripRequest,
) (*pb.CreateTripResponse, error) {
	fareID := req.GetRideFareID()
	userID := req.GetUserID()
	// 1. fetch and validate the ride fares
	rideFare, err := h.svc.GetAndValidateFare(ctx, fareID, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get and validate fare: %v", err)
	}
	// 2. call create trip
	trip, err := h.svc.CreateTrip(ctx, rideFare)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create trip: %v", err)
	}

	// 3. we also need to initialize an empty driver to the trip
	trip.Driver = &pb.TripDriver{}

	// 4. add a comment at the end of the function to publish an event on the Async Comms module.

	return &pb.CreateTripResponse{
		TripID: trip.ID.Hex(),
	}, nil
}
