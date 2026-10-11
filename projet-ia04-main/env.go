package projet

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const UI_WIDTH = 350

type Cell struct {
	X             int
	Y             int
	Country_owner *Country
	Group_owner   *Group_population
	Terrain       int // 0 = terre, 1 = eau, 2 = frontière
}

// Environment represents the simulation Environment.
type Environment struct {
	Grid          [][]*Cell
	Width         int
	Height        int
	CellSize      int
	Countries     []*Country
	Groups        []*Group_population
	Current_tour  int
	MigrationLogs []string
	mu            sync.RWMutex
}

type Game struct {
	Env          *Environment
	Sim          *Simulation
	Mu           *sync.RWMutex
	Btn_X, Btn_Y int
	Btn_W, Btn_H int
	IsHovering   bool
}

const path = "afrique.png"

func NewGame(Env *Environment, Sim *Simulation, mu *sync.RWMutex) *Game {
	game := &Game{
		Env:   Env,
		Sim:   Sim,
		Btn_W: 60,
		Btn_H: 40,
		Mu:    mu,
	}
	game.Btn_X = 20
	game.Btn_Y = Env.Height - game.Btn_H - 20
	return game
}
func (g *Game) Update() error {
	// Vérifier si la souris est sur le bouton
	mx, my := ebiten.CursorPosition()

	g.IsHovering = mx >= g.Btn_X && mx <= g.Btn_X+g.Btn_W &&
		my >= g.Btn_Y && my <= g.Btn_Y+g.Btn_H

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && g.IsHovering {
		g.Sim.IsRunning = !g.Sim.IsRunning
		fmt.Printf("State Simulation: %v\n", g.Sim.IsRunning)
	}

	if g.Sim != nil && g.Sim.IsRunning {
		temp := time.Since(g.Sim.LastTourTime)
		if temp >= g.Sim.TourDuration {
			g.Mu.Lock()
			g.Sim.RunOneTour()
			g.Mu.Unlock()
			g.Sim.LastTourTime = time.Now()
		}
	}
	return nil
}
func NewEnvironment(width, height, cellSize int) *Environment {
	Env := &Environment{
		Width:         width,
		Height:        height,
		CellSize:      cellSize,
		Current_tour:  0,
		Countries:     make([]*Country, 0),
		Groups:        make([]*Group_population, 0),
		MigrationLogs: make([]string, 0),
	}

	// Grid 2D
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}

	Env.Grid = make([][]*Cell, height)
	for y := 0; y < height; y++ {
		Env.Grid[y] = make([]*Cell, width)
		for x := 0; x < width; x++ {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()

			// valeurs RGBA sont sur 16 bits (0–65535), donc on ramène à 0–255
			rv := uint8(r >> 8)
			gv := uint8(g >> 8)
			bv := uint8(b >> 8)

			var terrainType int
			if rv > 220 && gv > 220 && bv > 220 {
				terrainType = 1 // eau
			} else if rv < 50 && gv < 50 && bv < 50 {
				terrainType = 0 // terre
			} else {
				terrainType = 2 // frontière
			}
			Env.Grid[y][x] = &Cell{
				X:       x,
				Y:       y,
				Terrain: terrainType,
			}

		}
	}

	rand.Seed(time.Now().UnixNano())
	return Env
}
func (env *Environment) AddLog(message string) {
	env.MigrationLogs = append(env.MigrationLogs, message)

	// Limiter la taille des logs à 20 entrées
	if len(env.MigrationLogs) > 20 {
		env.MigrationLogs = env.MigrationLogs[1:]
	}
}
func (env *Environment) ExpandWater(probability float64) {
	var newFloodedCells []*Cell

	for y := 0; y < env.Height; y++ {
		for x := 0; x < env.Width; x++ {
			cell := env.Grid[y][x]

			if cell.Terrain == 1 {
				dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
				for _, d := range dirs {
					nx, ny := x+d[0], y+d[1]
					if nx >= 0 && nx < env.Width && ny >= 0 && ny < env.Height {
						neighbor := env.Grid[ny][nx]

						if neighbor.Terrain != 1 {
							if rand.Float64() < probability {
								newFloodedCells = append(newFloodedCells, neighbor)
							}
						}
					}
				}
			}
		}
	}

	for _, cell := range newFloodedCells {
		cell.Terrain = 1
	}
}

var AfricanCountryNames = []string{
	"Tunisia", "Algeria", "Morocco", "Libya", "Egypt",
	"Sahara occidental", "Mauritania", "Mali", "Niger", "Chad", "Sudan",
	"Eritrea", "Senegal", "Burkina Faso", "Nigeria", "Ethiopia", "Cameroon", "Guinea-Bissau", "Guinea", "Benin", "Somalia", "Ghana", "Togo", "Ivory Coast",
	"Central African Republic", "Sierra Leone", "Liberia",
	"DR Congo", "Kenya", "Uganda", "Congo", "Equatorial Guinea", "Gabon",
	"Tanzania", "Burundi", "Rwanda",
	"Angola", "Zambia", "Malawi", "Mozambique", "Zimbabwe", "Namibia", "Botswana",
	"South Africa", "Eswatini", "Lesotho",
}

func (env *Environment) AssignCountries() {
	countryID := 1
	for y := 0; y < env.Height; y++ {
		for x := 0; x < env.Width; x++ {
			cell := env.Grid[y][x]
			if cell.Terrain == 0 && cell.Country_owner == nil {
				name := AfricanCountryNames[(countryID-1)%len(AfricanCountryNames)]
				newCountry := NewCountry(name, 5)
				newCountry.Id = countryID

				countryID++

				env.floodFillAssign(cell, newCountry)
				env.Countries = append(env.Countries, newCountry)

				landCount := len(newCountry.Cells)
				newCountry.Max_popu = landCount / 500
				if newCountry.Max_popu < 2 && newCountry.Max_popu > 1 {
					newCountry.Max_popu = 2
				} else if newCountry.Max_popu < 1 {
					newCountry.Max_popu = 1
				}
				fmt.Printf("Initialized %s: Area=%d cells -> MaxPop=%d\n", newCountry.Name, landCount, newCountry.Max_popu)
			}
		}
	}
}

func (env *Environment) floodFillAssign(start *Cell, country *Country) {
	stack := []*Cell{start}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current.Terrain != 0 || current.Country_owner != nil {
			continue
		}

		current.Country_owner = country
		country.Cells = append(country.Cells, current)

		dirs := [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
		for _, d := range dirs {
			nx, ny := current.X+d[0], current.Y+d[1]
			if nx >= 0 && nx < env.Width && ny >= 0 && ny < env.Height {
				neighbor := env.Grid[ny][nx]
				if neighbor.Terrain == 0 && neighbor.Country_owner == nil {
					stack = append(stack, neighbor)
				}
			}
		}
	}
}
func (env *Environment) DetectCountryNeighbors() {
	fmt.Println(" Recherche pays voisins...")
	connectionsFound := 0

	const searchRadius = 6

	for y := 0; y < env.Height; y += 2 {
		for x := 0; x < env.Width; x += 2 {
			cell := env.Grid[y][x]

			if cell.Terrain != 2 {
				continue
			}

			foundCountries := make(map[int]*Country)

			for dy := -searchRadius; dy <= searchRadius; dy++ {
				for dx := -searchRadius; dx <= searchRadius; dx++ {
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < env.Width && ny >= 0 && ny < env.Height {
						neighborCell := env.Grid[ny][nx]

						if neighborCell.Country_owner != nil {
							foundCountries[neighborCell.Country_owner.Id] = neighborCell.Country_owner
						}
					}
				}
			}

			if len(foundCountries) >= 2 {
				countries := []*Country{}
				for _, c := range foundCountries {
					countries = append(countries, c)
				}

				for i := 0; i < len(countries); i++ {
					for j := i + 1; j < len(countries); j++ {
						c1 := countries[i]
						c2 := countries[j]

						if !c1.Neighbors[c2.Id] {
							c1.Neighbors[c2.Id] = true
							c2.Neighbors[c1.Id] = true
							connectionsFound++
							fmt.Printf("   -> Connect: %s <--> %s\n", c1.Name, c2.Name)
						}
					}
				}
			}
		}
	}
	fmt.Printf("Trouver %d paire connecte.\n", connectionsFound)
}

func (env *Environment) CountryHasLand(c *Country) bool {
	// Si on trouve au moins 1 case valide, le pays est considéré comme "vivant".
	for y := 0; y < env.Height; y++ {
		for x := 0; x < env.Width; x++ {
			cell := env.Grid[y][x]
			// C'est notre pays, ce n'est pas de l'eau (0), et c'est libre
			if cell.Country_owner == c && cell.Terrain == 0 && cell.Group_owner == nil {
				return true // Il reste de l'espoir !
			}
		}
	}
	return false
}

func (e *Environment) FindBestNeighbor(group *Group_population) *Country {
	currentCountry := group.Current_country
	var bestC *Country
	maxScore := -999.0

	for neighborID := range currentCountry.Neighbors {
		c := e.GetCountryByID(neighborID)
		if c == nil {
			continue
		}

		score := 0.0
		safetyScore := float64(c.Max_popu - c.Current_popu)

		// Literate groups consider economy and safety and they don't care about borders
		if group.IsLiterate {
			economyScore := c.CalculateEconomy()
			score = (economyScore * 2.0) + safetyScore
			if c.Current_popu >= c.Max_popu {
				score -= 20.0
			}
		} else if !group.IsLiterate && !group.IsIllegal { // Non-literate but legal groups prioritize safety
			score = safetyScore * 2.0
			if !c.Border {
				score -= 5.0
			}

		} else { // Illegal groups prioritize open borders above all else
			score = safetyScore * 0.5
			if c.Border {
				score += 50.0
			}
		}

		if score > maxScore {
			maxScore = score
			bestC = c
		}
	}
	return bestC
}

func (e *Environment) CountValidCells(c *Country) int {
	count := 0
	for _, cell := range c.Cells {
		if cell.Terrain != 1 && cell.Group_owner == nil {
			count++
		}
	}

	return count
}

func (e *Environment) GetCell(x, y int) *Cell {
	if x >= 0 && x < e.Width && y >= 0 && y < e.Height {
		return e.Grid[y][x]
	}
	return nil
}

// Mettre à jour la carte des groupes pour un groupe donné (1x1)
func (e *Environment) UpdateGroupMap(g *Group_population, c *Country) {
	e.mu.Lock()
	defer e.mu.Unlock()
	cell := e.GetCell(g.Pos.X, g.Pos.Y)
	if cell != nil {
		cell.Group_owner = g
		cell.Country_owner = c
	}
}

// Retirer un groupe de la carte des groupes (1x1)
func (e *Environment) RemoveGroupFromMap(g *Group_population) {
	e.mu.Lock()
	defer e.mu.Unlock()
	cell := e.GetCell(g.Pos.X, g.Pos.Y)
	if cell != nil && cell.Group_owner == g {
		cell.Group_owner = nil
	}
}

var countryColors = []color.RGBA{
	{244, 67, 54, 255}, {233, 30, 99, 255}, {156, 39, 176, 255}, {103, 58, 183, 255},
	{63, 81, 181, 255}, {33, 150, 243, 255}, {3, 169, 244, 255}, {0, 188, 212, 255},
	{0, 150, 136, 255}, {76, 175, 80, 255}, {139, 195, 74, 255}, {205, 220, 57, 255},
	{255, 235, 59, 255}, {255, 193, 7, 255}, {255, 152, 0, 255}, {255, 87, 34, 255},
	{121, 85, 72, 255}, {158, 158, 158, 255}, {96, 125, 139, 255}, {183, 28, 28, 255},
	{136, 14, 79, 255}, {74, 20, 140, 255}, {49, 27, 146, 255}, {26, 35, 126, 255},
	{13, 71, 161, 255}, {1, 87, 155, 255}, {0, 96, 100, 255}, {0, 77, 64, 255},
	{27, 94, 32, 255}, {51, 105, 30, 255}, {130, 119, 23, 255}, {245, 127, 23, 255},
	{255, 111, 0, 255}, {191, 54, 12, 255}, {215, 204, 200, 255}, {239, 235, 233, 255},
	{128, 203, 196, 255}, {100, 181, 246, 255}, {179, 136, 255, 255}, {244, 143, 177, 255},
	{255, 204, 128, 255}, {255, 171, 145, 255}, {121, 134, 203, 255}, {174, 213, 129, 255},
	{255, 138, 128, 255}, {77, 182, 172, 255}, {144, 202, 249, 255}, {129, 199, 132, 255},
	{255, 241, 118, 255}, {255, 183, 77, 255}, {229, 115, 115, 255}, {186, 104, 200, 255},
	{66, 165, 245, 255}, {38, 166, 154, 255}, {102, 187, 106, 255}, {255, 238, 88, 255},
	{255, 167, 38, 255}, {240, 98, 146, 255}, {255, 112, 67, 255}, {161, 136, 127, 255},
}

func countryColor(id int) color.RGBA {
	return countryColors[id%len(countryColors)]
}
func (g *Game) Draw(screen *ebiten.Image) {
	const groupDisplaySize = 5.0

	// Background
	for y := 0; y < g.Env.Height; y++ {
		for x := 0; x < g.Env.Width; x++ {
			cell := g.Env.Grid[y][x]
			bgColor := color.RGBA{0, 0, 0, 0}

			if cell.Terrain == 1 {
				bgColor = color.RGBA{30, 144, 255, 255}
			} else if cell.Terrain == 2 {
				isOpen := false
				isClosed := false
				hasNeighbor := false

				dirs := [][2]int{{0, 1}, {0, -1}, {1, 0}, {-1, 0}}
				uniqueCountries := make(map[int]*Country)

				for _, d := range dirs {
					nx, ny := x+d[0], y+d[1]
					if nx >= 0 && nx < g.Env.Width && ny >= 0 && ny < g.Env.Height {
						neighbor := g.Env.Grid[ny][nx]
						if neighbor.Country_owner != nil {
							uniqueCountries[neighbor.Country_owner.Id] = neighbor.Country_owner
						}
					}
				}

				for _, c := range uniqueCountries {
					hasNeighbor = true
					if c.Border {
						isOpen = true
					} else {
						isClosed = true
					}
				}

				if !hasNeighbor {
					// Pas de pays voisin -> GRIS
					bgColor = color.RGBA{60, 60, 60, 255}
				} else if isOpen && !isClosed {
					// Tout ouvrir -> VERT
					bgColor = color.RGBA{0, 255, 0, 255}
				} else if !isOpen && isClosed {
					// Tout fermer -> ROUGE
					bgColor = color.RGBA{255, 0, 0, 255}
				} else {
					// Conflit -> ORANGE
					bgColor = color.RGBA{255, 165, 0, 255}
				}

			} else {

				if cell.Country_owner != nil {
					bgColor = countryColor(cell.Country_owner.Id)
				} else {
					bgColor = color.RGBA{20, 20, 20, 255}
				}
			}

			ebitenutil.DrawRect(screen, float64(x*g.Env.CellSize), float64(y*g.Env.CellSize), float64(g.Env.CellSize), float64(g.Env.CellSize), bgColor)
		}
	}

	// Agents (Groupes)
	for _, group := range g.Env.Groups {
		var groupColor color.RGBA

		if group.IsIllegal {
			groupColor = color.RGBA{255, 0, 0, 255} // RED - Illegal
		} else if group.IsLiterate {
			groupColor = color.RGBA{0, 0, 255, 255} // BLUE - Literate
		} else {
			groupColor = color.RGBA{255, 255, 255, 255} // WHITE - Non-literate
		}

		offset := (groupDisplaySize - float64(g.Env.CellSize)) / 2
		drawX := float64(group.Pos.X*g.Env.CellSize) - offset
		drawY := float64(group.Pos.Y*g.Env.CellSize) - offset

		ebitenutil.DrawRect(screen, drawX, drawY, groupDisplaySize, groupDisplaySize, groupColor)
	}
	g.DrawLegend(screen)

	// Bouton Play/Pause
	var btnColor color.RGBA
	if g.Sim.IsRunning {
		if g.IsHovering {
			btnColor = color.RGBA{255, 100, 100, 255}
		} else {
			btnColor = color.RGBA{200, 50, 50, 255}
		}
	} else {
		if g.IsHovering {
			btnColor = color.RGBA{100, 255, 100, 255}
		} else {
			btnColor = color.RGBA{50, 200, 50, 255}
		}
	}

	ebitenutil.DrawRect(screen, float64(g.Btn_X), float64(g.Btn_Y), float64(g.Btn_W), float64(g.Btn_H), btnColor)

	msg := " > "
	if g.Sim.IsRunning {
		msg = " || "
	}
	ebitenutil.DebugPrintAt(screen, msg, g.Btn_X+20, g.Btn_Y+12)

	statusMsg := "PAUSED"
	if g.Sim.IsRunning {
		statusMsg = fmt.Sprintf("TOUR: %d", g.Sim.CurrentTour)
	}
	ebitenutil.DebugPrintAt(screen, statusMsg, g.Btn_X+g.Btn_W+10, g.Btn_Y+12)

}
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.Env.Width + UI_WIDTH, g.Env.Height
}

// Draw a legend for the countries on the screen.
func (g *Game) DrawLegend(screen *ebiten.Image) {
	uiX := float64(g.Env.Width)

	ebitenutil.DrawRect(screen, uiX, 0, float64(UI_WIDTH), float64(g.Env.Height), color.RGBA{30, 30, 40, 255})

	ebitenutil.DrawRect(screen, uiX, 0, 2, float64(g.Env.Height), color.White)

	startTextX := int(uiX) + 20
	ebitenutil.DebugPrintAt(screen, "=== DETAILED INFORMATION ===", startTextX, 20)
	ebitenutil.DrawLine(screen, float64(startTextX), float64(startTextX+15), float64(startTextX+20), float64(startTextX+15), color.White)

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("CURRENT TOUR: %d", g.Sim.CurrentTour), startTextX, 50)

	mx, my := ebiten.CursorPosition()

	if mx >= 0 && mx < g.Env.Width && my >= 0 && my < g.Env.Height {
		cell := g.Env.GetCell(mx, my)

		if cell != nil {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Corr: (%d, %d)", mx, my), startTextX, 80)

			terrainStr := "Land"
			if cell.Terrain == 1 {
				terrainStr = "Water"
			} else if cell.Terrain == 2 {
				terrainStr = "Border"
			}
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Terrain: %s", terrainStr), startTextX, 100)

			if cell.Country_owner != nil {
				c := cell.Country_owner

				cColor := countryColor(c.Id)
				ebitenutil.DrawRect(screen, float64(startTextX), 130, 20, 20, cColor)

				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Country: %s", c.Name), startTextX+30, 130)
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("ID: %d", c.Id), startTextX, 160)

				borderStatus := "OPEN"
				if !c.Border {
					borderStatus = "CLOSED"
				}
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Border: %s", borderStatus), startTextX, 180)

				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Population: %d / %d", c.Current_popu, c.Max_popu), startTextX, 200)
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Deaths: %d", c.Total_deaths), startTextX, 220)

				validLand := g.Env.CountValidCells(c)
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Valid Land: %d cells", validLand), startTextX, 240)

				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Number of neighbors: %d", len(c.Neighbors)), startTextX, 270)
				yPos := 290

				var neighborList []*Country
				for nid := range c.Neighbors {
					n := g.Env.GetCountryByID(nid)
					if n != nil {
						neighborList = append(neighborList, n)
					}
				}
				sort.Slice(neighborList, func(i, j int) bool {
					return neighborList[i].Name < neighborList[j].Name
				})

				for _, neighbor := range neighborList {
					ebitenutil.DebugPrintAt(screen, fmt.Sprintf("- %s", neighbor.Name), startTextX+10, yPos)
					yPos += 15
				}

			} else if cell.Terrain == 0 {
				ebitenutil.DebugPrintAt(screen, " No Man's Land", startTextX, 130)
			}

			if cell.Group_owner != nil {
				groupInfoY := 500
				ebitenutil.DebugPrintAt(screen, "--- GROUP INFO ---", startTextX, groupInfoY)
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Group ID: %d", cell.Group_owner.Id), startTextX, groupInfoY+20)
				ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Current Country: %s", cell.Group_owner.Current_country.Name), startTextX, groupInfoY+40)

				if cell.Group_owner.Is_flooded {
					ebitenutil.DebugPrintAt(screen, "WARNING: Flood!", startTextX, groupInfoY+60)
				}
			}
		}
	} else {
		ebitenutil.DebugPrintAt(screen, "Move your mouse over the map", startTextX, 80)
		ebitenutil.DebugPrintAt(screen, "to see detailed information.", startTextX, 100)
	}

	logStartY := 600

	ebitenutil.DebugPrintAt(screen, "=== Migration Diary ===", startTextX, logStartY)

	ebitenutil.DrawLine(screen, float64(startTextX), float64(logStartY+15), float64(startTextX+300), float64(logStartY+15), color.White)

	currentY := logStartY + 25

	for _, log := range g.Env.MigrationLogs {
		ebitenutil.DebugPrintAt(screen, log, startTextX, currentY)
		currentY += 15
	}
}

func (env *Environment) CalculateCurrentLandArea() int {
	landCount := 0
	for y := 0; y < env.Height; y++ {
		for x := 0; x < env.Width; x++ {
			if env.Grid[y][x].Terrain == 0 {
				landCount++
			}
		}
	}
	return landCount
}

// CountryCSVData holds data from the CSV file
type CountryCSVData struct {
	Name         string
	Population   int
	LiteracyRate float64
}

// Mapping des noms anglais vers noms français du CSV
var NameMapping = map[string]string{
	"Morocco":                  "Maroc",
	"Algeria":                  "Algérie",
	"Tunisia":                  "Tunisie",
	"Libya":                    "Libye",
	"Egypt":                    "Égypte",
	"Mauritania":               "Mauritanie",
	"Mali":                     "Mali",
	"Niger":                    "Niger",
	"Chad":                     "Tchad",
	"Sudan":                    "Soudan",
	"Senegal":                  "Sénégal",
	"Gambia":                   "Gambie",
	"Guinea-Bissau":            "Guinée-Bissau",
	"Guinea":                   "Guinée",
	"Sierra Leone":             "Sierra Leone",
	"Liberia":                  "Liberia",
	"Ivory Coast":              "Côte d'Ivoire",
	"Burkina Faso":             "Burkina Faso",
	"Ghana":                    "Ghana",
	"Togo":                     "Togo",
	"Benin":                    "Bénin",
	"Nigeria":                  "Nigeria",
	"Cameroon":                 "Cameroun",
	"Central African Republic": "Centrafrique",
	"South Sudan":              "Soudan du Sud",
	"Eritrea":                  "Érythrée",
	"Djibouti":                 "Djibouti",
	"Equatorial Guinea":        "Guinée équatoriale",
	"Gabon":                    "Gabon",
	"Congo":                    "Congo (Brazzaville)",
	"DR Congo":                 "Congo (RDC)",
	"Uganda":                   "Ouganda",
	"Kenya":                    "Kenya",
	"Somalia":                  "Somalie",
	"Ethiopia":                 "Éthiopie",
	"Rwanda":                   "Rwanda",
	"Burundi":                  "Burundi",
	"Tanzania":                 "Tanzanie",
	"Angola":                   "Angola",
	"Zambia":                   "Zambie",
	"Malawi":                   "Malawi",
	"Mozambique":               "Mozambique",
	"Zimbabwe":                 "Zimbabwe",
	"Botswana":                 "Botswana",
	"Namibia":                  "Namibie",
	"South Africa":             "Afrique du Sud",
	"Lesotho":                  "Lesotho",
	"Eswatini":                 "Eswatini",
	"Madagascar":               "Madagascar",
	"Comoros":                  "Comores",
	"Mauritius":                "Maurice",
	"Seychelles":               "Seychelles",
	"Cabo Verde":               "Cap-Vert",
	"Sao Tome & Principe":      "Sao Tomé-et-Principe",
}

// LoadCountryDataFromCSV loads population and literacy data from CSV
func LoadCountryDataFromCSV(csvPath string) (map[string]CountryCSVData, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.Comma = ','
	reader.LazyQuotes = true

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}

	data := make(map[string]CountryCSVData)

	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		if len(record) < 3 {
			continue
		}

		name := strings.TrimSpace(record[0])
		population, _ := strconv.Atoi(strings.TrimSpace(record[1]))
		literacy, _ := strconv.ParseFloat(strings.TrimSpace(record[2]), 64)

		data[name] = CountryCSVData{
			Name:         name,
			Population:   population,
			LiteracyRate: literacy,
		}
	}

	return data, nil
}

// InitializeCountryStats loads CSV and assigns stats to countries
func (env *Environment) InitializeCountryStats(csvPath string) error {
	csvData, err := LoadCountryDataFromCSV(csvPath)
	if err != nil {
		return err
	}

	// Find min and max population to scale agents proportionally
	minPop := 999999999
	maxPop := 0
	for _, data := range csvData {
		if data.Population < minPop {
			minPop = data.Population
		}
		if data.Population > maxPop {
			maxPop = data.Population
		}
	}

	for _, country := range env.Countries {
		// Map English name to French CSV name
		csvName, exists := NameMapping[country.Name]
		if !exists {
			csvName = country.Name
		}

		if data, found := csvData[csvName]; found {
			country.LiteracyRate = data.LiteracyRate
			country.BasePopulation = data.Population

			// Scale Max_popu proportionally to real population
			// Use log scale to avoid huge differences
			// Range: 5 agents (smallest) to 100 agents (largest)
			popRatio := float64(data.Population-minPop) / float64(maxPop-minPop)
			country.Max_popu = 5 + int(popRatio*95) // 5 to 100 agents

			fmt.Printf("Loaded %s: Pop=%d, Literacy=%.1f%%, MaxAgents=%d\n",
				country.Name, data.Population, data.LiteracyRate, country.Max_popu)
		} else {
			// Default values
			country.LiteracyRate = 50.0
			country.BasePopulation = 1000000
			country.Max_popu = 10
			fmt.Printf("Default stats for %s (not found in CSV)\n", country.Name)
		}
	}

	return nil
}
