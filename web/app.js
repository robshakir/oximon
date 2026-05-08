const spo2El = document.getElementById('spo2-val');
const hrEl = document.getElementById('hr-val');
const statusEl = document.getElementById('status');

const canvas = document.getElementById('waveform-canvas');
const ctx = canvas.getContext('2d');

const spo2Canvas = document.getElementById('spo2-canvas');
const spo2Ctx = spo2Canvas.getContext('2d');
const spo2Min = document.getElementById('spo2-min');
const spo2Mean = document.getElementById('spo2-mean');
const spo2Max = document.getElementById('spo2-max');
const spo2HistCanvas = document.getElementById('spo2-hist-canvas');
const spo2HistCtx = spo2HistCanvas.getContext('2d');

const hrCanvas = document.getElementById('hr-canvas');
const hrCtx = hrCanvas.getContext('2d');
const hrMin = document.getElementById('hr-min');
const hrMean = document.getElementById('hr-mean');
const hrMax = document.getElementById('hr-max');
const hrHistCanvas = document.getElementById('hr-hist-canvas');
const hrHistCtx = hrHistCanvas.getContext('2d');

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
    spo2HistCanvas.width = spo2HistCanvas.parentElement.clientWidth;
    spo2HistCanvas.height = spo2HistCanvas.parentElement.clientHeight;
    
    hrCanvas.width = hrCanvas.parentElement.clientWidth;
    hrCanvas.height = hrCanvas.parentElement.clientHeight;
    hrHistCanvas.width = hrHistCanvas.parentElement.clientWidth;
    hrHistCanvas.height = hrHistCanvas.parentElement.clientHeight;
    
    drawTrend(spo2Ctx, spo2Canvas, spo2Data, '#0ea5e9', 90, 100);
    drawHistogram(spo2HistCtx, spo2HistCanvas, spo2Data, '#0ea5e9');
    
    drawTrend(hrCtx, hrCanvas, hrData, '#ef4444', 50, 120);
    drawHistogram(hrHistCtx, hrHistCanvas, hrData, '#ef4444');
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

function updateStats(data, minEl, meanEl, maxEl) {
    if (data.length === 0) return;
    const min = Math.min(...data);
    const max = Math.max(...data);
    const mean = data.reduce((a, b) => a + b, 0) / data.length;
    
    minEl.textContent = min;
    maxEl.textContent = max;
    meanEl.textContent = Math.round(mean);
}

function drawHistogram(ctx, cnv, data, colour) {
    ctx.clearRect(0, 0, cnv.width, cnv.height);
    if (data.length < 2) return;

    const numBins = 15;
    const min = Math.min(...data);
    const max = Math.max(...data);
    // Add small epsilon to range to avoid dividing by 0 if all values are identical
    const range = (max - min) || 1; 
    const binSize = range / numBins;
    
    const bins = new Array(numBins).fill(0);
    for (let val of data) {
        let binIndex = Math.floor((val - min) / binSize);
        if (binIndex >= numBins) binIndex = numBins - 1;
        bins[binIndex]++;
    }

    const maxCount = Math.max(...bins) || 1;
    const barWidth = cnv.width / numBins;

    ctx.fillStyle = colour;
    ctx.shadowBlur = 5;
    ctx.shadowColor = colour.replace(')', ', 0.3)').replace('rgb', 'rgba');
    if (colour.startsWith('#')) {
        ctx.shadowColor = colour + '40';
    }

    for (let i = 0; i < numBins; i++) {
        const barHeight = (bins[i] / maxCount) * cnv.height * 0.9; // max 90% of height
        if (barHeight === 0) continue;
        
        const x = i * barWidth;
        const y = cnv.height - barHeight;
        
        ctx.fillRect(x + 1, y, barWidth - 2, barHeight);
    }
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
            updateStats(spo2Data, spo2Min, spo2Mean, spo2Max);
            drawHistogram(spo2HistCtx, spo2HistCanvas, spo2Data, '#0ea5e9');
        } else if (data.type === 'pulse') {
            hrEl.textContent = data.value;
            hrData.push(data.value);
            if (hrData.length > TREND_POINTS) hrData.shift();
            drawTrend(hrCtx, hrCanvas, hrData, '#ef4444', 50, 120);
            updateStats(hrData, hrMin, hrMean, hrMax);
            drawHistogram(hrHistCtx, hrHistCanvas, hrData, '#ef4444');
            
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
