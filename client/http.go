package client

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

// Reminder represents the CLI client reminder
type Reminder struct {
	ID         int           `json:"id"`
	Title      string        `json:"title"`
	Message    string        `json:"message"`
	Duration   time.Duration `json:"duration"`
	CreatedAt  time.Time     `json:"created_at"`
	ModifiedAt time.Time     `json:"modified_at"`
}

func (r Reminder) String() string {
	bs, err := json.Marshal(&r)
	if err != nil {
		log.Fatalf("could not marshal json: %v", err)
	}
	var buff bytes.Buffer
	err = json.Indent(&buff, bs, "", "\t")
	if err != nil {
		log.Fatalf("could not indent json: %v", err)
	}
	return buff.String()
}

func (r Reminder) body() io.Reader {
	bs, err := json.Marshal(&r)
	if err != nil {
		log.Fatalf("could not marshal json: %v", err)
	}
	return bytes.NewReader(bs)
}

// httpRoundTripper represents the HTTP interceptor for the CLI HTTP client
type httpRoundTripper struct {
	healthURI         string
	originalTransport http.RoundTripper
}

func (roundTripper *httpRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := http.Get(roundTripper.healthURI)
	if err != nil || res.StatusCode != http.StatusOK {
		log.Fatalf("backend api is not available: %v", err)
	}
	res, err = roundTripper.originalTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// HTTPClient represents the HTTP client which communicates with reminders backend API
type HTTPClient struct {
	client     *http.Client
	BackendURI string
}

// NewHTTPClient creates a new instance of HTTPClient
func NewHTTPClient(uri string) HTTPClient {
	roundTripper := &httpRoundTripper{
		healthURI:         uri + "/health",
		originalTransport: http.DefaultTransport,
	}
	return HTTPClient{
		BackendURI: uri,
		client: &http.Client{
			Transport: roundTripper,
		},
	}
}

// Create calls the create API endpoint
func (c HTTPClient) Create(title, message string, duration time.Duration) Reminder {
	reminder := Reminder{
		Title:    title,
		Message:  message,
		Duration: duration,
	}
	req := newReq(
		http.MethodPost,
		c.BackendURI+"/reminders/create",
		reminder.body(),
	)
	res, err := c.client.Do(req)
	if err != nil && err != io.EOF {
		log.Fatalf("could not call create api endpoint: %v", err)
	}
	checkStatusCode(res, http.StatusCreated)

	var r Reminder
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil && err != io.EOF {
		log.Fatalf("could not decode response body: %v", err)
	}
	return r
}

// Edit calls the edit API endpoint
func (c HTTPClient) Edit(id int, title, message string, duration time.Duration) Reminder {
	reminder := Reminder{
		ID:       id,
		Title:    title,
		Message:  message,
		Duration: duration,
	}
	req := newReq(
		http.MethodPut,
		c.BackendURI+"/reminders/edit",
		reminder.body(),
	)
	res, err := c.client.Do(req)
	if err != nil && err != io.EOF {
		log.Fatalf("could not call edit api endpoint: %v", err)
	}
	checkStatusCode(res, http.StatusOK)

	var r Reminder
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil && err != io.EOF {
		log.Fatalf("could not decode response body: %v", err)
	}
	return r
}

// Fetch calls the fetch API endpoint
func (c HTTPClient) Fetch(ids []int) []Reminder {
	type IDs struct {
		IDs []int `json:"ids"`
	}
	body := IDs{IDs: ids}
	bs, err := json.Marshal(&body)
	if err != nil {
		log.Fatalf("could not marshal json: %v", err)
	}
	req := newReq(
		http.MethodPost,
		c.BackendURI+"/reminders/fetch",
		bytes.NewReader(bs),
	)
	res, err := c.client.Do(req)
	if err != nil && err != io.EOF {
		log.Fatalf("could not call fetch api endpoint: %v", err)
	}
	checkStatusCode(res, http.StatusOK)

	var rs []Reminder
	err = json.NewDecoder(res.Body).Decode(&rs)
	if err != nil && err != io.EOF {
		log.Fatalf("could not decode response body: %v", err)
	}
	return rs
}

// Delete calls the delete API endpoint
func (c HTTPClient) Delete(ids []int) {
	type IDs struct {
		IDs []int `json:"ids"`
	}
	body := IDs{IDs: ids}
	bs, err := json.Marshal(&body)
	if err != nil {
		log.Fatalf("could not marshal json: %v", err)
	}
	req := newReq(
		http.MethodDelete,
		c.BackendURI+"/reminders/delete",
		bytes.NewReader(bs),
	)
	res, err := c.client.Do(req)
	if err != nil && err != io.EOF {
		log.Fatalf("could not call delete api endpoint: %v", err)
	}
	checkStatusCode(res, http.StatusNoContent)
}

// newReq creates a new HTTP request to work with later on
func newReq(method, uri string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		log.Fatalf("could not create http request: %v", err)
	}
	return req
}

// checkStatusCode checks whether the response status code equals to expected one
func checkStatusCode(res *http.Response, statusCode int) {
	if res.StatusCode != statusCode {
		log.Fatalf(
			"unexpected response code: %d, expected: %d",
			res.StatusCode,
			statusCode,
		)
	}
}
