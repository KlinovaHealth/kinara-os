"use client";

import Image from "next/image";
import Link from "next/link";
import { useEffect } from "react";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

function useReveal() {
  useEffect(() => {
    const els = document.querySelectorAll(".reveal");
    const io = new IntersectionObserver(
      (entries) =>
        entries.forEach((e) => {
          if (e.isIntersecting) {
            e.target.classList.add("visible");
            io.unobserve(e.target);
          }
        }),
      { threshold: 0.06 }
    );
    els.forEach((el) => io.observe(el));
    return () => io.disconnect();
  }, []);
}

/* ── Data ─────────────────────────────────────────────────────── */

const STORIES = [
  {
    label: "Health",
    color: "#10B981",
    headline: "A shortage averted before anyone noticed.",
    body: "A nurse in a rural clinic enters stock levels on a feature phone. Kinara OS reads the shortfall, raises a logistics query, verifies it against the nearest warehouse, and dispatches a vehicle. No phone call, no spreadsheet, no ministry surrendering its records.",
  },
  {
    label: "Agriculture",
    color: "#F59E0B",
    headline: "A harvest that moves itself.",
    body: "A confirmed yield becomes a transport demand the moment it is recorded. How much, from which district, to which market. The logistics domain receives the signal under governance. The grain moves before the window closes.",
  },
  {
    label: "Logistics",
    color: "#3B82F6",
    headline: "A vehicle routed before anyone asked.",
    body: "A warehouse registers a surplus. A clinic registers a shortfall. Kinara OS raises the match, checks the policy, and routes the vehicle. No coordinator in the middle. No phone call. The record of the decision is in the log before the driver leaves.",
  },
  {
    label: "Maritime",
    color: "#0EA5E9",
    headline: "Cargo released. Truck dispatched. No call made.",
    body: "A container clears customs at the port. A logistics instruction fires inland before the container leaves the dock. No manual handover. No missing shipment. Two operators coordinated without either seeing the other's data.",
  },
];

const DOMAINS = [
  {
    name: "Health",
    services: 40,
    steward: "Ministry of Health",
    desc: "Patients, visits, labs, immunisation, outbreak detection.",
    color: "#10B981",
    href: "/domains/health",
  },
  {
    name: "Agriculture",
    services: 40,
    steward: "Ministry of Agriculture",
    desc: "Plots, inputs, harvest, market price, subsidy.",
    color: "#F59E0B",
    href: "/domains/agriculture",
  },
  {
    name: "Logistics",
    services: 36,
    steward: "Ministry of Transport",
    desc: "Routing, delivery, warehousing, returns, forecasting.",
    color: "#3B82F6",
    href: "/domains/logistics",
  },
  {
    name: "Maritime",
    services: 36,
    steward: "Port Authority",
    desc: "Vessels, berths, cargo, customs, trade finance.",
    color: "#0EA5E9",
    href: "/domains/maritime",
  },
];

const STEPS = [
  {
    n: "01",
    head: "A nurse counts the shelf.",
    body: "Stock entered on a feature phone over WhatsApp or SMS. No app, no signal required. Attributed to a named health worker.",
  },
  {
    n: "02",
    head: "The record clears the edge.",
    body: "Held locally, queued, forwarded and reconciled the moment a signal returns. No record is ever lost.",
  },
  {
    n: "03",
    head: "The join is authorised.",
    body: "The health stockout raises a logistics question. Kinara Core checks the lawful basis, permits the join, and writes the audit entry.",
  },
  {
    n: "04",
    head: "Logistics answers.",
    body: "The warehouse holds it. A vehicle is routed. Movement authorised. Movement logged.",
  },
  {
    n: "05",
    head: "The nurse is told.",
    body: "The answer arrives on the same phone: a dispatch reference and an arrival window. Elapsed time: minutes.",
  },
];

/* ── Page ─────────────────────────────────────────────────────── */

export default function Home() {
  useReveal();

  return (
    <>
      <Nav />

      <main className="pt-[108px]">

        {/* ═══════════════════════════════════════════════════════════
            HERO  — full-bleed dark, gradient glow, dot grid
        ═══════════════════════════════════════════════════════════ */}
        <section
          className="relative overflow-hidden"
          style={{
            background:
              "radial-gradient(ellipse 80% 60% at 75% -10%, rgba(224,86,28,0.22) 0%, transparent 60%), radial-gradient(ellipse 60% 60% at 10% 90%, rgba(14,165,233,0.08) 0%, transparent 55%), #080F1E",
          }}
        >
          {/* Dot grid overlay */}
          <div className="absolute inset-0 dot-grid pointer-events-none" />

          {/* Thin orange line at very top */}
          <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-[#E0561C] to-transparent" />

          <div className="relative max-w-[1440px] mx-auto px-8 pt-24 pb-0">
            {/* Eyebrow */}
            <div className="inline-flex items-center gap-2 mb-8">
              <span className="w-6 h-px bg-[#E0561C]" />
              <p className="text-[12px] font-[600] uppercase tracking-[0.18em] text-[#8A98A8]">
                Building Africa&apos;s digital infrastructure
              </p>
            </div>

            {/* Headline */}
            <div className="mb-8 max-w-5xl">
              <h1 className="text-[72px] md:text-[108px] lg:text-[130px] font-[800] tracking-[-0.05em] leading-[0.91]">
                <span className="text-white block">Africa runs on data</span>
                <span className="block" style={{
                  background: "linear-gradient(135deg, #FF6B35 0%, #E0561C 40%, #FF9A6C 100%)",
                  WebkitBackgroundClip: "text",
                  WebkitTextFillColor: "transparent",
                  backgroundClip: "text",
                }}>
                  that never connects.
                </span>
              </h1>
            </div>

            {/* Sub + CTAs */}
            <div className="grid md:grid-cols-2 gap-10 mb-16 max-w-5xl">
              <p className="text-[18px] md:text-[20px] text-[#8A98A8] leading-relaxed">
                Kinara OS is a jurisdiction-aware, policy-governed coordination layer that lets health, agriculture, logistics, and maritime services operate independently while sharing only authorized, auditable, purpose-limited information.
              </p>
              <div className="flex flex-col sm:flex-row md:flex-col lg:flex-row gap-4 items-start self-end pb-1">
                <Link
                  href="/contact"
                  className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors whitespace-nowrap"
                >
                  Request a briefing
                </Link>
                <Link
                  href="/platform"
                  className="link-arrow inline-flex items-center text-[14px] font-[600] text-[#8A98A8] hover:text-white transition-colors whitespace-nowrap py-3.5"
                >
                  See the architecture <span>&rarr;</span>
                </Link>
              </div>
            </div>

            {/* Trust strip */}
            <div className="flex flex-wrap items-center gap-y-2 border-t border-[#1A2A40] pt-6 mb-0 pb-10">
              {[
                "152 services in production",
                "4 governed domains",
                "144 isolated databases, one per service",
                "1 live tenant",
              ].map((item, i) => (
                <span key={i} className="text-[13px] text-[#5D6E82] flex items-center">
                  {i > 0 && <span className="mx-4 text-[#1A2A40]">&middot;</span>}
                  {item}
                </span>
              ))}
            </div>
          </div>

          {/* Architecture diagram — floats out of hero */}
          <div className="relative max-w-[1440px] mx-auto px-8">
            <div
              className="relative overflow-hidden"
              style={{
                border: "1px solid rgba(255,255,255,0.08)",
                background: "#0D1828",
                boxShadow: "0 0 80px rgba(224,86,28,0.08), 0 40px 80px rgba(0,0,0,0.4)",
              }}
            >
              {/* Vertical label */}
              <div className="absolute left-0 top-0 bottom-0 w-14 border-r border-[#1A2A40] flex items-end pb-6 z-10">
                <p
                  className="text-[9px] font-[600] uppercase tracking-[0.18em] text-[#3A4F68] whitespace-nowrap"
                  style={{ writingMode: "vertical-rl", transform: "rotate(180deg)" }}
                >
                  Kinara OS &nbsp;&rarr;&nbsp; Powered by Kinara Core
                </p>
              </div>
              <div className="pl-14">
                <Image
                  src="/diagrams/architecture-en.png"
                  alt="Kinara OS system: four sovereign domains coordinated by Kinara Core"
                  width={1600}
                  height={760}
                  className="w-full h-auto"
                  priority
                />
              </div>
            </div>

            {/* Below-diagram links */}
            <div className="grid grid-cols-2 border-x border-b border-[#1A2A40]">
              <Link
                href="/contact"
                className="link-arrow flex items-center justify-between px-6 py-4 border-r border-[#1A2A40] text-[13px] font-[600] text-[#5D6E82] hover:text-white hover:bg-[#0D1828] transition-colors"
              >
                Request a briefing <span className="text-[#E0561C]">&rarr;</span>
              </Link>
              <Link
                href="/platform"
                className="link-arrow flex items-center justify-between px-6 py-4 text-[13px] font-[600] text-[#5D6E82] hover:text-white hover:bg-[#0D1828] transition-colors"
              >
                See the architecture <span className="text-[#E0561C]">&rarr;</span>
              </Link>
            </div>
          </div>

          {/* Hero bottom fade */}
          <div className="h-24 bg-gradient-to-b from-transparent to-white pointer-events-none" />
        </section>

        {/* ═══════════════════════════════════════════════════════════
            THREE STORIES  — white, magazine layout
        ═══════════════════════════════════════════════════════════ */}
        <section className="bg-white px-8 py-24">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-16 reveal">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">
                Why it matters
              </p>
              <h2 className="text-[48px] md:text-[64px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] max-w-2xl">
                Four stories. One system.
              </h2>
            </div>

            <div className="grid sm:grid-cols-2 md:grid-cols-4 gap-px bg-[#C3CEDA]">
              {STORIES.map((s, i) => (
                <div
                  key={i}
                  className="bg-white p-8 group card-lift cursor-default"
                >
                  <div
                    className="inline-flex items-center gap-2 mb-6 px-3 py-1 text-[11px] font-[700] uppercase tracking-[0.14em]"
                    style={{ background: s.color + "15", color: s.color }}
                  >
                    {s.label}
                  </div>
                  <h3 className="text-[22px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-snug mb-4">
                    {s.headline}
                  </h3>
                  <p className="text-[15px] text-[#5D6E82] leading-relaxed">{s.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ═══════════════════════════════════════════════════════════
            EARLY WARNING  — dark feature section
        ═══════════════════════════════════════════════════════════ */}
        <section
          className="relative px-8 py-24 overflow-hidden"
          style={{ background: "#080F1E" }}
        >
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto reveal">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  Early warning
                </p>
                <h2 className="text-[40px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-8">
                  An outbreak detected before anyone calls it one.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-6">
                  A cluster of clinic visits rises in three districts. A logistics record shows unusual movement from the same area. An agricultural record shows a food security shock two weeks prior. A vessel from an affected region cleared the port ten days before that.
                </p>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-8">
                  No single domain sees the pattern. Kinara OS does. The signal exists in the data that already flows through the system every day. It does not require a new data collection effort. It requires a governed way to ask the question across boundaries.
                </p>
                <p className="text-[15px] font-[700] text-white border-l-2 border-[#E0561C] pl-5">
                  That is weeks earlier than traditional surveillance catches it.
                </p>
                <div className="mt-8">
                  <Link href="/early-warning" className="inline-flex items-center gap-2 text-[14px] font-[700] text-[#10B981] hover:text-[#34D399] transition-colors">
                    See the full signal chain &rarr;
                  </Link>
                </div>
              </div>

              {/* Signal chain */}
              <div className="space-y-2">
                <div className="flex items-center gap-4 px-5 pb-1">
                  <div className="flex-shrink-0 w-24 text-right">
                    <span className="text-[10px] font-[600] uppercase tracking-[0.15em] text-[#3A4F68]">Days elapsed</span>
                  </div>
                  <div className="flex-shrink-0 w-px" />
                  <div className="flex-1">
                    <span className="text-[10px] font-[600] uppercase tracking-[0.15em] text-[#3A4F68]">Signal</span>
                  </div>
                </div>
                {[
                  { domain: "Maritime", color: "#0EA5E9", signal: "Vessel from affected region clears port", day: "Day 0" },
                  { domain: "Agriculture", color: "#F59E0B", signal: "Food security shock recorded in three districts", day: "Day 14" },
                  { domain: "Logistics", color: "#3B82F6", signal: "Unusual movement pattern flagged on inland routes", day: "Day 18" },
                  { domain: "Health", color: "#10B981", signal: "Clinic visit cluster rises across same districts", day: "Day 21" },
                  { domain: "Kinara Core", color: "#E0561C", signal: "Cross-domain pattern raised. Early warning issued.", day: "Day 21" },
                ].map((row, i) => (
                  <div
                    key={i}
                    className="flex items-start gap-4 p-5 border border-[#1A2A40]"
                    style={{ background: i === 4 ? "rgba(224,86,28,0.08)" : "rgba(26,42,64,0.4)", borderColor: i === 4 ? "rgba(224,86,28,0.3)" : undefined }}
                  >
                    <div className="flex-shrink-0 w-24 text-right">
                      <span className="text-[11px] font-[700] text-[#3A4F68]">{row.day}</span>
                    </div>
                    <div
                      className="flex-shrink-0 w-px self-stretch"
                      style={{ background: row.color }}
                    />
                    <div className="flex-1">
                      <p
                        className="text-[10px] font-[700] uppercase tracking-[0.14em] mb-1"
                        style={{ color: row.color }}
                      >
                        {row.domain}
                      </p>
                      <p className={`text-[14px] leading-snug ${i === 4 ? "text-white font-[700]" : "text-[#5D6E82]"}`}>
                        {row.signal}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ═══════════════════════════════════════════════════════════
            THE GOVERNANCE LAYER  — light plane, two-column
        ═══════════════════════════════════════════════════════════ */}
        <section className="bg-[#F5F8FB] px-8 py-24 border-y border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start reveal">
              {/* Left: domain card */}
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  The system
                </p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  The Kinara Governance Layer
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-6">
                  Kinara Core sits at the intersection of four sovereign domains. It never holds data; it governs the questions that cross between them. Every join is checked against policy, executed with minimum disclosure, and written to an immutable audit log.
                </p>
                <Link
                  href="/governance"
                  className="link-arrow text-[14px] font-[700] text-[#E0561C] hover:text-[#c94d19] transition-colors"
                >
                  Read the governance model <span>&rarr;</span>
                </Link>
              </div>

              {/* Right: domain cards */}
              <div className="space-y-2">
                {DOMAINS.map((d) => (
                  <Link
                    key={d.name}
                    href={d.href}
                    className="group flex items-center gap-4 bg-white border border-[#C3CEDA] p-5 card-lift block"
                  >
                    <div
                      className="flex-shrink-0 w-1 self-stretch rounded-full"
                      style={{ background: d.color }}
                    />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-baseline gap-3 mb-0.5">
                        <p className="text-[17px] font-[800] text-[#0F1B2D] tracking-[-0.02em]">
                          {d.name}
                        </p>
                        <p className="text-[12px] font-[600] text-[#8A98A8] uppercase tracking-[0.08em]">
                          {d.services} services
                        </p>
                      </div>
                      <p className="text-[13px] text-[#5D6E82]">{d.desc}</p>
                    </div>
                    <span
                      className="text-[20px] text-[#C3CEDA] group-hover:text-[#E0561C] transition-colors flex-shrink-0"
                    >
                      &rarr;
                    </span>
                  </Link>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ═══════════════════════════════════════════════════════════
            BY THE NUMBERS  — white, large orange figures
        ═══════════════════════════════════════════════════════════ */}
        <section className="bg-white px-8 py-24">
          <div className="max-w-[1440px] mx-auto reveal">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-px bg-[#C3CEDA] border border-[#C3CEDA]">
              {[
                { n: "152", l: "services", sub: "in production" },
                { n: "144", l: "isolated databases", sub: "one per service" },
                { n: "4",   l: "domains",   sub: "fully governed" },
                { n: "72h", l: "of records", sub: "held offline at the edge" },
              ].map((s) => (
                <div key={s.n} className="bg-white p-10 group">
                  <p
                    className="text-[64px] font-[800] tracking-[-0.05em] leading-none mb-1"
                    style={{ color: "#E0561C" }}
                  >
                    {s.n}
                  </p>
                  <p className="text-[18px] font-[700] text-[#0F1B2D] tracking-[-0.02em]">
                    {s.l}
                  </p>
                  <p className="text-[13px] text-[#8A98A8] mt-0.5">{s.sub}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ═══════════════════════════════════════════════════════════
            HOW IT WORKS  — dark, step-by-step
        ═══════════════════════════════════════════════════════════ */}
        <section
          className="relative px-8 py-24 overflow-hidden"
          style={{
            background:
              "radial-gradient(ellipse 70% 80% at 90% 50%, rgba(224,86,28,0.10) 0%, transparent 60%), #080F1E",
          }}
        >
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-50" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start reveal">
              {/* Left sticky */}
              <div className="md:sticky md:top-32">
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  How it works
                </p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-8">
                  One request. Five steps. No custody surrendered.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-8">
                  Follow one stock request from a nurse at a rural clinic all the way to a logistics dispatch. Watch the audit log fill without any ministry seeing another&apos;s raw records.
                </p>
                <blockquote className="border-l-2 border-[#E0561C] pl-5">
                  <p className="text-[18px] font-[700] text-white leading-snug">
                    Elapsed time is minutes. No one filed a request. No ministry gave up custody.
                  </p>
                </blockquote>
              </div>

              {/* Right: steps */}
              <div className="space-y-2">
                {STEPS.map((step, i) => (
                  <div
                    key={i}
                    className="flex gap-5 p-6 border border-[#1A2A40] hover:border-[#E0561C]/30 transition-colors"
                    style={{ background: "rgba(26,42,64,0.4)" }}
                  >
                    <div className="flex-shrink-0">
                      <span
                        className="text-[11px] font-[800] block mt-0.5"
                        style={{ color: "#E0561C" }}
                      >
                        {step.n}
                      </span>
                    </div>
                    <div>
                      <p className="text-[16px] font-[700] text-white mb-1.5 tracking-[-0.01em]">
                        {step.head}
                      </p>
                      <p className="text-[14px] text-[#5D6E82] leading-relaxed">
                        {step.body}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Bottom fade to white */}
          <div className="absolute bottom-0 left-0 right-0 h-20 bg-gradient-to-b from-transparent to-white pointer-events-none" />
        </section>

        {/* ═══════════════════════════════════════════════════════════
            PROOF  — white, editorial
        ═══════════════════════════════════════════════════════════ */}
        <section className="bg-white px-8 py-24">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center reveal">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  In production
                </p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  Two tenants. One live system.
                </h2>

                {/* Klinova — for-profit */}
                <div className="mb-8 pl-5 border-l-2 border-[#E0561C]">
                  <p className="text-[12px] font-[700] uppercase tracking-[0.12em] text-[#E0561C] mb-2">
                    Klinova &nbsp;&middot;&nbsp; For-profit operator
                  </p>
                  <p className="text-[16px] text-[#5D6E82] leading-relaxed">
                    Klinova is the principal commercial tenant on Kinara OS, building health-sector coordination products for clinics, hospitals, insurance providers, pharmacies, doctors, and delivery networks across the continent. Every feature Klinova ships runs on the same governed stack available to future tenants.
                  </p>
                </div>

                {/* Village Health Access — non-profit */}
                <div className="mb-10 pl-5 border-l-2 border-[#10B981]">
                  <p className="text-[12px] font-[700] uppercase tracking-[0.12em] text-[#10B981] mb-2">
                    Village Health Access &nbsp;&middot;&nbsp; Non-profit operator
                  </p>
                  <p className="text-[16px] text-[#5D6E82] leading-relaxed">
                    A live community clinic network. Real patients. Real records. Stock entered via WhatsApp by named field workers. Reports drawn directly from the operational database. No manual export, no spreadsheet.
                  </p>
                </div>

                <Link
                  href="/evidence"
                  className="link-arrow text-[15px] font-[700] text-[#E0561C] hover:text-[#c94d19] transition-colors"
                >
                  See the evidence <span>&rarr;</span>
                </Link>
              </div>

              {/* Audit card visual */}
              <div className="bg-[#F5F8FB] border border-[#C3CEDA] p-8">
                <p className="text-[11px] font-[700] uppercase tracking-[0.14em] text-[#8A98A8] mb-6">
                  Audit log · Village Health Access
                </p>
                <div className="space-y-3">
                  {[
                    { ts: "08:14:03", action: "STOCK_ENTRY", worker: "A. Diallo · Clinic 04", status: "OK" },
                    { ts: "08:14:09", action: "CROSS_DOMAIN_JOIN", worker: "Kinara Core → Logistics", status: "AUTHORISED" },
                    { ts: "08:14:11", action: "DISPATCH_RAISED", worker: "Warehouse W-07 → Vehicle V-12", status: "OK" },
                    { ts: "08:17:22", action: "DELIVERY_ETA_SENT", worker: "→ A. Diallo · Clinic 04", status: "OK" },
                  ].map((row, i) => (
                    <div
                      key={i}
                      className="flex items-center gap-4 py-3 border-b border-[#C3CEDA] last:border-0 font-mono text-[12px]"
                    >
                      <span className="text-[#8A98A8] flex-shrink-0 w-16">{row.ts}</span>
                      <span className="font-[700] text-[#0F1B2D] flex-shrink-0 w-40 truncate">{row.action}</span>
                      <span className="text-[#5D6E82] flex-1 truncate hidden sm:block">{row.worker}</span>
                      <span
                        className="flex-shrink-0 text-[10px] font-[700] px-2 py-0.5"
                        style={{
                          background: row.status === "AUTHORISED" ? "#E0561C15" : "#10B98115",
                          color: row.status === "AUTHORISED" ? "#E0561C" : "#10B981",
                        }}
                      >
                        {row.status}
                      </span>
                    </div>
                  ))}
                </div>
                <p className="text-[12px] text-[#8A98A8] mt-5">
                  Every entry is immutable. Every join is attributed.
                </p>
              </div>
            </div>
          </div>
        </section>

        {/* ═══════════════════════════════════════════════════════════
            CLOSE CTA  — dark, full-bleed
        ═══════════════════════════════════════════════════════════ */}
        <section
          className="relative px-8 py-28 overflow-hidden"
          style={{
            background:
              "radial-gradient(ellipse 60% 80% at 30% 50%, rgba(224,86,28,0.18) 0%, transparent 60%), #080F1E",
          }}
        >
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto text-center reveal">
            <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-8">
              Connect Africa
            </p>
            <h2 className="text-[52px] md:text-[80px] lg:text-[96px] font-[800] tracking-[-0.05em] text-white leading-[0.93] mb-8 max-w-4xl mx-auto">
              Bring us the problem that crosses two ministries.
            </h2>
            <p className="text-[18px] text-[#5D6E82] mb-12 max-w-xl mx-auto leading-relaxed">
              That is the one we are built for. A briefing is 45 minutes. We will show you the running system, not slides.
            </p>
            <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
              <Link
                href="/contact"
                className="bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors"
              >
                Request a briefing
              </Link>
              <Link
                href="/about"
                className="link-arrow text-[15px] font-[600] text-[#5D6E82] hover:text-white transition-colors py-4"
              >
                About Kinara OS <span>&rarr;</span>
              </Link>
            </div>
            <p className="text-[13px] text-[#3A4F68] mt-10">
              Kinara OS is built and owned by Klinova &nbsp;&middot;&nbsp; kinaraos.com
            </p>
          </div>
        </section>
      </main>

      <Footer />
    </>
  );
}
