import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "For Commercial Partners · Kinara OS",
  description: "Metered, scoped, auditable access to demand signals and claim verification. Kinara OS sells the verification, never the record.",
};

export default function PartnersPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Commercial partners
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Metered, scoped, auditable access to demand signals and claim verification.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              Kinara OS sells the verification, never the record. Partners never receive identifiable individual data.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="The access model" description="what partners actually get" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8 max-w-2xl">
              The verification, not the record.
            </h2>
            <div className="grid md:grid-cols-2 gap-4">
              {[
                {
                  heading: "Demand signals, not patient data.",
                  body: "A pharmaceutical partner can see that a drug category is under-stocked in a region. They cannot see which patients need it.",
                },
                {
                  heading: "Claim verification, not the claim.",
                  body: "An insurer can verify that a visit occurred and a procedure was carried out. They receive the verification, not the clinical record.",
                },
                {
                  heading: "Metered and scoped.",
                  body: "Every API call is metered, attributed, and scoped to the minimum necessary data. Partners cannot broaden their own access.",
                },
                {
                  heading: "Auditable from both sides.",
                  body: "The tenant that produced the data can see every query run against it. The audit log is not editable by either party.",
                },
              ].map((item) => (
                <div key={item.heading} className="border border-[#C3CEDA] rounded-lg p-6">
                  <p className="text-[16px] font-[700] text-[#0F1B2D] mb-2">{item.heading}</p>
                  <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Permitted use cases" description="where the model works" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8 max-w-xl">
              What this enables.
            </h2>
            <ul className="space-y-4 max-w-2xl">
              {[
                "Pharmaceutical demand forecasting by drug category and region",
                "Agricultural input supply chain optimisation against confirmed harvest data",
                "Insurance claim verification against health and logistics records",
                "Development finance portfolio monitoring against operational outcomes",
              ].map((item) => (
                <li key={item} className="flex items-start gap-3">
                  <span className="mt-1 flex-shrink-0 w-5 h-5 rounded-full bg-[#E0561C] flex items-center justify-center">
                    <svg width="10" height="8" viewBox="0 0 10 8" fill="none">
                      <path d="M1 4l2.5 2.5L9 1" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
                    </svg>
                  </span>
                  <p className="text-[17px] text-[#0F1B2D] leading-snug">{item}</p>
                </li>
              ))}
            </ul>
          </div>
        </section>

        <CTABand
          headline="Discuss a specific access scope."
          sub="Bring us the verification or signal you need. We will tell you whether it is permitted and at what cost."
          cta="Request a briefing"
          href="/contact"
          note="Kinara OS is built and owned by Klinova."
        />
      </main>
      <Footer />
    </>
  );
}
