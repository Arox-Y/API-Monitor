import { CelestialGlow } from "@/components/landing/celestial-glow";
import { Hero } from "@/components/landing/hero";
import { ProductPreview } from "@/components/landing/product-preview";
import { SiteHeader } from "@/components/landing/site-header";

/**
 * Marketing landing page.
 *
 * Replaces the previous `redirect("/login")`. Signed-out visitors land here;
 * the header routes them to /login or /register.
 *
 * This entire page is a Server Component tree — the only JavaScript shipped to
 * the browser is the mobile navigation drawer.
 */
export default function LandingPage() {
  return (
    <div className="relative min-h-screen overflow-x-hidden bg-space-900 text-white antialiased">
      <CelestialGlow />
      <SiteHeader />
      <main>
        <Hero />
        <ProductPreview />
      </main>
    </div>
  );
}
