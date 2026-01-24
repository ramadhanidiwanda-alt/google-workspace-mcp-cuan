// Copyright 2025 Google LLC
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"time"
)

// TimeService provides time-related utilities
type TimeService struct{}

// NewTimeService creates a new TimeService instance
func NewTimeService() *TimeService {
	return &TimeService{}
}

// GetCurrentDateResult contains the result of GetCurrentDate
type GetCurrentDateResult struct {
	UTC   string `json:"utc"`
	Local string `json:"local"`
}

// GetCurrentDate returns the current date in both UTC and local timezone
func (s *TimeService) GetCurrentDate() ToolResponse {
	now := time.Now()

	result := GetCurrentDateResult{
		UTC:   now.UTC().Format("2006-01-02"),
		Local: now.Format("2006-01-02"),
	}

	return JSONResponse(result)
}

// GetCurrentTimeResult contains the result of GetCurrentTime
type GetCurrentTimeResult struct {
	UTC      string `json:"utc"`
	Local    string `json:"local"`
	Timezone string `json:"timezone"`
}

// GetCurrentTime returns the current time in both UTC and local timezone
func (s *TimeService) GetCurrentTime() ToolResponse {
	now := time.Now()
	zone, _ := now.Zone()

	result := GetCurrentTimeResult{
		UTC:      now.UTC().Format(time.RFC3339),
		Local:    now.Format(time.RFC3339),
		Timezone: zone,
	}

	return JSONResponse(result)
}

// GetTimeZoneResult contains the result of GetTimeZone
type GetTimeZoneResult struct {
	Name   string `json:"name"`
	Offset int    `json:"offset"` // Offset in seconds from UTC
}

// GetTimeZone returns the local timezone information
func (s *TimeService) GetTimeZone() ToolResponse {
	now := time.Now()
	zone, offset := now.Zone()

	result := GetTimeZoneResult{
		Name:   zone,
		Offset: offset,
	}

	return JSONResponse(result)
}
