let eventSource;
const eventCounts = { WRITE: 0, CREATE: 0, REMOVE: 0 };
const eventTimes = {};

function startWatching() {
    const dir = document.getElementById('directory').value;
    if (!dir) {
        alert('Please enter a directory to watch.');
        return;
    }
    fetch('/watch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ dir })
    }).then(res => {
        if (res.ok) {
            alert('Started watching: ' + dir);
            startEventStream();
        } else {
            alert('Failed to start watching. Check the directory.');
        }
    });
}

function startEventStream() {
    if (eventSource) eventSource.close();
    eventSource = new EventSource('/events');

    eventSource.onopen = function() {
        console.log("Connected to /events");
    };

    eventSource.onmessage = function(event) {
        console.log("Event received:", event.data); // Verifica que los datos llegan
        const events = document.getElementById('events');
        const data = event.data.match(/\[(.*?)\] (\w+)\s"(.*?)"\s\(Last Modified:\s(.*?)\)/);
        if (data) {
            const time = data[1]; // Hora del evento
            const action = data[2]; // Tipo de acción
            const file = data[3]; // Nombre del archivo

            // Actualizar contadores
            if (eventCounts[action] !== undefined) {
                eventCounts[action]++;
                updateEventTypeChart();
            }

            // Actualizar tiempos
            const date = time.split(" ")[0];
            if (!eventTimes[date]) {
                eventTimes[date] = 0;
            }
            eventTimes[date]++;
            updateEventTimeChart();

            // Mostrar en el área de eventos
            const formattedEvent = `Time: ${time}\nAction: ${action}\nFile: ${file}\n\n`;
            events.value += formattedEvent;
            events.scrollTop = events.scrollHeight;
        } else {
            console.warn("Event data did not match expected format:", event.data);
        }
    };

    eventSource.onerror = function() {
        console.error("Error connecting to /events");
    };
}

// Gráfico de Tipos de Eventos (Barras)
const eventTypeChart = new Chart(document.getElementById('eventTypeChart'), {
    type: 'bar',
    data: {
        labels: ['WRITE', 'CREATE', 'REMOVE'],
        datasets: [{
            label: 'Cantidad de Eventos',
            data: [0, 0, 0], // Inicialmente vacío
            backgroundColor: ['#1e90ff', '#32cd32', '#ff6347'], // Colores personalizados
            borderWidth: 1,
        }]
    },
    options: {
        plugins: {
            legend: {
                display: true,
                position: 'top',
                labels: {
                    font: {
                        size: 14,
                        weight: 'bold',
                    },
                    color: '#333'
                }
            },
        },
        scales: {
            x: {
                title: {
                    display: true,
                    text: 'Tipo de Evento',
                    font: {
                        size: 14,
                        weight: 'bold',
                    },
                    color: '#333',
                }
            },
            y: {
                title: {
                    display: true,
                    text: 'Cantidad',
                    font: {
                        size: 14,
                        weight: 'bold',
                    },
                    color: '#333',
                },
                beginAtZero: true
            }
        },
        responsive: true,
        maintainAspectRatio: false, // Asegura que sea responsivo
    }
});

function updateEventTypeChart() {
    eventTypeChart.data.datasets[0].data = [
        eventCounts.WRITE,
        eventCounts.CREATE,
        eventCounts.REMOVE
    ];
    eventTypeChart.update();
}

// Gráfico de Eventos por Día
const eventTimeChart = new Chart(document.getElementById('eventTimeChart'), {
    type: 'bar',
    data: {
        labels: [], // Fechas de eventos
        datasets: [{
            label: 'Eventos por Día',
            data: [], // Cantidad de eventos
            backgroundColor: 'rgba(255, 165, 0, 0.7)', 
            borderColor: 'rgba(255, 165, 0, 1)', 
            borderWidth: 1,
        }]
    },
    options: {
        plugins: {
            legend: {
                display: false,
            }
        },
        scales: {
            x: {
                title: {
                    display: true,
                    text: 'Días',
                    font: {
                        size: 14,
                        weight: 'bold',
                    },
                    color: '#333',
                }
            },
            y: {
                title: {
                    display: true,
                    text: 'Cantidad de Eventos',
                    font: {
                        size: 14,
                        weight: 'bold',
                    },
                    color: '#333',
                },
                beginAtZero: true
            }
        },
        responsive: true,
        maintainAspectRatio: false, 
    }
});

function updateEventTimeChart() {
    eventTimeChart.data.labels = Object.keys(eventTimes);
    eventTimeChart.data.datasets[0].data = Object.values(eventTimes);
    eventTimeChart.update();
}
