/**
 * globe.ts – Minimalist dark 3D globe with glowing proxy chain arcs.
 *
 * Renders via Three.js + three-globe.  The globe has no external texture;
 * land and ocean are differentiated only by material colour.
 *
 * Arc pipeline (innermost → outermost in the onion chain):
 *   Guard  ──►  Middle  ──►  Exit
 *
 * Glow is achieved through:
 *   - A high-contrast saturated colour on a near-black background.
 *   - A semi-transparent duplicate arc at 2× stroke width behind each arc.
 */

import * as THREE from 'three';
import ThreeGlobe from 'three-globe';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

// ─────────────────────────────────────────────────────────────────────────────
// Public types
// ─────────────────────────────────────────────────────────────────────────────

export interface ProxyNode {
  latitude:    number;
  longitude:   number;
  countryCode: string;
  role:        string;
  ipAddress:   string;
}

export interface ProxyChain {
  guard:  ProxyNode;
  middle: ProxyNode;
  exit:   ProxyNode;
}

// ─────────────────────────────────────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────────────────────────────────────

const COLOR = {
  sky:          '#000000',
  globe:        0x10243a,
  atmosphere:   '#0d2f7a',
  arcGuardMid:  '#00d4ff',   // cyan
  arcMidExit:   '#00ffa0',   // teal-green
  glowSuffix:   '66',        // 40 % opacity hex suffix for glow layer
  pointGuard:   '#ff3c3c',
  pointMiddle:  '#ffb700',
  pointExit:    '#3cff6e',
  labelFg:      '#e0e0ff',
} as const;

const ARC_ALTITUDE   = 0.38;
const ARC_STROKE     = 0.55;
const ARC_GLOW_MULT  = 2.8;  // glow layer stroke multiplier
const DASH_LENGTH    = 0.45;
const DASH_GAP       = 0.22;
const DASH_ANIM_MS   = 1700;
const POINT_RADIUS   = 0.55;
const POINT_ALTITUDE = 0.025;

// ─────────────────────────────────────────────────────────────────────────────
// GlobeRenderer
// ─────────────────────────────────────────────────────────────────────────────

export class GlobeRenderer {
  private readonly renderer: THREE.WebGLRenderer;
  private readonly scene:    THREE.Scene;
  private readonly camera:   THREE.PerspectiveCamera;
  private readonly controls: OrbitControls;
  private readonly globe:    ThreeGlobe;
  private rafId = 0;

  constructor(private readonly canvas: HTMLCanvasElement) {
    // ── Renderer ──────────────────────────────────────────────────────────
    this.renderer = new THREE.WebGLRenderer({ canvas, antialias: true, alpha: false });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    this.renderer.setSize(canvas.clientWidth, canvas.clientHeight, false);
    this.renderer.setClearColor(COLOR.sky, 1);
    this.renderer.outputColorSpace = THREE.SRGBColorSpace;

    // ── Scene ─────────────────────────────────────────────────────────────
    this.scene = new THREE.Scene();
    this.scene.background = new THREE.Color(COLOR.sky);

    // Minimal lighting: soft ambient + one directional (rim-light feel).
    this.scene.add(new THREE.AmbientLight(0xffffff, 0.25));
    const rim = new THREE.DirectionalLight(0x4466cc, 0.9);
    rim.position.set(-200, 100, 200);
    this.scene.add(rim);

    // ── Camera ────────────────────────────────────────────────────────────
    this.camera = new THREE.PerspectiveCamera(42, this.aspect(), 1, 10_000);
    this.camera.position.z = 295;

    // ── Controls ──────────────────────────────────────────────────────────
    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.controls.enableDamping    = true;
    this.controls.dampingFactor    = 0.06;
    this.controls.rotateSpeed      = 0.45;
    this.controls.autoRotate       = true;
    this.controls.autoRotateSpeed  = 0.35;
    this.controls.minDistance      = 150;
    this.controls.maxDistance      = 550;
    this.controls.enablePan        = false;

    // ── Globe ─────────────────────────────────────────────────────────────
    this.globe = new ThreeGlobe({ animateIn: false })
      .showAtmosphere(true)
      .atmosphereColor(COLOR.atmosphere)
      .atmosphereAltitude(0.14);

    // Custom material: no external texture, solid near-black sphere.
    (this.globe.globeMaterial() as THREE.MeshPhongMaterial).color.setHex(COLOR.globe);
    const globeMaterial = this.globe.globeMaterial() as THREE.MeshPhongMaterial;
    globeMaterial.emissive.setHex(0x07172a);
    globeMaterial.specular = new THREE.Color(0x315070);
    globeMaterial.shininess = 12;

    this.scene.add(this.globe as unknown as THREE.Object3D);

    // ── Start ─────────────────────────────────────────────────────────────
    window.addEventListener('resize', this.onResize);
    this.animate();
  }

  // ── Public API ───────────────────────────────────────────────────────────

  /** Render the Guard → Middle → Exit chain with glowing arcs and node dots. */
  updateChain(chain: ProxyChain): void {
    const { guard, middle, exit } = chain;

    // Two arcs + two glow-shadow copies (rendered behind, wider & semi-transparent).
    const arcs = [
      // Glow layers (drawn first so they sit behind the bright arc).
      {
        startLat: guard.latitude,  startLng: guard.longitude,
        endLat:   middle.latitude, endLng:   middle.longitude,
        color: COLOR.arcGuardMid + COLOR.glowSuffix,
        stroke: ARC_STROKE * ARC_GLOW_MULT,
        glow: true,
      },
      {
        startLat: middle.latitude, startLng: middle.longitude,
        endLat:   exit.latitude,   endLng:   exit.longitude,
        color: COLOR.arcMidExit + COLOR.glowSuffix,
        stroke: ARC_STROKE * ARC_GLOW_MULT,
        glow: true,
      },
      // Bright foreground arcs.
      {
        startLat: guard.latitude,  startLng: guard.longitude,
        endLat:   middle.latitude, endLng:   middle.longitude,
        color: COLOR.arcGuardMid,
        stroke: ARC_STROKE,
        glow: false,
      },
      {
        startLat: middle.latitude, startLng: middle.longitude,
        endLat:   exit.latitude,   endLng:   exit.longitude,
        color: COLOR.arcMidExit,
        stroke: ARC_STROKE,
        glow: false,
      },
    ];

    const points = [
      { lat: guard.latitude,  lng: guard.longitude,  color: COLOR.pointGuard,  label: `Guard · ${guard.countryCode}` },
      { lat: middle.latitude, lng: middle.longitude, color: COLOR.pointMiddle, label: `Middle · ${middle.countryCode}` },
      { lat: exit.latitude,   lng: exit.longitude,   color: COLOR.pointExit,   label: `Exit · ${exit.countryCode}` },
    ];

    this.globe
      .arcsData(arcs)
      .arcColor('color')
      .arcAltitude(ARC_ALTITUDE)
      .arcStroke((d: object) => (d as { stroke: number }).stroke)
      .arcDashLength(DASH_LENGTH)
      .arcDashGap(DASH_GAP)
      .arcDashAnimateTime(DASH_ANIM_MS)
      .arcsTransitionDuration(600)

      .pointsData(points)
      .pointColor('color')
      .pointAltitude(POINT_ALTITUDE)
      .pointRadius(POINT_RADIUS)
      .pointsMerge(false);
  }

  /** Remove all arcs and points (e.g. while disconnected). */
  clearChain(): void {
    this.globe.arcsData([]).pointsData([]);
  }

  dispose(): void {
    cancelAnimationFrame(this.rafId);
    this.controls.dispose();
    this.renderer.dispose();
    window.removeEventListener('resize', this.onResize);
  }

  // ── Private ──────────────────────────────────────────────────────────────

  private animate = (): void => {
    this.rafId = requestAnimationFrame(this.animate);
    this.controls.update();
    this.renderer.render(this.scene, this.camera);
  };

  private onResize = (): void => {
    this.renderer.setSize(this.canvas.clientWidth, this.canvas.clientHeight, false);
    this.camera.aspect = this.aspect();
    this.camera.updateProjectionMatrix();
  };

  private aspect(): number {
    return this.canvas.clientWidth / (this.canvas.clientHeight || 1);
  }
}
