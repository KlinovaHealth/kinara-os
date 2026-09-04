import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import StatTile from "@/components/StatTile";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "Health Domain · Kinara OS",
  description: "40 health services on Kinara OS. Patients, visits, labs, immunisation, outbreak detection. Steward: Ministry of Health.",
};

export default function HealthDomainPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Domain · Health
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Health records that stay with the Ministry, and coordinate with everything else.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              The Health domain covers patient records, clinic visits, laboratory results, immunisation data, and outbreak indicators. The Ministry of Health remains the steward.
            </p>
            <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl mt-4">
              Kinara OS is a secure, configurable healthcare operations platform that helps organizations coordinate multilingual care, clinician workflows, referrals, prescriptions, pharmacy access, payments, consent, reporting, and privacy-preserving health intelligence.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="What is covered" description="services in this domain" />
            <div className="grid md:grid-cols-2 gap-4 mb-10">
              {["Patient registration and identity", "Clinic visit records", "Laboratory results and referrals", "Immunisation schedules and coverage", "Drug stock and supply chain", "Outbreak indicators and alerts"].map((item) => (
                <div key={item} className="flex items-center gap-3 border border-[#C3CEDA] rounded-lg px-5 py-4 bg-white">
                  <span className="w-2 h-2 rounded-full bg-[#E0561C] flex-shrink-0" />
                  <p className="text-[15px] text-[#0F1B2D]">{item}</p>
                </div>
              ))}
            </div>

            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
              <StatTile figure="40" label="services in this domain" />
              <StatTile figure="36" label="tenant-isolated databases" />
              <StatTile figure="72h" label="records held at edge, offline" />
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Cross-domain joins" description="what health data enables" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-6 max-w-2xl">
              Health data is the input to logistics, agriculture, and maritime decisions.
            </h2>
            <div className="space-y-4 max-w-2xl text-[17px] text-[#5D6E82] leading-relaxed">
              <p>A stockout in a clinic is a logistics query: what is in the nearest warehouse. Kinara OS governs that join.</p>
              <p>A nutrition indicator in health data is a signal for the agriculture domain: which crop categories are under-supplied. Kinara OS governs that join.</p>
              <p>Both joins are authorised, checked against policy, and written to the audit log before they execute.</p>
            </div>
          </div>
        </section>

        <CTABand
          headline="See the Health domain in a briefing."
          sub="We will walk through the domain architecture and show a live cross-domain query from Health to Logistics."
          cta="Request a briefing"
          href="/contact"
        />
      </main>
      <Footer />
    </>
  );
}
