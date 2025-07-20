package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	authhandler "github.com/ocenb/marketplace/internal/handlers/auth"
	listinghandler "github.com/ocenb/marketplace/internal/handlers/listing"
	"github.com/ocenb/marketplace/internal/models"
	"github.com/ocenb/marketplace/tests/suite"
)

func TestMarketplaceWorkflow(t *testing.T) {
	s := suite.New(t)

	// 1. Register User
	registerReq := authhandler.RegisterRequest{Login: "testuser", Password: "password123"}
	registerBody, _ := json.Marshal(registerReq)
	resp, err := s.Client.Post(s.BaseURL+"/auth/register", "application/json", bytes.NewReader(registerBody))
	if err != nil {
		s.Fatalf("Failed to register user: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		s.Fatalf("Registration expected 201 Created, got %d", resp.StatusCode)
	}
	err = resp.Body.Close()
	if err != nil {
		s.Errorf("Failed to close response body: %v", err)
	}

	// 2. Login User
	loginReq := authhandler.LoginRequest{Login: "testuser", Password: "password123"}
	loginBody, _ := json.Marshal(loginReq)
	resp, err = s.Client.Post(s.BaseURL+"/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		s.Fatalf("Failed to login user: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		s.Fatalf("Login expected 200 OK, got %d", resp.StatusCode)
	}

	var loginResp authhandler.LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResp)
	if err != nil {
		s.Fatalf("Failed to decode login response: %v", err)
	}
	err = resp.Body.Close()
	if err != nil {
		s.Errorf("Failed to close response body: %v", err)
	}
	if loginResp.Token == "" {
		s.Fatalf("Login response token is empty")
	}

	authToken := "Bearer " + loginResp.Token

	// 3. Create Listing
	createListingReq := listinghandler.CreateListingRequest{
		Title:       "Test Listing 1",
		Description: "A description for test listing 1.",
		ImageURL:    "https://images.unsplash.com/photo-1752564627655-168bd1be3202?q=80&w=928&auto=format&fit=crop&ixlib=rb-4.1.0&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%3D%3D",
		Price:       150000,
	}
	createListingBody, _ := json.Marshal(createListingReq)
	req, err := http.NewRequest(http.MethodPost, s.BaseURL+"/listing", bytes.NewReader(createListingBody))
	if err != nil {
		s.Fatalf("Failed to create new request for listing creation: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)

	resp, err = s.Client.Do(req)
	if err != nil {
		s.Fatalf("Failed to create listing: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		s.Fatalf("Create Listing expected 201 Created, got %d", resp.StatusCode)
	}

	var createListingRes models.Listing
	err = json.NewDecoder(resp.Body).Decode(&createListingRes)
	if err != nil {
		s.Fatalf("Failed to decode create listing response: %v", err)
	}
	err = resp.Body.Close()
	if err != nil {
		s.Errorf("Failed to close response body: %v", err)
	}

	// 4. Create Listing with Bad Image URL
	createBadImageListingReq := listinghandler.CreateListingRequest{
		Title:       "Listing with Bad Image",
		Description: "This listing has an invalid image URL.",
		ImageURL:    "not-a-valid-url",
		Price:       200000,
	}
	createBadImageListingBody, _ := json.Marshal(createBadImageListingReq)
	req, err = http.NewRequest(http.MethodPost, s.BaseURL+"/listing", bytes.NewReader(createBadImageListingBody))
	if err != nil {
		s.Fatalf("Failed to create new request for bad image listing: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)

	resp, err = s.Client.Do(req)
	if err != nil {
		s.Fatalf("Failed to send bad image listing request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		s.Fatalf("Create Listing with bad image expected 400 Bad Request, got %d", resp.StatusCode)
	}
	err = resp.Body.Close()
	if err != nil {
		s.Errorf("Failed to close response body: %v", err)
	}

	// 5. Create Listing Without Token
	createUnauthorizedListingReq := listinghandler.CreateListingRequest{
		Title:       "Unauthorized Attempt",
		Description: "This listing should not be created.",
		ImageURL:    "http://example.com/unauth.jpg",
		Price:       300000,
	}
	createUnauthorizedListingBody, _ := json.Marshal(createUnauthorizedListingReq)
	req, err = http.NewRequest(http.MethodPost, s.BaseURL+"/listing", bytes.NewReader(createUnauthorizedListingBody))
	if err != nil {
		s.Fatalf("Failed to create request for unauthorized listing creation: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = s.Client.Do(req)
	if err != nil {
		s.Fatalf("Failed to send unauthorized listing request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		s.Fatalf("Create Listing without token expected 401 Unauthorized, got %d", resp.StatusCode)
	}
	err = resp.Body.Close()
	if err != nil {
		s.Errorf("Failed to close response body: %v", err)
	}

	// 6. Get All Listings
	req, err = http.NewRequest(http.MethodGet, s.BaseURL+"/listing/feed", nil)
	if err != nil {
		s.Fatalf("Failed to create new request for listing feed: %v", err)
	}
	req.Header.Set("Authorization", authToken)

	resp, err = s.Client.Do(req)
	if err != nil {
		s.Fatalf("Failed to get listings feed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		s.Fatalf("Get Listings Feed expected 200 OK, got %d", resp.StatusCode)
	}

	var feedResp models.ListingsFeed
	err = json.NewDecoder(resp.Body).Decode(&feedResp)
	if err != nil {
		s.Fatalf("Failed to decode listings feed response: %v", err)
	}
	err = resp.Body.Close()
	if err != nil {
		s.Errorf("Failed to close response body: %v", err)
	}

	if len(feedResp.Listings) == 0 {
		s.Fatalf("Expected at least one listing in the feed, got 0")
	}

	found := false
	for _, l := range feedResp.Listings {
		if l.ID == createListingRes.ID {
			found = true
			if l.Title != createListingReq.Title {
				s.Errorf("Listing title mismatch: expected %q, got %q", createListingReq.Title, l.Title)
			}
			if l.Price != createListingReq.Price {
				s.Errorf("Listing price mismatch: expected %d, got %d", createListingReq.Price, l.Price)
			}
			if !l.IsOwner {
				s.Errorf("IsOwner expected to be true for the created listing, got false")
			}
			break
		}
	}

	if !found {
		s.Fatalf("Created listing with ID %q not found in feed", createListingRes.ID)
	}
}
