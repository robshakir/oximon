const spo2El = document.getElementById('spo2-val');
const hrEl = document.getElementById('hr-val');
const statusEl = document.getElementById('status');

const canvas = document.getElementById('waveform-canvas');
const ctx = canvas.getContext('2d');

const spo2Canvas = document.getElementById('spo2-canvas');
const spo2Ctx = spo2Canvas.getContext('2d');

const hrCanvas = document.getElementById('hr-canvas');
const hrCtx = hrCanvas.getContext('2d');

let waveformData = [];
const MAX_POINTS = 300; // How many points to show on screen for waveform
let animationId;

let spo2Data = [];
let hrData = [];
const TREND_POINTS = 60; // Show last 60 readings

// Resize canvas to match display size
function resizeCanvas() {
    canvas.width = canvas.parentElement.clientWidth;
    canvas.height = canvas.parentElement.clientHeight;
    
    spo2Canvas.width = spo2Canvas.parentElement.clientWidth;
    spo2Canvas.height = spo2Canvas.parentElement.clientHeight;
    
    hrCanvas.width = hrCanvas.parentElement.clientWidth;
    hrCanvas.height = hrCanvas.parentElement.clientHeight;
    
    drawTrend(spo2Ctx, spo2Canvas, spo2Data, '#0ea5e9', 90, 100);
    drawTrend(hrCtx, hrCanvas, hrData, '#ef4444', 50, 120);
}
window.addEventListener('resize', resizeCanvas);
resizeCanvas();

function drawTrend(ctx, cnv, data, colour, defaultMin, defaultMax) {
    ctx.clearRect(0, 0, cnv.width, cnv.height);
    if (data.length < 2) return;

    // Use default boundaries, but allow auto-scaling if values exceed them
    let min = Math.min(...data, defaultMin);
    let max = Math.max(...data, defaultMax);
    if (max - min < 5) max = min + 5;
    const range = max - min;
    
    const stepX = cnv.width / TREND_POINTS;

    ctx.beginPath();
    ctx.strokeStyle = colour;
    ctx.lineWidth = 3;
    ctx.lineJoin = 'round';
    ctx.lineCap = 'round';

    for (let i = 0; i < data.length; i++) {
        // Draw from right to left (newest on the right)
        const x = cnv.width - ((data.length - 1 - i) * stepX);
        const normalisedY = (data[i] - min) / range;
        const y = cnv.height - (normalisedY * cnv.height * 0.8) - (cnv.height * 0.1);

        if (i === 0) {
            ctx.moveTo(x, y);
        } else {
            ctx.lineTo(x, y);
        }
    }
    
    // Add glow effect
    ctx.shadowBlur = 10;
    ctx.shadowColor = colour.replace(')', ', 0.5)').replace('rgb', 'rgba'); // simple hack for hex glow
    if (colour.startsWith('#')) {
        ctx.shadowColor = colour + '80'; // hex transparency
    }
    
    ctx.stroke();
    ctx.shadowBlur = 0;
}

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
            spo2Data.push(data.value);
            if (spo2Data.length > TREND_POINTS) spo2Data.shift();
            drawTrend(spo2Ctx, spo2Canvas, spo2Data, '#0ea5e9', 90, 100);
        } else if (data.type === 'pulse') {
            hrEl.textContent = data.value;
            hrData.push(data.value);
            if (hrData.length > TREND_POINTS) hrData.shift();
            drawTrend(hrCtx, hrCanvas, hrData, '#ef4444', 50, 120);
            
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
        const normalisedY = (waveformData[i] - min) / range;
        const y = canvas.height - (normalisedY * canvas.height * 0.8) - (canvas.height * 0.1);

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
