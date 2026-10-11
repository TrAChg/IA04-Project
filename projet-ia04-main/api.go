package projet

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

// StartAPI lance le serveur HTTP sur une goroutine distincte.
func StartAPI(sim *Simulation, mu *sync.RWMutex) {
	mux := http.NewServeMux()

	// 1. GET /status: état actuel de la simulation
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		status := map[string]interface{}{
			"tour":            sim.CurrentTour,
			"is_running":      sim.IsRunning,
			"total_groups":    len(sim.Env.Groups),
			"total_countries": len(sim.Env.Countries),
		}
		jsonResponse(w, status)
	})

	// 2. GET /countries: information pour tous les pays
	mux.HandleFunc("GET /countries", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		var list []map[string]interface{}
		for _, c := range sim.Env.Countries {
			c.mu.Lock()
			literate := 0
			nonLiterate := 0
			illegal := 0

			for _, g := range c.PopulationList {
				if g.IsLiterate {
					literate++
				} else if g.IsIllegal {
					illegal++
				} else {
					nonLiterate++
				}
			}

			currentPop := c.Current_popu
			isClosed := !c.Border
			c.mu.Unlock()

			indicator := c.CalculateIndicator(currentPop, isClosed)

			data := map[string]interface{}{
				"id":           c.Id,
				"name":         c.Name,
				"pop":          currentPop,
				"max_pop":      c.Max_popu,
				"border":       c.Border,
				"deaths":       c.Total_deaths,
				"literate":     literate,
				"non_literate": nonLiterate,
				"illegal":      illegal,
				"indicator":    indicator,
			}
			list = append(list, data)
		}
		jsonResponse(w, list)
	})

	// 3. GET /groups: information pour tous les groupes
	mux.HandleFunc("GET /groups", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		var list []map[string]interface{}
		for _, g := range sim.Env.Groups {
			countryName := "Unknown"
			if g.Current_country != nil {
				countryName = g.Current_country.Name
			}
			data := map[string]interface{}{
				"id":      g.Id,
				"x":       g.Pos.X,
				"y":       g.Pos.Y,
				"country": countryName,
				"type":    "Non-Literate",
			}
			if g.IsLiterate {
				data["type"] = "Literate"
			}
			if g.IsIllegal {
				data["status"] = "Illegal"
			} else {
				data["status"] = "Legal"
			}
			list = append(list, data)
		}
		jsonResponse(w, list)
	})

	// 4. GET /history: historique des statistiques par tour
	mux.HandleFunc("GET /history", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()

		var snapshots []StatsSnapshot
		for _, snap := range sim.History {
			snapshots = append(snapshots, snap)
		}

		sort.Slice(snapshots, func(i, j int) bool {
			return snapshots[i].Tour < snapshots[j].Tour
		})

		jsonResponse(w, snapshots)
	})
	// 5. GET /group/{id}: details d'un groupe spécifique
	mux.HandleFunc("GET /group/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "Invalid ID format", http.StatusBadRequest)
			return
		}

		mu.RLock()
		defer mu.RUnlock()

		var foundGroup *Group_population
		for _, g := range sim.Env.Groups {
			if g.Id == id {
				foundGroup = g
				break
			}
		}

		if foundGroup == nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		countryName := "None"
		if foundGroup.Current_country != nil {
			countryName = foundGroup.Current_country.Name
		}

		response := map[string]interface{}{
			"id":         foundGroup.Id,
			"position":   foundGroup.Pos, // {X, Y}
			"country":    countryName,
			"is_flooded": foundGroup.Is_flooded,
		}
		jsonResponse(w, response)
	})

	// 6. GET /logs: récupérer les logs de migration récents
	mux.HandleFunc("GET /logs", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		jsonResponse(w, sim.Env.MigrationLogs)
	})

	// 7. POST /control/toggle: Pause / Play
	mux.HandleFunc("POST /control/toggle", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		sim.IsRunning = !sim.IsRunning
		newState := sim.IsRunning
		mu.Unlock()

		jsonResponse(w, map[string]bool{"is_running": newState})
	})

	fs := http.FileServer(http.Dir("./html"))
	mux.Handle("/", fs)

	s := &http.Server{
		Addr:           ":12000",
		Handler:        corsMiddleware(mux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Println("API Server started on http://localhost:12000")
	log.Println("Try: http://localhost:12000/status")
	log.Println("Try: http://localhost:12000/group/1 (to get group with ID 1)")
	log.Println("Try: http://localhost:12000/countries")
	log.Println("Try: http://localhost:12000/history")
	log.Println("Try: http://localhost:12000/groups")
	log.Println("Try: http://localhost:12000/logs")
	log.Println("Use POST /control/toggle to pause/resume the simulation")

	go func() {
		if err := s.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
