import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "Domaine Santé · Kinara OS" };

export default function FrHealthPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Domaine · Santé</p>
          <h1 className="text-[64px] md:text-[80px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-4xl">
            Des dossiers de santé qui restent au Ministère, et coordonnent avec tout le reste.
          </h1>
          <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">Le domaine Santé couvre les dossiers patients, les visites en clinique, les résultats de laboratoire, les données de vaccination et les indicateurs épidémiques. Le Ministère de la Santé reste le dépositaire.</p>
          <p className="text-[18px] text-[#5D6E82] leading-relaxed max-w-2xl mt-4">Kinara OS est une plateforme d&apos;opérations de santé sécurisée et configurable qui aide les organisations à coordonner les soins multilingues, les flux de travail cliniques, les orientations, les prescriptions, l&apos;accès aux pharmacies, les paiements, le consentement, le reporting et l&apos;intelligence sanitaire préservant la vie privée.</p>
        </section>
        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <div className="grid md:grid-cols-2 gap-4 mb-10">
              {["Enregistrement et identité des patients", "Dossiers de visites en clinique", "Résultats de laboratoire et références", "Calendriers et couverture vaccinale", "Stock et chaîne d'approvisionnement en médicaments", "Indicateurs et alertes épidémiques"].map((item) => (
                <div key={item} className="flex items-center gap-3 border border-[#C3CEDA] rounded-lg px-5 py-4 bg-white" style={{ borderLeft: "4px solid #059669" }}>
                  <p className="text-[15px] text-[#0F1B2D]">{item}</p>
                </div>
              ))}
            </div>
            <div className="grid grid-cols-3 border border-[#C3CEDA]">
              {[{ n: "40", l: "services dans ce domaine" }, { n: "36", l: "bases de données isolées" }, { n: "72h", l: "enregistrements en périphérie, hors ligne" }].map((s, i) => (
                <div key={s.n} className={`p-8 bg-white ${i < 2 ? "border-r border-[#C3CEDA]" : ""}`}>
                  <p className="text-[48px] font-[700] leading-none mb-2 text-[#E0561C]">{s.n}</p>
                  <p className="text-[14px] text-[#5D6E82]">{s.l}</p>
                </div>
              ))}
            </div>
          </div>
        </section>
        <section className="max-w-[1440px] mx-auto px-8 py-20">
          <div className="grid md:grid-cols-2 gap-16 items-end">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">Voir le domaine Santé lors d&apos;un entretien.</h2>
            <div>
              <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-8">Nous vous montrerons l&apos;architecture du domaine et une requête en direct de la Santé vers la Logistique.</p>
              <a href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-8 py-4 hover:bg-[#c94d19] transition-colors">Demander un entretien</a>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
