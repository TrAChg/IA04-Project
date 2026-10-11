# Simulation Multi-Agents : Monter de l'eau

**Sujet :** Modélisation des flux migratoires causés par la montée des eaux et analyse des réponses géopolitiques des nations africaines.

---

## 1. Objectifs de la Simulation
Cette simulation a pour but d'observer l'évolution démographique et politique du continent africain face à montée du niveau de la mer. Elle repose sur une approche **Multi-Agents (SMA)** où deux types d'entités interagissent :

1.  **Les Groupes de Population (Migrants) :** Cherchent à survivre et à trouver un refuge optimal.
2.  **Les Pays (Gouvernements) :** Cherchent à maximiser leur stabilité économique tout en gérant la pression migratoire et la sécurité.

---

## 2. Les Agents et Leurs Caractéristiques

### A. Les Groupes de Population
Les populations sont divisées en groupes mobiles. Chaque groupe appartient à une catégorie influençant son comportement et son impact économique.

| Type | Couleur | Apport Économique | Vision (Foresight) | Comportement |
| :--- | :---: | :---: | :---: | :--- |
| **Literate (Instruits)** | 🔵 Bleu | **+1.0** | **Élevée (5 cases)** | Prévoyants, cherchent des pays riches et sûrs. Privilégiés aux frontières (VIP) (Ils pourraient être autorisés à passer la frontière gratuitement.). |
| **Non-Literate (Non-instruits)** | ⚪ Blanc | **0.0** | Faible (1 case) | Réactifs, migrent par nécessité immédiate. Cherchent la sécurité (espace disponible). |
| **Illegal (Illégaux)** | 🔴 Rouge | **-0.3** | Faible (1 case)| Ils cherchent un pays aux frontières ouvertes car ils veulent rentrer légalement. |

### B. Les Pays (États)
Chaque pays est un agent intelligent qui gère son territoire et sa politique frontalière.

* **Capacité (Max_Popu) :** Le nombre maximum de groupes que le pays peut soutenir (basé sur sa surface terrestre: Plus de terres vides => Plus de groupes: max_popu = landCount (Nombre cells sur ce pays) / 500 ).
* **Indicateur de Succès :** Une valeur calculée à chaque tour pour décider d'ouvrir ou fermer les frontières.
* **Politique Frontalière :**
    * *OPEN (Ouverte) :* Pas de coût, mais risque de surpopulation.
    * *CLOSED (Fermée) :* Coût de sécurité élevé, risque de décès pour les migrants, mais protège de la surpopulation.

---

## 3. Règles et Mécanismes

### A. Dynamique de l'Eau (Environnement)
L'eau se propage de manière probabiliste à partir des cases d'eau existantes vers les terres adjacentes.
* **Vitesse :** L'algorithme d'inondation s'exécute plusieurs fois par tour.
* **Probabilité :** Une case terre voisine de l'eau a **15%** de chance d'être inondée à chaque itération.

### B. Prise de Décision des Pays (IA Gouvernementale)
À chaque tour, chaque pays évalue deux scénarios (Ouvrir vs Fermer) et choisit celui qui maximise son **Indicateur de Performance ($I$)**.

**Formule de l'Indicateur :**

$$I = E_{pays} - P_{densité} - P_{sécurité}$$

Où :

1.  **$E_{pays}$ (Économie)** : Somme des apports des groupes présents (Literate +1, Illegal -0.3, Non-literate: 0).
2.  **$P_{densité}$ (Pénalité de Surpopulation)** :
    * Seuil de confort (*OptimalDensity*) : **70%** de la capacité max.
    * Si la population dépasse ce seuil, une pénalité exponentielle est appliquée pour simuler la pression sur les infrastructures.
    * *Formule :* $P_{densité} = (Ratio_{actuel} - 0.40) \times MaxPop \times 5.0$
    * $Ratio_{actuel}$: Le pourcentage actuel de la population s'il dépasse 70 %.
3.  **$P_{sécurité}$ (Coût de Sécurité)** :
    * Si Frontière **FERMÉE** : Coût fixe (**-2.0**).
    * Si Frontière **OUVERTE** : Coût nul (**0**).

> **Logique :** Un pays ferme ses frontières dès qu'il atteint ~70% de sa capacité pour éviter la pénalité de densité, sauf si l'apport économique des migrants (Literate) compense le coût de sécurité.

### C. Stratégie de Migration (Choix de la Destination)
Lorsqu'un groupe doit migrer (inondation proche), il évalue ses voisins selon un **Score d'Attractivité ($S$)**.

**1. Pour les Groupes Instruits (Literate) :**
$$S = (2 \times E_{pays}) + Sécurité_{pays} - Pénalité_{Fermeture}$$
* Ils privilégient l'économie. Ils évitent les frontières fermées sauf nécessité absolue.
* $ Pénalité_{Fermeture}$: -20

**2. Pour les Groupes Non-Instruits (Non-literate):**
$$S = Sécurité_{pays}*2 - Pénalité_{Fermeture}$$
* $Sécurité_{pays} = MaxPop - CurrentPop$ (Places libres).
* $Pénalité_{Fermeture}$: -5
* Ils veulent un pays suffisamment sûr pour qu'ils puissent y vivre.

**3. Pour les Groupes Illégaux :**
$$S = 1/2 *Sécurité_{pays} + Bonus_{Ouverture}$$
* $Sécurité_{pays} = MaxPop - CurrentPop$ (Places libres).
* $Bonus_{Ouverture}$: +50
* Ils cherchent avant tout de l'espace libre.
* Les illégaux préfèrent les frontières ouvertes pour tenter de redevenir légaux ("régularisation").

### D. Passage des Frontières et Mortalité
Le passage d'une frontière comporte des risques définis par le statut du pays cible :

1.  **Frontière OUVERTE :**
    * Passage sûr.
    * Le migrant garde son statut (ou devient Légal s'il était Illégal).

2.  **Frontière FERMÉE :**
    * **Literate (VIP) :** Passage autorisé (Brain Gain). Reste Légal.
    * **Non-Literate :**
        * **40% de chance de mourir** à la frontière.
        * **60% de chance de passer**, mais devient **Illegal** (Rouge).

### E. Tableau Récapitulatif des API

Ce tableau résume tous les points de terminaison (endpoints) disponibles pour la communication entre le Backend (Go) et le Frontend (HTML/JS).

| Endpoint | Méthode | Description / Fonction | Données Retournées (JSON) | Lien d'Accès (Test) |
| :--- | :---: | :--- | :--- | :--- |
| **/status** | `GET` | Récupère l'état global. | Tour, État,... Map. | http://localhost:12000/status |
| **/countries** | `GET` | Détails complets des pays (Classement). | Liste Pays, Pop, Indicateurs. | http://localhost:12000/countries |
| **/groups** | `GET` | Position et type de tous les agents (Carte). | Coordonnées X/Y, Type, Statut. | http://localhost:12000/groups |
| **/history** | `GET` | Historique des stats tour par tour (Graphiques). | Liste Snapshots (Stats globales). | http://localhost:12000/history |
| **/logs** | `GET` | Derniers journaux d'événements. | Liste de Strings (Logs). | http://localhost:12000/logs |
| **/control/toggle** | `POST` | Pause / Reprise de la simulation. | Nouvel état (`is_running`). | *(Requête POST - Non cliquable)* |
| **/group/{id}** | `GET` | Détails d'un groupe spécifique (Debug). | Position, Pays, Inondé. | http://localhost:12000/group/0 |
# Guide d'Utilisation

Ce guide explique comment installer, lancer et interagir avec la simulation de migration climatique via le tableau de bord web.

---

## 1. Prérequis techniques

Avant de lancer la simulation, assurez-vous que les éléments suivants sont installés sur votre machine :

* **Go (Golang)** : Version 1.18 ou supérieure.
    * *Vérifier l'installation :* Ouvrez un terminal et tapez `go version`.
* **Navigateur Web** : Google Chrome, Firefox, ou Edge (pour le tableau de bord).

---

## 2. Installation et Configuration

1.  **Ouvrir le terminal** dans le dossier racine du projet (là où se trouvent `main.go`, `go.mod`, etc.).
2.  **Télécharger les dépendances** nécessaires (Ebiten, etc.) en exécutant la commande suivante :

    ```bash
    go mod tidy
    ```
3.  **Vérifier la structure des fichiers** :
    Assurez-vous que le dossier `html`, `cmd`,... contient bien les sous-dossiers et fichiers suivants :
    ```text
    Projet_simulation/
    │
    ├── go.mod                  # module Go
    ├── go.sum                  # Checksum dependencies
    ├── cmd                       
        └── simulation/
            └── main.go         # Fonction principale
    ├── agent.go                # Logic Country et Group_population
    ├── env.go                  # Logic ENV, lire map, logic water
    ├── simu.go                 # Faire iteration sur simulation, StatsSnapshot
    ├── api.go                  # Server API (Backend)
    │
    ├── afrique.png             # Img pour lire des donneéss
    │
    ├── readme.md               
    │
    └── html/                   # Interface sur website
        ├── index.html          # Dashboard & Logs & Map Visualizer
        ├── analytics.html      # Charts
        ├── countries.html      # Information détaillées sur chaque pays
        │
        └── js/
            └── sea.js          # Logic Javascript (Fetch API, ...)
    ```

---

## 3. Lancement de la Simulation

Pour démarrer le programme, exécutez la commande suivante dans votre terminal :

```bash
cd Nom_de_votre_projet
go run cmd/simulation/main.go
```
Le terminal affichera le message : API Server started on http://localhost:12000.
## 4. Scénario de Simulation
La simulation se déroule en tours (tours de jeu).

* **Tours 1 :** Début de la montée des eaux. Premiers déplacements internes. Les pays commencent à se remplir.
* **Tours 100 :** Crise migratoire. Les pays côtiers disparaissent. Les pays voisins ferment leurs frontières (atteinte du seuil de 40%). Augmentation massive des décès et des populations illégales.
* **Tours 150 :** Saturation continentale. Les pays "refuges" (au centre) sont surpeuplés. L'économie globale chute à cause du poids des populations illégales et des coûts de sécurité.

---

## 5. Indicateurs de Suivi (Dashboard)
Les données suivantes sont collectées en temps réel via l'API :


* **Total Population vs Décès :** Mesure l'impact humanitaire:
* **Surface Terrestre :** Mesure l'impact monter de l'eau direct: 

* **Politiques Frontalières :** Nombre de pays en mode "Ouvert" vs "Fermé".
* **Classement des Pays :** Basé sur la population survivante et la stabilité.

## 6. Architecture Technique & Concurrence

Pour respecter les principes des Systèmes Multi-Agents (SMA) et optimiser les performances, la simulation utilise intensivement la concurrence native de Go (Goroutines).

* **Cycle de vie (RunOneTour) :**
    1.  **Environnement :** Mise à jour synchrone de la montée des eaux.
    2.  **Agents Pays (Gouvernements) :** Chaque pays exécute sa prise de décision (`Live`) dans sa propre **Goroutine**.
    3.  **Agents Groupes (Migrants) :** Chaque groupe de population calcule ses déplacements et interagit avec l'environnement dans une **Goroutine indépendante**.
    
* **Synchronisation :**
    * L'utilisation de `sync.WaitGroup` garantit que tous les agents ont terminé leurs actions avant de passer au tour suivant.
    * L'accès concurrentiel à la grille (Grid) et aux ressources partagées est protégé par des **Mutex (`sync.Mutex` & `sync.RWMutex`)** pour éviter les "Race Conditions".