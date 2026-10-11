package projet

import (
	"fmt"
	"math/rand"
	"sync"
)

const (
	// Migration response codes
	MigrationRejected = 0
	MigrationAccepted = 1
	MigrationDied     = 2

	// Scoring weights
	OptimalDensity = 0.7
	SecurityCost   = 2.0
	DensityPenalty = 2.0
)

type Point struct {
	X int
	Y int
}

// La demande de migration est Envoyée par le groupe au pays de destination pour faire une demande d'immigration.
type MigrationRequest struct {
	Group     *Group_population
	ReplyChan chan int
}

// Agent est une interface publique. Dans simu.go, nous aurons list[]Agent
type Agent interface {
	Live(Env *Environment)
}

type Country struct {
	Id             int
	Name           string
	Neighbors      map[int]bool
	Current_popu   int
	Max_popu       int
	Border         bool // TRUE = OPEN BORDER, FALSE = CLOSED BORDER
	Total_deaths   int
	LiteracyRate   float64 // Taux d'alphabétisation (0-100)
	BasePopulation int     // Population réelle du pays (from CSV)
	MigrationQueue chan MigrationRequest // Channel pour les demandes d'immigration
	mu             sync.Mutex
	PopulationList []*Group_population
	Cells          []*Cell
}

type Group_population struct {
	Id              int
	Pos             Point
	Current_country *Country
	Is_flooded      bool
	StatusChan      chan bool
	IsLiterate      bool // True = Blue, False = White
	IsIllegal       bool // True = Red
}

func NewCountry(name string, maxPop int) *Country {
	return &Country{
		Name:           name,
		Max_popu:       maxPop,
		Current_popu:   0,
		Border:         false,
		Total_deaths:   0,
		MigrationQueue: make(chan MigrationRequest, 1000),
		PopulationList: make([]*Group_population, 0),
		Neighbors:      make(map[int]bool),
		Cells:          make([]*Cell, 0),
	}
}
func (e *Environment) GetCountryByID(id int) *Country {
	for _, c := range e.Countries {
		if c.Id == id {
			return c
		}
	}
	return nil
}

func NewGroupPopulation(id int, pos Point, country *Country) *Group_population {
	group := &Group_population{
		Id:              id,
		Pos:             pos,
		Current_country: country,
		Is_flooded:      false,
		IsLiterate:      false, // Default, will be set in InitializePopulation
		IsIllegal:       false, // Default legal
	}
	if country != nil {
		country.AddGroup(group)
	}
	return group
}

// Caluler le score économique du pays en fonction de sa population
func (c *Country) CalculateEconomy() float64 {
	score := 0.0
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, g := range c.PopulationList {
		if g.IsIllegal {
			score -= 0.3
		} else if g.IsLiterate {
			score += 1.0
		} else {
			score += 0.3 // Non-literate 0 contribution
		}
	}
	return score
}

func (c *Country) AddGroup(group *Group_population) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.PopulationList = append(c.PopulationList, group)
	c.Current_popu += 1
}

func (c *Country) RemoveGroup(group *Group_population) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, g := range c.PopulationList {
		if g == group {
			c.PopulationList = append(c.PopulationList[:i], c.PopulationList[i+1:]...)
			c.Current_popu -= 1
			break
		}
	}
}
func (c *Country) Listen(env *Environment, wg *sync.WaitGroup) {
	defer wg.Done()

	for req := range c.MigrationQueue {
		fmt.Printf("[TOUR %d] %s reçoit une demande d'émigration du Group %d\n	", c.Id, c.Name, req.Group.Id)
		accepted := c.processMigrationRequest(env, req.Group)
		req.ReplyChan <- accepted
	}
}

func (c *Country) Live(Env *Environment) {
	c.deliberateBorder(Env)
}

// PredictImmigration évalue le nombre de pays voisins susceptibles
func (c *Country) PredictImmigration(env *Environment) int {
	prediction := 0

	for neighborID := range c.Neighbors {
		neighbor := env.GetCountryByID(neighborID)
		if neighbor == nil {
			continue
		}
		if neighbor.Current_popu >= neighbor.Max_popu || !neighbor.Border {
			prediction += 1
		}

		// Vérifier le pourcentage de cellules inondées
		floodedCells := 0
		for _, cell := range neighbor.Cells {
			if cell.Terrain == 1 {
				floodedCells++
			}
		}
		floodRatio := float64(floodedCells) / float64(len(neighbor.Cells))
		if floodRatio > 0.3 {
			prediction += 2
		}
	}

	if prediction < 1 {
		prediction = 1
	}

	if prediction > 5 {
		prediction = 5
	}

	return prediction
}

// projectedPop: Population projetée
// isClosed: Indique si la frontière est fermée
func (c *Country) CalculateIndicator(projectedPop int, isClosed bool) float64 {
	score := 0.0

	realPop := projectedPop
	if realPop > c.Max_popu {
		realPop = c.Max_popu
	}

	currentEcoScore := 0.0
	c.mu.Lock()
	for _, g := range c.PopulationList {
		if g.IsIllegal {
			currentEcoScore -= 0.3
		} else if g.IsLiterate {
			currentEcoScore += 1.0
		} else {
			currentEcoScore += 0.0
		}
	}
	c.mu.Unlock()
	if realPop > c.Current_popu {
		addedPeople := float64(realPop - c.Current_popu)
		currentEcoScore += (addedPeople * 0.5)
	}
	score += currentEcoScore

	occupancyRate := float64(projectedPop) / float64(c.Max_popu)
	if occupancyRate > OptimalDensity {
		excessRatio := occupancyRate - OptimalDensity
		score -= (excessRatio * float64(c.Max_popu) * DensityPenalty)
	}

	if isClosed {
		score -= SecurityCost
	}

	return score
}

func (c *Country) deliberateBorder(env *Environment) {
	projectedInflow := c.PredictImmigration(env)

	// Open border
	popIfOpen := c.Current_popu + projectedInflow
	scoreOpen := c.CalculateIndicator(popIfOpen, false)

	// Close border
	popIfClosed := c.Current_popu
	scoreClosed := c.CalculateIndicator(popIfClosed, true)

	c.mu.Lock()
	defer c.mu.Unlock()

	if scoreOpen >= scoreClosed {
		c.Border = true
	} else {
		c.Border = false
	}
}

// Debug: Imprimer les statistiques du territoire du pays
func (c *Country) PrintTerritoryStats(env *Environment) {
	totalOwned := 0
	flooded := 0
	occupied := 0
	valid := 0

	for y := 0; y < env.Height; y++ {
		for x := 0; x < env.Width; x++ {
			cell := env.GetCell(x, y)
			if cell != nil && cell.Country_owner == c {
				totalOwned++
				if cell.Terrain == 1 {
					flooded++
				} else if cell.Group_owner != nil {
					occupied++
				} else {
					valid++
				}
			}
		}
	}
	fmt.Printf("   [DEBUG %s] Total: %d | Flood: %d | Occupied: %d | Valid: %d\n",
		c.Name, totalOwned, flooded, occupied, valid)
}

func (c *Country) processMigrationRequest(env *Environment, group *Group_population) int {
	c.mu.Lock()
	// Tester la capacité
	if c.Current_popu >= c.Max_popu {
		c.mu.Unlock()
		fmt.Printf(" Group %d -> %s refusé car la population est trop élevée (%d/%d).\n", group.Id, c.Name, c.Current_popu, c.Max_popu)
		return MigrationRejected
	}

	if !c.Border {
		if group.IsLiterate {
			group.IsIllegal = false // Literate groups can enter even if border is closed because any country wants them

		} else {
			if rand.Float64() < 0.4 {
				c.mu.Unlock()
				return MigrationDied // Died at border
			}
			group.IsIllegal = true
		}
	} else {
		group.IsIllegal = false
	}
	c.mu.Unlock()

	foundPos, ok := c.findPlacement(env)

	if ok {
		c.AddGroup(group)
		env.UpdateGroupMap(group, c)

		group.Pos = foundPos
		return MigrationAccepted
	}
	fmt.Printf(" Group %d -> %s refusé car ne peut pas trouver un emplacement valide.\n", group.Id, c.Name)
	c.PrintTerritoryStats(env)
	return MigrationRejected
}

func (c *Country) findPlacement(env *Environment) (Point, bool) {
	for i := 0; i < 100; i++ {
		randomIndex := rand.Intn(len(c.Cells))
		cell := c.Cells[randomIndex]

		if c.isSafeSpot(env, cell.X, cell.Y, 5) {
			return Point{X: cell.X, Y: cell.Y}, true
		}
	}

	// Find a safe spot within the country
	for _, cell := range c.Cells {
		if c.isSafeSpot(env, cell.X, cell.Y, 5) {
			return Point{X: cell.X, Y: cell.Y}, true
		}
	}

	return Point{}, false
}

func (c *Country) isSafeSpot(env *Environment, x, y int, radius int) bool {
	if !c.isValidSpot(env, x, y) {
		return false
	}

	// Check surrounding cells within radius
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			nx, ny := x+dx, y+dy
			if !c.isValidSpot(env, nx, ny) {
				fmt.Printf(" Spot (%d,%d) invalide autour de (%d,%d)\n", nx, ny, x, y)
				return false
			}
		}
	}
	return true
}

func (c *Country) isValidSpot(env *Environment, x, y int) bool {
	cell := env.GetCell(x, y)
	if cell == nil {
		return false
	}
	if cell.Country_owner != c {
		return false
	}
	if cell.Group_owner != nil {
		return false
	}
	if cell.Terrain == 1 {
		return false
	}
	return true
}

// GROUP

func (g *Group_population) Live(Env *Environment) {
	g.perceive(Env)
	g.deliberateAndAct(Env)
}

func (g *Group_population) perceive(Env *Environment) {
	myCell := Env.GetCell(g.Pos.X, g.Pos.Y)
	if myCell != nil && myCell.Terrain == 1 {
		g.Is_flooded = true
		return
	}

	// Vérifier les cellules environnantes
	var visionRadius int
	if g.IsLiterate {
		visionRadius = 10 // Blue groups have larger vision
	} else {
		visionRadius = 1 // White groups have smaller vision
	}
	sawWater := false

	for dy := -visionRadius; dy <= visionRadius; dy++ {
		for dx := -visionRadius; dx <= visionRadius; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}

			nx, ny := g.Pos.X+dx, g.Pos.Y+dy
			cell := Env.GetCell(nx, ny)
			if cell != nil && cell.Terrain == 1 {
				sawWater = true
				break
			}
		}
		if sawWater {
			break
		}
	}

	g.Is_flooded = sawWater
}

func (g *Group_population) deliberateAndAct(Env *Environment) {
	if g.Is_flooded {
		success := g.attemptMigration(Env)

		myCell := Env.GetCell(g.Pos.X, g.Pos.Y)
		if !success && myCell != nil && myCell.Terrain == 1 {
			g.die(Env)
		}
	}
}

func (g *Group_population) attemptMigration(env *Environment) bool {
	targetCountry := env.FindBestNeighbor(g)
	if targetCountry == nil {
		return false
	}
	fmt.Printf("[TOUR %d] Group %d (%s) demande d'émigration %s\n",
		env.Current_tour, g.Id, g.Current_country.Name, targetCountry.Name)

	// Envoyer la demande de migration
	replyChan := make(chan int)
	req := MigrationRequest{Group: g, ReplyChan: replyChan}

	targetCountry.MigrationQueue <- req

	// Attendre resultat
	status := <-replyChan

	if status == MigrationAccepted {
		oldCountryName := g.Current_country.Name
		g.Current_country.RemoveGroup(g)

		oldCell := env.GetCell(g.Pos.X, g.Pos.Y)
		if oldCell != nil && oldCell.Group_owner == g {
			oldCell.Group_owner = nil
		}

		g.Current_country = targetCountry
		g.Is_flooded = false

		statusStr := "Legal"
		if g.IsIllegal {
			statusStr = "ILLEGAL"
		}
		env.AddLog(fmt.Sprintf("[%d] Grp %d: %s -> %s (%s)", env.Current_tour, g.Id, oldCountryName, g.Current_country.Name, statusStr))
		return true
	} else if status == MigrationDied {
		env.AddLog(fmt.Sprintf("[%d] Grp %d: DIED at border of %s", env.Current_tour, g.Id, targetCountry.Name))
		g.die(env)
		return false
	}
	fmt.Printf(" [ECHEC] Group %d n'a pas pu migrer vers %s\n",
		g.Id, targetCountry.Name)
	return false
}

func (g *Group_population) die(env *Environment) {
	if g.Current_country != nil {
		g.Current_country.Total_deaths += 1
		g.Current_country.RemoveGroup(g)
	}
	env.RemoveGroupFromMap(g)
}
