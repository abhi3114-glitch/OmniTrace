document.addEventListener('DOMContentLoaded', () => {
    // Navigation
    const navLinks = document.querySelectorAll('.nav-links li');
    const views = document.querySelectorAll('.view');

    navLinks.forEach(link => {
        link.addEventListener('click', () => {
            // Remove active class
            navLinks.forEach(l => l.classList.remove('active'));
            views.forEach(v => v.classList.remove('active'));

            // Add active class
            link.classList.add('active');
            const viewId = link.getAttribute('data-view') + '-view';
            const view = document.getElementById(viewId);
            if (view) view.classList.add('active');

            if (link.getAttribute('data-view') === 'traces') {
                loadTraces();
            }
        });
    });

    // Close Modal
    document.querySelector('.close-modal').addEventListener('click', () => {
        document.getElementById('trace-detail-modal').classList.add('hidden');
    });

    // Initial Load
    // loadTraces();
});

async function loadTraces() {
    const list = document.getElementById('trace-list');
    list.innerHTML = '<div class="loading">Loading...</div>';

    try {
        const response = await fetch('/api/traces?limit=20');
        const traces = await response.json();

        list.innerHTML = '';
        if (!traces || traces.length === 0) {
            list.innerHTML = '<div class="empty">No traces found</div>';
            return;
        }

        traces.forEach(trace => {
            const item = document.createElement('div');
            item.className = 'trace-item ' + (trace.has_error ? 'error' : '');
            item.innerHTML = `
                <div class="trace-main">
                    <div class="trace-op">${escapeHtml(trace.root_operation || 'root')}</div>
                    <div class="trace-svc">${escapeHtml(trace.root_service || 'unknown')}</div>
                </div>
                <div class="trace-meta">
                    <div class="trace-dur">${formatDuration(trace.duration)}</div>
                    <div class="trace-time">${formatTime(trace.start_time)}</div>
                </div>
            `;
            item.addEventListener('click', () => showTraceDetail(trace.trace_id));
            list.appendChild(item);
        });
    } catch (err) {
        list.innerHTML = `<div class="error">Failed to load traces: ${err.message}</div>`;
    }
}

async function showTraceDetail(traceId) {
    const modal = document.getElementById('trace-detail-modal');
    const vis = document.getElementById('trace-vis');
    
    modal.classList.remove('hidden');
    vis.innerHTML = '<div class="loading">Loading trace details...</div>';

    try {
        const response = await fetch(`/api/traces/${traceId}`);
        const trace = await response.json();

        renderWaterfall(trace, vis);
    } catch (err) {
        vis.innerHTML = `<div class="error">Failed to load detail: ${err.message}</div>`;
    }
}

function renderWaterfall(trace, container) {
    if (!trace.spans || trace.spans.length === 0) {
        container.innerHTML = 'Empty trace';
        return;
    }

    // Sort spans by start time
    const spans = trace.spans.sort((a, b) => new Date(a.start_time) - new Date(b.start_time));
    
    // Calculate total duration for scaling
    const traceStart = new Date(trace.start_time).getTime();
    const traceEnd = new Date(trace.end_time).getTime();
    const totalDuration = traceEnd - traceStart;

    let html = `<h3>${escapeHtml(trace.root_span?.operation_name || 'Trace')} <small>(${formatDuration(totalDuration * 1000000)})</small></h3>`;
    html += '<div class="waterfall-container">';

    spans.forEach(span => {
        const start = new Date(span.start_time).getTime();
        const end = new Date(span.end_time).getTime();
        const duration = end - start;
        
        const leftPercent = ((start - traceStart) / totalDuration) * 100;
        const widthPercent = Math.max((duration / totalDuration) * 100, 0.5); // Min width 0.5%

        html += `
            <div class="waterfall-row">
                <div class="waterfall-label" title="${escapeHtml(span.operation_name)}">
                    ${escapeHtml(span.service_name)}: ${escapeHtml(span.operation_name)}
                </div>
                <div class="waterfall-bar-container">
                    <div class="waterfall-bar ${span.status === 'error' ? 'error' : ''}" 
                         style="left: ${leftPercent}%; width: ${widthPercent}%;">
                    </div>
                </div>
                <div style="width: 80px; text-align: right; font-size: 0.75rem;">
                    ${formatDuration(span.duration)}
                </div>
            </div>
        `;
    });
    html += '</div>';
    
    container.innerHTML = html;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.innerText = text;
    return div.innerHTML;
}

function formatDuration(nanos) {
    const ms = nanos / 1000000;
    if (ms < 1) return '<1ms';
    if (ms > 1000) return (ms/1000).toFixed(2) + 's';
    return ms.toFixed(1) + 'ms';
}

function formatTime(isoString) {
    const date = new Date(isoString);
    return date.toLocaleTimeString();
}
