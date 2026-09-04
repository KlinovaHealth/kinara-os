import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";

const inter = Inter({
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800"],
  variable: "--font-inter",
  display: "swap",
});

export const metadata: Metadata = {
  title: {
    default: "Kinara OS · Governed coordination for health, agriculture, logistics and maritime",
    template: "%s · Kinara OS",
  },
  description:
    "One record per person, shared across four domains, with an owner, a jurisdiction and an audit trail for every entry.",
  metadataBase: new URL("https://kinaraos.com"),
  alternates: {
    canonical: "/",
    languages: { fr: "/fr" },
  },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="en" className={inter.variable}>
      <body className="min-h-full flex flex-col bg-white text-[#0F1B2D] antialiased">
        {children}
      </body>
    </html>
  );
}
