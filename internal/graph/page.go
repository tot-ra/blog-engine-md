package graph

import (
	"fmt"
	"html"
)

func generateGraphHTML(siteTitle string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Graph View | %s</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; overflow: hidden; background: #ffffff; color: #111; }
        #graph-container { width: 100vw; height: 100vh; display: block; }
        .legend {
            position: fixed;
            left: 50%%;
            bottom: 16px;
            transform: translateX(-50%%);
            z-index: 10;
            display: flex;
            align-items: center;
            gap: 14px;
            background: rgba(255,255,255,0.95);
            padding: 8px 12px;
            border-radius: 999px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.15);
        }
        .legend-item { display: flex; align-items: center; gap: 6px; }
        .legend-dot { width: 10px; height: 10px; border-radius: 50%%; }
        .hint {
            position: fixed;
            top: 16px;
            left: 16px;
            z-index: 10;
            background: rgba(255,255,255,0.95);
            padding: 8px 12px;
            border-radius: 6px;
            box-shadow: 0 2px 8px rgba(0,0,0,0.15);
            font-size: 12px;
            color: #555;
            max-width: 280px;
        }
        .tooltip {
            position: fixed; padding: 6px 10px; background: rgba(0,0,0,0.85);
            color: white; border-radius: 4px; font-size: 13px;
            pointer-events: none; display: none; z-index: 20;
        }
        .back-link {
            position: fixed; top: 16px; right: 16px; z-index: 10;
            background: rgba(255,255,255,0.95); padding: 8px 16px;
            border-radius: 6px; box-shadow: 0 2px 8px rgba(0,0,0,0.15);
            text-decoration: none; color: #333; font-size: 14px;
        }
        .back-link:hover { background: #f0f0f0; }
        body.embed .back-link,
        body.embed .hint { display: none; }
        body.embed .legend {
            bottom: 8px;
            gap: 10px;
            padding: 6px 10px;
            font-size: 12px;
        }
        body.dark {
            background: #111315;
            color: #e8edf6;
        }
        body.dark .legend,
        body.dark .back-link,
        body.dark .hint {
            background: rgba(33, 37, 45, 0.94);
            color: #e8edf6;
            border: 1px solid rgba(255,255,255,0.08);
        }
        body.dark .hint { color: #c5ccd8; }
        body.dark .back-link:hover {
            background: rgba(45, 50, 60, 0.98);
        }
    </style>
    <script type="importmap">
    {
      "imports": {
        "three": "https://cdn.jsdelivr.net/npm/three@0.170.0/build/three.module.js",
        "three/addons/": "https://cdn.jsdelivr.net/npm/three@0.170.0/examples/jsm/"
      }
    }
    </script>
</head>
<body>
    <div class="hint">Drag to orbit · scroll to zoom · click a node to open</div>
    <div class="legend">
        <div class="legend-item"><div class="legend-dot" style="background:#4CAF50"></div> Blog</div>
        <div class="legend-item"><div class="legend-dot" style="background:#2196F3"></div> Docs</div>
        <div class="legend-item"><div class="legend-dot" style="background:#FF9800"></div> Tags</div>
    </div>
    <a href="/" class="back-link">← Back to site</a>
    <div id="tooltip" class="tooltip"></div>
    <canvas id="graph-container"></canvas>

    <script type="module">
    import * as THREE from 'three';
    import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

    (function() {
        const savedTheme = localStorage.getItem('theme');
        const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
        const isDarkTheme = savedTheme === 'dark' || (savedTheme !== 'light' && prefersDark);
        if (isDarkTheme) {
            document.body.classList.add('dark');
        }

        const embed = new URLSearchParams(window.location.search).get('embed') === '1';
        if (embed) {
            document.body.classList.add('embed');
        }

        const pathParts = window.location.pathname.split('/').filter(Boolean);
        const langPrefix = (pathParts.length > 0 && (pathParts[0] === 'ru' || pathParts[0] === 'en')) ? '/' + pathParts[0] : '';
        const backLink = document.querySelector('.back-link');
        if (backLink) {
            backLink.setAttribute('href', (langPrefix || '') + '/');
        }

        fetch((langPrefix || '') + '/graph.json')
            .then(r => r.json())
            .then(data => initGraph(data))
            .catch(err => console.error('Failed to load graph.json', err));

        function initGraph(data) {
            const canvas = document.getElementById('graph-container');
            const tooltip = document.getElementById('tooltip');
            const dark = document.body.classList.contains('dark');

            const scene = new THREE.Scene();
            scene.background = new THREE.Color(dark ? 0x111315 : 0xffffff);

            const camera = new THREE.PerspectiveCamera(55, window.innerWidth / window.innerHeight, 0.1, 2000);
            camera.position.set(0, 40, 140);

            const renderer = new THREE.WebGLRenderer({ canvas, antialias: true });
            renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
            renderer.setSize(window.innerWidth, window.innerHeight);

            const controls = new OrbitControls(camera, canvas);
            controls.enableDamping = true;
            controls.dampingFactor = 0.08;
            controls.minDistance = 20;
            controls.maxDistance = 500;
            controls.target.set(0, 0, 0);

            scene.add(new THREE.AmbientLight(0xffffff, dark ? 0.7 : 0.85));
            const key = new THREE.DirectionalLight(0xffffff, dark ? 0.55 : 0.45);
            key.position.set(40, 80, 60);
            scene.add(key);

            const nodeById = new Map();
            data.nodes.forEach(n => nodeById.set(n.id, n));

            const edgeColor = dark ? 0x6b7585 : 0xb0b7c3;
            const edgePositions = [];
            data.edges.forEach(e => {
                const s = typeof e.source === 'object' ? e.source : nodeById.get(e.source);
                const t = typeof e.target === 'object' ? e.target : nodeById.get(e.target);
                if (!s || !t) return;
                edgePositions.push(s.x || 0, s.y || 0, s.z || 0, t.x || 0, t.y || 0, t.z || 0);
            });
            if (edgePositions.length) {
                const edgeGeo = new THREE.BufferGeometry();
                edgeGeo.setAttribute('position', new THREE.Float32BufferAttribute(edgePositions, 3));
                scene.add(new THREE.LineSegments(
                    edgeGeo,
                    new THREE.LineBasicMaterial({ color: edgeColor, transparent: true, opacity: dark ? 0.28 : 0.22 })
                ));
            }

            const meshes = [];
            const sphereGeoCache = new Map();
            function sphereGeo(radius) {
                const key = radius.toFixed(2);
                if (!sphereGeoCache.has(key)) {
                    sphereGeoCache.set(key, new THREE.SphereGeometry(radius, 16, 12));
                }
                return sphereGeoCache.get(key);
            }

            data.nodes.forEach(n => {
                const radius = Math.max(0.6, (n.size || 3) * 0.28);
                const mat = new THREE.MeshStandardMaterial({
                    color: new THREE.Color(n.color || '#888888'),
                    roughness: 0.45,
                    metalness: 0.05
                });
                const mesh = new THREE.Mesh(sphereGeo(radius), mat);
                mesh.position.set(n.x || 0, n.y || 0, n.z || 0);
                mesh.userData = n;
                scene.add(mesh);
                meshes.push(mesh);
            });

            const raycaster = new THREE.Raycaster();
            const pointer = new THREE.Vector2();
            let hovered = null;
            let pointerDown = null;
            let moved = false;

            function setPointer(event) {
                const rect = canvas.getBoundingClientRect();
                pointer.x = ((event.clientX - rect.left) / rect.width) * 2 - 1;
                pointer.y = -((event.clientY - rect.top) / rect.height) * 2 + 1;
            }

            function pick() {
                raycaster.setFromCamera(pointer, camera);
                const hits = raycaster.intersectObjects(meshes, false);
                return hits.length ? hits[0].object : null;
            }

            function showTooltip(mesh, event) {
                if (!mesh) {
                    tooltip.style.display = 'none';
                    canvas.style.cursor = 'default';
                    return;
                }
                const n = mesh.userData;
                tooltip.style.display = 'block';
                tooltip.style.left = (event.clientX + 12) + 'px';
                tooltip.style.top = (event.clientY + 12) + 'px';
                tooltip.textContent = n.label + ' (' + n.type + ')';
                canvas.style.cursor = 'pointer';
            }

            canvas.addEventListener('pointerdown', e => {
                setPointer(e);
                pointerDown = { x: e.clientX, y: e.clientY };
                moved = false;
            });
            canvas.addEventListener('pointermove', e => {
                if (pointerDown) {
                    const dx = e.clientX - pointerDown.x;
                    const dy = e.clientY - pointerDown.y;
                    if (Math.abs(dx) > 3 || Math.abs(dy) > 3) moved = true;
                }
                setPointer(e);
                const hit = pick();
                if (hit !== hovered) {
                    if (hovered) hovered.scale.set(1, 1, 1);
                    hovered = hit;
                    if (hovered) hovered.scale.set(1.35, 1.35, 1.35);
                }
                showTooltip(hovered, e);
            });
            canvas.addEventListener('pointerup', e => {
                const wasMoved = moved;
                pointerDown = null;
                moved = false;
                if (wasMoved) return;
                setPointer(e);
                const hit = pick();
                if (!hit || !hit.userData || !hit.userData.url) return;
                const url = hit.userData.url;
                if (embed && window.parent && window.parent !== window) {
                    window.parent.postMessage({ type: 'blog-graph-navigate', url }, window.location.origin);
                    return;
                }
                window.location.href = url;
            });
            canvas.addEventListener('pointerleave', () => {
                if (hovered) hovered.scale.set(1, 1, 1);
                hovered = null;
                tooltip.style.display = 'none';
                canvas.style.cursor = 'default';
            });

            window.addEventListener('resize', () => {
                camera.aspect = window.innerWidth / window.innerHeight;
                camera.updateProjectionMatrix();
                renderer.setSize(window.innerWidth, window.innerHeight);
            });

            function animate() {
                requestAnimationFrame(animate);
                controls.update();
                renderer.render(scene, camera);
            }
            animate();
        }
    })();
    </script>
</body>
</html>`, html.EscapeString(siteTitle))
}
