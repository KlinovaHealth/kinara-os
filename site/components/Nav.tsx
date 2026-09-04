"use client";

import Image from "next/image";
import Link from "next/link";
import { useState } from "react";

const SUB_NAV = {
  en: [
    { label: "Kinara OS", href: "/" },
    { label: "Architecture", href: "/platform" },
    { label: "Governance", href: "/governance" },
    { label: "Domains", href: "/domains" },
    { label: "Who it is for", href: "/for/governments" },
    { label: "Evidence", href: "/evidence" },
  ],
  fr: [
    { label: "Kinara OS", href: "/fr" },
    { label: "Architecture", href: "/fr/platform" },
    { label: "Gouvernance", href: "/fr/governance" },
    { label: "Domaines", href: "/fr/domains" },
    { label: "Pour qui", href: "/fr/for/governments" },
    { label: "Preuves", href: "/fr/evidence" },
  ],
};

export default function Nav({ lang = "en" }: { lang?: "en" | "fr" }) {
  const [open, setOpen] = useState(false);
  const items = SUB_NAV[lang];
  const toggleHref = lang === "fr" ? "/" : "/fr";
  const toggleLabel = lang === "fr" ? "EN" : "FR";
  const ctaLabel = lang === "fr" ? "Demander un entretien" : "Request a briefing";
  const ctaHref = lang === "fr" ? "/fr/contact" : "/contact";

  return (
    <header className="fixed top-0 left-0 right-0 z-50 bg-white border-b border-[#C3CEDA]" style={{ borderTop: "3px solid #E0561C" }}>
      {/* Top bar */}
      <div className="max-w-[1440px] mx-auto px-8 h-16 flex items-center justify-between">
        <Link href={lang === "fr" ? "/fr" : "/"} className="flex items-center">
          <Image
            src="/logos/logo-kinara-os.jpeg"
            alt="Kinara OS"
            width={400}
            height={110}
            className="h-[58px] w-auto"
            priority
          />
        </Link>

        <div className="flex items-center gap-4">
          <Link
            href={toggleHref}
            className="hidden md:block text-[13px] font-[600] text-[#5D6E82] hover:text-[#0F1B2D] transition-colors"
          >
            {toggleLabel}
          </Link>
          <Link
            href={ctaHref}
            className="hidden md:block text-[13px] font-[700] bg-[#E0561C] text-white px-5 py-2.5 hover:bg-[#c94d19] transition-colors"
          >
            {ctaLabel}
          </Link>
          <button
            className="p-1.5 text-[#0F1B2D]"
            onClick={() => setOpen(!open)}
            aria-label="Toggle menu"
          >
            <svg width="20" height="14" viewBox="0 0 20 14" fill="none">
              {open ? (
                <path d="M1 1l18 12M19 1L1 13" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"/>
              ) : (
                <>
                  <line x1="0" y1="1" x2="20" y2="1" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"/>
                  <line x1="0" y1="7" x2="20" y2="7" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"/>
                  <line x1="0" y1="13" x2="20" y2="13" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"/>
                </>
              )}
            </svg>
          </button>
        </div>
      </div>

      {/* Sub-nav */}
      <div className="hidden md:block border-t border-[#C3CEDA]">
        <div className="max-w-[1440px] mx-auto px-8">
          <nav className="flex items-center gap-8 h-10 overflow-x-auto">
            {items.map((item, i) => (
              <Link
                key={item.href}
                href={item.href}
                className={`text-[13px] whitespace-nowrap transition-colors flex-shrink-0 ${
                  i === 0
                    ? "font-[700] text-[#E0561C] border-b-2 border-[#E0561C] pb-px"
                    : "font-[400] text-[#5D6E82] hover:text-[#0F1B2D]"
                }`}
              >
                {item.label}
              </Link>
            ))}
          </nav>
        </div>
      </div>

      {/* Mobile menu */}
      {open && (
        <div className="border-t border-[#C3CEDA] bg-white px-8 py-6 space-y-4">
          {items.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="block text-[15px] text-[#0F1B2D] py-2 border-b border-[#C3CEDA]"
              onClick={() => setOpen(false)}
            >
              {item.label}
            </Link>
          ))}
          <Link
            href={ctaHref}
            className="block text-[15px] font-[700] bg-[#E0561C] text-white px-5 py-3 text-center mt-4"
            onClick={() => setOpen(false)}
          >
            {ctaLabel}
          </Link>
        </div>
      )}
    </header>
  );
}
