import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import Header from "./components/Header";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Rose Seraphine Tucker — Retrospective",
  description:
    "An austere contemporary retrospective presenting the foundational and conceptual works of Rose Seraphine Tucker.",
  openGraph: {
    title: "Rose Seraphine Tucker — Retrospective",
    description:
      "An austere contemporary retrospective presenting the foundational and conceptual works of Rose Seraphine Tucker.",
    siteName: "Rose Seraphine Tucker Retrospective",
    type: "website",
    locale: "en_US",
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className={`${geistSans.variable} ${geistMono.variable}`}>
      <body>
        <Header />
        <main role="main">{children}</main>
        <footer role="contentinfo" />
      </body>
    </html>
  );
}

