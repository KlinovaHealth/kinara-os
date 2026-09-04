import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import StatTile from "@/components/StatTile";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "Logistics Domain · Kinara OS",
  description: "36 logistics services on Kinara OS. Routing, delivery, damage, returns, forecasting. Steward: Ministry of Transport.",
};

export default function LogisticsDomainPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Domain · Logistics
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Movement records that respond to health, agriculture, and trade demands.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              The Logistics domain covers routing, warehousing, delivery tracking, damage and returns, and demand forecasting. The Ministry of Transport remains the steward.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="What is covered" description="services in this domain" />
            <div className="grid md:grid-cols-2 gap-4 mb-10">
              {["Route planning and vehicle dispatch", "Warehouse inventory and location", "Delivery confirmation and tracking", "Damage and spoilage recording", "Returns processing", "Demand forecasting"].map((item) => (
                <div key={item} className="flex items-center gap-3 border border-[#C3CEDA] rounded-lg px-5 py-4 bg-white">
                  <span className="w-2 h-2 rounded-full bg-[#E0561C] flex-shrink-0" />
                  <p className="text-[15px] text-[#0F1B2D]">{item}</p>
                </div>
              ))}
            </div>

            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
              <StatTile figure="36" label="services in this domain" />
              <StatTile figure="36" label="tenant-isolated databases" />
              <StatTile figure="4" label="demand signals from other domains" />
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Cross-domain joins" description="what logistics data responds to" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-6 max-w-2xl">
              The answer to a health stockout is a logistics dispatch.
            </h2>
            <div className="space-y-4 max-w-2xl text-[17px] text-[#5D6E82] leading-relaxed">
              <p>When a health stockout is confirmed, the logistics domain is queried: what is available in the nearest warehouse, and what vehicle can reach the clinic. That join is authorised by Kinara Core and logged before it executes.</p>
              <p>When a harvest yield is confirmed in agriculture, logistics receives a demand signal: how much needs to move, from which district, by which deadline.</p>
            </div>
          </div>
        </section>

        <CTABand
          headline="See the Logistics domain in a briefing."
          sub="We will show the full flow from a health stockout to a logistics dispatch. Live, not slides."
          cta="Request a briefing"
          href="/contact"
        />
      </main>
      <Footer />
    </>
  );
}
