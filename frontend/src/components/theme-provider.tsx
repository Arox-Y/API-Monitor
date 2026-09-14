"use client";

import { createContext, useCallback, useContext, useEffect, useState } from "react";

/**
 * Light/dark theme for the PRODUCT surface (everything behind /login).
 *
 * The marketing landing page is dark-only by design (ADR-015) and does not
 * use this at all. This toggles the `.dark` class that globals.css already
 * defines a full token palette for - see FRONTEND_DESIGN.md section 9, which
 * flagged that palette as complete but unreachable, because nothing ever set
 * the class. This is that wiring.
 */

type Theme = "light" | "dark" | "system";

type ThemeContextValue = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  /** The theme actually applied right now, after resolving "system". */
  resolvedTheme: "light" | "dark";
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

const STORAGE_KEY = "theme";

function getSystemTheme(): "light" | "dark" {
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "dark"
    : "light";
}

function applyTheme(resolved: "light" | "dark") {
  document.documentElement.classList.toggle("dark", resolved === "dark");
}

export function ThemeProvider({ children }: { children: React.ReactNode }) {
  // Defaults to "dark" - not "system" - so a fresh visitor's first look at
  // the product matches the always-dark landing page they likely just came
  // from, rather than defaulting to whatever their OS happens to prefer.
  // They can still switch to light at any time; this only sets the FIRST
  // impression.
  const [theme, setThemeState] = useState<Theme>("dark");
  const [resolvedTheme, setResolvedTheme] = useState<"light" | "dark">("dark");

  // Runs once on mount, on the CLIENT only, to read what was actually saved.
  // The blocking inline script in layout.tsx already applied the right class
  // before this component ever rendered - this just brings React's own state
  // into agreement with what is already on screen, so the toggle button
  // shows the correct current choice.
  useEffect(() => {
    const stored = window.localStorage.getItem(STORAGE_KEY) as Theme | null;
    const initial: Theme = stored ?? "dark";
    setThemeState(initial);
    setResolvedTheme(initial === "system" ? getSystemTheme() : initial);
  }, []);

  // If the user is on "system" and their OS theme changes while the tab is
  // open, follow it live rather than waiting for a reload.
  useEffect(() => {
    if (theme !== "system") return;

    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      const next = getSystemTheme();
      setResolvedTheme(next);
      applyTheme(next);
    };

    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [theme]);

  const setTheme = useCallback((next: Theme) => {
    setThemeState(next);
    window.localStorage.setItem(STORAGE_KEY, next);

    const resolved = next === "system" ? getSystemTheme() : next;
    setResolvedTheme(resolved);
    applyTheme(resolved);
  }, []);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, resolvedTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme(): ThemeContextValue {
  const ctx = useContext(ThemeContext);
  if (!ctx) {
    throw new Error("useTheme must be used inside <ThemeProvider>");
  }
  return ctx;
}

/**
 * The exact script text run inline, before hydration, to avoid a flash of
 * the wrong theme.
 *
 * WHY THIS HAS TO BE A BLOCKING INLINE SCRIPT.
 *
 * React does not paint anything until it hydrates, but the browser has
 * already rendered the server's HTML by then - which was generated with no
 * idea what theme this particular visitor chose last time. Waiting for
 * ThemeProvider's own useEffect above to run would mean the page visibly
 * flashes light, then switches to dark, on every single load.
 *
 * A plain synchronous <script> tag runs during HTML parsing, before the
 * browser paints, so the right class is already present at first paint.
 * This is the same technique next-themes uses internally - hand-written here
 * to avoid adding a dependency for four lines of logic (CLAUDE.md section 5).
 */
export const themeInitScript = `
(function () {
  try {
    var stored = localStorage.getItem('${STORAGE_KEY}');
    var theme = stored || 'dark';
    var resolved = theme === 'system'
      ? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
      : theme;
    if (resolved === 'dark') {
      document.documentElement.classList.add('dark');
    }
  } catch (e) {}
})();
`;
