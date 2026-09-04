import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "Pour les bailleurs · Kinara OS" };

export default function FrFundersPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Bailleurs de fonds et finance de développement</p>
          <h1 className="text-[64px] md:text-[80px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-4xl">
            Des rapports de résultats tirés de l&apos;enregistrement opérationnel lui-même.
          </h1>
          <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
            Le rapport est une vue sur les mêmes enregistrements que la clinique utilise pour fonctionner au quotidien. Traçable jusqu&apos;à la source, chaque chiffre.
          </p>
        </section>

        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-2xl">Chaque indicateur a un enregistrement source.</h2>
            <div className="grid md:grid-cols-2 lg:grid-cols-4 gap-4 mb-12">
              {[
                { heading: "Aucune extraction manuelle.", body: "Les rapports s'exécutent en tant que requêtes gouvernées sur des données opérationnelles en direct. Pas d'étape d'export, pas de tableur de consolidation." },
                { heading: "Attribution à l'agent de terrain.", body: "Entrée authentifiée et attribuée par un agent autorisé nommé, avec journalisation d'audit horodatée." },
                { heading: "Ré-audit à tout moment.", body: "Réexécutez la même requête dans six mois. Les enregistrements sous-jacents sont là, inchangés, avec leurs horodatages d'origine." },
                { heading: "Rapports agrégés respectueux de la vie privée.", body: "Le reporting agrégé est conçu pour empêcher l'accès aux enregistrements individuels identifiables, sous réserve des seuils de confidentialité approuvés, des contrôles d'accès et du droit applicable." },
              ].map((item) => (
                <div key={item.heading} className="border border-[#C3CEDA] rounded-lg p-6 bg-white">
                  <p className="text-[16px] font-[700] text-[#0F1B2D] mb-2">{item.heading}</p>
                  <p className="text-[14px] text-[#5D6E82] leading-relaxed">{item.body}</p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="max-w-[1440px] mx-auto px-8 py-20">
          <div className="grid md:grid-cols-2 gap-16 items-end">
            <h2 className="text-[52px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">Voir un rapport de résultats tiré d&apos;un locataire actif.</h2>
            <div>
              <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-8">Nous vous montrerons la requête, les enregistrements sous-jacents et le journal d&apos;audit pour la même session.</p>
              <a href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-8 py-4 hover:bg-[#c94d19] transition-colors">Demander un entretien</a>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
