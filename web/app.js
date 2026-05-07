const spo2El = document.getElementById('spo2-val');
const hrEl = document.getElementById('hr-val');
const statusEl = document.getElementById('status');
const canvas = document.getElementById('waveform-canvas');
const ctx = canvas.getContext('2d');

let waveformData = [];
const MAX_POINTS = 300; // How many points to show on screen
let animationId;

// Resize canvas to match display size
function resizeCanvas() {
    canvas.width = canvas.parentElement.clientWidth;
    canvas.height = canvas.parentElement.clientHeight;
}
window.addEventListener('resize', resizeCanvas);
resizeCanvas();

function connect() {
    const wsProto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${wsProto}//${location.host}/ws`);

    ws.onopen = () => {
        statusEl.textContent = 'Connected';
        statusEl.className = 'status connected';
    };

    ws.onclose = () => {
        statusEl.textContent = 'Disconnected';
        statusEl.className = 'status disconnected';
        setTimeout(connect, 3000);
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        
        if (data.type === 'spo2') {
            spo2El.textContent = data.value;
        } else if (data.type === 'pulse') {
            hrEl.textContent = data.value;
            // Briefly add class to trigger heartbeat animation
            const hrCard = document.querySelector('.hr-card');
            hrCard.classList.remove('beating');
            void hrCard.offsetWidth; // trigger reflow
            hrCard.classList.add('beating');
        } else if (data.type === 'waveform') {
            waveformData.push(data.value);
            if (waveformData.length > MAX_POINTS) {
                waveformData.shift();
            }
        }
    };
}

function drawWaveform() {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    
    if (waveformData.length < 2) {
        animationId = requestAnimationFrame(drawWaveform);
        return;
    }

    // Auto-scale Y axis
    let min = Math.min(...waveformData);
    let max = Math.max(...waveformData);
    if (max - min < 10) {
        max = min + 10;
    }
    const range = max - min;
    
    const stepX = canvas.width / MAX_POINTS;

    ctx.beginPath();
    ctx.strokeStyle = '#ef4444'; // Accent red
    ctx.lineWidth = 3;
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';

    for (let i = 0; i < waveformData.length; i++) {
        const x = i * stepX;
        // Invert Y since canvas 0 is top
        const normalizedY = (waveformData[i] - min) / range;
        const y = canvas.height - (normalizedY * canvas.height * 0.8) - (canvas.height * 0.1);

        if (i === 0) {
            ctx.moveTo(x, y);
        } else {
            ctx.lineTo(x, y);
        }
    }
    
    // Add glow effect
    ctx.shadowBlur = 10;
    ctx.shadowColor = 'rgba(239, 68, 68, 0.5)';
    ctx.stroke();
    
    // Reset shadow for next frame
    ctx.shadowBlur = 0;

    animationId = requestAnimationFrame(drawWaveform);
}

connect();
drawWaveform();
