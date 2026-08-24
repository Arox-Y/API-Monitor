/**
 * CelestialGlow renders the decorative lighting behind the hero: a domed arc,
 * vertical light beams, and drifting stardust.
 *
 * Server Component — purely presentational, no interactivity, so it ships zero
 * JavaScript to the browser.
 *
 * The particle positions are a fixed table rather than Math.random(). Random
 * values would differ between the server render and the client hydration,
 * producing a React hydration mismatch.
 */

/** Fixed stardust positions. Deterministic on purpose — see file header. */
const PARTICLES = [
  { left: "12%", top: "26%", size: 2, delay: "0s", opacity: 0.5 },
  { left: "22%", top: "48%", size: 1, delay: "1.4s", opacity: 0.35 },
  { left: "31%", top: "16%", size: 2, delay: "2.7s", opacity: 0.45 },
  { left: "44%", top: "58%", size: 1, delay: "0.8s", opacity: 0.3 },
  { left: "57%", top: "22%", size: 2, delay: "3.4s", opacity: 0.5 },
  { left: "68%", top: "44%", size: 1, delay: "2.1s", opacity: 0.4 },
  { left: "76%", top: "18%", size: 2, delay: "4.2s", opacity: 0.45 },
  { left: "86%", top: "52%", size: 1, delay: "1.1s", opacity: 0.3 },
  { left: "92%", top: "30%", size: 2, delay: "3.0s", opacity: 0.4 },
] as const;

/** Vertical light pillars falling from the arc. */
const BEAMS = [
  { left: "18%", width: "1px", height: "62%", delay: "0s" },
  { left: "34%", width: "2px", height: "78%", delay: "1.5s" },
  { left: "50%", width: "1px", height: "88%", delay: "3s" },
  { left: "66%", width: "2px", height: "74%", delay: "2.2s" },
  { left: "82%", width: "1px", height: "58%", delay: "0.7s" },
] as const;

export function CelestialGlow() {
  return (
    <div
      aria-hidden="true"
      className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[860px] overflow-hidden"
    >
      {/* Broad radial bloom. The outer element owns the centring transform so the
          breathing animation (which also uses transform) cannot fight it. */}
      <div className="absolute left-1/2 top-[-420px] h-[820px] w-[1240px] -translate-x-1/2">
        <div
          className="animate-glow-breathe h-full w-full rounded-[50%] blur-3xl"
          style={{
            background:
              "radial-gradient(ellipse at 50% 60%, rgba(56,189,248,0.42) 0%, rgba(96,165,250,0.28) 35%, rgba(129,140,248,0.12) 58%, transparent 74%)",
          }}
        />
      </div>

      {/* The dome rim — a hairline arc catching the light along its top edge. */}
      <div className="absolute left-1/2 top-[-340px] h-[680px] w-[1120px] -translate-x-1/2">
        <div className="h-full w-full rounded-[50%] border-t border-sky-300/35 shadow-[0_0_140px_50px_rgba(56,189,248,0.14)]" />
      </div>

      {/* A tighter, brighter core arc sitting just inside the rim. */}
      <div className="absolute left-1/2 top-[-300px] h-[560px] w-[820px] -translate-x-1/2">
        <div className="h-full w-full rounded-[50%] border-t border-cyan-200/45 shadow-[0_0_90px_20px_rgba(103,232,249,0.12)]" />
      </div>

      {/* Vertical beams fading downward into the hero. */}
      {BEAMS.map((beam) => (
        <div
          key={beam.left}
          className="animate-beam-fade absolute top-0"
          style={{
            left: beam.left,
            width: beam.width,
            height: beam.height,
            animationDelay: beam.delay,
            background:
              "linear-gradient(to bottom, rgba(125,211,252,0.30) 0%, rgba(96,165,250,0.10) 45%, transparent 100%)",
          }}
        />
      ))}

      {/* Stardust. */}
      {PARTICLES.map((p) => (
        <div
          key={`${p.left}-${p.top}`}
          className="animate-particle-drift absolute rounded-full bg-white"
          style={{
            left: p.left,
            top: p.top,
            width: p.size,
            height: p.size,
            opacity: p.opacity,
            animationDelay: p.delay,
            boxShadow: "0 0 6px 1px rgba(186,230,253,0.65)",
          }}
        />
      ))}

      {/* Fade the glow into the page background so the arc has no hard edge. */}
      <div className="absolute inset-x-0 bottom-0 h-56 bg-gradient-to-b from-transparent to-space-900" />
    </div>
  );
}
