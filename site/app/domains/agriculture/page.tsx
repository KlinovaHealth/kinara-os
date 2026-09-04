import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import StatTile from "@/components/StatTile";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "Agriculture Domain · Kinara OS",
  description: "40 agriculture services on Kinara OS. Plots, inputs, harvest, market price, subsidy. Steward: Ministry of Agriculture.",
};

export default function AgricultureDomainPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Domain · Agriculture
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Agricultural records that inform logistics, health, and trade.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              The Agriculture domain covers plot registration, input records, harvest data, market prices, and subsidy programmes. The Ministry of Agriculture remains the steward.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="What is covered" description="services in this domain" />
            <div className="grid md:grid-cols-2 gap-4 mb-10">
              {["Plot and parcel registration", "Input supply and distribution", "Harvest yield records", "Market price indices", "Subsidy eligibility and payment", "Extension officer activity logs"].map((item) => (
                <div key={item} className="flex items-center gap-3 border border-[#C3CEDA] rounded-lg px-5 py-4 bg-white">
                  <span className="w-2 h-2 rounded-full bg-[#E0561C] flex-shrink-0" />
                  <p className="text-[15px] text-[#0F1B2D]">{item}</p>
                </div>
              ))}
            </div>

            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
              <StatTile figure="40" label="services in this domain" />
              <StatTile figure="36" label="tenant-isolated databases" />
              <StatTile figure="0" label="manual export steps per report run" />
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Cross-domain joins" description="what agriculture data enables" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-6 max-w-2xl">
              A harvest record is a transport demand. A subsidy record is an input for health.
            </h2>
            <div className="space-y-4 max-w-2xl text-[17px] text-[#5D6E82] leading-relaxed">
              <p>A confirmed harvest yield is a logistics question: how much needs to move, from where, by when. Kinara OS governs that join.</p>
              <p>A nutrition indicator in health data corresponds to a crop category signal in agriculture data. That correlation can be computed under governance without either domain seeing the other&apos;s raw records.</p>
            </div>
          </div>
        </section>

        <CTABand
          headline="See the Agriculture domain in a briefing."
          sub="We will show the domain architecture and a live cross-domain query from Agriculture to Logistics."
          cta="Request a briefing"
          href="/contact"
        />
      </main>
      <Footer />
    </>
  );
}
