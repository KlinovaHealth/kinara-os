import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = {
  title: "Evidence · Kinara OS",
  description: "Two live tenants on Kinara OS. Real records, real operations, the same stack every future tenant will use.",
};

export default function EvidencePage() {
  return (
    <>
      <Nav />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="bg-white px-8 pt-20 pb-16 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
              Evidence
            </p>
            <h1 className="text-[56px] md:text-[80px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.93] mb-8 max-w-4xl">
              Not a pilot deck. Two live tenants.
            </h1>
            <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl">
              Two operators, two real tenancies, the same governed stack every future tenant will use. Nothing staged, nothing sandboxed.
            </p>
          </div>
        </section>

        {/* ── Tenant 1: Village Health Access ──────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                {/* Label */}
                <div className="flex items-start gap-4 mb-8">
                  <div className="w-1 flex-shrink-0 bg-[#10B981] mt-1" style={{ height: "52px" }} />
                  <div>
                    <p className="text-[12px] font-[700] uppercase tracking-[0.14em] text-[#10B981]">
                      Village Health Access &nbsp;&middot;&nbsp; Non-profit operator
                    </p>
                    <p className="text-[14px] text-[#8A98A8]">Live tenancy &middot; Kinara OS</p>
                  </div>
                </div>

                <h2 className="text-[36px] md:text-[48px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  A live operator, on a live tenancy.
                </h2>
                <div className="text-[17px] text-[#5D6E82] leading-relaxed space-y-4 mb-10">
                  <p>
                    A live community clinic network. Real patients. Real records. Stock entered via WhatsApp by named field workers. Reports drawn directly from the operational database. No manual export, no spreadsheet.
                  </p>
                  <p>
                    The funder report is a query over the same data the clinic uses to run its day.
                  </p>
                </div>
              </div>

              {/* Stats */}
              <div className="grid grid-cols-2 gap-px bg-[#C3CEDA] border border-[#C3CEDA]">
                {[
                  { n: "40", l: "health services", sub: "in their domain" },
                  { n: "0",  l: "manual export steps", sub: "per funder report" },
                  { n: "72h", l: "of offline records", sub: "queued at the edge" },
                  { n: "1",  l: "live tenancy", sub: "same stack as all future tenants" },
                ].map((s) => (
                  <div key={s.n} className="bg-[#F5F8FB] p-8">
                    <p className="text-[48px] font-[800] tracking-[-0.05em] leading-none mb-1" style={{ color: "#10B981" }}>
                      {s.n}
                    </p>
                    <p className="text-[15px] font-[700] text-[#0F1B2D]">{s.l}</p>
                    <p className="text-[12px] text-[#8A98A8] mt-0.5">{s.sub}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Tenant 2: Klinova ────────────────────────────────── */}
        <section className="bg-white px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                {/* Label */}
                <div className="flex items-start gap-4 mb-8">
                  <div className="w-1 flex-shrink-0 bg-[#E0561C] mt-1" style={{ height: "52px" }} />
                  <div>
                    <p className="text-[12px] font-[700] uppercase tracking-[0.14em] text-[#E0561C]">
                      Klinova
                    </p>
                    <p className="text-[14px] text-[#8A98A8]">For-profit operator &middot; live tenancy</p>
                  </div>
                </div>

                <h2 className="text-[36px] md:text-[48px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  A live operator, on a live tenancy.
                </h2>
                <div className="text-[17px] text-[#5D6E82] leading-relaxed space-y-4 mb-10">
                  <p>
                    Klinova is the principal commercial tenant. It connects clinics, hospitals, insurance providers, pharmacies, doctors, and delivery networks into a single coordinated health system. Patient triage, digital prescriptions, referral routing, and partner payments all run on the same Kinara OS stack.
                  </p>
                  <p>
                    Every product Klinova ships uses the same governance model, the same audit log, and the same metered access controls that every future tenant inherits on day one.
                  </p>
                </div>
              </div>

              {/* Stats */}
              <div className="grid grid-cols-2 gap-px bg-[#C3CEDA] border border-[#C3CEDA]">
                {[
                  { n: "4", l: "partner types", sub: "clinics, pharmacies, doctors, delivery" },
                  { n: "48h", l: "payment cycle", sub: "billed and settled per consultation" },
                  { n: "0",  l: "setup fee", sub: "for pilot partners" },
                  { n: "1",  l: "live tenancy", sub: "same stack as all future tenants" },
                ].map((s) => (
                  <div key={s.l} className="bg-white p-8">
                    <p className="text-[48px] font-[800] tracking-[-0.05em] leading-none mb-1" style={{ color: "#E0561C" }}>
                      {s.n}
                    </p>
                    <p className="text-[15px] font-[700] text-[#0F1B2D]">{s.l}</p>
                    <p className="text-[12px] text-[#8A98A8] mt-0.5">{s.sub}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Verifiable ───────────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  What is verifiable
                </p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-6">
                  Every figure traced back to source.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  A funder or auditor can re-derive any reported number by running the same query against the same operational database. Nothing is pre-aggregated into a separate reporting layer.
                </p>
              </div>
              <ul className="space-y-0 border border-[#C3CEDA]">
                {[
                  "Every visit record attributed to a named health worker at a named facility",
                  "Every stock entry timestamped at point of capture, not point of upload",
                  "Every cross-domain query that produced a logistics outcome is in the audit log",
                  "Every report figure can be re-derived by running the same query again",
                ].map((item, i) => (
                  <li
                    key={i}
                    className="flex items-start gap-5 px-6 py-5 border-b border-[#C3CEDA] last:border-0 bg-white"
                  >
                    <span className="flex-shrink-0 w-5 h-5 mt-0.5 rounded-full bg-[#E0561C] flex items-center justify-center">
                      <svg width="10" height="8" viewBox="0 0 10 8" fill="none">
                        <path d="M1 4l2.5 2.5L9 1" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                      </svg>
                    </span>
                    <p className="text-[16px] text-[#0F1B2D] leading-snug">{item}</p>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </section>

        {/* ── CTA ──────────────────────────────────────────────── */}
        <section
          className="relative px-8 py-24 overflow-hidden"
          style={{
            background:
              "radial-gradient(ellipse 60% 80% at 30% 50%, rgba(224,86,28,0.15) 0%, transparent 60%), #080F1E",
          }}
        >
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  Request a demonstration
                </p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  We will show you the tenant, the records, and the audit log.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  Everything is live, nothing is staged. A briefing is 45 minutes.
                </p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link
                  href="/contact"
                  className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors"
                >
                  Request a briefing
                </Link>
                <p className="text-[13px] text-[#3A4F68] mt-2">
                  Kinara OS is built and owned by Klinova &nbsp;&middot;&nbsp; kinaraos.com
                </p>
              </div>
            </div>
          </div>
        </section>

      </main>
      <Footer />
    </>
  );
}
