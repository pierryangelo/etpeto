package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

const serviceVersion = "0.1.0-dev"

type ReadinessChecker interface {
	Ready(context.Context) error
}

type HealthBody struct {
	Status string `json:"status" example:"ok" doc:"Current health state"`
}

type HealthOutput struct {
	Status int
	Body   HealthBody
}

type ServiceInfoBody struct {
	Name    string `json:"name" example:"Etpeto" doc:"Product name"`
	Version string `json:"version" example:"0.1.0-dev" doc:"Running service version"`
}

type ServiceInfoOutput struct {
	Body ServiceInfoBody
}

func registerHealth(api huma.API, readiness ReadinessChecker) {
	huma.Register(api, huma.Operation{
		OperationID: "get-liveness",
		Method:      http.MethodGet,
		Path:        "/livez",
		Summary:     "Check process liveness",
		Tags:        []string{"Operations"},
	}, func(context.Context, *struct{}) (*HealthOutput, error) {
		return &HealthOutput{
			Status: http.StatusOK,
			Body:   HealthBody{Status: "ok"},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-readiness",
		Method:      http.MethodGet,
		Path:        "/readyz",
		Summary:     "Check dependency readiness",
		Tags:        []string{"Operations"},
	}, func(ctx context.Context, _ *struct{}) (*HealthOutput, error) {
		if err := readiness.Ready(ctx); err != nil {
			return &HealthOutput{
				Status: http.StatusServiceUnavailable,
				Body:   HealthBody{Status: "unavailable"},
			}, nil
		}
		return &HealthOutput{
			Status: http.StatusOK,
			Body:   HealthBody{Status: "ok"},
		}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-service-info",
		Method:      http.MethodGet,
		Path:        "/api/v1/service-info",
		Summary:     "Describe the running Etpeto service",
		Tags:        []string{"System"},
	}, func(context.Context, *struct{}) (*ServiceInfoOutput, error) {
		return &ServiceInfoOutput{Body: ServiceInfoBody{
			Name:    "Etpeto",
			Version: serviceVersion,
		}}, nil
	})
}
