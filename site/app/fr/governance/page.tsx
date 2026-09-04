import Nav from "@/components/Nav";
import Footer from "@/components/Footer";

export const metadata = { title: "Modèle de gouvernance · Kinara OS" };

const GUARANTEES = [
  "Aucun ministère ne détient l'ensemble des quatre jeux de données.",
  "Chaque jointure interdomaine est autorisée et enregistrée.",
  "Les locataires sont séparés au niveau de la base de données, pas de la requête.",
  "Kinara OS est conçu pour prendre en charge la résidence des données respectant la juridiction et le traitement transfrontalier contrôlé, conformément au droit applicable et à la politique du client.",
];

export default function FrGovernancePage() {
  return (
    <>
      <Nav lang="fr" />
      <main className="pt-[100px]">
        <section className="max-w-[1440px] mx-auto px-8 pt-16 pb-20">
          <p className="text-[11px] font-[800] uppercase tracking-[0.12em] text-[#8A98A8] mb-6">Gouvernance</p>
          <h1 className="text-[64px] md:text-[80px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-8 max-w-4xl">
            Chaque jointure interdomaine est autorisée. Chaque autorisation est enregistrée.
          </h1>
          <p className="text-[20px] text-[#5D6E82] leading-relaxed max-w-2xl">
            Kinara OS ne présuppose pas la confiance entre ministères. Il applique les conditions dans lesquelles les données peuvent franchir une frontière, le consigne et l&apos;attribue.
          </p>
        </section>

        <section className="bg-[#F5F8FB] px-8 py-20">
          <div className="max-w-[1440px] mx-auto">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-10 max-w-2xl">Politique d&apos;abord. Jointure ensuite. Journal toujours.</h2>
            <div className="space-y-4">
              {[
                { step: "01", heading: "Un service de domaine émet une requête interdomaine.", body: "Le service demandeur envoie la question à Kinara Core, pas directement à l'autre domaine. Il inclut l'identité du demandeur, la requête spécifique et la base légale déclarée." },
                { step: "02", heading: "Core vérifie la politique.", body: "La politique de gouvernance pour ce demandeur, ce type de données et cet objet est vérifiée. Si aucune politique n'autorise la jointure, la requête est refusée. Le refus est également consigné." },
                { step: "03", heading: "La jointure est exécutée et le résultat retourné.", body: "Core exécute la jointure avec le minimum de données nécessaires. Le domaine répondant retourne la réponse à Core, pas directement au demandeur." },
                { step: "04", heading: "L'entrée d'audit est consignée.", body: "Avant que la réponse ne soit retournée au demandeur, l'entrée d'audit est écrite. Elle enregistre ce qui a été demandé, ce qui a été retourné, qui a demandé et quand. Cette entrée ne peut être modifiée par aucune des deux parties." },
              ].map((item) => (
                <div key={item.step} className="flex gap-5 border border-[#C3CEDA] rounded-lg p-6 bg-white">
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

        <section className="max-w-[1440px] mx-auto px-8 py-20">
          <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95] mb-12 max-w-xl">Quatre engagements. Par écrit.</h2>
          <ol className="space-y-6 max-w-2xl">
            {GUARANTEES.map((item, i) => (
              <li key={i} className="flex items-start gap-5">
                <div className="w-[3px] bg-[#E0561C] self-stretch rounded-full flex-shrink-0 mt-1" />
                <p className="text-[22px] font-[700] tracking-[-0.02em] text-[#0F1B2D] leading-snug">{item}</p>
              </li>
            ))}
          </ol>
        </section>

        <section className="max-w-[1440px] mx-auto px-8 py-12 border-t border-[#C3CEDA]">
          <div className="grid md:grid-cols-2 gap-16 items-end">
            <h2 className="text-[48px] font-[700] tracking-[-0.04em] text-[#0F1B2D] leading-[0.95]">Examiner le modèle de gouvernance lors d&apos;un entretien.</h2>
            <div>
              <p className="text-[18px] text-[#5D6E82] leading-relaxed mb-8">Amenez vos équipes juridiques, de protection des données ou techniques. Nous parcourrons le modèle et répondrons aux questions spécifiques.</p>
              <a href="/fr/contact" className="inline-block bg-[#E0561C] text-white text-[14px] font-[700] px-8 py-4 hover:bg-[#c94d19] transition-colors">Demander un entretien</a>
            </div>
          </div>
        </section>
      </main>
      <Footer lang="fr" />
    </>
  );
}
