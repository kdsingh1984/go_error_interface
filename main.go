package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type loginSchema struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ClientError is an error whose details to be shared with client.
type ClientError interface {
	Error() string
	// ResponseBody returns response body.
	ResponseBody() ([]byte, error)
	// ResponseHeaders returns http status code and headers.
	ResponseHeaders() (int, map[string]string)
}

type HTTPError struct {
	Cause  error  `json:"-"`
	Detail string `json:"detail"`
	Status int    `json:"_"`
}

func (e *HTTPError) Error() string {
	if e.Cause == nil {
		return e.Detail
	}
	return e.Detail + " : " + e.Cause.Error()
}

func (e *HTTPError) ResponseBody() ([]byte, error) {
	body, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("Error parsing response body: %v", err)
	}
	return body, err
}

func (e *HTTPError) ResponseHeaders() (int, map[string]string) {
	return e.Status, map[string]string{
		"Content-Type": "application/json; charset=utf-8",
	}
}

func NewHTTPError(err error, status int, detail string) *HTTPError {
	return &HTTPError{
		Cause:  err,
		Detail: detail,
		Status: status,
	}
}

type rootHndler func(w http.ResponseWriter, r *http.Request) error

func loginUser(username string, password string) (bool, error) {
	if username != "kamal" {
		return false, errors.New("Nor user")
	}
	return true, nil
}

func (fn rootHndler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := fn(w, r)
	if err == nil {
		return
	}
	log.Printf("AN error occured %w", err)

	clientError, ok := err.(ClientError)
	if !ok {
		w.WriteHeader(500)
		return
	}

	body, err := clientError.ResponseBody()
	if err != nil {
		log.Printf("An error occured: %v", err)
		w.WriteHeader(500)
		return
	}

	status, header := clientError.ResponseHeaders()
	for k, v := range header {
		w.Header().Set(k, v)
	}
	w.WriteHeader(status)
	w.Write(body)
}

func loginHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return NewHTTPError(nil, 405, "Method not allowed") // Return 405 Method Not Allowed.
	}
	// Read request body.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("Request body read error: %v", err)
	}

	// Parse body as json.
	var schema loginSchema
	if err = json.Unmarshal(body, &schema); err != nil {
		return NewHTTPError(err, 400, "Bad request: Invalid JSON")
	}

	ok, err := loginUser(schema.Username, schema.Password)
	if err != nil {
		return fmt.Errorf("LoginUser DB error: %v", err)
	}

	if !ok {
		return NewHTTPError(nil, 401, "Wrong Password")
	}
	w.WriteHeader(200) // Successfully logged in.
	return nil
}

func main() {
	http.Handle("/login/", rootHndler(loginHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
