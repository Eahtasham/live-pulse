/**
 * Generates a stable browser fingerprint that survives incognito mode.
 *
 * Combines multiple signals that remain constant across normal and
 * incognito windows on the same device + browser:
 *   - Canvas rendering (GPU/font fingerprint)
 *   - Screen geometry
 *   - Timezone
 *   - Platform / language / hardware concurrency
 *   - WebGL renderer
 *
 * The result is a hex-encoded SHA-256 hash.
 */
export async function generateFingerprint(): Promise<string> {
  const signals: string[] = [];

  // 1. Canvas fingerprint
  try {
    const canvas = document.createElement("canvas");
    canvas.width = 200;
    canvas.height = 50;
    const ctx = canvas.getContext("2d");
    if (ctx) {
      ctx.textBaseline = "top";
      ctx.font = "14px Arial";
      ctx.fillStyle = "#f60";
      ctx.fillRect(0, 0, 200, 50);
      ctx.fillStyle = "#069";
      ctx.fillText("LivePulse fingerprint", 2, 15);
      ctx.fillStyle = "rgba(102,204,0,0.7)";
      ctx.fillText("LivePulse fingerprint", 4, 17);
      signals.push(canvas.toDataURL());
    }
  } catch {
    // canvas blocked
  }

  // 2. Screen geometry
  signals.push(`${screen.width}x${screen.height}x${screen.colorDepth}`);
  signals.push(`${screen.availWidth}x${screen.availHeight}`);

  // 3. Timezone
  signals.push(Intl.DateTimeFormat().resolvedOptions().timeZone);
  signals.push(String(new Date().getTimezoneOffset()));

  // 4. Platform, language, hardware
  signals.push(navigator.platform);
  signals.push(navigator.language);
  signals.push(String(navigator.hardwareConcurrency || 0));

  // 5. WebGL renderer
  try {
    const gl = document.createElement("canvas").getContext("webgl");
    if (gl) {
      const dbg = gl.getExtension("WEBGL_debug_renderer_info");
      if (dbg) {
        signals.push(gl.getParameter(dbg.UNMASKED_VENDOR_WEBGL) || "");
        signals.push(gl.getParameter(dbg.UNMASKED_RENDERER_WEBGL) || "");
      }
    }
  } catch {
    // webgl blocked
  }

  // 6. Installed media types (codec support varies per browser/device)
  try {
    const types = [
      "video/webm; codecs=vp8",
      "video/mp4; codecs=avc1",
      "audio/ogg; codecs=vorbis",
    ];
    for (const t of types) {
      signals.push(`${t}:${MediaSource.isTypeSupported(t)}`);
    }
  } catch {
    // MediaSource unavailable
  }

  // Hash
  const raw = signals.join("|||");
  const encoded = new TextEncoder().encode(raw);
  const hashBuffer = await crypto.subtle.digest("SHA-256", encoded);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  return hashArray.map((b) => b.toString(16).padStart(2, "0")).join("");
}

/**
 * Returns a stable client ID for the current browser+device.
 *
 * Uses the fingerprint as the primary identifier. Falls back to a
 * localStorage UUID if crypto.subtle is unavailable (e.g. non-HTTPS).
 */
export async function getStableClientId(): Promise<string> {
  try {
    return await generateFingerprint();
  } catch {
    // Fallback for environments without crypto.subtle (HTTP, older browsers)
    const key = "livepulse_client_id";
    let id = localStorage.getItem(key);
    if (!id) {
      id = crypto.randomUUID();
      localStorage.setItem(key, id);
    }
    return id;
  }
}
