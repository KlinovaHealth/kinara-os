import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import StatTile from "@/components/StatTile";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "Maritime Domain · Kinara OS",
  description: "36 maritime services on Kinara OS. Vessels, berths, cargo, customs, trade finance. Steward: Port Authority.",
};

export default function MaritimeDomainPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Domain · Maritime
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Port and vessel records that connect trade to inland logistics and customs.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              The Maritime domain covers vessel scheduling, berth management, cargo records, customs clearance, and trade finance. The Port Authority remains the steward.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="What is covered" description="services in this domain" />
            <div className="grid md:grid-cols-2 gap-4 mb-10">
              {["Vessel scheduling and berth allocation", "Cargo manifest and tracking", "Customs declaration and clearance", "Trade finance and letter of credit", "Port authority inspections", "Container release authorisation"].map((item) => (
                <div key={item} className="flex items-center gap-3 border border-[#C3CEDA] rounded-lg px-5 py-4 bg-white">
                  <span className="w-2 h-2 rounded-full bg-[#E0561C] flex-shrink-0" />
                  <p className="text-[15px] text-[#0F1B2D]">{item}</p>
                </div>
              ))}
            </div>

            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
              <StatTile figure="36" label="services in this domain" />
              <StatTile figure="36" label="tenant-isolated databases" />
              <StatTile figure="0" label="manual handover steps to logistics" />
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Cross-domain joins" description="what maritime data connects to" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-6 max-w-2xl">
              A container cleared at the port is a logistics instruction.
            </h2>
            <div className="space-y-4 max-w-2xl text-[17px] text-[#5D6E82] leading-relaxed">
              <p>When customs releases a container, the logistics domain receives a movement instruction. That join is authorised by Kinara Core without a manual handover between the port authority and the transport ministry.</p>
              <p>Trade finance events in maritime can trigger supply chain records in agriculture and logistics. The same governance model applies: the join is authorised, checked against policy, and logged.</p>
            </div>
          </div>
        </section>

        <CTABand
          headline="See the Maritime domain in a briefing."
          sub="We will show the port-to-logistics join live, with the audit log."
          cta="Request a briefing"
          href="/contact"
        />
      </main>
      <Footer />
    </>
  );
}
