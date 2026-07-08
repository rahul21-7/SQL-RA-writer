let scene, camera, renderer, particles, dataCore;
const crystalGroup = new THREE.Group();

const SQL_KEYWORDS = ["SELECT", "FROM", "WHERE", "JOIN", "GROUP BY", "ORDER BY", "COUNT", "SUM", "RA", "π", "σ", "γ", "⨝"];

// Initialization of the 3D Experience
function initThree() {
    const container = document.getElementById('canvas-container');
    scene = new THREE.Scene();
    camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 1000);
    renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setSize(window.innerWidth, window.innerHeight);
    renderer.setPixelRatio(window.devicePixelRatio);
    container.appendChild(renderer.domElement);

    const gridGeometry = new THREE.BufferGeometry();
    const vertices = [];
    for (let x = -80; x < 80; x += 4) {
        for (let z = -80; z < 80; z += 4) {
            vertices.push(x, -8, z);
        }
    }
    gridGeometry.setAttribute('position', new THREE.Float32BufferAttribute(vertices, 3));
    const gridMaterial = new THREE.PointsMaterial({
        color: 0x10b981,
        size: 0.08,
        transparent: true,
        opacity: 0.2
    });
    particles = new THREE.Points(gridGeometry, gridMaterial);
    scene.add(particles);

    dataCore = new THREE.Group();
    const cylinderGeom = new THREE.CylinderGeometry(3.5, 3.5, 1.2, 32);
    const cylinderMat = new THREE.MeshPhongMaterial({
        color: 0x10b981,
        transparent: true,
        opacity: 0.4,
        emissive: 0x10b981,
        emissiveIntensity: 0.5,
        shininess: 100
    });

    for(let i=0; i<3; i++) {
        const c = new THREE.Mesh(cylinderGeom, cylinderMat);
        c.position.y = (i - 1) * 1.8;
        dataCore.add(c);
    }

    SQL_KEYWORDS.forEach((word, index) => {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        canvas.width = 256;
        canvas.height = 64;
        ctx.fillStyle = 'rgba(16, 185, 129, 0.8)';
        ctx.font = 'Bold 40px Outfit';
        ctx.fillText(word, 10, 50);

        const texture = new THREE.CanvasTexture(canvas);
        const spriteMat = new THREE.SpriteMaterial({ map: texture, transparent: true });
        const sprite = new THREE.Sprite(spriteMat);
        
        const angle = (index / SQL_KEYWORDS.length) * Math.PI * 2;
        const radius = 8 + Math.random() * 2;
        sprite.position.set(Math.cos(angle) * radius, (Math.random() - 0.5) * 5, Math.sin(angle) * radius);
        sprite.scale.set(4, 1, 1);
        
        sprite.userData = { angle: angle, radius: radius, speed: 0.01 + Math.random() * 0.01 };
        crystalGroup.add(sprite);
    });
    
    crystalGroup.add(dataCore);
    scene.add(crystalGroup);

    const pointLight = new THREE.PointLight(0x10b981, 2, 50);
    pointLight.position.set(0, 5, 5);
    scene.add(pointLight);
    scene.add(new THREE.AmbientLight(0x404040));

    camera.position.z = 25;
    camera.position.y = 5;

    animate();
}

function animate() {
    requestAnimationFrame(animate);
    
    const time = Date.now() * 0.001;
    const positions = particles.geometry.attributes.position.array;
    for (let i = 0; i < positions.length; i += 3) {
        const x = positions[i];
        const z = positions[i + 2];
        positions[i + 1] = -8 + Math.sin(x * 0.1 + time) * Math.cos(z * 0.1 + time) * 1.5;
    }
    particles.geometry.attributes.position.needsUpdate = true;

    crystalGroup.rotation.y += 0.002;
    
    crystalGroup.children.forEach(child => {
        if (child.isSprite) {
            child.userData.angle += child.userData.speed;
            child.position.x = Math.cos(child.userData.angle) * child.userData.radius;
            child.position.z = Math.sin(child.userData.angle) * child.userData.radius;
            child.position.y += Math.sin(time + child.userData.angle) * 0.01;
        }
    });

    dataCore.children.forEach((c, idx) => {
        c.scale.setScalar(1 + Math.sin(time * 2 + idx) * 0.03);
    });

    renderer.render(scene, camera);
}

const enterForgeBtn = document.getElementById('enter-forge');
const forgeBtn = document.getElementById('forge-btn');
const executeBtn = document.getElementById('execute-btn');
const questionInput = document.getElementById('question-input');
const feedbackWidget = document.getElementById('feedback-controls');
const accuracyVal = document.getElementById('accuracy-val');
const raOutput = document.getElementById('ra-output');
const sqlOutput = document.getElementById('sql-output');
const seedBtn = document.getElementById('seed-btn');
const schemaList = document.getElementById('schema-list');
const schemaStatus = document.getElementById('schema-status');
const benchmarkTable = document.getElementById('benchmark-table');

let currentModel = "standard";
let lastGeneration = null;

enterForgeBtn.onclick = () => {
    document.body.classList.add('forge-active');
    gsap.to(camera.position, { z: 18, y: 12, duration: 1.5, ease: "power2.inOut" });
    gsap.to(crystalGroup.position, { y: 10, duration: 1.5, ease: "power2.inOut" });
    fetchSchema();
    fetchBenchmarks();
};

async function fetchBenchmarks() {
    try {
        const resp = await fetch('/api/benchmarks');
        const data = await resp.json();
        renderBenchmarks(data);
    } catch (e) {
        benchmarkTable.innerText = "Results not yet available.";
    }
}

function renderBenchmarks(data) {
    if (!data || !data.qwen || !data.llama) {
        benchmarkTable.innerHTML = `<div class="no-benchmarks" style="padding: 15px; color: #888; text-align: center; font-size: 0.9em; line-height: 1.4;">
            No local benchmark data found.<br>Run <code style="color: #10b981; font-family: monospace;">python python/eval.py</code> to generate results.
        </div>`;
        return;
    }
    const q = data.qwen.your_model;
    const l = data.llama.your_model;

    let html = `<table>
        <thead>
            <tr>
                <th>Metric</th>
                <th>Expert Forge</th>
                <th>Base Logic</th>
                <th>Delta</th>
            </tr>
        </thead>
        <tbody>
            ${renderRow("Exact Match %", q.exact_match_pct, l.exact_match_pct)}
            ${renderRow("Execution %", q.exec_accuracy, l.exec_accuracy)}
            ${renderRow("RA Valid %", q.ra_valid_rate, l.ra_valid_rate)}
        </tbody>
    </table>`;
    benchmarkTable.innerHTML = html;
}

function renderRow(label, qVal, lVal) {
    const diff = (qVal - lVal).toFixed(1);
    const arrow = diff > 0 ? "▲" : (diff < 0 ? "▼" : "=");
    const cls = diff > 0 ? "delta-up" : (diff < 0 ? "delta-down" : "");
    return `<tr>
        <td>${label}</td>
        <td>${qVal}%</td>
        <td>${lVal}%</td>
        <td class="${cls}">${arrow} ${Math.abs(diff)}%</td>
    </tr>`;
}

async function fetchSchema() {
    try {
        const resp = await fetch('/api/schema');
        const data = await resp.json();
        schemaList.innerHTML = data.schema.replace(/\n/g, '<br>');
        schemaStatus.innerText = "ONLINE";
    } catch (e) {
        schemaList.innerText = "Failed to scan DB.";
        schemaStatus.innerText = "ERROR";
    }
}

seedBtn.onclick = async () => {
    seedBtn.disabled = true;
    seedBtn.innerText = "SEEDING...";
    try {
        const resp = await fetch('/api/seed', { method: 'POST' });
        if (resp.ok) {
            fetchSchema();
        }
    } finally {
        seedBtn.disabled = false;
        seedBtn.innerText = "SEED SAMPLE DB";
    }
};

questionInput.oninput = () => {
    const intensity = Math.min(questionInput.value.length / 100, 1) + 0.5;
    dataCore.children.forEach(c => {
        c.material.emissiveIntensity = intensity;
    });
    gsap.to(crystalGroup.scale, {
        x: 1 + intensity * 0.1, y: 1 + intensity * 0.1, z: 1 + intensity * 0.1,
        duration: 0.3
    });
};

forgeBtn.onclick = async () => {
    const q = questionInput.value.trim();
    if (!q) return;

    document.getElementById('loader').classList.remove('hidden');
    executeBtn.classList.remove('ready');
    executeBtn.disabled = true;

    try {
        const resp = await fetch('/api/query', {
            method: 'POST',
            body: JSON.stringify({ question: q, model: currentModel, db_id: "postgres" })
        });
        const data = await resp.json();
        raOutput.innerText = data.ra;
        sqlOutput.innerText = data.sql;
        lastGeneration = { question: q, ra: data.ra, sql: data.sql };
        executeBtn.disabled = false;
        executeBtn.classList.add('ready');
        gsap.to(particles.material, { opacity: 0.6, duration: 0.2, yoyo: true, repeat: 1 });
    } finally {
        document.getElementById('loader').classList.add('hidden');
    }
};

executeBtn.onclick = async () => {
    if (!executeBtn.classList.contains('ready')) return;
    const sql = sqlOutput.innerText;
    document.getElementById('loader').classList.remove('hidden');
    try {
        const resp = await fetch('/api/execute', { method: 'POST', body: JSON.stringify({ sql }) });
        const data = await resp.json();
        renderResults(data);
    } finally {
        document.getElementById('loader').classList.add('hidden');
    }
};

document.querySelectorAll('.fb-btn').forEach(btn => {
    btn.onclick = async () => {
        if (!lastGeneration) return;
        const rating = btn.dataset.rating;
        await fetch('/api/feedback', {
            method: 'POST',
            body: JSON.stringify({
                question: lastGeneration.question, db_id: "postgres",
                predicted_ra: lastGeneration.ra, generated_sql: lastGeneration.sql, rating: rating
            })
        });
        
        const color = rating === "2" ? 0x00ff00 : (rating === "1" ? 0xffff00 : 0xff0000);
        dataCore.children.forEach(c => {
             gsap.to(c.material.color, { r: (color >> 16 & 255) / 255, g: (color >> 8 & 255) / 255, b: (color & 255) / 255, duration: 0.5, yoyo: true, repeat: 1 });
        });
    };
});

function renderResults(data) {
    const head = document.getElementById('table-head');
    const body = document.getElementById('table-body');
    head.innerHTML = ""; body.innerHTML = "";
    
    if (data.error) {
        body.innerHTML = `<tr><td style="color:red">Execution Error: ${data.error}</td></tr>`;
        return;
    }

    const hRow = document.createElement('tr');
    data.columns.forEach(c => {
        const th = document.createElement('th'); th.innerText = c;
        hRow.appendChild(th);
    });
    head.appendChild(hRow);

    data.results.forEach(row => {
        const rRow = document.createElement('tr');
        data.columns.forEach(c => {
            const td = document.createElement('td'); 
            td.innerText = row[c] === null ? 'NULL' : row[c];
            rRow.appendChild(td);
        });
        body.appendChild(rRow);
    });
    document.getElementById('row-count').innerText = `${data.results.length} rows`;
}

document.querySelectorAll('.model-option').forEach(opt => {
    opt.onclick = () => {
        document.querySelectorAll('.model-option').forEach(o => o.classList.remove('active'));
        opt.classList.add('active');
        currentModel = opt.dataset.mode;
    };
});

window.onresize = () => {
    camera.aspect = window.innerWidth / window.innerHeight;
    camera.updateProjectionMatrix();
    renderer.setSize(window.innerWidth, window.innerHeight);
};

initThree();

document.getElementById('copy-sql').onclick = () => {
    navigator.clipboard.writeText(sqlOutput.innerText);
    const btn = document.getElementById('copy-sql');
    const old = btn.innerText;
    btn.innerText = "✔️";
    setTimeout(() => btn.innerText = old, 1000);
};
