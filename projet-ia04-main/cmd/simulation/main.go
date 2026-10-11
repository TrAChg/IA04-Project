package main

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"projet"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	const width = 1025
	const height = 1025
	const uiWidth = 350
	ebiten.SetWindowSize(width+uiWidth, height)
	const cellSize = 1
	var mu sync.RWMutex

	rand.Seed(time.Now().UnixNano())

	env := projet.NewEnvironment(width, height, cellSize)

	env.AssignCountries()
	env.DetectCountryNeighbors()

	// Charger les données CSV pour population et taux d'alphabétisation
	err := env.InitializeCountryStats("docs/data_afrique.csv")
	if err != nil {
		log.Printf("Warning: Could not load CSV data: %v\n", err)
	}

	sim := projet.NewSimulation(env)
	sim.TourDuration = 500 * time.Millisecond
	sim.InitializePopulation(60) // 60% of max_population
	projet.StartAPI(sim, &mu)

	game := projet.NewGame(env, sim, &mu)
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("Simulation IA04: Sea Level Rise - Africa")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
