import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import StatTile from "@/components/StatTile";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "For Funders and Development Finance · Kinara OS",
  description: "Outcome reporting drawn from the operational record itself. Every figure is auditable back to source.",
};

export default function FundersPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Funders and development finance
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Outcome reporting drawn from the operational record itself.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              The report is a view over the same records the clinic uses to run its day. Auditable back to source, every figure.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="The problem" description="what current reporting produces" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-6 max-w-2xl">
              A spreadsheet nobody can trace back to a source.
            </h2>
            <div className="text-[18px] text-[#5D6E82] leading-relaxed space-y-4 max-w-2xl mb-10">
              <p>
                Current reporting is a chain of manual extracts. A data officer pulls figures from one system. Another checks them against a second. A third consolidates in Excel. By the time the report reaches you, the original record is three hand-offs away.
              </p>
              <p>
                Kinara OS replaces that chain. Reporting queries run against the same operational databases the system uses to function, with the same governance controls.
              </p>
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="What changes" description="the audit-ready record" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8 max-w-2xl">
              Every metric has a source record.
            </h2>
            <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4 mb-12">
              {[
                {
                  heading: "No manual extraction.",
                  body: "Reports run as governed queries over live operational data. No export step, no consolidation spreadsheet.",
                },
                {
                  heading: "Attribution to the field worker.",
                  body: "Authenticated, attributed entry by a named authorized worker, with timestamped audit logging.",
                },
                {
                  heading: "Re-audit at any time.",
                  body: "Run the same query again in six months. The underlying records are there, unchanged, with their original timestamps.",
                },
                {
                  heading: "Privacy-safe aggregate reporting.",
                  body: "Aggregate reporting is designed to prevent access to identifiable individual records, subject to approved privacy thresholds, access controls, and applicable law.",
                },
              ].map((item) => (
                <div key={item.heading} className="border border-[#C3CEDA] rounded-lg p-6">
                  <p className="text-[16px] font-[700] text-[#0F1B2D] mb-2">{item.heading}</p>
                  <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                </div>
              ))}
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <StatTile figure="152" label="services producing auditable records" />
              <StatTile figure="144" label="independent databases" />
              <StatTile figure="4" label="governed domains" />
              <StatTile figure="0" label="manual export steps in a report run" />
            </div>
          </div>
        </section>

        <CTABand
          headline="See an outcome report drawn from a live tenant."
          sub="We will show you the query, the underlying records, and the audit log for the same session."
          cta="Request a briefing"
          href="/contact"
          note="Kinara OS is built and owned by Klinova."
        />
      </main>
      <Footer />
    </>
  );
}
