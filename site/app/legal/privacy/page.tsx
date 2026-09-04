import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = {
  title: "Privacy Policy · Kinara OS",
  description: "Kinara OS privacy policy. How we handle data on kinaraos.com.",
};

export default function PrivacyPage() {
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
              Privacy Policy
            </h1>
            <div className="prose prose-sm max-w-none text-[#5D6E82] leading-relaxed space-y-6 text-[16px]">
              <p className="text-[14px] text-[#8A98A8]">Last updated: 1 September 2026</p>

              <p>
                This policy describes how Klinova Health LLC (&quot;Klinova&quot;, &quot;we&quot;, &quot;us&quot;) handles personal data collected through the kinaraos.com website.
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">What we collect</h2>
              <p>
                When you submit the contact form, we collect your name, organisation, email address, and the information you provide about your coordination problem. We use this information to respond to your briefing request and for no other purpose.
              </p>
              <p>
                We do not use tracking pixels, third-party advertising networks, or cross-site tracking on this website.
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">How we use it</h2>
              <p>
                Contact form submissions are used to respond to briefing requests. We do not sell, share, or transfer this information to third parties except as necessary to deliver our response (for example, an email service provider).
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Tenant data</h2>
              <p>
                This policy covers the kinaraos.com marketing website only. If you are a Kinara OS tenant or operator, the data governance terms in your tenancy agreement govern the handling of operational data. That agreement is separate from this policy.
              </p>

              <h2 className="text-[20px] font-[700] text-[#0F1B2D] mt-8 mb-3">Contact</h2>
              <p>
                Questions about this policy may be directed to Klinova Health LLC via the contact form on this site.
              </p>
            </div>
          </div>
        </section>
      </main>
      <Footer />
    </>
  );
}
