package projet

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var ReportMilestones = map[int]bool{
	1:   true,
	50:  true,
	100: true,
	150: true,
}

type StatsSnapshot struct {
	Tour               int
	TotalLandArea      int
	TotalPopulation    int
	TotalDeaths        int
	OpenBorders        int
	ClosedBorders      int
	CollapsedCountries int
}

type Simulation struct {
	Env          *Environment
	CurrentTour  int
	TourDuration time.Duration
	LastTourTime time.Time
	IsRunning    bool
	History      map[int]StatsSnapshot
}

func NewSimulation(Env *Environment) *Simulation {
	sim := &Simulation{
		Env:          Env,
		CurrentTour:  0,
		TourDuration: 500 * time.Millisecond,
		LastTourTime: time.Now(),
		IsRunning:    true,
		History:      make(map[int]StatsSnapshot),
	}

	sim.CaptureSnapshot(0)
	return sim
}
func (s *Simulation) CaptureSnapshot(tour int) {
	snapshot := StatsSnapshot{
		Tour: tour,
	}

	snapshot.TotalLandArea = s.Env.CalculateCurrentLandArea()

	for _, c := range s.Env.Countries {
		c.mu.Lock()
		snapshot.TotalPopulation += c.Current_popu
		snapshot.TotalDeaths += c.Total_deaths
		if c.Current_popu > 0 {
			if c.Border {
				snapshot.OpenBorders++
			} else {
				snapshot.ClosedBorders++
			}
		} else if c.Current_popu == 0 {
			snapshot.CollapsedCountries++
		}
		c.mu.Unlock()
	}

	s.History[tour] = snapshot
	s.PrintSnapshotReport(snapshot)
}

func (s *Simulation) PrintSnapshotReport(snap StatsSnapshot) {
	fmt.Println("\n========================================")
	fmt.Printf("   Report - TOUR %d\n", snap.Tour)
	fmt.Println("========================================")
	fmt.Printf("Totals cells : %d cells\n", snap.TotalLandArea)
	fmt.Printf("Totals lives  : %d groups\n", snap.TotalPopulation)
	fmt.Printf("Totals deaths       : %d groups\n", snap.TotalDeaths)
	fmt.Printf("Border Open          : %d countries\n", snap.OpenBorders)
	fmt.Printf("Border Closed        : %d countries\n", snap.ClosedBorders)
	fmt.Printf("Country collapsed       : %d countries\n", snap.CollapsedCountries)
	fmt.Println("========================================")
}

// InitializePopulation crée une population initiale basée sur les données réelles
func (s *Simulation) InitializePopulation(percentage int) {
	totalGroups := 0

	for _, country := range s.Env.Countries {
		// Population cible = pourcentage du Max_popu
		targetPop := (country.Max_popu * percentage) / 100
		if targetPop < 1 {
			targetPop = 1
		}

		// Nombre de lettrés basé sur le VRAI taux d'alphabétisation du pays
		literacyRate := country.LiteracyRate / 100.0 // Convertir en 0-1
		if literacyRate <= 0 {
			literacyRate = 0.5 // Default 50% si pas de données
		}
		literateCount := int(float64(targetPop) * literacyRate)

		fmt.Printf("Init %s: Max=%d -> Creating %d groups (%d Literate = %.1f%%)\n",
			country.Name, country.Max_popu, targetPop, literateCount, country.LiteracyRate)

		created := 0
		attempts := 0
		for created < targetPop && attempts < targetPop*10 {
			attempts++
			if len(country.Cells) > 0 {
				idx := rand.Intn(len(country.Cells))
				cell := country.Cells[idx]

				if cell.Terrain == 0 && cell.Group_owner == nil {
					newGroup := NewGroupPopulation(totalGroups, Point{X: cell.X, Y: cell.Y}, country)
					// Les premiers 'literateCount' agents sont lettrés
					if created < literateCount {
						newGroup.IsLiterate = true
					} else {
						newGroup.IsLiterate = false
					}
					s.Env.Groups = append(s.Env.Groups, newGroup)
					s.Env.UpdateGroupMap(newGroup, country)

					totalGroups++
					created++
				}
			}
		}
	}
	fmt.Printf("Total Population Initialized: %d groups\n", totalGroups)
	s.CaptureSnapshot(0)
}

func (s *Simulation) RunOneTour() {
	s.CurrentTour++
	s.Env.Current_tour = s.CurrentTour
	fmt.Printf("\n--- TOUR %d ---\n", s.CurrentTour)
	// EAU LOGIC
	s.UpdateWaterLogic()

	// AGENT LOGIC
	var wgCountry sync.WaitGroup
	for _, country := range s.Env.Countries {
		wgCountry.Add(1)
		go func(c *Country) {
			defer wgCountry.Done()
			c.Live(s.Env)
		}(country)
	}
	wgCountry.Wait()

	var wgListen sync.WaitGroup
	for _, country := range s.Env.Countries {
		wgListen.Add(1)
		go country.Listen(s.Env, &wgListen)
	}

	var wgGroups sync.WaitGroup
	for _, group := range s.Env.Groups {
		wgGroups.Add(1)
		go func(g *Group_population) {
			defer wgGroups.Done()
			g.Live(s.Env)
		}(group)
	}

	wgGroups.Wait()
	for _, country := range s.Env.Countries {
		close(country.MigrationQueue)
	}
	wgListen.Wait()

	for _, country := range s.Env.Countries {
		country.MigrationQueue = make(chan MigrationRequest, 1000)
	}
	// CLEANUP & STATS
	s.CleanupDeadGroups()
	if ReportMilestones[s.CurrentTour] {
		s.CaptureSnapshot(s.CurrentTour)
		if s.CurrentTour == 150 {
			s.PrintFinalComparison()
			s.IsRunning = false
			fmt.Println("End simulation.")
		}
	}
	s.PrintStats()
}
func (s *Simulation) PrintFinalComparison() {
	fmt.Println("\n####################################################################")
	fmt.Println("                  Report Comparison Over Milestones")
	fmt.Println("####################################################################")
	fmt.Printf("%-10s | %-15s | %-12s | %-10s | %-10s | %-10s\n",
		"TOUR", "Totals (Cells)", "Popu", "Deaths", "Open/Closed", "COLLAPSED")
	fmt.Println("--------------------------------------------------------------------")

	// 0, 50, 100, 150
	milestones := []int{1, 50, 100, 150}
	for _, m := range milestones {
		if snap, ok := s.History[m]; ok {
			fmt.Printf("%-10d | %-15d | %-12d | %-10d | %d | %d | %d\n",
				snap.Tour,
				snap.TotalLandArea,
				snap.TotalPopulation,
				snap.TotalDeaths,
				snap.OpenBorders,
				snap.ClosedBorders,
				snap.CollapsedCountries)
		}
	}
	fmt.Println("####################################################################")
}
func (s *Simulation) UpdateWaterLogic() {

	speed := 4
	floodProbability := 0.05

	for i := 0; i < speed; i++ {
		s.Env.ExpandWater(floodProbability)
	}
}

func (s *Simulation) CleanupDeadGroups() {
	aliveGroups := []*Group_population{}

	for _, g := range s.Env.Groups {
		// Si le groupe appartient toujours à un pays
		if g.Current_country != nil {
			isAlive := false
			g.Current_country.mu.Lock()
			for _, countryGroup := range g.Current_country.PopulationList {
				if countryGroup == g {
					isAlive = true
					break
				}
			}
			g.Current_country.mu.Unlock()

			if isAlive {
				aliveGroups = append(aliveGroups, g)
			} else {
				// Si le groupe n'appartient plus à un pays (mort), assurez-vous de le supprimer de la carte
				s.Env.RemoveGroupFromMap(g)
			}
		}
	}
	// Mettre à jour la liste principale
	s.Env.Groups = aliveGroups
}

// PrintStats affiche les statistiques globales dans le terminal
func (s *Simulation) PrintStats() {
	totalPop := 0
	totalDeaths := 0

	for _, c := range s.Env.Countries {
		c.mu.Lock()
		realPop := c.Current_popu
		totalPop += realPop
		totalDeaths += c.Total_deaths
		c.mu.Unlock()
	}
}
