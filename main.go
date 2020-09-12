//  A simple REST API built for the kubucation YouTube channel
//  + GET /coasters returns list of coasters as JSON
//  + GET /coasters/{id} returns details of specific coaster as JSON
//  + POST /coasters accepts a new coaster to be added
//  + POST /coasters returns status 415 if content is not application/json
//  + GET /admin requires basic auth
//  GET /coasters/random redirects (Status 302) to a random coaster

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type coaster struct {
	Name       string `json:"name"`
	Manufactur string `json:"manufactur"`
	ID         string `json:"id"`
	InPark     string `json:"inPark"`
	Height     int    `json:"height"`
}

type coasterHandlers struct {
	sync.Mutex
	store map[string]coaster
}

func (h *coasterHandlers) coasters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "POST":
		h.post(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
}
func (h *coasterHandlers) post(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	ct := r.Header.Get("content-type")
	if ct != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("Need type of application/json but got '%s'", ct)))
	}

	var coasterBody coaster
	err = json.Unmarshal(bodyBytes, &coasterBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}
	h.Lock()
	coasterBody.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	h.store[coasterBody.ID] = coasterBody
	defer h.Unlock()
}

func (h *coasterHandlers) getRandomCoaster(w http.ResponseWriter, r *http.Request) {
	ids := make([]string, len(h.store))

	h.Lock()
	i := 0
	for id := range h.store {
		ids[i] = id
		i++
	}
	h.Unlock()

	var target string
	if len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if len(ids) == 1 {
		target = ids[0]
	} else {
		rand.Seed(time.Now().UnixNano())
		target = ids[rand.Intn(len(ids)-1)]
	}

	w.Header().Add("content-type", "application/json")
	w.Header().Add("location", fmt.Sprintf("/coasters/%s", target))
	w.WriteHeader(http.StatusFound)
}

func (h *coasterHandlers) getCoaster(w http.ResponseWriter, r *http.Request) {
	paths := strings.Split(r.URL.String(), "/")
	if len(paths) != 3 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("The url that you request is not found"))
		return
	}
	id := paths[2]

	if id == "random" {
		h.getRandomCoaster(w, r)
		return
	}

	h.Lock()
	dataCoaster, ok := h.store[id]
	h.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("No content found for ID, '%s'", id)))
		return
	}

	jsonBytes, err := json.Marshal(dataCoaster)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)

}

func (h *coasterHandlers) get(w http.ResponseWriter, r *http.Request) {
	h.Lock()
	coasters := make([]coaster, len(h.store))
	i := 0
	for _, coaster := range h.store {
		coasters[i] = coaster
		i++
	}
	h.Unlock()
	jsonBytes, err := json.Marshal(coasters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func newCoasterHandlers() *coasterHandlers {
	return &coasterHandlers{store: map[string]coaster{}}
}

type adminPortal struct {
	password string
}

func newAdminPortal() *adminPortal {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("need to set env variable for ADMIN_PASSWORD")
	}
	return &adminPortal{password: password}
}
func (a *adminPortal) handler(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok || username != "admin" || password != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 unauthorized"))
		return
	}
	w.Write([]byte("<div><h2 style='color:orange;'>Wellcome, admin</h2></div>"))
}

func main() {
	admin := newAdminPortal()
	coasterHandler := newCoasterHandlers()
	http.HandleFunc("/coasters", coasterHandler.coasters)
	http.HandleFunc("/coasters/", coasterHandler.getCoaster)
	http.HandleFunc("/admin", admin.handler)
	port := ":8080"
	err := http.ListenAndServe(port, nil)
	fmt.Println(fmt.Sprintf("Server running on http://localhost%s 🐹", port))
	if err != nil {
		panic(err)
	}
}
