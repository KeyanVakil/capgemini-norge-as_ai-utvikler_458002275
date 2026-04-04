function formatDate(dateStr) {
    const d = new Date(dateStr);
    const now = new Date();
    const diffMs = now - d;
    const diffMin = Math.floor(diffMs / 60000);
    const diffHr = Math.floor(diffMs / 3600000);

    if (diffMin < 1) return 'just now';
    if (diffMin < 60) return `${diffMin}m ago`;
    if (diffHr < 24) return `${diffHr}h ago`;

    return d.toLocaleDateString('en-US', {
        month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
    });
}

function formatDuration(ms) {
    if (ms < 1000) return `${ms}ms`;
    const seconds = (ms / 1000).toFixed(1);
    if (seconds < 60) return `${seconds}s`;
    const minutes = Math.floor(ms / 60000);
    const remainSec = ((ms % 60000) / 1000).toFixed(0);
    return `${minutes}m ${remainSec}s`;
}

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}
