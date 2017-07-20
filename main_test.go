package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mkobaly/hipchatBot/config"
	"github.com/mkobaly/hipchatBot/teamcity"
)

func TestHook(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.

	json := `{
    "event": "room_message",
    "item": {
        "message": {
            "date": "2017-07-14T15:04:49.484086+00:00",
            "from": {
                "id": 4513556,
                "links": {
                    "self": "https://api.hipchat.com/v2/user/4513556"
                },
                "mention_name": "MichaelKobaly",
                "name": "Michael Kobaly",
                "version": "00000000"
            },
            "id": "a9106252-693b-4c12-9bde-755eaaa07052",
            "mentions": [],
            "message": "/build status 21766",
            "type": "message"
        },
        "room": {
            "id": 4008322,
            "is_archived": false,
            "links": {
                "members": "https://api.hipchat.com/v2/room/4008322/member",
                "participants": "https://api.hipchat.com/v2/room/4008322/participant",
                "self": "https://api.hipchat.com/v2/room/4008322",
                "webhooks": "https://api.hipchat.com/v2/room/4008322/webhook"
            },
            "name": "Test_Deployments",
            "privacy": "private",
            "version": "PL5REA4D"
        }
    },
    "oauth_client_id": "8ad68a0b-f15c-4851-b6d4-b19b31867388",
    "webhook_id": 18544450
}`

	reader := strings.NewReader(json) //Convert string to reader

	req, err := http.NewRequest("POST", "/hook", reader)
	if err != nil {
		t.Fatal(err)
	}

	config := config.NewConfig("config.yaml")

	c := &Context{
		rooms:   make(map[string]*RoomConfig),
		builder: teamcity.New(config.Teamcity),
		cfg:     config,
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(c.hook)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rec, req)

	// Check the status code is what we expect.
	if status := rec.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestHealthCheck(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}

	c := &Context{
		rooms: make(map[string]*RoomConfig),
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rec := httptest.NewRecorder()
	handler := http.HandlerFunc(c.healthcheck)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rec, req)

	// Check the status code is what we expect.
	if status := rec.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	// expected := `{"alive": true}`
	// if rec.Body.String() != expected {
	// 	t.Errorf("handler returned unexpected body: got %v want %v",
	// 		rec.Body.String(), expected)
	// }
}
