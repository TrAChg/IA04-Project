EQUATIONS : 

Net impact(group populations) = Contributions(group populations) - Consumptions(group populations)

**For illegal immigrants:**
Net impact(illegal immigrant) = IllegalImmigrantPenalty = -0.3 (constant negative impact)

Total_impact() = sum (Net impact)

$$Carrying\ Capacity = (Land\ Area - Flooded\ Area) \times \text{Base\_Density} \times Infrastructure\_Index$$

$$ Saturation(country) = \frac{Current\ Population}{Carrying\ Capacity}$$

Consumptions(group populations) =  (1 - Integration_Progress) × Service_Quality_Factor

Contributions(group populations) = Literacy_Factor × Economic_Productivity × Integration_Progress

Where Literacy_Factor:
- Literate person: 1.0
- Non-literate person: 0.3 (still contributes some labor)

Death Number(country) = Number of agent dead in and only in the country


ATTRIBUTE OF EACH EQUATIONS : 

ATTRIBUT FIX: 

Economic_Productivity(C)= PIB_PPA_par_habitant(C)−PIB_min,Africa / (PIB_max,Africa−PIB_min,Africa)

Infrastructure_Index = AIDI_normalisé (valeur AIDI / 100)

Integration_Progress(C)=α⋅Edu_Capacity_Index(C)+β⋅Economic_Productivity(C)+γ⋅Infrastructure_Index(C)
with α=0.4 (éducation: écoles + universités + accès) β=0.3 (économie) γ=0.3 (infrastructure)

Service_Quality_Factor(C) = 0.5×Edu_Access_Index(C) + 0.5×Health_Access_Index(C)

with Edu_Access_Index(C)=0.4 × Primary_Enrollment/100 + 0.4 × Secondary_Enrollment/100 + 0.2 x Tertiary_Enrollment/100


ATRIBUT CHANGING :

DEPENDING ON THE ENVIRONMENT (changing but fix one time) :

Density_Threshold = 1.5 (Base Population / (Land_Area-Flooded Area))

DEPENDING ON THE AGENTS : 

Litteracy rate = current number of litterate agents/total of the current agents 

INDICATOR OF THE SIMULATION : 

Every country wants to maximize the next equations : 

**When border is OPEN:**
$$Indicator = \beta \cdot Total\_impact - (1 - \beta) \cdot Death\_Number - \delta \cdot \max(0, saturation - 1)$$

**When border is CLOSED (includes Security Cost):**
$$Indicator = \beta \cdot Total\_impact - (1 - \beta) \cdot Death\_Number - \delta \cdot \max(0, saturation - 1) - SecurityCost$$

**Constants:**
- β (Beta) = 0.5 (weight for economic impact vs deaths)
- δ (Delta) = 0.1 (penalty for overpopulation)
- SecurityCost = 0.02 (cost of maintaining closed borders)
- IllegalImmigrantPenalty = -0.3 (negative economic impact of illegal immigrants)
- IllegalCrossingDeathRisk = 0.3 (30% chance of death when crossing closed border)
- MaxTours = 150 (game ends at round 150)
- WaterStopTour = 100 (water stops rising at round 100)
- CollapseThreshold = -10.0 (country collapses if indicator falls below this)
- Non-literate contribution factor = 0.3 (non-literate people still contribute some labor)

COUNTRY COLLAPSE : 

A country is considered **collapsed** when:
$$Indicator < -10.0$$

Collapse conditions:
- Only checked after round 10 (to let countries stabilize)
- Indicator falls below -10.0
- OR all land is flooded (no remaining land cells)

When a country collapses:
- Is_Collapsed = true
- The country is displayed in **dark gray** color
- Population will try to flee to neighboring countries
- The country can no longer support its population

Rules : 
 
1 : Every family wants to stay away water
2 : Crossing the border illegally may cause death (30% death risk)
3 : Staying in a country where there is a too much people can cause death
4 : Every family wants to stay away a collapsed country
5 : A country can collapsed because of his total impact (Total_impact < 0)
6 : A country can collapsed because of the water
7 : Illegal immigrants have a fixed negative net impact (-0.3)
8 : Closing borders has a security cost (-0.05 on indicator)
9 : The simulation ends at round 100 (water stops rising)
10 : Countries are ranked by their final Indicator value

ILLEGAL IMMIGRATION :

When a population group attempts to cross into a country with **closed borders**:
1. Death risk check: 30% chance of death during crossing attempt
2. If survives: Enters as illegal immigrant (Is_Illegal = true)
3. Illegal immigrants contribute a fixed negative net impact (-0.3)
4. Illegal immigrants are displayed in **red** on the map

VISUAL REPRESENTATION :

Agent colors on the map:
- **Blue** : Legal literate population
- **White** : Legal non-literate population  
- **Red** : Illegal immigrants (crossed closed border)

VOTE

BORDER DECISION (deliberateBorder):

Each country decides to open or close its border based on maximizing its Indicator function:
1. Calculate Indicator if border is OPEN (allows legal immigration)
2. Calculate Indicator if border is CLOSED (includes security cost, may create illegal immigrants)
3. Choose the option that gives the higher Indicator value

GAME END :

At round 150:
1. Final ranking is displayed
2. Countries ranked by Indicator function (highest = best survival)
3. Display shows: Rank, Country Name, Indicator Value, Population, Collapsed Status

At round 100:
1. Water stops rising (flooding stops)

INITIALISATION :

1. Every Border has an aleatory state.
2. No country is flooded
3. Max 20 agents per country, distributed proportionally by land area
4. Every characteristic of each country is from real sources (CSV data)
5. Literacy rate is distributed according to country's literacy rate statistic
6. Tour duration: 100ms per round (fast simulation)

