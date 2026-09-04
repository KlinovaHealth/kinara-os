import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = {
  title: "Terms of Use · Kinara OS",
  description: "Terms of use for kinaraos.com.",
};

export default function TermsPage() {
  return (
    <>
      <Nav />
      <main className="pt-16">
        <section className="bg-white px-6 pt-24 pb-20">
          <div className="max-w-2xl mx-auto">
            <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">
              Legal
            </p>
            <h1 className="text-[40px] font-[800] tracking-[-0.03em] text-[#0F1B2D] leading-tight mb-8">
              Terms of Use
            </h1>
            <div className="prose prose-sm max-w-none text-[#5D6E82] leading-relaxed space-y-6 text-[16px]">
              <p className="text-[14px] text-[#8A98A8]">Last updated: 1 September 2026</p>

              <p>
                These terms govern use of the kinaraos.com website, operated by Klinova Health LLC. By accessing this site you accept these terms.
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Use of the site</h2>
              <p>
                This site provides information about the Kinara OS platform. You may read and share content for informational purposes. You may not reproduce, distribute, or create derivative works from the content without prior written permission from Klinova.
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">No warranty</h2>
              <p>
                The content on this site is provided for informational purposes only. Klinova makes no representations or warranties of any kind, express or implied, regarding the accuracy or completeness of any content.
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Tenancy agreements</h2>
              <p>
                Use of the Kinara OS platform as a tenant or operator is governed by a separate tenancy agreement. These website terms do not apply to platform use.
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Governing law</h2>
              <p>
                These terms are governed by the laws of the Commonwealth of Virginia, United States of America.
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Contact</h2>
              <p>
                Questions about these terms may be directed to Klinova Health LLC via the contact form on this site.
              </p>
            </div>
          </div>
        </section>
      </main>
      <Footer />
    </>
  );
}
