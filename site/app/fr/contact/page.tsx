import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "Contact · Kinara OS" };

export default function FrContactPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Contact</p>
          <h1 className="text-[64px] md:text-[80px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-3xl">
            Demander un entretien.
          </h1>
          <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl mb-12">
            45 minutes. Le système en fonctionnement, pas des diapositives. Amenez vos équipes techniques, juridiques ou stratégiques.
          </p>

          <div className="max-w-xl">
            <div className="bg-[#F5F8FB] border border-[#C3CEDA] p-8">
              <form action="https://formspree.io/f/placeholder" method="POST" className="space-y-5">
                <div>
                  <label className="block text-[13px] font-[600] text-[#0F1B2D] mb-1.5">Nom</label>
                  <input type="text" name="name" required className="w-full border border-[#C3CEDA] px-4 py-3 text-[15px] text-[#0F1B2D] bg-white focus:outline-none focus:border-[#0F1B2D] transition-colors" placeholder="Votre nom" />
                </div>
                <div>
                  <label className="block text-[13px] font-[600] text-[#0F1B2D] mb-1.5">Organisation</label>
                  <input type="text" name="organisation" required className="w-full border border-[#C3CEDA] px-4 py-3 text-[15px] text-[#0F1B2D] bg-white focus:outline-none focus:border-[#0F1B2D] transition-colors" placeholder="Ministère, fonds ou entreprise" />
                </div>
                <div>
                  <label className="block text-[13px] font-[600] text-[#0F1B2D] mb-1.5">Courriel</label>
                  <input type="email" name="email" required className="w-full border border-[#C3CEDA] px-4 py-3 text-[15px] text-[#0F1B2D] bg-white focus:outline-none focus:border-[#0F1B2D] transition-colors" placeholder="vous@organisation.gouv" />
                </div>
                <div>
                  <label className="block text-[13px] font-[600] text-[#0F1B2D] mb-1.5">Le problème de coordination que vous cherchez à résoudre</label>
                  <textarea name="problem" rows={4} className="w-full border border-[#C3CEDA] px-4 py-3 text-[15px] text-[#0F1B2D] bg-white focus:outline-none focus:border-[#0F1B2D] transition-colors resize-none" placeholder="Décrivez le problème de coordination interministérielle ou interdomaine auquel vous êtes confronté." />
                </div>
                <button type="submit" className="w-full bg-[#E0561C] text-white text-[15px] font-[700] px-7 py-4 hover:bg-[#c94d19] transition-colors">
                  Envoyer la demande
                </button>
              </form>
            </div>
            <p className="text-[13px] text-[#8A98A8] mt-4 text-center">Nous répondons sous deux jours ouvrables.</p>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
