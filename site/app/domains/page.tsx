import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import DomainCard from "@/components/DomainCard";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "Domains · Kinara OS",
  description: "Four governed domains: Health, Agriculture, Logistics, Maritime. 152 services, 144 databases, one coordination layer.",
};

const DOMAINS = [
  {
    name: "Health",
    services: 40,
    steward: "Ministry of Health",
    description: "Patients, visits, labs, immunisation, outbreak detection.",
    href: "/domains/health",
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M8 2h8v6h6v8h-6v6H8v-6H2v-8h6z"/>
      </svg>
    ),
  },
  {
    name: "Agriculture",
    services: 40,
    steward: "Ministry of Agriculture",
    description: "Plots, inputs, harvest, market price, subsidy.",
    href: "/domains/agriculture",
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M17 8C8 10 5.9 16.17 3.82 22"/>
        <path d="M21 3C17 3 9 5 5 12c3-5 8-7 16-9z"/>
      </svg>
    ),
  },
  {
    name: "Logistics",
    services: 36,
    steward: "Ministry of Transport",
    description: "Routing, delivery, damage, returns, forecasting.",
    href: "/domains/logistics",
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <rect x="1" y="3" width="15" height="13"/>
        <polygon points="16 8 20 8 23 11 23 16 16 16 16 8"/>
        <circle cx="5.5" cy="18.5" r="2.5"/>
        <circle cx="18.5" cy="18.5" r="2.5"/>
      </svg>
    ),
  },
  {
    name: "Maritime",
    services: 36,
    steward: "Port Authority",
    description: "Vessels, berths, cargo, customs, trade finance.",
    href: "/domains/maritime",
    icon: (
      <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <circle cx="12" cy="5" r="3"/>
        <line x1="12" y1="22" x2="12" y2="8"/>
        <path d="M5 12H2a10 10 0 0 0 20 0h-3"/>
      </svg>
    ),
  },
];

export default function DomainsPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-6xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Domains
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Four domains. 152 services. One coordination layer.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl mb-16">
              Each domain is sovereign. Each steward retains full custody of its own records. Kinara OS governs what crosses between them.
            </p>

            <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4">
              {DOMAINS.map((d) => (
                <DomainCard key={d.name} {...d} />
              ))}
            </div>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Cross-domain joins" description="how the domains interact" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8 max-w-2xl">
              The joins that matter cross domain lines.
            </h2>
            <div className="grid md:grid-cols-2 gap-4">
              {[
                { from: "Health", to: "Logistics", example: "A stockout in a clinic triggers a logistics query. What is in the nearest warehouse? Is a vehicle available?" },
                { from: "Agriculture", to: "Logistics", example: "A harvest record triggers a transport demand. How much produce needs to move, from where, by when?" },
                { from: "Health", to: "Agriculture", example: "A nutrition indicator in health data correlates with a crop failure signal in agriculture data." },
                { from: "Maritime", to: "Logistics", example: "A container clears the port. Customs releases it. A logistics instruction is raised without a manual handover." },
              ].map((item) => (
                <div key={item.from + item.to} className="bg-white border border-[#C3CEDA] rounded-lg p-6">
                  <p className="text-[12px] font-[800] uppercase tracking-[0.1em] text-[#E0561C] mb-3">
                    {item.from} &rarr; {item.to}
                  </p>
                  <p className="text-[15px] text-[#5D6E82] leading-relaxed">{item.example}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <CTABand
          headline="See the domains in a briefing."
          sub="We will show you the domain architecture and a live cross-domain query."
          cta="Request a briefing"
          href="/contact"
          note="Kinara OS is built and owned by Klinova."
        />
      </main>
      <Footer />
    </>
  );
}
