import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "About · Kinara OS",
  description: "Kinara OS is built and owned by Klinova. A coordination layer for African public institutions.",
};

export default function AboutPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              About
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Built to solve the coordination problem. Not to replace the ministries that have it.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              Kinara OS is built and owned by Klinova. We work with African public institutions to build coordination infrastructure that respects and reinforces institutional sovereignty.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Our position" description="what we are and are not" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8 max-w-2xl">
              We do not want your data. We want to govern the join.
            </h2>
            <div className="text-[18px] text-[#5D6E82] leading-relaxed space-y-4 max-w-2xl">
              <p>
                The standard approach to public sector data coordination asks institutions to centralise their data into a shared platform. That approach requires institutions to give up custody, and it fails because they will not.
              </p>
              <p>
                Our position is different. Each institution keeps its own systems and its own data. What we operate is the layer that governs when and how a question may cross between them.
              </p>
              <p>
                That is not a compromise. It is the only model that works in a multi-ministry, multi-jurisdiction environment where sovereignty over records is non-negotiable.
              </p>
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Klinova" description="who builds and owns Kinara OS" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-6 max-w-xl">
              Klinova builds and operates Kinara OS.
            </h2>
            <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl">
              Klinova is the entity that builds, maintains, and operates the Kinara OS platform. All tenants contract with Klinova. All data governance commitments are made by Klinova. The system is designed so that Klinova itself cannot read any tenant&apos;s operational data without a logged, authorised query.
            </p>
          </div>
        </section>

        <CTABand
          headline="Talk to us."
          sub="Bring us the coordination problem. We will tell you directly whether and how Kinara OS addresses it."
          cta="Request a briefing"
          href="/contact"
          note="Kinara OS is built and owned by Klinova."
        />
      </main>
      <Footer />
    </>
  );
}
