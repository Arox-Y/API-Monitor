import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { Providers } from "@/components/providers";
import { ThemeProvider, themeInitScript } from "@/components/theme-provider";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "API Monitor",
  description: "Monitor the health of your APIs and websites.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      // suppressHydrationWarning is scoped to exactly this element. It is
      // needed because the inline script below adds the "dark" class to
      // <html> before React ever runs, so React's hydration check would
      // otherwise see a class attribute it did not itself render and warn
      // about a mismatch that is not actually a bug.
      suppressHydrationWarning
    >
      <head>
        {/* Must run before paint, so it is a plain blocking script - see
            theme-provider.tsx for why a <Script> component would be too
            late and cause a visible flash of the wrong theme. */}
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body className="min-h-full flex flex-col">
        <ThemeProvider>
          <Providers>{children}</Providers>
        </ThemeProvider>
      </body>
    </html>
  );
}
