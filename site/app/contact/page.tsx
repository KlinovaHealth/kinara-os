import Nav from "@/components/Nav";
import Footer from "@/components/Footer";
import SectionRail from "@/components/SectionRail";

export const metadata = {
  title: "Contact · Kinara OS",
  description: "Request a briefing on Kinara OS. A 45-minute session with the running system.",
};

export default function ContactPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">

        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-4xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Contact
            </p>
            <h1 className="text-[52px] md:text-[64px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-[1.05] mb-8 max-w-3xl">
              Request a briefing.
            </h1>
            <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl mb-12">
              45 minutes. The running system, not slides. Bring your technical, legal, or strategic teams.
            </p>

            <div className="max-w-xl">
              <div className="bg-[#F5F8FB] border border-[#C3CEDA] rounded-xl p-8">
                <SectionRail label="Briefing request" description="tell us what you are working on" />

                <form
                  action="https://formspree.io/f/placeholder"
                  method="POST"
                  className="space-y-5 mt-6"
                >
                  <div>
                    <label className="block text-[13px] font-[600] text-[#0F1B2D] mb-1.5">
                      Name
                    </label>
                    <input
                      type="text"
                      name="name"
                      required
                      className="w-full border border-[#C3CEDA] rounded-md px-4 py-3 text-[15px] text-[#0F1B2D] bg-white focus:outline-none focus:border-[#0F1B2D] transition-colors"
                      placeholder="Your name"
                    />
                  </div>

                  <div>
                    <label className="block text-[13px] font-[600] text-[#0F1B2D] mb-1.5">
                      Organisation
                    </label>
                    <input
                      type="text"
                      name="organisation"
                      required
                      className="w-full border border-[#C3CEDA] rounded-md px-4 py-3 text-[15px] text-[#0F1B2D] bg-white focus:outline-none focus:border-[#0F1B2D] transition-colors"
                      placeholder="Ministry, fund, or company"
                    />
                  </div>

                  <div>
                    <label className="block text-[13px] font-[600] text-[#0F1B2D] mb-1.5">
                      Email
                    </label>
                    <input
                      type="email"
                      name="email"
                      required
                      className="w-full border border-[#C3CEDA] rounded-md px-4 py-3 text-[15px] text-[#0F1B2D] bg-white focus:outline-none focus:border-[#0F1B2D] transition-colors"
                      placeholder="you@organisation.gov"
                    />
                  </div>

                  <div>
                    <label className="block text-[13px] font-[600] text-[#0F1B2D] mb-1.5">
                      The coordination problem you are trying to solve
                    </label>
                    <textarea
                      name="problem"
                      rows={4}
                      className="w-full border border-[#C3CEDA] rounded-md px-4 py-3 text-[15px] text-[#0F1B2D] bg-white focus:outline-none focus:border-[#0F1B2D] transition-colors resize-none"
                      placeholder="Describe the cross-ministry or cross-domain coordination problem you are facing."
                    />
                  </div>

                  <button
                    type="submit"
                    className="w-full bg-[#E0561C] text-white text-[15px] font-[700] px-7 py-4 rounded-md hover:bg-[#c94d19] transition-colors"
                  >
                    Send request
                  </button>
                </form>
              </div>

              <p className="text-[13px] text-[#8A98A8] mt-4 text-center">
                We respond within two business days.
              </p>
            </div>
          </div>
        </section>
      </main>
      <Footer />
    </>
  );
}
