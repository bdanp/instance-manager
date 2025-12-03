package webserver

func getIndexHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AWS Instance Manager</title>
    <link rel="stylesheet" href="/css/style.css">
</head>
<body>
    <div class="container">
        <header>
            <h1>Instance Manager</h1>
            <p class="subtitle">Manage your instances effortlessly</p>
        </header>

        <nav class="tabs">
            <button class="tab-btn active" data-tab="instances">Instances</button>
            <button class="tab-btn" data-tab="create">Create Instance</button>
        </nav>

        <!-- Instances Tab -->
        <div id="instances-tab" class="tab-content active">
            <div class="card">
                <div class="card-header">
                    <h2>Running Instances</h2>
                    <button class="btn btn-primary" onclick="refreshInstances()">🔄 Refresh</button>
                </div>
                <div id="instances-list" class="instances-grid">
                    <p class="loading">Loading instances...</p>
                </div>
            </div>
        </div>

        <!-- Create Instance Tab -->
        <div id="create-tab" class="tab-content">
            <div class="card">
                <h2>Create New Instance</h2>
                <form id="create-form" class="form">
                    <div class="form-group">
                        <label for="provider">Provider</label>
                        <select id="provider" class="input">
                            <option value="aws">AWS</option>
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="instance-type">Instance Type</label>
                        <select id="instance-type" class="input">
                            <option value="t2.nano">t2.nano</option>
                            <option value="t2.micro">t2.micro</option>
                            <option value="t2.small">t2.small</option>
                            <option value="t2.medium">t2.medium</option>
                            <option value="t3.nano">t3.nano</option>
                            <option value="t3.micro">t3.micro</option>
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="duration">Duration</label>
                        <input type="text" id="duration" class="input" placeholder="e.g., 1h, 30m, 2h30m" value="1h" required>
                    </div>

                    <div class="form-group">
                        <label for="public-key">SSH Public Key Path</label>
                        <input type="text" id="public-key" class="input" placeholder="e.g., ~/.ssh/id_rsa.pub" required>
                    </div>

                    <div class="form-group">
                        <label for="availability-zone">Availability Zone</label>
                        <select id="availability-zone" class="input">
                            <option value="us-east-1a">us-east-1a</option>
                            <option value="us-east-1b">us-east-1b</option>
                            <option value="us-east-1c">us-east-1c</option>
                            <option value="us-east-1d">us-east-1d</option>
                            <option value="us-east-1e">us-east-1e</option>
                            <option value="us-east-1f">us-east-1f</option>
                            <option value="us-west-1a">us-west-1a</option>
                            <option value="us-west-1c">us-west-1c</option>
                        </select>
                    </div>

                    <button type="submit" class="btn btn-success">🚀 Create Instance</button>
                </form>
            </div>
        </div>

        <!-- Messages -->
        <div id="message" class="message hidden"></div>
    </div>

    <script src="/js/app.js"></script>
</body>
</html>`
}

func getStyleCSS() string {
	return `* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
    padding: 20px;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
}

header {
    text-align: center;
    color: white;
    margin-bottom: 40px;
    animation: slideDown 0.5s ease-out;
}

header h1 {
    font-size: 2.5em;
    margin-bottom: 10px;
    text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
}

.subtitle {
    font-size: 1.1em;
    opacity: 0.9;
}

.tabs {
    display: flex;
    gap: 10px;
    margin-bottom: 30px;
    background: white;
    padding: 10px;
    border-radius: 10px;
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
}

.tab-btn {
    flex: 1;
    padding: 12px 20px;
    border: none;
    background: #f0f0f0;
    cursor: pointer;
    border-radius: 6px;
    font-size: 1em;
    font-weight: 500;
    transition: all 0.3s ease;
}

.tab-btn.active {
    background: #667eea;
    color: white;
}

.tab-btn:hover {
    background: #667eea;
    color: white;
}

.tab-content {
    display: none;
    animation: fadeIn 0.3s ease-out;
}

.tab-content.active {
    display: block;
}

@keyframes fadeIn {
    from {
        opacity: 0;
        transform: translateY(10px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

@keyframes slideDown {
    from {
        opacity: 0;
        transform: translateY(-20px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.card {
    background: white;
    border-radius: 10px;
    padding: 30px;
    box-shadow: 0 8px 16px rgba(0,0,0,0.1);
    margin-bottom: 20px;
}

.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
}

.card h2 {
    color: #333;
    margin-bottom: 20px;
    font-size: 1.5em;
}

.form {
    display: grid;
    gap: 20px;
}

.form-group {
    display: flex;
    flex-direction: column;
}

.form-group label {
    margin-bottom: 8px;
    color: #333;
    font-weight: 600;
}

.input {
    padding: 12px;
    border: 2px solid #ddd;
    border-radius: 6px;
    font-size: 1em;
    transition: border-color 0.3s;
}

.input:focus {
    outline: none;
    border-color: #667eea;
}

.btn {
    padding: 12px 24px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    font-size: 1em;
    font-weight: 600;
    transition: all 0.3s ease;
    white-space: nowrap;
}

.btn-primary {
    background: #667eea;
    color: white;
}

.btn-primary:hover {
    background: #5568d3;
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
}

.btn-success {
    background: #48bb78;
    color: white;
    width: 100%;
}

.btn-success:hover {
    background: #38a169;
    transform: translateY(-2px);
}

.btn-danger {
    background: #f56565;
    color: white;
    padding: 8px 16px;
    font-size: 0.9em;
}

.btn-danger:hover {
    background: #e53e3e;
}

.btn-info {
    background: #4299e1;
    color: white;
    padding: 8px 16px;
    font-size: 0.9em;
}

.btn-info:hover {
    background: #3182ce;
}

.btn:disabled, .btn[disabled] {
    background: #e2e8f0 !important;
    color: #a0aec0 !important;
    cursor: not-allowed !important;
    border: 1px solid #cbd5e1 !important;
    opacity: 0.7;
    box-shadow: none;
}

.instances-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
    gap: 20px;
}

.instance-card {
    background: #f7fafc;
    border: 2px solid #e2e8f0;
    border-radius: 8px;
    padding: 20px;
    transition: all 0.3s ease;
}

.instance-card:hover {
    border-color: #667eea;
    box-shadow: 0 4px 12px rgba(102, 126, 234, 0.2);
    transform: translateY(-4px);
}

.instance-id {
    font-weight: 700;
    color: #667eea;
    font-family: monospace;
    font-size: 0.9em;
    word-break: break-all;
    margin-bottom: 12px;
}

.instance-detail {
    display: flex;
    justify-content: space-between;
    padding: 8px 0;
    border-bottom: 1px solid #e2e8f0;
}

.instance-detail:last-child {
    border-bottom: none;
}

.instance-detail-label {
    font-weight: 600;
    color: #718096;
    min-width: 100px;
}

.instance-detail-value {
    color: #2d3748;
    text-align: right;
    font-family: monospace;
    flex: 1;
    margin-left: 10px;
}

.status {
    display: inline-block;
    padding: 4px 12px;
    border-radius: 20px;
    font-size: 0.85em;
    font-weight: 600;
}

.status.running {
    background: #c6f6d5;
    color: #22543d;
}

.status.stopped {
    background: #fed7d7;
    color: #742a2a;
}

.status.expired {
    background: #feebc8;
    color: #7c2d12;
}

.instance-actions {
    display: flex;
    gap: 10px;
    margin-top: 15px;
}

.instance-actions button {
    flex: 1;
}

.message {
    position: fixed;
    top: 20px;
    right: 20px;
    padding: 16px 24px;
    border-radius: 8px;
    color: white;
    font-weight: 600;
    z-index: 1000;
    animation: slideIn 0.3s ease-out;
}

.message.hidden {
    display: none;
}

.message.success {
    background: #48bb78;
}

.message.error {
    background: #f56565;
}

.message.info {
    background: #4299e1;
}

@keyframes slideIn {
    from {
        transform: translateX(400px);
        opacity: 0;
    }
    to {
        transform: translateX(0);
        opacity: 1;
    }
}

.loading {
    text-align: center;
    color: #a0aec0;
    padding: 40px 20px;
    font-size: 1.1em;
}

.empty {
    text-align: center;
    padding: 40px 20px;
    color: #718096;
}

@media (max-width: 768px) {
    .instances-grid {
        grid-template-columns: 1fr;
    }

    header h1 {
        font-size: 2em;
    }

    .tabs {
        flex-direction: column;
    }

    .card {
        padding: 20px;
    }

    .instance-detail {
        flex-direction: column;
    }

    .instance-detail-value {
        text-align: left;
        margin-left: 0;
        margin-top: 4px;
    }
}`
}

func getAppJS() string {
	return `var API_BASE = '/api';

document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', function() {
        const tab = this.dataset.tab;
        switchTab(tab, this);
    });
});

function switchTab(tab, btn) {
    document.querySelectorAll('.tab-content').forEach(el => {
        el.classList.remove('active');
    });
    document.querySelectorAll('.tab-btn').forEach(b => {
        b.classList.remove('active');
    });
    document.getElementById(tab + '-tab').classList.add('active');
    btn.classList.add('active');
    if (tab === 'instances') {
        refreshInstances();
    }
}

async function refreshInstances() {
    try {
        const response = await fetch(API_BASE + '/instances');
        const data = await response.json();
        if (!data.success) {
            showMessage('Error loading instances', 'error');
            return;
        }
        const instances = data.data || [];
        const list = document.getElementById('instances-list');
        if (instances.length === 0) {
            list.innerHTML = '<p class="empty">No instances running. Create one to get started!</p>';
            return;
        }
        list.innerHTML = instances.map(instance => createInstanceCard(instance)).join('');
    } catch (error) {
        showMessage('Failed to load instances: ' + error.message, 'error');
    }
}

function createInstanceCard(instance) {
    const isExpired = new Date(instance.expires_at) < new Date();
    const statusClass = isExpired ? 'expired' : (instance.state === 'running' ? 'running' : 'stopped');
    const statusText = isExpired ? 'Expired' : instance.state;
    let sshSection = '';
    if (instance.public_ip) {
        sshSection = '<div class="instance-detail"><span class="instance-detail-label">SSH:</span><span class="instance-detail-value">' + instance.username + '@' + instance.public_ip + '</span></div>';
    }
    return '<div class="instance-card">' +
        '<div class="instance-id">' + instance.id + '</div>' +
        '<div class="instance-detail">' +
        '<span class="instance-detail-label">Type:</span>' +
        '<span class="instance-detail-value">' + instance.instance_type + '</span>' +
        '</div>' +
        '<div class="instance-detail">' +
        '<span class="instance-detail-label">Status:</span>' +
        '<span class="status ' + statusClass + '">' + statusText + '</span>' +
        '</div>' +
        '<div class="instance-detail">' +
        '<span class="instance-detail-label">IP:</span>' +
        '<span class="instance-detail-value">' + (instance.public_ip || 'N/A') + '</span>' +
        '</div>' +
        '<div class="instance-detail">' +
        '<span class="instance-detail-label">Zone:</span>' +
        '<span class="instance-detail-value">' + instance.availability_zone + '</span>' +
        '</div>' +
        '<div class="instance-detail">' +
        '<span class="instance-detail-label">Expires:</span>' +
        '<span class="instance-detail-value">' + new Date(instance.expires_at).toLocaleString() + '</span>' +
        '</div>' +
        sshSection +
        '<div class="instance-actions">' +
        '<button class="btn btn-info" onclick="showExtendDialog(\'' + instance.id + '\')">⏰ Extend</button>' +
        '<button class="btn btn-danger" onclick="stopInstance(\'' + instance.id + '\')"' + (isExpired ? ' disabled title="Cannot stop an expired instance"' : '') + '>⛔ Stop</button>' +
        '<button class="btn btn-danger" onclick="terminateInstance(\'' + instance.id + '\')">🗑️ Terminate</button>' +
        '</div>' +
        '</div>';
}

document.getElementById('create-form').addEventListener('submit', async function(e) {
    e.preventDefault();
    const instanceType = document.getElementById('instance-type').value;
    const duration = document.getElementById('duration').value;
    const publicKey = document.getElementById('public-key').value;
    const availabilityZone = document.getElementById('availability-zone').value;
    const provider = document.getElementById('provider').value;
    try {
        showMessage('Creating instance... Please wait', 'info');
        const response = await fetch(API_BASE + '/instances/create', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                instance_type: instanceType,
                duration: duration,
                public_key_path: publicKey,
                availability_zone: availabilityZone,
                provider: provider,
            }),
        });
        const data = await response.json();
        if (!data.success) {
            showMessage('Error: ' + data.error, 'error');
            return;
        }
        showMessage('Instance created! ID: ' + data.data.id, 'success');
        document.getElementById('create-form').reset();

        // Switch to instances tab immediately
        document.querySelector('[data-tab="instances"]').click();

        // Refresh instances immediately and then every 5 seconds for 2 minutes to catch IP assignment
        refreshInstances();
        var refreshCount = 0;
        var quickRefreshInterval = setInterval(() => {
            refreshInstances();
            refreshCount++;
            if (refreshCount >= 24) { // 24 * 5 seconds = 2 minutes
                clearInterval(quickRefreshInterval);
            }
        }, 5000);
    } catch (error) {
        showMessage('Failed to create instance: ' + error.message, 'error');
    }
});

async function showExtendDialog(instanceId) {
    const duration = prompt('Enter duration to extend (e.g., 1h, 30m):', '1h');
    if (!duration) return;
    try {
        const response = await fetch(API_BASE + '/instances/extend?instance_id=' + instanceId, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                duration: duration,
            }),
        });
        const data = await response.json();
        if (!data.success) {
            showMessage('Error: ' + data.error, 'error');
            return;
        }
        showMessage('Instance TTL extended successfully!', 'success');
        refreshInstances();
    } catch (error) {
        showMessage('Failed to extend instance: ' + error.message, 'error');
    }
}

async function stopInstance(instanceId) {
    if (!confirm('Are you sure you want to Stop this instance?')) return;
    try {
        const response = await fetch(API_BASE + '/instances/stop?instance_id=' + instanceId, {
            method: 'POST',
        });
        const data = await response.json();
        if (!data.success) {
            showMessage('Error: ' + data.error, 'error');
            return;
        }
        showMessage('Instance Stopd successfully!', 'success');
        refreshInstances();
    } catch (error) {
        showMessage('Failed to Stop instance: ' + error.message, 'error');
    }
}

async function terminateInstance(instanceId) {
    if (!confirm('Are you sure you want to TERMINATE this instance? This cannot be undone.')) return;
    try {
        const response = await fetch(API_BASE + '/instances/terminate?instance_id=' + instanceId, {
            method: 'POST',
        });
        const data = await response.json();
        if (!data.success) {
            showMessage('Error: ' + data.error, 'error');
            return;
        }
        showMessage('Instance terminated successfully!', 'success');
        refreshInstances();
    } catch (error) {
        showMessage('Failed to terminate instance: ' + error.message, 'error');
    }
}

function showMessage(message, type) {
    if (!type) type = 'info';
    const msgEl = document.getElementById('message');
    msgEl.textContent = message;
    msgEl.className = 'message ' + type;
    setTimeout(() => {
        msgEl.classList.add('hidden');
    }, 4000);
}

window.addEventListener('load', () => {
    refreshInstances();
});

setInterval(refreshInstances, 30000);`
}
