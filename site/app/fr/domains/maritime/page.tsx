import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "Domaine Maritime · Kinara OS" };

export default function FrMaritimePage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Domaine · Maritime</p>
          <h1 className="text-[64px] md:text-[80px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-4xl">
            Des enregistrements portuaires et navals qui relient le commerce à la logistique intérieure et aux douanes.
          </h1>
          <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">Le domaine Maritime couvre la planification des navires, la gestion des postes à quai, les enregistrements de cargaisons, le dédouanement et le financement du commerce. L&apos;Autorité portuaire reste le dépositaire.</p>
        </section>
        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-4 mb-10">
              {["Planification des navires et allocation des postes à quai", "Manifeste et suivi des cargaisons", "Déclaration en douane et dédouanement", "Finance du commerce et lettre de crédit", "Inspections de l'autorité portuaire", "Autorisation de mainlevée des conteneurs"].map((item) => (
                <div key={item} className="flex items-center gap-3 border border-[#C3CEDA] rounded-lg px-5 py-4 bg-white" style={{ borderLeft: "4px solid #0891B2" }}>
                  <p className="text-[15px] text-[#0F1B2D]">{item}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
        <section className="max-w-[1440px] mx-auto px-8 py-20">
          <div className="grid md:grid-cols-2 gap-16 items-end">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">Voir le domaine Maritime lors d&apos;un entretien.</h2>
            <div>
              <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-8">Nous vous montrerons la jointure port-logistique en direct, avec le journal d&apos;audit.</p>
              <a href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-8 py-4 hover:bg-[#c94d19] transition-colors">Demander un entretien</a>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
