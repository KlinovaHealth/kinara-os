import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = {
  title: "Disease Surveillance & Early Warning · Kinara OS",
  description: "Cross-domain outbreak detection. Kinara OS surfaces epidemic signals weeks before single-domain surveillance by correlating health, logistics, agriculture and maritime data under governance.",
};

const SIGNAL_CHAIN = [
  { domain: "Maritime", color: "#0EA5E9", day: "Day 0",  signal: "Vessel from an affected region clears port customs. Crew manifest recorded. No flag raised. No single domain has context.", detail: "The maritime domain records every vessel arrival, crew list, and port of origin. In isolation, this is routine data. Its significance only emerges in combination with what follows." },
  { domain: "Agriculture", color: "#F59E0B", day: "Day 14", signal: "A food security shock is recorded across three inland districts: lower yields, disrupted supply. Logistics domain sees unusual inbound demand.", detail: "Malnutrition and immune suppression precede disease spikes by two to four weeks in documented outbreak patterns. An agriculture signal this size, from these districts, is a known precursor. No health system sees it in time without a cross-domain query." },
  { domain: "Logistics", color: "#3B82F6", day: "Day 18", signal: "Unusual movement pattern flagged on inland routes: higher-than-baseline traffic from the districts, no corresponding economic event.", detail: "Movement data from the logistics domain shows population displacement before health facilities register it. Field workers travelling between districts, informal traders shifting routes. These appear in transport records first." },
  { domain: "Health", color: "#10B981", day: "Day 21", signal: "Clinic visit cluster rises sharply across the same three districts. Multiple facilities. Similar presenting symptoms. No cross-district coordination.", detail: "Individual clinics see their own queue. Without a cross-facility signal, each treats a local spike as a local issue. The pattern is invisible inside any single health facility's records." },
  { domain: "Kinara Core", color: "#E0561C", day: "Day 21", signal: "Cross-domain pattern raised. Four correlated signals across maritime, agriculture, logistics and health. Early warning issued to the health steward.", detail: "Kinara Core correlates the four signals under governance. Each join is authorised, the minimum data is disclosed, and the alert is attributed in the audit log. The health steward receives the warning with the evidence chain attached. No raw data from any other domain is disclosed." },
];

const WHY_IT_MATTERS = [
  { heading: "Traditional surveillance is single-domain.", body: "Health information systems track clinic visits and lab results. They cannot see the food security shock that preceded the cluster by two weeks, or the movement pattern that explains its geography. The signals exist. The connection does not." },
  { heading: "The data collection is already done.", body: "Kinara OS does not require a new surveillance programme. The maritime, agriculture, logistics and health records that produce the early warning already flow through the system every day as part of normal operations. The outbreak signal is a query, not a new data source." },
  { heading: "Every join is governed and attributed.", body: "Correlating health data with logistics or agriculture data is a cross-domain query. Kinara Core checks the lawful basis, executes with minimum disclosure, and writes the audit entry before the result is returned. The health steward can see exactly what was asked and what was disclosed." },
  { heading: "Weeks, not days.", body: "The maritime-to-health signal chain in documented outbreak patterns spans 14 to 28 days. A system that sees the Day 0 maritime signal and correlates it with Day 14 agriculture data issues a warning on Day 21. Single-domain surveillance issues the same warning on Day 28 to 35, when the cluster is already visible in clinic queues." },
];

const USE_CASES = [
  { label: "Epidemic preparedness", color: "#10B981", desc: "Correlate health cluster data with logistics movement and maritime arrivals to detect outbreaks 7 to 14 days earlier than clinic-only surveillance." },
  { label: "Food security and disease correlation", color: "#F59E0B", desc: "Match agriculture yield shocks to downstream health outcomes. Identify at-risk populations before malnutrition presents clinically." },
  { label: "Vector and transmission mapping", color: "#3B82F6", desc: "Use logistics movement data to model disease transmission routes. Identify which transport corridors correlate with spread patterns." },
  { label: "Port health screening", color: "#0EA5E9", desc: "Cross-reference maritime crew manifests and port-of-origin data with active outbreak locations to inform port health decisions." },
  { label: "Supply chain readiness", color: "#E0561C", desc: "When a health alert is raised, the logistics domain already holds warehouse stock levels and vehicle availability. Response dispatch begins before the formal request is filed." },
];

export default function EarlyWarningPage() {
  return (
    <>
      <Nav />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section
          className="relative overflow-hidden px-8 pt-24 pb-0"
          style={{ background: "radial-gradient(ellipse 70% 60% at 80% 0%, rgba(16,185,129,0.12) 0%, transparent 55%), radial-gradient(ellipse 50% 60% at 10% 90%, rgba(14,165,233,0.06) 0%, transparent 55%), #080F1E" }}
        >
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="inline-flex items-center gap-2 mb-8">
              <span className="w-6 h-px bg-[#10B981]" />
              <p className="text-[12px] font-[600] uppercase tracking-[0.18em] text-[#8A98A8]">Disease surveillance · Early warning</p>
            </div>
            <h1 className="text-[64px] md:text-[96px] lg:text-[112px] font-[800] tracking-[-0.05em] leading-[0.91] mb-8 max-w-5xl">
              <span className="text-white block">An outbreak detected</span>
              <span className="block" style={{ background: "linear-gradient(135deg, #10B981 0%, #34D399 100%)", WebkitBackgroundClip: "text", WebkitTextFillColor: "transparent", backgroundClip: "text" }}>
                before anyone calls it one.
              </span>
            </h1>
            <div className="grid md:grid-cols-2 gap-10 mb-16 max-w-5xl">
              <p className="text-[18px] md:text-[20px] text-[#8A98A8] leading-relaxed">
                Kinara OS surfaces epidemic signals weeks before single-domain surveillance by correlating health, logistics, agriculture, and maritime data under governance. No new data collection. No shared databases. Every join authorised and logged.
              </p>
              <div className="flex flex-col gap-4 items-start self-end pb-1">
                <Link href="/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors">
                  Request a briefing
                </Link>
                <Link href="/domains/health" className="text-[14px] font-[600] text-[#8A98A8] hover:text-white transition-colors">
                  See the Health domain &rarr;
                </Link>
              </div>
            </div>
          </div>
          <div className="h-20 bg-gradient-to-b from-transparent to-white pointer-events-none" />
        </section>

        {/* ── Why it matters ────────────────────────────────────── */}
        <section className="bg-white px-8 py-24 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">The problem</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  The signals exist. The connection does not.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-6">
                  Every documented epidemic has a precursor signal that appears in data other than the health system: in food prices, in transport patterns, in port arrivals. That data exists. It is collected daily. It sits in separate systems with separate stewards.
                </p>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  Traditional surveillance waits for the health system to see enough cases to raise a flag. By then, the outbreak is already in the community. Kinara OS sees the correlated signal across all four domains before the health system has a cluster to report.
                </p>
              </div>
              <div className="space-y-0 border border-[#C3CEDA]">
                {WHY_IT_MATTERS.map((item, i) => (
                  <div key={i} className="bg-white px-7 py-6 border-b border-[#C3CEDA] last:border-0">
                    <p className="text-[16px] font-[700] text-[#0F1B2D] mb-2">{item.heading}</p>
                    <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Signal chain ──────────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-24 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-14">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">How it works</p>
              <h2 className="text-[40px] md:text-[56px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] max-w-3xl">
                Follow one outbreak signal across four domains.
              </h2>
            </div>
            <div className="space-y-0 border border-[#C3CEDA]">
              {SIGNAL_CHAIN.map((row, i) => (
                <div
                  key={i}
                  className="grid md:grid-cols-2 border-b border-[#C3CEDA] last:border-0"
                  style={{ borderLeftWidth: "3px", borderLeftStyle: "solid", borderLeftColor: row.color, background: i === 4 ? "rgba(224,86,28,0.03)" : "white" }}
                >
                  <div className="px-8 py-6 md:border-r border-[#C3CEDA]">
                    <p className="text-[11px] font-[700] uppercase tracking-[0.15em] mb-1" style={{ color: row.color }}>{row.domain}</p>
                    <p className="text-[12px] text-[#8A98A8] mb-3">{row.day} from first signal</p>
                    <p className={`text-[16px] font-[700] leading-snug ${i === 4 ? "text-[#E0561C]" : "text-[#0F1B2D]"}`}>{row.signal}</p>
                  </div>
                  <div className="px-8 py-6">
                    <p className="text-[14px] text-[#5D6E82] leading-relaxed">{row.detail}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Use cases ─────────────────────────────────────────── */}
        <section className="bg-white px-8 py-24 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-14">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">Applications</p>
              <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] max-w-2xl">
                What this enables across domains.
              </h2>
            </div>
            <div className="grid md:grid-cols-3 gap-px bg-[#C3CEDA]">
              {USE_CASES.map((uc) => (
                <div key={uc.label} className="bg-white p-8">
                  <div className="w-8 h-1 mb-5 rounded-full" style={{ background: uc.color }} />
                  <p className="text-[17px] font-[800] text-[#0F1B2D] mb-3 tracking-[-0.02em]">{uc.label}</p>
                  <p className="text-[14px] text-[#5D6E82] leading-relaxed">{uc.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Governance note ───────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-24 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">Data governance</p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  No domain surrenders custody to produce the warning.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-6">
                  The maritime steward does not share its crew manifests with the health system. The agriculture steward does not share its yield records with the logistics system. Each domain retains full custody of its own data throughout.
                </p>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  Kinara Core asks a governed question across the boundary, receives a minimum-disclosure answer, and issues the alert. Every step is in the audit log. The health steward can see exactly what was asked, what domains were queried, and what minimum data was used to produce the signal.
                </p>
              </div>
              <div style={{ background: "#080F1E" }} className="relative p-10 overflow-hidden">
                <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
                <div className="relative space-y-5">
                  <p className="text-[11px] font-[700] uppercase tracking-[0.14em] text-[#8A98A8]">Audit entry · Early warning query</p>
                  {[
                    { label: "Query raised by", value: "Kinara Core · Pattern engine" },
                    { label: "Domains queried", value: "Maritime, Agriculture, Logistics, Health" },
                    { label: "Data disclosed", value: "Minimum: aggregate signals only" },
                    { label: "Raw records disclosed", value: "None" },
                    { label: "Lawful basis checked", value: "Yes · Policy ref 04-EW" },
                    { label: "Alert issued to", value: "Health steward only" },
                    { label: "Audit entry", value: "Immutable · Cannot be altered" },
                  ].map((row, i) => (
                    <div key={i} className="flex items-start justify-between gap-8 border-b border-[#1A2A40] pb-4 last:border-0 last:pb-0">
                      <p className="text-[13px] text-[#5D6E82]">{row.label}</p>
                      <p className="text-[13px] font-[600] text-white text-right">{row.value}</p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ── CTA ──────────────────────────────────────────────── */}
        <section
          className="relative px-8 py-24 overflow-hidden"
          style={{ background: "radial-gradient(ellipse 60% 80% at 30% 50%, rgba(16,185,129,0.12) 0%, transparent 60%), #080F1E" }}
        >
          <div className="absolute inset-0 dot-grid pointer-events-none opacity-40" />
          <div className="relative max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-center">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#10B981] mb-6">Request a demonstration</p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  We will run the cross-domain query live.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  A 45-minute briefing. We walk through the signal chain, show the governance layer authorising the join, and display the audit log in real time. Nothing staged.
                </p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link href="/contact" className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors">
                  Request a briefing
                </Link>
                <Link href="/governance" className="inline-block border border-[#3A4F68] text-[#8A98A8] text-[15px] font-[600] px-9 py-4 hover:border-[#8A98A8] hover:text-white transition-colors">
                  Read the governance model
                </Link>
                <p className="text-[13px] text-[#3A4F68] mt-2">Kinara OS is built and owned by Klinova &nbsp;&middot;&nbsp; kinaraos.com</p>
              </div>
            </div>
          </div>
        </section>

      </main>
      <Footer />
    </>
  );
}
