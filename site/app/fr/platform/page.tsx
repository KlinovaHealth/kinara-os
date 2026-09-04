import Image from "next/image";
import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "Architecture de la plateforme · Kinara OS" };

export default function FrPlatformPage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Plateforme</p>
          <h1 className="text-[64px] md:text-[80px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-4xl">
            L&apos;architecture d&apos;une couche de coordination gouvernée.
          </h1>
          <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl mb-16">
            152 microservices Go. 144 bases de données isolées par locataire. Un Kinara Core qui gouverne chaque jointure interdomaine.
          </p>

          <div className="relative rounded-xl overflow-hidden border border-[#C3CEDA] bg-[#F5F8FB] mb-4">
            <Image src="/diagrams/architecture-fr.png" alt="Architecture système Kinara OS" width={1600} height={800} className="w-full h-auto" />
          </div>
          <p className="text-[13px] text-[#8A98A8] text-center mb-16">
            Les enregistrements entrent sous mandat. Les décisions sortent sous responsabilité.
          </p>

          <div className="grid grid-cols-2 md:grid-cols-4 border border-[#C3CEDA]">
            {[
              { n: "152", l: "services en production" },
              { n: "144", l: "bases de données isolées, une par service" },
              { n: "4",   l: "domaines gouvernés" },
              { n: "72h", l: "données en périphérie, hors ligne" },
            ].map((s, i) => (
              <div key={s.n} className={`p-8 ${i < 3 ? "border-r border-[#C3CEDA]" : ""}`}>
                <p className="text-[48px] font-[700] tracking-[-0.04em] leading-none mb-2 text-[#E0561C]">{s.n}</p>
                <p className="text-[14px] text-[#5D6E82]">{s.l}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-10 max-w-2xl">Quatre couches. Une règle.</h2>
            <div className="space-y-4">
              {[
                { label: "Services de domaine", heading: "Chaque domaine possède ses propres services et bases de données.", body: "La santé exploite 40 services sur ses propres bases. L&apos;agriculture de même. Aucun service d&apos;un domaine ne peut interroger une base de données d&apos;un autre domaine." },
                { label: "Kinara Core", heading: "La couche de gouvernance qui autorise et enregistre chaque jointure interdomaine.", body: "Lorsqu&apos;un service de santé doit adresser une question logistique, il envoie la requête à Kinara Core. Core vérifie la base légale, autorise ou refuse, et consigne l&apos;entrée d&apos;audit." },
                { label: "Runtime en périphérie", heading: "Les enregistrements créés là où la connectivité est la plus mauvaise.", body: "Des travailleurs de terrain sur des téléphones basiques. Des cliniques sans connexion fiable. Le runtime en périphérie conserve les enregistrements localement, les met en file d&apos;attente et les synchronise au retour de la connectivité." },
                { label: "Kinara IA", heading: "Intelligence opérationnelle sans autorité clinique.", body: "Kinara IA prend en charge l&apos;accueil multilingue, la traduction, la navigation dans les soins, la documentation et les informations opérationnelles. Les cliniciens agréés demeurent responsables du diagnostic, de la prescription et du traitement." },
              ].map((item) => (
                <div key={item.label} className="border border-[#C3CEDA] rounded-lg p-6 bg-white">
                  <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#E0561C] mb-2">{item.label}</p>
                  <p className="text-[18px] font-[700] text-[#0F1B2D] mb-2 leading-snug" dangerouslySetInnerHTML={{ __html: item.heading }} />
                  <p className="text-[15px] text-[#5D6E82] leading-relaxed" dangerouslySetInnerHTML={{ __html: item.body }} />
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="max-w-[1440px] mx-auto px-8 py-20">
          <div className="grid md:grid-cols-2 gap-16 items-end">
            <h2 className="text-[52px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">Voir le système en fonctionnement.</h2>
            <div>
              <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-8">Un entretien technique de 45 minutes. Nous parcourrons l&apos;architecture en direct, pas sur des diapositives.</p>
              <a href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-8 py-4 hover:bg-[#c94d19] transition-colors">Demander un entretien</a>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
