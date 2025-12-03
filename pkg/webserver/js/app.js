// app.js
// Fetch and render instances, including provider field
function refreshInstances() {
    const grid = document.getElementById('instances-list');
    grid.innerHTML = '<p class="loading">Loading instances...</p>';
    fetch('/api/instances')
        .then(res => res.json())
        .then(data => {
            if (!data.success || !Array.isArray(data.data)) {
                grid.innerHTML = '<p class="error">Failed to load instances.</p>';
                return;
            }
            if (data.data.length === 0) {
                grid.innerHTML = '<p>No instances found.</p>';
                return;
            }
            grid.innerHTML = '';
            data.data.forEach(instance => {
                const card = document.createElement('div');
                card.className = 'instance-card';
                card.innerHTML = `
                    <div class="instance-id"><b>ID:</b> ${instance.id}</div>
                    <div class="instance-detail"><span class="instance-detail-label">Type:</span> <span class="instance-detail-value">${instance.instance_type}</span></div>
                    <div class="instance-detail"><span class="instance-detail-label">Provider:</span> <span class="instance-detail-value">${instance.provider || 'aws'}</span></div>
                    <div class="instance-detail"><span class="instance-detail-label">State:</span> <span class="instance-detail-value">${instance.state}</span></div>
                    <div class="instance-detail"><span class="instance-detail-label">Public IP:</span> <span class="instance-detail-value">${instance.public_ip || '-'}</span></div>
                    <div class="instance-detail"><span class="instance-detail-label">Zone:</span> <span class="instance-detail-value">${instance.availability_zone}</span></div>
                    <div class="instance-detail"><span class="instance-detail-label">Expires At:</span> <span class="instance-detail-value">${instance.expires_at}</span></div>
                `;
                grid.appendChild(card);
            });
        })
        .catch(() => {
            grid.innerHTML = '<p class="error">Failed to load instances.</p>';
        });
}

document.addEventListener('DOMContentLoaded', function() {
    refreshInstances();
    // Add more event listeners as needed
});
