package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

func main() {
	log.Println("starting server")

	api := NewAPI()

	http.HandleFunc("/api", api.Handler)

	http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil)
}

func (a *API) Handler(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("q")
	data, hit, err := a.getData(r.Context(), query)
	log.Println(query)
	if err != nil {
		log.Printf("get data: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp := APIresponse{Cache: hit, Data: data}

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		log.Printf("encode response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

}

func (a *API) getData(ctx context.Context, query string) ([]nominatimResponse, bool, error) {
	// is query cached?
	value, err := a.cache.Get(ctx, query).Result()
	if err == redis.Nil {
		//call external api
		apiAddress := fmt.Sprintf("https://nominatim.openstreetmap.org/search?country=%s&format=json", url.PathEscape(query))

		resp, err := http.Get(apiAddress)
		if err != nil {
			return nil, false, err
		}

		data := make([]nominatimResponse, 0)
		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			return nil, false, err
		}

		//set in redis
		encodedData, err := json.Marshal(data)
		if err != nil {
			return nil, false, err
		}

		err = a.cache.Set(ctx, query, encodedData, time.Second*15).Err()
		if err != nil {
			panic(err)
		}

		//return resp
		return data, false, nil
	} else if err != nil {
		log.Printf("call redis: %v", err)
		return nil, false, err
	} else {
		// build response
		data := make([]nominatimResponse, 0)
		err := json.Unmarshal([]byte(value), &data)
		if err != nil {
			return nil, false, err
		}

		//return response
		return data, true, nil
	}

}

type APIresponse struct {
	Cache bool                `json:"cache"`
	Data  []nominatimResponse `json:"resp"`
}

type nominatimResponse struct {
	PlaceID     int      `json:"place_id"`
	Licence     string   `json:"licence"`
	OsmType     string   `json:"osm_type"`
	OsmID       int      `json:"osm_id"`
	Boundingbox []string `json:"boundingbox"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	DisplayName string   `json:"display_name"`
	Class       string   `json:"class"`
	Type        string   `json:"type"`
	Importance  float64  `json:"importance"`
	Icon        string   `json:"icon"`
}

type API struct {
	cache *redis.Client
}

func NewAPI() *API {
	var opts *redis.Options
	if os.Getenv("LOCAL") == "true" {
		reddisAddress := fmt.Sprintf("%s:6379", os.Getenv("REDIS_URL"))
		opts = &redis.Options{
			Addr:     reddisAddress,
			Password: "", // no password set
			DB:       0,  // use default DB
		}

	} else {
		opts = &redis.Options{
			Addr:     "",
			Password: "", // no password set
			DB:       0,  // use default DB
		}
	}

	rdb := redis.NewClient(opts)
	return &API{rdb}
}
