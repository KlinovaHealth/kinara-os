import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";
import GuaranteeList from "@/components/GuaranteeList";
import CTABand from "@/components/CTABand";

export const metadata = {
  title: "Governance Model · Kinara OS",
  description: "How Kinara OS governs every cross-domain data join. Policy-checked, audit-logged, jurisdiction-bound.",
};

const GUARANTEES = [
  "No single ministry holds all four datasets.",
  "Every cross-domain join is authorised and recorded.",
  "Tenants are separated at the database, not the query.",
  "Kinara OS is designed to support jurisdiction-aware data residency and controlled cross-border processing in accordance with applicable law and customer policy.",
];

export default function GovernancePage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Governance
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Every cross-domain join is authorised. Every authorisation is recorded.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
              Kinara OS does not assume trust between ministries. It enforces the conditions under which data may cross a boundary, logs it, and attributes it.
            </p>
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="The model" description="how governance works in practice" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8 max-w-2xl">
              Policy first. Join second. Log always.
            </h2>
            <div className="space-y-4">
              {[
                {
                  step: "01",
                  heading: "A domain service raises a cross-domain request.",
                  body: "The requesting service sends the question to Kinara Core, not directly to the other domain. It includes the requestor identity, the specific query, and the stated lawful basis.",
                },
                {
                  step: "02",
                  heading: "Core checks the policy.",
                  body: "The governance policy for that requestor, that data type, and that purpose is checked. If no policy permits the join, the request is denied. Denial is also logged.",
                },
                {
                  step: "03",
                  heading: "The join is executed and the result returned.",
                  body: "Core executes the join with the minimum data necessary to answer the question. The responding domain returns the answer to Core, not to the requester directly.",
                },
                {
                  step: "04",
                  heading: "The audit entry is written.",
                  body: "Before the response is returned to the requester, the audit entry is written. It records what was asked, what was returned, who asked, and when. This entry cannot be modified by either party.",
                },
              ].map((item) => (
                <div key={item.step} className="flex gap-5 border border-[#C3CEDA] rounded-lg p-6">
                  <div className="flex-shrink-0 text-[13px] font-[800] text-[#E0561C]">{item.step}</div>
                  <div>
                    <p className="text-[16px] font-[700] text-[#0F1B2D] mb-2">{item.heading}</p>
                    <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="bg-white px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Commitments" description="what is contractually guaranteed" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-10 max-w-xl">
              Four guarantees, in writing.
            </h2>
            <GuaranteeList items={GUARANTEES} />
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-6 py-20">
          <div className="max-w-4xl mx-auto">
            <SectionRail label="Tenant separation" description="why it is harder than row-level security" />
            <h2 className="text-[36px] md:text-[44px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-6 max-w-2xl">
              Separated at the database, not the query.
            </h2>
            <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl">
              Row-level security on a shared database means that the right SQL query, the wrong join, or a query planner leak can expose data that should not be visible. Kinara OS separates tenants at the database level. There is no query that can cross the boundary, because the databases are not shared.
            </p>
          </div>
        </section>

        <CTABand
          headline="Review the governance model in a briefing."
          sub="Bring your legal, data protection, or technical teams. We will walk through the model and answer specific questions."
          cta="Request a briefing"
          href="/contact"
          note="Kinara OS is built and owned by Klinova."
        />
      </main>
      <Footer />
    </>
  );
}
