const API_URL = "http://localhost:12000";
let isRunning = false;
let popChart, landChart, borderChart; 

document.addEventListener("DOMContentLoaded", () => {
    updateGlobalStatus();
    if (document.getElementById('popChart')) initCharts();
    if (document.getElementById('toggleBtn')) initDashboardControls();

    setInterval(() => {
        updateGlobalStatus(); 
        if (document.getElementById('logBox')) updateLogs();
        if (document.getElementById('kpi-pop')) updateKPIs();
        if (document.getElementById('popChart')) updateCharts();
        if (document.getElementById('countryTableBody')) updateCountriesTable();
    }, 1000);
    if (document.getElementById('simCanvas')) {
        initMapVisualizer();
    }
});

// GET /status in api.go
async function updateGlobalStatus() {
    try {
        const res = await fetch(`${API_URL}/status`);
        const data = await res.json();
        isRunning = data.is_running;

        // Update Navbar Info
        document.getElementById('tourDisplay').innerText = `Tour: ${data.tour}`;
        const statusEl = document.getElementById('globalStatus');
        statusEl.innerText = isRunning ? "RUNNING" : "PAUSED";
        statusEl.className = isRunning ? "navbar-text fw-bold me-3 text-success" : "navbar-text fw-bold me-3 text-danger";

        // Update Dashboard Button if exists
        const btn = document.getElementById('toggleBtn');
        if (btn) {
            if (isRunning) {
                btn.innerHTML = '<i class="bi bi-pause-circle"></i> Pause Simulation';
                btn.className = "btn btn-lg btn-outline-danger w-100 py-3 fw-bold";
            } else {
                btn.innerHTML = '<i class="bi bi-play-circle"></i> Resume Simulation';
                btn.className = "btn btn-lg btn-outline-success w-100 py-3 fw-bold";
            }
        }
    } catch (e) { console.log("Connecting..."); }
}

function initDashboardControls() {
    document.getElementById('toggleBtn').addEventListener('click', async () => {
        await fetch(`${API_URL}/control/toggle`, { method: 'POST' });
        updateGlobalStatus();
    });
}

async function updateLogs() {
    try {
        const res = await fetch(`${API_URL}/logs`);
        const logs = await res.json();
        const logBox = document.getElementById('logBox');
        logBox.innerHTML = logs.slice().reverse().map(l => `<div>> ${l}</div>`).join('');
    } catch (e) {}
}

async function updateKPIs() { // GET /history in api.go
    try {
        const res = await fetch(`${API_URL}/history`);
        const history = await res.json();
        if (history.length > 0) {
            const latest = history[history.length - 1]; 
            document.getElementById('kpi-pop').innerText = latest.TotalPopulation;
            document.getElementById('kpi-death').innerText = latest.TotalDeaths;
            document.getElementById('kpi-land').innerText = latest.TotalLandArea;
        }
    } catch (e) {}
}


// GET /history in api.go
function initCharts() {
    const commonOptions = {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        elements: { point: { radius: 0 } },
        interaction: { mode: 'index', intersect: false },
        plugins: { legend: { position: 'top' } }
    };

    popChart = new Chart(document.getElementById('popChart'), {
        type: 'line',
        data: { labels: [], datasets: [
            { label: 'Population', data: [], borderColor: '#0d6efd', borderWidth: 2 },
            { label: 'Deaths', data: [], borderColor: '#dc3545', borderWidth: 2 }
        ]},
        options: commonOptions
    });

    landChart = new Chart(document.getElementById('landChart'), {
        type: 'line',
        data: { labels: [], datasets: [{ label: 'Land Area', data: [], borderColor: '#198754', backgroundColor: 'rgba(25,135,84,0.1)', fill: true }] },
        options: commonOptions
    });

    borderChart = new Chart(document.getElementById('borderChart'), {
        type: 'line',
        data: { labels: [], datasets: [
            { label: 'Open', data: [], borderColor: '#0dcaf0' },
            { label: 'Closed', data: [], borderColor: '#fd7e14' }
        ]},
        options: commonOptions
    });
}

async function updateCharts() {
    try {
        const res = await fetch(`${API_URL}/history`);
        const history = await res.json();
        if (!history || history.length === 0) return;

        const labels = history.map(h => h.Tour);
        
        popChart.data.labels = labels;
        popChart.data.datasets[0].data = history.map(h => h.TotalPopulation);
        popChart.data.datasets[1].data = history.map(h => h.TotalDeaths);
        popChart.update();

        landChart.data.labels = labels;
        landChart.data.datasets[0].data = history.map(h => h.TotalLandArea);
        landChart.update();

        borderChart.data.labels = labels;
        borderChart.data.datasets[0].data = history.map(h => h.OpenBorders);
        borderChart.data.datasets[1].data = history.map(h => h.ClosedBorders);
        borderChart.update();
    } catch (e) {}
}

// GET /countries in api.go
async function updateCountriesTable() {
    try {
        const res = await fetch(`${API_URL}/countries`);
        let countries = await res.json();
        countries.sort((a, b) => b.pop - a.pop); // Sort by Population

        const tbody = document.getElementById('countryTableBody');
        tbody.innerHTML = countries.map((c, index) => {
            const occupancy = Math.round((c.pop / c.max_pop) * 100);
            let badgeClass = c.border ? "bg-success" : "bg-danger";
            let badgeText = c.border ? "OPEN" : "CLOSED";
            const indicatorFixed = c.indicator ? c.indicator.toFixed(2) : "0.00";

            // Progress Bar Color
            let barColor = "bg-primary";
            if(occupancy > 70) barColor = "bg-warning";
            if(occupancy > 90) barColor = "bg-danger";

            return `
                <tr>
                    <td><strong>#${index + 1}</strong></td>
                    <td>${c.name}</td>
                    <td>${c.pop}</td>
                    <td>${c.max_pop}</td>
                    <td>
                        <div class="progress" style="height: 20px;">
                            <div class="progress-bar ${barColor}" style="width: ${Math.min(occupancy, 100)}%">${occupancy}%</div>
                        </div>
                    </td>
                    <td class="text-primary">${c.literate}</td>
                    <td class="text-secondary">${c.non_literate}</td>
                    <td class="text-danger fw-bold">${c.illegal}</td>
                    <td><strong>${indicatorFixed}</strong></td>
                    <td class="text-danger fw-bold">${c.deaths}</td>
                    <td><span class="badge ${badgeClass}">${badgeText}</span></td>
                </tr>
            `;
        }).join('');
    } catch (e) {}
}