import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import Link from "next/link";

export const metadata = {
  title: "For Healthcare Partners · Klinova",
  description: "Clinics, pharmacies, doctors, and delivery providers join the Klinova network and receive verified patient referrals.",
};

const PARTNER_TYPES = [
  {
    type: "Clinics & hospitals",
    tag: "Referrals + dashboard",
    color: "#10B981",
    desc: "Receive pre-triaged patient referrals ready for in-person care. Digital patient dossiers arrive before they walk in the door.",
  },
  {
    type: "Pharmacies",
    tag: "Digital Rx + delivery",
    color: "#E0561C",
    desc: "Receive digital prescriptions directly from Klinova doctors. Zero paper, instant notification, faster dispensing.",
  },
  {
    type: "Doctors & nurses",
    tag: "Teleconsult income",
    color: "#3B82F6",
    desc: "Consult patients by chat, voice, or video on your own schedule. Klinova handles triage, records, and payment.",
  },
  {
    type: "Delivery & transport",
    tag: "Last-mile delivery",
    color: "#F59E0B",
    desc: "Deliver medications from partner pharmacies to patients. Join the network and receive steady, verified delivery requests.",
  },
  {
    type: "Insurance companies",
    tag: "Claims verification",
    color: "#8B5CF6",
    desc: "Receive claim verification signals against health records you do not hold. The check crosses the boundary; the record stays in the clinic.",
  },
  {
    type: "Governments & NGOs",
    tag: "Population health",
    color: "#0EA5E9",
    desc: "Coordinate public health programs across clinic networks, supply chains, and community health workers. Aggregate signals, not raw records.",
  },
];

const DASHBOARD_FEATURES = [
  {
    title: "Patient referral queue",
    desc: "See incoming Klinova referrals in real time. Accept or reschedule with one click. Patient dossier attached.",
  },
  {
    title: "Appointment management",
    desc: "Full booking calendar linked to your actual capacity. Patients see your real-time availability.",
  },
  {
    title: "Digital health records",
    desc: "Access patient history, prior consults, and prescriptions. No clipboard, no paper.",
  },
  {
    title: "Pharmacy & Rx module",
    desc: "Issue digital prescriptions that route automatically to the nearest partner pharmacy.",
  },
  {
    title: "Analytics & reporting",
    desc: "Track consultations, referrals, revenue, and patient outcomes by day, week, or month.",
  },
  {
    title: "Payments & billing",
    desc: "Klinova handles patient billing and sends your share within 48 hours of each consultation.",
  },
];

export default function PartnersPage() {
  return (
    <>
      <Nav />
      <main className="pt-[108px]">

        {/* ── Hero ─────────────────────────────────────────────── */}
        <section className="bg-white px-8 pt-20 pb-0">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-end pb-16 border-b border-[#C3CEDA]">
              <div>
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  For healthcare partners
                </p>
                <h1 className="text-[56px] md:text-[72px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.93] mb-8">
                  Join Africa&apos;s healthcare grid.
                </h1>
                <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-10 max-w-lg">
                  Clinics, pharmacies, doctors, employers, and delivery providers join the Klinova network and receive verified patient referrals.
                </p>
                <div className="flex flex-wrap gap-4">
                  <Link
                    href="/contact"
                    className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors"
                  >
                    Apply to be a partner
                  </Link>
                  <Link
                    href="/contact"
                    className="inline-block border border-[#C3CEDA] text-[#0F1B2D] text-[14px] font-[600] px-7 py-3.5 hover:border-[#0F1B2D] transition-colors"
                  >
                    Talk to our team
                  </Link>
                </div>
              </div>

              {/* Quick benefits */}
              <div className="space-y-0 border border-[#C3CEDA]">
                {[
                  { label: "No setup fee", sub: "for pilot partners" },
                  { label: "Instant patient referrals", sub: "from day one" },
                  { label: "Revenue share model", sub: "paid within 48 hours" },
                ].map((b, i) => (
                  <div
                    key={i}
                    className="flex items-center gap-5 px-6 py-5 border-b border-[#C3CEDA] last:border-0"
                  >
                    <div
                      className="flex-shrink-0 w-2 h-2 rounded-full"
                      style={{ background: "#E0561C" }}
                    />
                    <div>
                      <p className="text-[16px] font-[700] text-[#0F1B2D]">{b.label}</p>
                      <p className="text-[13px] text-[#8A98A8]">{b.sub}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        {/* ── Partner types ────────────────────────────────────── */}
        <section className="bg-[#F5F8FB] px-8 py-20 border-y border-[#C3CEDA]">
          <div className="max-w-[1440px] mx-auto">
            <div className="mb-12">
              <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-4">
                Partner types
              </p>
              <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">
                Whether you have a clinic, pharmacy, lab, or motorbike, there&apos;s a place for you.
              </h2>
            </div>

            <div className="grid md:grid-cols-3 gap-px bg-[#C3CEDA]">
              {PARTNER_TYPES.map((p) => (
                <div key={p.type} className="bg-white p-8 group card-lift cursor-default">
                  <div className="flex items-start justify-between mb-5">
                    <div
                      className="w-1 self-stretch mr-4 flex-shrink-0 rounded-full"
                      style={{ background: p.color, minHeight: "48px" }}
                    />
                    <div className="flex-1">
                      <h3 className="text-[20px] font-[800] text-[#0F1B2D] tracking-[-0.02em] mb-1">
                        {p.type}
                      </h3>
                      <span
                        className="inline-block text-[11px] font-[700] uppercase tracking-[0.12em] px-2.5 py-1 mb-4"
                        style={{ background: p.color + "18", color: p.color }}
                      >
                        {p.tag}
                      </span>
                      <p className="text-[15px] text-[#5D6E82] leading-relaxed">{p.desc}</p>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* ── Dashboard ────────────────────────────────────────── */}
        <section className="bg-white px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-16 items-start">
              <div className="md:sticky md:top-32">
                <p className="text-[11px] font-[700] uppercase tracking-[0.18em] text-[#E0561C] mb-6">
                  Clinic partner dashboard
                </p>
                <h2 className="text-[40px] md:text-[52px] font-[800] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-6">
                  Everything your clinic needs, in one place.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed mb-8">
                  Klinova&apos;s partner dashboard is built around how clinics actually work: appointments, referrals, prescriptions, and performance, all in one screen.
                </p>
                <Link
                  href="/contact"
                  className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-7 py-3.5 hover:bg-[#c94d19] transition-colors"
                >
                  Apply to be a partner
                </Link>
              </div>

              <div className="space-y-0 border border-[#C3CEDA]">
                {DASHBOARD_FEATURES.map((f, i) => (
                  <div
                    key={i}
                    className="px-7 py-6 border-b border-[#C3CEDA] last:border-0 hover:bg-[#F5F8FB] transition-colors"
                  >
                    <div className="flex items-start gap-4">
                      <span className="text-[12px] font-[800] text-[#E0561C] flex-shrink-0 mt-0.5 w-5 text-right">
                        {String(i + 1).padStart(2, "0")}
                      </span>
                      <div>
                        <p className="text-[16px] font-[700] text-[#0F1B2D] mb-1">{f.title}</p>
                        <p className="text-[14px] text-[#5D6E82] leading-relaxed">{f.desc}</p>
                      </div>
                    </div>
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
                  Ready to join
                </p>
                <h2 className="text-[44px] md:text-[56px] font-[800] tracking-[-0.04em] text-white leading-[0.95] mb-6">
                  Apply as a pilot partner. No setup fee.
                </h2>
                <p className="text-[17px] text-[#5D6E82] leading-relaxed">
                  Pilot partners pay no setup fee. You receive referrals from day one and earn a share of every consultation billed through the network.
                </p>
              </div>
              <div className="flex flex-col gap-4 items-start">
                <Link
                  href="/contact"
                  className="inline-block bg-[#E0561C] text-white text-[15px] font-[700] px-9 py-4 hover:bg-[#c94d19] transition-colors"
                >
                  Apply to be a partner
                </Link>
                <Link
                  href="/contact"
                  className="inline-block border border-[#3A4F68] text-[#8A98A8] text-[15px] font-[600] px-9 py-4 hover:border-[#8A98A8] hover:text-white transition-colors"
                >
                  Talk to our team
                </Link>
                <p className="text-[13px] text-[#3A4F68] mt-2">
                  Klinova runs on Kinara OS &nbsp;&middot;&nbsp; kinaraos.com
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
