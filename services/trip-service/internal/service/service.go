package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	tripTypes "ride-sharing/services/trip-service/pkg/types"
	"ride-sharing/shared/proto/trip"
	"ride-sharing/shared/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type service struct {
	repo domain.TripRepository
}

func NewService(repo domain.TripRepository) *service {
	return &service{
		repo: repo,
	}
}

func (s *service) CreateTrip(ctx context.Context, fare *domain.RideFareModel) (*domain.TripModel, error) {
	trip := &domain.TripModel{
		ID:       primitive.NewObjectID(),
		UserID:   fare.UserID,
		Status:   "pending",
		RideFare: fare,
		Driver:   &trip.TripDriver{},
	}

	return s.repo.CreateTrip(ctx, trip)
}

func (s *service) GetRoute(ctx context.Context, pickup, destination *types.Coordinate, useOSRMApi bool) (*tripTypes.OsrmApiResponse, error) {
	if !useOSRMApi {
		// Return a simple mock response in case we don't want to rely on an external API
		return &tripTypes.OsrmApiResponse{
			Routes: []struct {
				Distance float64 `json:"distance"`
				Duration float64 `json:"duration"`
				Geometry struct {
					Coordinates [][]float64 `json:"coordinates"`
				} `json:"geometry"`
			}{
				{
					Distance: 5.0, // 5km
					Duration: 600, // 10 minutes
					Geometry: struct {
						Coordinates [][]float64 `json:"coordinates"`
					}{
						Coordinates: [][]float64{
							{pickup.Latitude, pickup.Longitude},
							{destination.Latitude, destination.Longitude},
						},
					},
				},
			},
		}, nil
	}

	baseURL := "http://router.project-osrm.org/route/v1/driving"
	// baseURLAlternative := "https://osrm.selfmadeengineer.com/route/v1/driving"

	url := fmt.Sprintf("%s/%f,%f;%f,%f?overview=full&geometries=geojson",
		baseURL,
		pickup.Longitude, pickup.Latitude,
		destination.Longitude, destination.Latitude)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch route from: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var routeResp tripTypes.OsrmApiResponse
	if err := json.Unmarshal(body, &routeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %v", err)
	}

	return &routeResp, nil
}

func (s *service) EstimatePackagesPriceWithRoute(route *tripTypes.OsrmApiResponse) []*domain.RideFareModel {
	baseFares := getBaseFare()
	estimatedFares := make([]*domain.RideFareModel, len(baseFares))

	for i, f := range baseFares {
		estimatedFares[i] = estimateFareRoute(f, route)
	}

	return estimatedFares
}

func (s *service) GenerateTripFares(
	ctx context.Context,
	rideFares []*domain.RideFareModel,
	userID string,
	route *tripTypes.OsrmApiResponse,
) ([]*domain.RideFareModel, error) {

	fares := make([]*domain.RideFareModel, len(rideFares))

	for i, f := range rideFares {
		fare := &domain.RideFareModel{
			ID:                primitive.NewObjectID(),
			UserID:            userID,
			PackageSlug:       f.PackageSlug,
			TotalPriceInCents: f.TotalPriceInCents,
			Route:             route,
		}

		if err := s.repo.SaveRideFare(ctx, fare); err != nil {
			return nil, fmt.Errorf("failed to save ride fare: %v", err)
		}

		fares[i] = fare
	}

	return fares, nil
}

func (s *service) GetAndValidateFare(ctx context.Context, fareID, userID string) (*domain.RideFareModel, error) {

	fare, err := s.repo.GetRideFareByID(ctx, fareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride fare: %w", err)
	}

	if fare == nil {
		return nil, fmt.Errorf("ride fare not found")
	}

	// user fare validation (user is owner of this fare?)
	if fare.UserID != userID {
		return nil, fmt.Errorf("user is not the owner of this fare")
	}

	return fare, nil
}

func estimateFareRoute(fare *domain.RideFareModel, route *tripTypes.OsrmApiResponse) *domain.RideFareModel {

	pricingCfg := tripTypes.DefaultPricingConfig()
	carPackagePrice := fare.TotalPriceInCents

	distanceKm := route.Routes[0].Distance
	durationInMinutes := route.Routes[0].Duration

	// distance
	distanceFare := distanceKm * pricingCfg.PricePerUnitOfDistance
	// time
	timeFare := durationInMinutes * pricingCfg.PricingPerMinute
	// car price
	totalPrice := carPackagePrice + distanceFare + timeFare

	return &domain.RideFareModel{
		PackageSlug:       fare.PackageSlug,
		TotalPriceInCents: totalPrice,
	}
}

func getBaseFare() []*domain.RideFareModel {
	return []*domain.RideFareModel{
		{
			PackageSlug:       "suv",
			TotalPriceInCents: 200,
		},
		{
			PackageSlug:       "sedan",
			TotalPriceInCents: 350,
		},
		{
			PackageSlug:       "van",
			TotalPriceInCents: 400,
		},
		{
			PackageSlug:       "luxury",
			TotalPriceInCents: 500,
		},
	}
}
