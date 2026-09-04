import Image from "next/image";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import StatTile from "@/components/StatTile";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "Platform Architecture · Kinara OS",
  description: "152 Go microservices. 144 tenant-isolated databases. One governed coordination layer across health, agriculture, logistics and maritime.",
};

export default function PlatformPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Platform
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              The architecture of a governed coordination layer.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              152 Go microservices. 144 tenant-isolated databases. One Kinara Core that governs every cross-domain join.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-6xl mx-auto">
            <SectionRail label="Architecture" description="how the system is structured" />
            <div className="relative rounded-xl overflow-hidden border border-[#C3CEDA] bg-white mb-4">
              <Image
                src="/diagrams/architecture-en.png"
                alt="Kinara OS full system architecture diagram"
                width={1600}
                height={800}
                className="w-full h-auto"
              />
            </div>
            <p className="text-[13px] text-[#8A98A8] text-center mb-12">
              Records move inward under mandate. Decisions move back out under accountability.
            </p>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              <StatTile figure="152" label="services in production" />
              <StatTile figure="144" label="isolated databases, one per service" />
              <StatTile figure="4" label="governed domains" />
              <StatTile figure="72h" label="records held at edge, offline" />
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Core components" description="what each layer does" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8 max-w-xl">
              Four layers. One rule.
            </h2>
            <div className="space-y-4">
              {[
                {
                  label: "Domain services",
                  heading: "Each domain owns its own services and databases.",
                  body: "Health runs 40 services against its own 36 databases. Agriculture the same. No service in one domain can query a database in another. The separation is enforced at the connection layer, not the application layer.",
                },
                {
                  label: "Kinara Core",
                  heading: "The governance layer that permits and records every cross-domain join.",
                  body: "When a health service needs to raise a logistics question, it sends the request to Kinara Core. Core checks the lawful basis, verifies the requestor, permits or denies, and writes the audit entry. The join happens through Core, never directly between domains.",
                },
                {
                  label: "Edge runtime",
                  heading: "Records created where connectivity is worst.",
                  body: "Field workers on feature phones. Clinics without reliable data connections. The edge runtime holds records locally, queues them, and reconciles the moment connectivity returns. No record is lost. No entry requires a live connection at the moment of capture.",
                },
                {
                  label: "Kinara AI",
                  heading: "Operational intelligence without clinical authority.",
                  body: "Kinara AI supports multilingual intake, translation, care navigation, documentation, and operational insight. Licensed clinicians remain responsible for diagnosis, prescribing, and treatment.",
                },
              ].map((item) => (
                <div key={item.label} className="border border-[#C3CEDA] rounded-lg p-6">
                  <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#E0561C] mb-2">
                    {item.label}
                  </p>
                  <p className="text-[18px] font-[700] text-[#0F1B2D] mb-2 leading-snug">
                    {item.heading}
                  </p>
                  <p className="text-[15px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Technical choices" description="why we built it this way" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8 max-w-xl">
              Go. PostgreSQL. No shared databases.
            </h2>
            <div className="grid md:grid-cols-2 gap-4">
              {[
                { heading: "Go microservices", body: "Low memory, fast startup, suitable for constrained environments. Each service is independently deployable and independently failure-isolated." },
                { heading: "PostgreSQL per tenant", body: "Every tenant gets its own database. Row-level separation is not sufficient for data that different ministries must never see together." },
                { heading: "WhatsApp and SMS inputs", body: "The entry point is the device the worker already has. Not a custom app, not a tablet programme, not a funded device rollout." },
                { heading: "Audit log as first-class output", body: "The log is not a side-effect. It is a primary output of every cross-domain operation. It cannot be disabled, altered, or selectively written." },
              ].map((item) => (
                <div key={item.heading} className="border border-[#C3CEDA] rounded-lg p-6">
                  <p className="text-[16px] font-[700] text-[#0F1B2D] mb-2">{item.heading}</p>
                  <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <CTABand
          headline="See the running system."
          sub="A 45-minute technical briefing. We will walk through the architecture live, not on slides."
          cta="Request a briefing"
          href="/contact"
          note="Kinara OS is built and owned by Klinova."
        />
      </main>
      <Footer />
    </>
  );
}
