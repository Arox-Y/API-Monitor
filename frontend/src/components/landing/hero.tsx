import Link from "next/link";
import { Play, Sparkle } from "lucide-react";

/**
 * Hero: badge, headline, subheadline, CTA pair.
 *
 * Server Component — no state, no effects, no event handlers.
 */
export function Hero() {
  return (
    <section className="relative mx-auto max-w-4xl px-4 pt-20 pb-12 text-center sm:px-6 md:pt-28">
      {/* Frosted pill badge */}
      <div className="mb-8 flex justify-center">
        <span className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-1.5 text-xs text-slate-300 backdrop-blur-sm">
          <Sparkle className="size-3 text-sky-300" aria-hidden="true" />
          Early Access Beta
        </span>
      </div>

      {/* Headline. The gradient span carries the accent; the first line stays
          plain white so the eye lands on the promise, not the effect. */}
      <h1 className="text-balance text-5xl font-bold leading-[1.05] tracking-tight text-white md:text-6xl lg:text-7xl">
        <span
          className="block"
          style={{ textShadow: "0 0 40px rgba(148,197,255,0.25)" }}
        >
          Ship Faster With
        </span>
        <span className="mt-2 block bg-gradient-to-b from-white via-sky-100 to-sky-300 bg-clip-text text-transparent">
          Uptime You Trust
        </span>
      </h1>

      {/* Subheadline */}
      <p className="mx-auto mt-6 max-w-[600px] text-pretty text-base leading-relaxed text-slate-400 md:text-lg">
        A minimal monitoring system that turns noisy endpoint checks into clear,
        glowing, effortless signals — so you find out your API is down before
        your users tell you.
      </p>

      {/* CTA group */}
      <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
        <Link
          href="/register"
          className="inline-flex w-full items-center justify-center rounded-full bg-white px-6 py-3 text-sm font-medium text-slate-950 transition-all duration-200 hover:scale-[1.03] hover:bg-slate-100 hover:shadow-[0_0_28px_rgba(96,165,250,0.45)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-400 sm:w-auto"
        >
          Get Started
        </Link>

        <Link
          href="#preview"
          className="inline-flex w-full items-center justify-center gap-2 rounded-full border border-white/15 bg-white/5 px-6 py-3 text-sm font-medium text-white backdrop-blur-sm transition-all duration-200 hover:bg-white/10 hover:shadow-[0_0_24px_rgba(96,165,250,0.25)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-400 sm:w-auto"
        >
          <Play className="size-3.5" aria-hidden="true" />
          Watch Demo
        </Link>
      </div>
    </section>
  );
}
