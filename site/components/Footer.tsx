"use client";

import Image from "next/image";
import Link from "next/link";

const cols = [
  {
    heading: "Platform",
    links: [
      { label: "Architecture", href: "/platform" },
      { label: "Governance", href: "/governance" },
      { label: "Domains", href: "/domains" },
    ],
  },
  {
    heading: "Who it is for",
    links: [
      { label: "Governments", href: "/for/governments" },
      { label: "Funders", href: "/for/funders" },
      { label: "Partners", href: "/for/partners" },
    ],
  },
  {
    heading: "Company",
    links: [
      { label: "Evidence", href: "/evidence" },
      { label: "About", href: "/about" },
      { label: "Contact", href: "/contact" },
    ],
  },
  {
    heading: "Legal",
    links: [
      { label: "Privacy", href: "/legal/privacy" },
      { label: "Terms", href: "/legal/terms" },
    ],
  },
];

export default function Footer({ lang = "en" }: { lang?: "en" | "fr" }) {
  const prefix = lang === "fr" ? "/fr" : "";

  return (
    <footer className="border-t border-[#C3CEDA] bg-white pt-14 pb-10 px-8">
      <div className="max-w-[1440px] mx-auto">
        <div className="grid grid-cols-2 md:grid-cols-5 gap-10 mb-14">
          <div className="col-span-2 md:col-span-1">
            <Link href={prefix + "/"} className="inline-block mb-5">
              <Image
                src="/logos/logo-kinara-os.jpeg"
                alt="Kinara OS"
                width={275}
                height={75}
                className="h-[60px] w-auto"
              />
            </Link>
            <p className="text-[13px] text-[#8A98A8] leading-relaxed max-w-[180px]">
              Built and owned by Klinova.
            </p>
          </div>

          {cols.map((col) => (
            <div key={col.heading}>
              <p className="text-[11px] font-[700] uppercase tracking-[0.1em] text-[#0F1B2D] mb-4">
                {col.heading}
              </p>
              <ul className="space-y-3">
                {col.links.map((link) => (
                  <li key={link.href}>
                    <Link
                      href={prefix + link.href}
                      className="text-[14px] text-[#5D6E82] hover:text-[#0F1B2D] transition-colors"
                    >
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="border-t border-[#C3CEDA] pt-6 flex flex-col md:flex-row items-start md:items-center justify-between gap-4">
          <p className="text-[13px] text-[#8A98A8]">
            &copy; {new Date().getFullYear()} Klinova Health LLC &nbsp;&middot;&nbsp; kinaraos.com
          </p>
          <Link
            href={lang === "fr" ? "/" : "/fr"}
            className="text-[13px] font-[500] text-[#5D6E82] hover:text-[#0F1B2D] transition-colors"
          >
            {lang === "fr" ? "English" : "Français"}
          </Link>
        </div>
      </div>
    </footer>
  );
}
