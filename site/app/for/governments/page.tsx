import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = {
  title: "Who Kinara OS is for · Kinara OS",
  description: "Kinara OS connects clinics, insurers, logistics operators, agribusinesses, port authorities, and public bodies into a single coordinated system without any of them surrendering the data they own.",
};

const OPERATORS = [
  {
    type: "Clinics & hospital networks",
    color: "#10B981",
    desc: "Share referral signals, patient routing, and stock levels across facilities without centralising records. Each clinic stays the owner of its own data.",
  },
  {
    type: "Insurance providers",
    color: "#E0561C",
    desc: "Receive verified claim signals against health records you do not hold. The verification crosses the boundary; the record stays with the clinic.",
  },
  {
    type: "Logistics & transport operators",
    color: "#3B82F6",
    desc: "Receive demand signals from health and agriculture systems. Route vehicles against confirmed need, not estimates. Every dispatch is logged.",
  },
  {
    type: "Agribusinesses & cooperatives",
    color: "#F59E0B",
    desc: "Connect confirmed harvest data to market routing and input supply. The record stays with the farmer; the signal reaches the buyer.",
  },
  {
    type: "Port authorities & maritime operators",
    color: "#0EA5E9",
    desc: "Coordinate cargo release with inland logistics in real time. Two operators, one signal, no shared records, full audit trail.",
  },
  {
    type: "Public bodies & regulators",
    color: "#8B5CF6",
    desc: "Coordinate across agencies without building a central repository. Each body retains custody. Every cross-boundary query is attributed and logged.",
  },
];

const HOW_IT_WORKS = [
  {
    heading: "Each operator keeps its own systems.",
    body: "Health records stay in health systems. Logistics data stays in logistics systems. Kinara OS does not move or copy them.",
  },
  {
    heading: "Queries cross boundaries under policy.",
    body: "When a cross-system question is permitted, Kinara Core checks the policy, executes the join with minimum disclosure, and writes the audit entry.",
  },
  {
    heading: "Every action is attributed.",
    body: "Who asked, what they asked, what was returned. All of it is in the audit log. Neither party can alter it.",
  },
  {
    heading: "Works from the edge.",
    body: "Field workers on feature phones, offline for 72 hours. Records queue locally and reconcile when connectivity returns.",
  },
];

export default function GovernmentsPage() {
  return (
    <>
      <Nav />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="bg-white px-8 pt-20 pb-16 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
              Who it is for
            </p>
            <h1 className="text-[56px] md:text-[80px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.93] mb-8 max-w-4xl">
              Built for any operator whose data needs to cross a boundary.
            </h1>
            <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl mb-10">
              Kinara OS connects clinics, insurers, logistics companies, agribusinesses, port operators, and public bodies into one coordinated system. No operator surrenders the data it owns. No central repository is created.
            </p>
            <div className="flex flex-wrap gap-4">
              <Link
                href="/contact"
                className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors"
              >
                Request a briefing
              </Link>
              <Link
                href="/platform"
                className="inline-block border border-[#C3CEDA] text-[#0F1B2D] text-[14px] font-[600] px-7 py-3.5 hover:border-[#0F1B2D] transition-colors"
              >
                See the architecture
              </Link>
            </div>
          </div>
        </section>

        {/* ── The real problem ─────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  The problem
                </p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8">
                  The data exists. The connection does not.
                </h2>
                <div className="text-[17px] text-[#5D6E82] leading-relaxed space-y-4">
                  <p>
                    A health stockout is also a logistics failure. A cargo release at the port is also a routing signal for inland transport. A confirmed harvest is also a demand forecast for agri-input suppliers.
                  </p>
                  <p>
                    Every one of these problems is solvable with data that already exists inside separate organisations. The problem is crossing the boundary legally, quickly, and with a record of it having happened.
                  </p>
                  <p>
                    This applies equally to a private clinic network, an insurance company, a logistics operator, and a port authority. The coordination failure is not a government problem. It is an infrastructure problem.
                  </p>
                </div>
              </div>

              {/* Pull quote */}
              <div
                className="relative p-10"
                style={{ background: "#080F1E" }}
              >
                <div className="absolute inset-0 dot-grid pointer-events-none opacity-50" />
                <div className="relative">
                  <div className="w-8 h-px bg-[#E0561C] mb-8" />
                  <p className="text-[28px] font-[800] text-white leading-snug tracking-[-0.03em] mb-8">
                    Africa&apos;s grids are not disconnected because the data does not exist. They are disconnected because there is no governed way to let it cross an organisational boundary.
                  </p>
                  <p className="text-[14px] font-[600] text-[#E0561C] uppercase tracking-[0.12em]">
                    Kinara OS is the crossing.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* ── Operator types ───────────────────────────────────── */}
        <section className="bg-white px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-12">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">
                Operator types
              </p>
              <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] max-w-2xl">
                Private, public, or both. The stack is the same.
              </h2>
            </div>
            <div className="grid md:grid-cols-3 gap-px bg-[#C3CEDA]">
              {OPERATORS.map((op) => (
                <div key={op.type} className="bg-white p-8 group card-lift cursor-default">
                  <div
                    className="w-8 h-1 mb-5 rounded-full"
                    style={{ background: op.color }}
                  />
                  <h3 className="text-[18px] font-[800] text-[#0F1B2D] tracking-[-0.02em] mb-3">
                    {op.type}
                  </h3>
                  <p className="text-[14px] text-[#5D6E82] leading-relaxed">{op.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── How it works ─────────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-b border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  How it works
                </p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-6">
                  A governed join layer, not a central repository.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-8">
                  No operator gives up custody of what it owns. What changes is that a verified question can now cross a boundary, under policy, with an immutable record of it having happened.
                </p>
                <Link
                  href="/governance"
                  className="text-[14px] font-[700] text-[#E0561C] hover:text-[#c94d19] transition-colors"
                >
                  Read the governance model &rarr;
                </Link>
              </div>
              <div className="space-y-0 border border-[#C3CEDA]">
                {HOW_IT_WORKS.map((item, i) => (
                  <div key={i} className="bg-white px-7 py-6 border-b border-[#C3CEDA] last:border-0">
                    <p className="text-[16px] font-[700] text-[#0F1B2D] mb-1.5">{item.heading}</p>
                    <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                  </div>
                ))}
              </div>
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
                  Start a conversation
                </p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  Bring us the problem that crosses two organisations.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  A briefing is 45 minutes. We will show you the running system, not slides. Public or private, the conversation is the same.
                </p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link
                  href="/contact"
                  className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors"
                >
                  Request a briefing
                </Link>
                <Link
                  href="/evidence"
                  className="inline-block border border-[#3A4F68] text-[#8A98A8] text-[15px] font-[600] px-9 py-4 hover:border-[#8A98A8] hover:text-white transition-colors"
                >
                  See who is already on it
                </Link>
              </div>
            </div>
          </div>
        </section>

      </main>
      <Footer />
    </>
  );
}
